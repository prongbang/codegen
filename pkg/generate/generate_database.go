package generate

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/prongbang/codegen/pkg/command"
	"github.com/prongbang/codegen/pkg/filex"
	"github.com/prongbang/codegen/pkg/tools"
	"github.com/prongbang/codegen/template"
	"github.com/pterm/pterm"
)

// DatabaseGenerator scaffolds internal/database for an existing project, so a
// project created without a database can gain one later.
type DatabaseGenerator interface {
	Init(databases []string) error
}

type databaseGenerator struct {
	FileX  filex.FileX
	Cmd    command.Command
	Wire   tools.Installer
	Runner tools.Runner
}

// databaseSourceFile maps a database to the file that proves it is already set
// up in the project.
func databaseSourceFile(db string) string {
	switch db {
	case template.DatabaseMariaDB:
		return "mariadb.go"
	case template.DatabaseMongoDB:
		return "mongodb.go"
	case template.DatabaseInfluxDB3:
		return "influxdb3.go"
	}
	return ""
}

// existingDatabases reports which databases the project already scaffolds.
func existingDatabases(fx filex.FileX, rootDir string) []string {
	var found []string
	for _, db := range template.SupportedDatabases() {
		path := filepath.Join(rootDir, "internal", "database", databaseSourceFile(db))
		if fx.IsExist(path) {
			found = append(found, db)
		}
	}
	return found
}

// templateDelta renders tmpl with and without the given database and returns
// the block the database adds. Deriving the fragment from the same template the
// project generator uses keeps the two from drifting apart.
func templateDelta(tmpl string, db string) (string, error) {
	with, err := template.RenderText(tmpl, template.Project{Databases: []string{db}})
	if err != nil {
		return "", err
	}
	without, err := template.RenderText(tmpl, template.Project{})
	if err != nil {
		return "", err
	}

	a, b := string(with), string(without)

	prefix := 0
	for prefix < len(a) && prefix < len(b) && a[prefix] == b[prefix] {
		prefix++
	}

	suffix := 0
	for suffix < len(a)-prefix && suffix < len(b)-prefix &&
		a[len(a)-1-suffix] == b[len(b)-1-suffix] {
		suffix++
	}

	return a[prefix : len(a)-suffix], nil
}

func (d *databaseGenerator) Init(databases []string) error {
	rootDir, module, err := getProjectContext(d.FileX)
	if err != nil {
		return err
	}

	current := existingDatabases(d.FileX, rootDir)
	merged := mergeDatabases(current, databases)

	var added []string
	for _, db := range merged {
		if !containsString(current, db) {
			added = append(added, db)
		}
	}
	if len(added) == 0 {
		pterm.Info.Printfln("Nothing to do: %s already set up", strings.Join(databases, ", "))
		return nil
	}

	data := template.Project{Module: module, Databases: merged}

	// MariaDB brings the migration runner with it, so a project that adds the
	// database later ends up identical to one created with it.
	configs := getDatabaseConfig(rootDir, data)
	for _, config := range getMigrationConfig(rootDir, data) {
		// Never overwrite a runner the project has since edited, nor a baseline
		// migration it may have replaced.
		if d.FileX.IsExist(config.Path) {
			continue
		}
		configs = append(configs, config)
	}

	spinner, _ := pterm.DefaultSpinner.Start(fmt.Sprintf("Initialize database %s", strings.Join(added, ", ")))
	for _, config := range configs {
		dir := filepath.Dir(config.Path)
		if err := d.FileX.EnsureDir(dir); err != nil {
			spinner.Fail(err)
			return err
		}
		if err := WriteFile(d.FileX, config.Path, config.Template, config.Data); err != nil {
			spinner.Fail(err)
			return err
		}
	}
	spinner.Success()

	if err := d.installDatabaseModules(rootDir, added); err != nil {
		return err
	}

	if err := bindDatabaseConfiguration(d.FileX, rootDir, added); err != nil {
		return err
	}

	if err := bindDatabaseWiring(d.FileX, rootDir, module); err != nil {
		return err
	}

	if data.HasMariaDB() {
		if err := bindStartupMigrations(d.FileX, rootDir); err != nil {
			return err
		}
	}

	// go mod tidy + wire
	if err := d.FileX.Chdir(rootDir); err != nil {
		return err
	}
	if err := d.Wire.Install(); err != nil {
		return err
	}
	return d.Runner.Run()
}

// databaseModules returns the "module@version" requirements a database adds,
// derived from the same go.mod template the project generator renders so the
// two cannot drift apart.
func databaseModules(added []string) ([]string, error) {
	var modules []string
	for _, db := range added {
		fragment, err := templateDelta(template.ModTemplate, db)
		if err != nil {
			return nil, err
		}
		for _, line := range strings.Split(fragment, "\n") {
			line = strings.TrimSpace(line)
			if idx := strings.Index(line, "//"); idx >= 0 {
				line = strings.TrimSpace(line[:idx])
			}
			name, version, ok := strings.Cut(line, " ")
			if !ok || name == "" || version == "" {
				continue
			}
			modules = append(modules, name+"@"+version)
		}
	}
	return modules, nil
}

// installDatabaseModules resolves the database dependencies with "go get".
// Editing go.mod directly is not enough: go mod tidy will not upgrade a
// requirement that is already pinned, so adding a database to a project that
// already has one can leave two modules providing the same package (the
// pre-split google.golang.org/genproto is the usual culprit). "go get" performs
// the upgrade and records the checksums.
func (d *databaseGenerator) installDatabaseModules(rootDir string, added []string) error {
	modules, err := databaseModules(added)
	if err != nil {
		return err
	}
	if len(modules) == 0 {
		return nil
	}
	if err := d.FileX.Chdir(rootDir); err != nil {
		return err
	}

	spinner, _ := pterm.DefaultSpinner.Start("Resolving database dependencies")
	for _, module := range modules {
		if _, err := d.Cmd.Run("go", "get", module); err != nil {
			spinner.Fail(err)
			return fmt.Errorf("go get %s: %w", module, err)
		}
	}

	// Adding a database to a project that already has one can leave a stale
	// requirement that a fresh resolve would never pick, and two modules then
	// provide the same package. Upgrading the older module clears it.
	if out, err := d.Cmd.Run("go", "mod", "tidy"); err != nil {
		stale := ambiguousModule(out)
		if stale == "" {
			spinner.Fail(err)
			return fmt.Errorf("go mod tidy: %w", err)
		}
		if _, err := d.Cmd.Run("go", "get", stale+"@latest"); err != nil {
			spinner.Fail(err)
			return fmt.Errorf("go get %s@latest: %w", stale, err)
		}
		pterm.Info.Printfln("Upgraded %s to resolve an ambiguous import", stale)
	}

	spinner.Success()
	return nil
}

// ambiguousModule extracts the outdated module from a "go mod tidy" failure of
// the form:
//
//	ambiguous import: found package P in multiple modules:
//		example.com/old v0.0.0-... (/path)
//		example.com/old/sub v0.0.0-... (/path)
//
// The first module listed is the one that still bundles the package after it
// was split out, so upgrading it is what resolves the clash.
func ambiguousModule(output string) string {
	lines := strings.Split(output, "\n")
	for i, line := range lines {
		if !strings.Contains(line, "ambiguous import:") || i+1 >= len(lines) {
			continue
		}
		candidate := strings.TrimSpace(lines[i+1])
		if name, _, ok := strings.Cut(candidate, " "); ok && strings.Contains(name, ".") {
			return name
		}
	}
	return ""
}

// bindDatabaseConfiguration appends the configuration struct fields and the
// development/production yml blocks for the newly added databases.
func bindDatabaseConfiguration(fx filex.FileX, rootDir string, added []string) error {
	type target struct {
		path   string
		tmpl   string
		marker string
	}
	targets := []target{
		{filepath.Join(rootDir, "configuration", "configuration.go"), template.ConfigurationTemplate, "//+codegen:struct configuration"},
		{filepath.Join(rootDir, "configuration", "development.yml"), template.ConfigurationDevelopmentTemplate, ""},
		{filepath.Join(rootDir, "configuration", "production.yml"), template.ConfigurationProductionTemplate, ""},
	}

	for _, t := range targets {
		if !fx.IsExist(t.path) {
			continue
		}
		text := fx.ReadFile("/" + t.path)

		for _, db := range added {
			fragment, err := templateDelta(t.tmpl, db)
			if err != nil {
				return err
			}
			// The delta loses its surrounding newlines to the common prefix and
			// suffix, so re-attach them when splicing the block back in.
			block := strings.Trim(fragment, "\n")
			if block == "" || strings.Contains(text, block) {
				continue
			}

			switch {
			case t.marker == "":
				text = strings.TrimRight(text, "\n") + "\n\n" + block + "\n"
			case strings.Contains(text, t.marker):
				text = strings.Replace(text, t.marker, t.marker+"\n"+block, 1)
			default:
				pterm.Warning.Printfln("%s has no %s marker; add the %s section manually",
					filepath.Base(t.path), t.marker, db)
			}
		}

		spinner, _ := pterm.DefaultSpinner.Start(fmt.Sprintf("Binding file %s", filepath.Base(t.path)))
		if err := fx.WriteFile("/"+t.path, []byte(text)); err != nil {
			spinner.Fail(err)
			return err
		}
		spinner.Success()
	}
	return nil
}

// bindDatabaseWiring gives CreateApp its database.Drivers parameter and makes
// cmd/api/main.go build and close the driver. It is a no-op for projects that
// already take the driver.
func bindDatabaseWiring(fx filex.FileX, rootDir string, module string) error {
	databaseImport := fmt.Sprintf(`"%s/internal/database"`, module)

	for _, name := range []string{"wire.go", "wire_gen.go"} {
		path := filepath.Join(rootDir, name)
		if !fx.IsExist(path) {
			continue
		}
		text := fx.ReadFile("/" + path)
		if strings.Contains(text, "CreateApp(dbDriver database.Drivers)") {
			continue
		}
		if !strings.Contains(text, "func CreateApp() app.App {") {
			pterm.Warning.Printfln("%s has an unexpected CreateApp signature; add database.Drivers manually", name)
			continue
		}

		text = strings.Replace(text,
			"func CreateApp() app.App {",
			"func CreateApp(dbDriver database.Drivers) app.App {", 1)
		if !strings.Contains(text, databaseImport) {
			text = replaceFirstMarker(text, wireImportMarkers(), func(marker string) string {
				return fmt.Sprintf("%s\n\t%s", databaseImport, marker)
			})
		}

		spinner, _ := pterm.DefaultSpinner.Start(fmt.Sprintf("Binding file %s", name))
		if err := fx.WriteFile("/"+path, []byte(text)); err != nil {
			spinner.Fail(err)
			return err
		}
		spinner.Success()
	}

	mainPath := filepath.Join(rootDir, "cmd", "api", "main.go")
	if !fx.IsExist(mainPath) {
		return nil
	}
	text := fx.ReadFile("/" + mainPath)
	if strings.Contains(text, "database.NewDatabaseDriver()") {
		return nil
	}

	if !strings.Contains(text, "CreateApp()") {
		pterm.Warning.Printfln("cmd/api/main.go has an unexpected CreateApp call; pass the database driver manually")
		return nil
	}
	text = strings.Replace(text, "CreateApp()", "CreateApp(dbDriver)", 1)
	text = replaceFirstMarker(text, mainDatabaseMarkers(), func(marker string) string {
		return fmt.Sprintf("dbDriver := database.NewDatabaseDriver()\n\t\t\tdefer dbDriver.Close()\n\n\t\t\t%s", marker)
	})
	if !strings.Contains(text, databaseImport) {
		text = replaceFirstMarker(text, mainImportMarkers(), func(marker string) string {
			return fmt.Sprintf("%s\n\t%s", databaseImport, marker)
		})
	}

	spinner, _ := pterm.DefaultSpinner.Start("Binding file main.go")
	if err := fx.WriteFile("/"+mainPath, []byte(text)); err != nil {
		spinner.Fail(err)
		return err
	}
	spinner.Success()
	return nil
}

func mainImportMarkers() []string {
	return []string{"//+codegen:import main:package", "// +codegen:import main:package"}
}

func mainDatabaseMarkers() []string {
	return []string{"//+codegen:func main:database", "// +codegen:func main:database"}
}

func containsString(list []string, value string) bool {
	for _, item := range list {
		if item == value {
			return true
		}
	}
	return false
}

// mergeDatabases keeps the project's existing databases and adds the requested
// ones, preserving the canonical order.
func mergeDatabases(current []string, requested []string) []string {
	var merged []string
	for _, db := range template.SupportedDatabases() {
		if containsString(current, db) || containsString(requested, db) {
			merged = append(merged, db)
		}
	}
	return merged
}

func NewDatabaseGenerator(fileX filex.FileX, cmd command.Command, wire tools.Installer, runner tools.Runner) DatabaseGenerator {
	return &databaseGenerator{
		FileX:  fileX,
		Cmd:    cmd,
		Wire:   wire,
		Runner: runner,
	}
}
