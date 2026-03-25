package mod

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/prongbang/codegen/pkg/filex"
)

func TestModNewAppPath(t *testing.T) {
	if got := (Mod{AppPath: "internal/app"}).NewAppPath(); got != "internal" {
		t.Fatalf("unexpected new app path: %s", got)
	}
	if got := (Mod{AppPath: "internal/demo"}).NewAppPath(); got != "internal/demo" {
		t.Fatalf("unexpected custom app path: %s", got)
	}
}

func TestGetModule(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "internal", "app", "api"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module github.com/acme/demo\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(filepath.Join(root, "internal", "app", "api")); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(cwd) })

	got := GetModule(filex.NewFileX())
	if got.Module != "github.com/acme/demo" || got.AppPath != "internal/app" || got.Name != "demo" {
		t.Fatalf("unexpected mod: %+v", got)
	}
}
