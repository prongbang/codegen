package creator

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/prongbang/codegen/pkg/filex"
	"github.com/prongbang/codegen/pkg/mod"
	"github.com/prongbang/codegen/pkg/option"
)

type mockFileX struct {
	getwd    string
	getwdErr error
	ensureErr error
	writeErr error
	writes   map[string][]byte
}

func (m *mockFileX) EnsureDir(string) error                { return m.ensureErr }
func (m *mockFileX) ReadFile(string) string                { return "" }
func (m *mockFileX) Getwd() (string, error)                { return m.getwd, m.getwdErr }
func (m *mockFileX) Chdir(string) error                    { return nil }
func (m *mockFileX) IsExist(string) bool                   { return false }
func (m *mockFileX) IsDirExist(string) bool                { return false }
func (m *mockFileX) WriteFile(name string, data []byte) error {
	if m.writeErr != nil {
		return m.writeErr
	}
	if m.writes == nil {
		m.writes = map[string][]byte{}
	}
	m.writes[name] = data
	return nil
}

func TestCreatorCreateWritesFile(t *testing.T) {
	root := t.TempDir()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(cwd) })

	cr := New(filex.NewFileX())
	err = cr.Create(Config{
		Pkg: option.Package{
			Name:   "device_service",
			Module: mod.Mod{},
		},
		Filename: "server.go",
		Template: []byte("package demo"),
	})
	if err != nil {
		t.Fatal(err)
	}

	target := filepath.Join(root, "deviceservice", "server.go")
	data, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "package demo" {
		t.Fatalf("unexpected file content: %s", string(data))
	}
}

func TestCreatorCreateGetwdError(t *testing.T) {
	want := errors.New("getwd failed")
	cr := New(&mockFileX{getwdErr: want})
	err := cr.Create(Config{Pkg: option.Package{Name: "demo"}, Filename: "a.go", Template: []byte("x")})
	if !errors.Is(err, want) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCreatorCreateEnsureDirError(t *testing.T) {
	want := errors.New("ensure failed")
	cr := New(&mockFileX{getwd: "/tmp", ensureErr: want})
	err := cr.Create(Config{Pkg: option.Package{Name: "demo"}, Filename: "a.go", Template: []byte("x")})
	if !errors.Is(err, want) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCreatorCreateWriteError(t *testing.T) {
	want := errors.New("write failed")
	cr := New(&mockFileX{getwd: "/tmp", writeErr: want})
	err := cr.Create(Config{Pkg: option.Package{Name: "demo"}, Filename: "a.go", Template: []byte("x")})
	if !errors.Is(err, want) {
		t.Fatalf("unexpected error: %v", err)
	}
}
