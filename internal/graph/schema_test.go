package graph

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/prongbang/codegen/internal/analyzer"
	"github.com/prongbang/codegen/internal/loader"
)

func writeFile(t *testing.T, name, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(name), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(name, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func chdir(t *testing.T, dir string) {
	t.Helper()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(cwd)
	})
}

func TestBuilderHandlesAliasEmbeddedInlineStructAndIgnoreTags(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)
	writeFile(t, filepath.Join(dir, "go.mod"), "module github.com/acme/demo\n\ngo 1.23.4\n")
	writeFile(t, filepath.Join(dir, "model", "types.go"), `package model

type UUID string

type Base struct {
	ID UUID `+"`json:\"id\"`"+`
}

type Payload struct {
	Base
	Hidden string `+"`json:\"hidden\" swaggerignore:\"true\"`"+`
	Meta struct {
		Version int `+"`json:\"version\"`"+`
	} `+"`json:\"meta\"`"+`
	Labels map[string]int `+"`json:\"labels,omitempty\"`"+`
}
`)

	mod, err := loader.Load([]string{"./..."})
	if err != nil {
		t.Fatal(err)
	}
	ref := &analyzer.TypeRef{Kind: analyzer.TypeNamed, Package: "github.com/acme/demo/model", Name: "Payload"}
	builder := NewBuilder(mod)
	schema := builder.Build(ref)
	if schema.Ref == "" {
		t.Fatalf("expected ref schema: %+v", schema)
	}
	component := builder.Components["model_Payload"]
	if component == nil {
		t.Fatal("expected model_Payload component")
	}
	if _, ok := component.Properties["id"]; !ok {
		t.Fatalf("expected embedded id property: %+v", component.Properties)
	}
	if _, ok := component.Properties["hidden"]; ok {
		t.Fatalf("expected hidden property to be ignored: %+v", component.Properties)
	}
	meta := component.Properties["meta"]
	if meta == nil || meta.Type != "object" {
		t.Fatalf("expected inline meta object: %+v", meta)
	}
	if meta.Properties["version"] == nil {
		t.Fatalf("expected meta.version property: %+v", meta.Properties)
	}
	if component.Properties["labels"] == nil || component.Properties["labels"].AdditionalProperties == nil {
		t.Fatalf("expected labels additionalProperties: %+v", component.Properties["labels"])
	}
	if base := builder.Components["model_Base"]; base == nil || base.Properties["id"] == nil {
		t.Fatalf("expected model_Base component with id: %+v", base)
	}
}

func TestBuilderHandlesFormTagsAndFileHeaders(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)
	writeFile(t, filepath.Join(dir, "go.mod"), "module github.com/acme/demo\n\ngo 1.23.4\n")
	writeFile(t, filepath.Join(dir, "upload", "types.go"), `package upload

import (
	"mime/multipart"
)

type ImportEnergyConsumption struct {
	UserRequestInfo any                   `+"`json:\"-\"`"+`
	Token           string                `+"`form:\"token\" validate:\"required\"`"+`
	File            *multipart.FileHeader `+"`form:\"file\" validate:\"required\"`"+`
}
`)

	mod, err := loader.Load([]string{"./..."})
	if err != nil {
		t.Fatal(err)
	}
	ref := &analyzer.TypeRef{Kind: analyzer.TypeNamed, Package: "github.com/acme/demo/upload", Name: "ImportEnergyConsumption"}
	builder := NewBuilder(mod)
	schema := builder.Build(ref)
	if schema.Ref == "" {
		t.Fatalf("expected ref schema: %+v", schema)
	}
	component := builder.Components["upload_ImportEnergyConsumption"]
	if component == nil {
		t.Fatal("expected upload_ImportEnergyConsumption component")
	}
	if _, ok := component.Properties["token"]; !ok {
		t.Fatalf("expected token form field: %+v", component.Properties)
	}
	fileProp := component.Properties["file"]
	if fileProp == nil || fileProp.Type != "string" || fileProp.Format != "binary" {
		t.Fatalf("expected binary file field: %+v", fileProp)
	}
	if len(component.Required) != 2 || component.Required[0] != "file" || component.Required[1] != "token" {
		t.Fatalf("unexpected required fields: %+v", component.Required)
	}
}
