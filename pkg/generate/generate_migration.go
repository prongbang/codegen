package generate

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/ettle/strcase"
	"github.com/prongbang/codegen/pkg/filex"
	"github.com/prongbang/codegen/template"
	"github.com/pterm/pterm"
)

// MigrationGenerator scaffolds the startup migration runner for an existing
// project and creates new migration files inside it.
type MigrationGenerator interface {
	Init() error
	New(name string) error
}

type migrationGenerator struct {
	FileX filex.FileX
}

// migrationNameRE mirrors the pattern bun's migrate.Discover accepts. A file it
// rejects aborts the whole run, so the name is checked before the file lands.
var migrationNameRE = regexp.MustCompile(`^[0-9a-z_\-]+$`)

// getMigrationConfig returns the migration files for a project with MariaDB.
// Shared by project creation and "codegen migration init" so both produce the
// same runner.
func getMigrationConfig(rootDir string, data template.Project) []FileConfig {
	// The runner applies its files over the bun MariaDB connection, so there is
	// nothing to generate without it.
	if !data.HasMariaDB() {
		return nil
	}

	return []FileConfig{
		{
			Path:     fmt.Sprintf("%s/internal/database/migrations.go", rootDir),
			Template: template.DatabaseMigrationsTemplate,
			Data:     data,
		},
		{
			Path:     fmt.Sprintf("%s/migrations/embed.go", rootDir),
			Template: template.MigrationsEmbedTemplate,
			Data:     data,
		},
		{
			Path:     fmt.Sprintf("%s/migrations/README.md", rootDir),
			Template: template.MigrationsReadmeTemplate,
			Data:     data,
		},
		{
			// //go:embed sql/*.sql fails to build when the glob matches nothing,
			// so the directory ships with a no-op migration.
			Path:     fmt.Sprintf("%s/migrations/sql/0001_init.up.sql", rootDir),
			Template: template.MigrationsInitSQLTemplate,
			Data:     data,
		},
	}
}

func (m *migrationGenerator) Init() error {
	rootDir, module, err := getProjectContext(m.FileX)
	if err != nil {
		return err
	}

	if !m.FileX.IsExist(filepath.Join(rootDir, "internal", "database", "mariadb.go")) {
		return fmt.Errorf("migrations run over the MariaDB connection, add it first with \"codegen database init -d mariadb\"")
	}

	data := template.Project{Module: module, Databases: []string{template.DatabaseMariaDB}}

	var pending []FileConfig
	for _, config := range getMigrationConfig(rootDir, data) {
		// Re-running must not overwrite a runner the project has since edited,
		// nor the baseline file it may have replaced.
		if m.FileX.IsExist(config.Path) {
			continue
		}
		pending = append(pending, config)
	}

	if len(pending) > 0 {
		spinner, _ := pterm.DefaultSpinner.Start("Initialize migrations")
		for _, config := range pending {
			if err := m.FileX.EnsureDir(filepath.Dir(config.Path)); err != nil {
				spinner.Fail(err)
				return err
			}
			if err := WriteFile(m.FileX, config.Path, config.Template, config.Data); err != nil {
				spinner.Fail(err)
				return err
			}
		}
		spinner.Success()
	} else {
		pterm.Info.Println("Nothing to do: migrations already set up")
	}

	return bindStartupMigrations(m.FileX, rootDir)
}

func (m *migrationGenerator) New(name string) error {
	rootDir, _, err := getProjectContext(m.FileX)
	if err != nil {
		return err
	}

	sqlDir := filepath.Join(rootDir, "migrations", "sql")
	if !m.FileX.IsDirExist(sqlDir) {
		return fmt.Errorf("%s not found, run \"codegen migration init\" first", filepath.Join("migrations", "sql"))
	}

	fileName := strcase.ToSnake(strings.ReplaceAll(name, " ", "_"))
	if !migrationNameRE.MatchString(fileName) {
		return fmt.Errorf("invalid migration name %q, expected lowercase letters, digits, %q and %q", name, "_", "-")
	}

	entries, err := os.ReadDir(sqlDir)
	if err != nil {
		return err
	}
	version := nextMigrationVersion(entries)

	path := filepath.Join(sqlDir, fmt.Sprintf("%s_%s.up.sql", version, fileName))
	if m.FileX.IsExist(path) {
		return fmt.Errorf("%s already exists", path)
	}

	data := template.Any{"Version": version, "Name": fileName}

	spinner, _ := pterm.DefaultSpinner.Start(fmt.Sprintf("Create migration %s_%s.up.sql", version, fileName))
	if err := WriteFile(m.FileX, path, template.MigrationsSQLTemplate, data); err != nil {
		spinner.Fail(err)
		return err
	}
	spinner.Success()

	pterm.Info.Printfln("Edit %s, then rebuild — the runner only sees files compiled into the binary",
		filepath.Join("migrations", "sql", fmt.Sprintf("%s_%s.up.sql", version, fileName)))
	return nil
}

// versionPrefixRE captures the leading version of a migration filename. bun
// accepts up to 14 digits; anything else in the directory is not a migration.
var versionPrefixRE = regexp.MustCompile(`^(\d{1,14})_`)

// nextMigrationVersion returns the highest version present plus one, zero
// padded to the width the directory already uses so the ascending filename
// order stays the run order.
func nextMigrationVersion(entries []os.DirEntry) string {
	highest := 0
	width := 4

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		matches := versionPrefixRE.FindStringSubmatch(entry.Name())
		if matches == nil {
			continue
		}
		value, err := strconv.Atoi(matches[1])
		if err != nil {
			continue
		}
		if value > highest {
			highest = value
		}
		if len(matches[1]) > width {
			width = len(matches[1])
		}
	}

	return fmt.Sprintf("%0*d", width, highest+1)
}

// bindStartupMigrations makes cmd/api/main.go run the migrations right after
// the database driver connects. It is a no-op for a project that already does.
func bindStartupMigrations(fx filex.FileX, rootDir string) error {
	mainPath := filepath.Join(rootDir, "cmd", "api", "main.go")
	if !fx.IsExist(mainPath) {
		return nil
	}

	text := fx.ReadFile(mainPath)
	if strings.Contains(text, "StartupMigrations") {
		return nil
	}

	const call = "// Database migrations (always on; RUN_MIGRATIONS=false opts out).\n\t\t\tdatabase.StartupMigrations(dbDriver)"

	updated := replaceFirstMarker(text, mainDatabaseMarkers(), func(marker string) string {
		return fmt.Sprintf("%s\n\n\t\t\t%s", call, marker)
	})
	if updated == text {
		// A project generated before the marker existed still has the driver, so
		// anchor on it rather than giving up.
		const anchor = "defer dbDriver.Close()"
		if !strings.Contains(text, anchor) {
			pterm.Warning.Println("cmd/api/main.go has no database driver; call database.StartupMigrations(dbDriver) manually")
			return nil
		}
		updated = strings.Replace(text, anchor, anchor+"\n\n\t\t\t"+call, 1)
	}

	spinner, _ := pterm.DefaultSpinner.Start("Binding file main.go")
	if err := fx.WriteFile(mainPath, []byte(updated)); err != nil {
		spinner.Fail(err)
		return err
	}
	spinner.Success()
	return nil
}

func NewMigrationGenerator(fileX filex.FileX) MigrationGenerator {
	return &migrationGenerator{FileX: fileX}
}
