package loader

import (
	"bufio"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

type TypeSpecRef struct {
	Pkg  *Package
	File *ast.File
	Spec *ast.TypeSpec
}

type Package struct {
	Dir        string
	Name       string
	ImportPath string
	Files      []*ast.File
}

type Module struct {
	RootDir    string
	ModulePath string
	Fset       *token.FileSet
	Packages   map[string]*Package
	Types      map[string]*TypeSpecRef
}

func typeKey(importPath, name string) string {
	return importPath + ":" + name
}

func (m *Module) FindPackage(importPath string) *Package {
	return m.Packages[importPath]
}

func (m *Module) FindType(importPath, name string) *TypeSpecRef {
	return m.Types[typeKey(importPath, name)]
}

func Load(patterns []string) (*Module, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	rootDir, modulePath, err := findModuleRoot(cwd)
	if err != nil {
		return nil, err
	}

	dirs, err := collectDirs(rootDir, cwd, patterns)
	if err != nil {
		return nil, err
	}

	mod := &Module{
		RootDir:    rootDir,
		ModulePath: modulePath,
		Fset:       token.NewFileSet(),
		Packages:   map[string]*Package{},
		Types:      map[string]*TypeSpecRef{},
	}

	for _, dir := range dirs {
		files, err := parseDir(mod.Fset, dir)
		if err != nil {
			return nil, err
		}
		if len(files) == 0 {
			continue
		}
		name := files[0].Name.Name
		importPath, err := moduleImportPath(rootDir, modulePath, dir)
		if err != nil {
			return nil, err
		}
		pkg := &Package{
			Dir:        dir,
			Name:       name,
			ImportPath: importPath,
			Files:      files,
		}
		mod.Packages[pkg.ImportPath] = pkg

		for _, file := range files {
			for _, decl := range file.Decls {
				gen, ok := decl.(*ast.GenDecl)
				if !ok || gen.Tok != token.TYPE {
					continue
				}
				for _, spec := range gen.Specs {
					typeSpec, ok := spec.(*ast.TypeSpec)
					if !ok {
						continue
					}
					mod.Types[typeKey(pkg.ImportPath, typeSpec.Name.Name)] = &TypeSpecRef{
						Pkg:  pkg,
						File: file,
						Spec: typeSpec,
					}
				}
			}
		}
	}

	return mod, nil
}

func findModuleRoot(start string) (string, string, error) {
	dir := start
	for {
		goMod := filepath.Join(dir, "go.mod")
		if stat, err := os.Stat(goMod); err == nil && !stat.IsDir() {
			modulePath, err := readModulePath(goMod)
			if err != nil {
				return "", "", err
			}
			return dir, modulePath, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", "", fmt.Errorf("go.mod not found")
		}
		dir = parent
	}
}

func readModulePath(goMod string) (string, error) {
	file, err := os.Open(goMod)
	if err != nil {
		return "", err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(line, "module ") {
			continue
		}
		modulePath := strings.TrimSpace(strings.TrimPrefix(line, "module "))
		if modulePath != "" {
			return modulePath, nil
		}
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}
	return "", fmt.Errorf("module path not found in %s", goMod)
}

func collectDirs(rootDir, cwd string, patterns []string) ([]string, error) {
	if len(patterns) == 0 {
		patterns = []string{"./..."}
	}

	seen := map[string]bool{}
	dirs := []string{}
	for _, pattern := range patterns {
		if pattern == "" {
			continue
		}
		if pattern == "./..." || pattern == "." {
			matched, err := walkDirs(rootDir)
			if err != nil {
				return nil, err
			}
			for _, dir := range matched {
				if !seen[dir] {
					seen[dir] = true
					dirs = append(dirs, dir)
				}
			}
			continue
		}
		if strings.HasSuffix(pattern, "/...") {
			base := strings.TrimSuffix(pattern, "/...")
			abs, err := filepath.Abs(filepath.Join(cwd, base))
			if err != nil {
				return nil, err
			}
			matched, err := walkDirs(abs)
			if err != nil {
				return nil, err
			}
			for _, dir := range matched {
				if !seen[dir] {
					seen[dir] = true
					dirs = append(dirs, dir)
				}
			}
			continue
		}

		abs, err := filepath.Abs(filepath.Join(cwd, pattern))
		if err != nil {
			return nil, err
		}
		info, err := os.Stat(abs)
		if err != nil {
			return nil, err
		}
		dir := abs
		if !info.IsDir() {
			dir = filepath.Dir(abs)
		}
		if !seen[dir] {
			seen[dir] = true
			dirs = append(dirs, dir)
		}
	}

	return dirs, nil
}

func walkDirs(root string) ([]string, error) {
	dirs := []string{}
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			name := d.Name()
			if name == ".git" || name == "vendor" || strings.HasPrefix(name, ".") {
				if path == root {
					return nil
				}
				return filepath.SkipDir
			}
			hasGo, err := dirHasGoFiles(path)
			if err != nil {
				return err
			}
			if hasGo {
				dirs = append(dirs, path)
			}
		}
		return nil
	})
	return dirs, err
}

func dirHasGoFiles(dir string) (bool, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false, err
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasSuffix(name, ".go") && !strings.HasSuffix(name, "_test.go") {
			return true, nil
		}
	}
	return false, nil
}

func parseDir(fset *token.FileSet, dir string) ([]*ast.File, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	files := []*ast.File{}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		path := filepath.Join(dir, name)
		file, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
		if err != nil {
			return nil, err
		}
		files = append(files, file)
	}
	return files, nil
}

func moduleImportPath(rootDir, modulePath, dir string) (string, error) {
	rel, err := filepath.Rel(rootDir, dir)
	if err != nil {
		return "", err
	}
	if rel == "." {
		return modulePath, nil
	}
	return modulePath + "/" + filepath.ToSlash(rel), nil
}
