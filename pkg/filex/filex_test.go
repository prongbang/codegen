//go:build !test
// +build !test

package filex

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFileXOperations(t *testing.T) {
	fx := NewFileX()
	root := t.TempDir()

	dir := filepath.Join(root, "nested", "dir")
	if err := fx.EnsureDir(dir); err != nil {
		t.Fatal(err)
	}
	if !fx.IsDirExist(dir) {
		t.Fatalf("expected dir to exist: %s", dir)
	}

	file := filepath.Join(dir, "hello.txt")
	if err := fx.WriteFile(file, []byte("hello")); err != nil {
		t.Fatal(err)
	}
	if !fx.IsExist(file) {
		t.Fatalf("expected file to exist: %s", file)
	}
	if got := fx.ReadFile(file); got != "hello" {
		t.Fatalf("unexpected file contents: %q", got)
	}
	if got := fx.ReadFile(filepath.Join(root, "missing.txt")); got != "" {
		t.Fatalf("expected empty read for missing file, got %q", got)
	}

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(cwd) })

	if err := fx.Chdir(root); err != nil {
		t.Fatal(err)
	}
	gotwd, err := fx.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	resolvedRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatal(err)
	}
	if gotwd != resolvedRoot {
		t.Fatalf("unexpected working dir: %s", gotwd)
	}
}
