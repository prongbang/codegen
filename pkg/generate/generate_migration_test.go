package generate

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/prongbang/codegen/pkg/filex"
	"github.com/prongbang/codegen/template"
)

func TestGetMigrationConfigNeedsMariaDB(t *testing.T) {
	if got := getMigrationConfig("/tmp/demo", template.Project{}); got != nil {
		t.Fatalf("expected no migration files without a database, got %v", got)
	}
	if got := getMigrationConfig("/tmp/demo", template.Project{Databases: []string{template.DatabaseMongoDB}}); got != nil {
		t.Fatalf("expected no migration files for mongodb only, got %v", got)
	}

	configs := getMigrationConfig("/tmp/demo", template.Project{
		Module:    "github.com/acme/demo",
		Databases: []string{template.DatabaseMariaDB},
	})
	want := []string{
		"/tmp/demo/internal/database/migrations.go",
		"/tmp/demo/migrations/embed.go",
		"/tmp/demo/migrations/README.md",
		"/tmp/demo/migrations/sql/0001_init.up.sql",
	}
	if len(configs) != len(want) {
		t.Fatalf("expected %d files, got %d", len(want), len(configs))
	}
	for i, path := range want {
		if configs[i].Path != path {
			t.Fatalf("file %d: expected %s, got %s", i, path, configs[i].Path)
		}
	}
}

// The runner is the one generated file that must compile, and it imports the
// project's own migrations package, so the module has to reach the template.
func TestMigrationsRunnerTemplateIsValidGo(t *testing.T) {
	data := template.Project{Module: "github.com/acme/demo", Databases: []string{template.DatabaseMariaDB}}

	for _, tmpl := range []string{template.DatabaseMigrationsTemplate, template.MigrationsEmbedTemplate} {
		src := mustRender(t, tmpl, data)
		if _, err := parser.ParseFile(token.NewFileSet(), "generated.go", src, parser.AllErrors); err != nil {
			t.Fatalf("generated file does not parse: %v\n%s", err, src)
		}
	}

	src := string(mustRender(t, template.DatabaseMigrationsTemplate, data))
	if !strings.Contains(src, `migrationsrc "github.com/acme/demo/migrations"`) {
		t.Fatalf("runner does not import the project's migrations package:\n%s", src)
	}
	if !strings.Contains(src, "WithMarkAppliedOnSuccess(true)") {
		t.Fatal("runner must record a migration only after its SQL succeeds")
	}
}

func TestNextMigrationVersion(t *testing.T) {
	cases := []struct {
		name  string
		files []string
		want  string
	}{
		{"empty directory", nil, "0001"},
		{"after the baseline", []string{"0001_init.up.sql"}, "0002"},
		{"highest wins, not the count", []string{"0001_a.up.sql", "0007_b.up.sql", "0003_c.up.sql"}, "0008"},
		{"down files do not double-count", []string{"0002_a.up.sql", "0002_a.down.sql"}, "0003"},
		{"non-migrations are ignored", []string{"README.md", "0001_a.up.sql"}, "0002"},
		{"keeps the width already in use", []string{"000001_a.up.sql"}, "000002"},
		{"widens when the sequence overflows", []string{"9999_a.up.sql"}, "10000"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			for _, name := range tc.files {
				writeFile(t, filepath.Join(dir, name), "SELECT 1;\n")
			}
			entries, err := os.ReadDir(dir)
			if err != nil {
				t.Fatal(err)
			}
			if got := nextMigrationVersion(entries); got != tc.want {
				t.Fatalf("expected %s, got %s", tc.want, got)
			}
		})
	}
}

func TestMigrationInitScaffoldsAndBindsMain(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "go.mod"), "module github.com/acme/demo\n")
	writeFile(t, filepath.Join(root, "internal", "database", "mariadb.go"), "package database\n")
	writeFile(t, filepath.Join(root, "cmd", "api", "main.go"), `package main

func main() {
	dbDriver := database.NewDatabaseDriver()
	defer dbDriver.Close()

	//+codegen:func main:database

	apps := demo.CreateApp(dbDriver)
	apps.StartAPI()
}
`)
	chdir(t, root)

	if err := NewMigrationGenerator(filex.NewFileX()).Init(); err != nil {
		t.Fatal(err)
	}

	for _, name := range []string{
		filepath.Join("internal", "database", "migrations.go"),
		filepath.Join("migrations", "embed.go"),
		filepath.Join("migrations", "README.md"),
		filepath.Join("migrations", "sql", "0001_init.up.sql"),
	} {
		if _, err := os.Stat(filepath.Join(root, name)); err != nil {
			t.Fatalf("expected %s to be generated: %v", name, err)
		}
	}

	main := readFile(t, filepath.Join(root, "cmd", "api", "main.go"))
	if !strings.Contains(main, "database.StartupMigrations(dbDriver)") {
		t.Fatalf("main.go was not bound:\n%s", main)
	}
	// The call has to land after the driver exists and before the app starts.
	if strings.Index(main, "StartupMigrations") < strings.Index(main, "defer dbDriver.Close()") ||
		strings.Index(main, "StartupMigrations") > strings.Index(main, "CreateApp") {
		t.Fatalf("StartupMigrations is in the wrong place:\n%s", main)
	}

	// Re-running must not duplicate the call or clobber an edited runner.
	writeFile(t, filepath.Join(root, "migrations", "sql", "0001_init.up.sql"), "-- edited\n")
	if err := NewMigrationGenerator(filex.NewFileX()).Init(); err != nil {
		t.Fatal(err)
	}
	if got := strings.Count(readFile(t, filepath.Join(root, "cmd", "api", "main.go")), "StartupMigrations"); got != 1 {
		t.Fatalf("expected a single StartupMigrations call, got %d", got)
	}
	if got := readFile(t, filepath.Join(root, "migrations", "sql", "0001_init.up.sql")); got != "-- edited\n" {
		t.Fatalf("re-run overwrote an existing migration: %q", got)
	}
}

// A project generated before the marker existed still has the driver, so the
// binder falls back to it rather than giving up.
func TestMigrationInitBindsMainWithoutMarker(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "go.mod"), "module github.com/acme/demo\n")
	writeFile(t, filepath.Join(root, "internal", "database", "mariadb.go"), "package database\n")
	writeFile(t, filepath.Join(root, "cmd", "api", "main.go"), `package main

func main() {
	dbDriver := database.NewDatabaseDriver()
	defer dbDriver.Close()

	apps := demo.CreateApp(dbDriver)
	apps.StartAPI()
}
`)
	chdir(t, root)

	if err := NewMigrationGenerator(filex.NewFileX()).Init(); err != nil {
		t.Fatal(err)
	}

	main := readFile(t, filepath.Join(root, "cmd", "api", "main.go"))
	if !strings.Contains(main, "database.StartupMigrations(dbDriver)") {
		t.Fatalf("main.go was not bound:\n%s", main)
	}
	if strings.Index(main, "StartupMigrations") > strings.Index(main, "CreateApp") {
		t.Fatalf("StartupMigrations must run before the app starts:\n%s", main)
	}
}

func TestMigrationInitRequiresMariaDB(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "go.mod"), "module github.com/acme/demo\n")
	chdir(t, root)

	err := NewMigrationGenerator(filex.NewFileX()).Init()
	if err == nil || !strings.Contains(err.Error(), "MariaDB") {
		t.Fatalf("expected a MariaDB requirement error, got %v", err)
	}
}

func TestMigrationNew(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "go.mod"), "module github.com/acme/demo\n")
	writeFile(t, filepath.Join(root, "migrations", "sql", "0001_init.up.sql"), "SELECT 1;\n")
	chdir(t, root)

	generator := NewMigrationGenerator(filex.NewFileX())
	if err := generator.New("Master Room Setup"); err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(root, "migrations", "sql", "0002_master_room_setup.up.sql")
	body := readFile(t, path)
	if !strings.Contains(body, "master_room_setup") || !strings.Contains(body, "0002") {
		t.Fatalf("skeleton was not filled in:\n%s", body)
	}
	// bun aborts the whole run on a name it cannot parse, so the generated one
	// must match the pattern it accepts.
	if !migrationNameRE.MatchString("master_room_setup") {
		t.Fatal("generated name is not one bun accepts")
	}

	// The version advances even when the name repeats, so nothing is clobbered.
	if err := generator.New("Master Room Setup"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "migrations", "sql", "0003_master_room_setup.up.sql")); err != nil {
		t.Fatalf("expected the repeat to take the next version: %v", err)
	}
	if got := readFile(t, path); got != body {
		t.Fatal("the repeat overwrote the earlier migration")
	}
}

func TestMigrationNewRejectsUnusableNames(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "go.mod"), "module github.com/acme/demo\n")
	writeFile(t, filepath.Join(root, "migrations", "sql", "0001_init.up.sql"), "SELECT 1;\n")
	chdir(t, root)

	if err := NewMigrationGenerator(filex.NewFileX()).New("../../etc/passwd"); err == nil {
		t.Fatal("expected an invalid name to be rejected")
	}
}

func TestMigrationNewWithoutInit(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "go.mod"), "module github.com/acme/demo\n")
	chdir(t, root)

	err := NewMigrationGenerator(filex.NewFileX()).New("master_room_setup")
	if err == nil || !strings.Contains(err.Error(), "migration init") {
		t.Fatalf("expected a pointer to \"codegen migration init\", got %v", err)
	}
}
