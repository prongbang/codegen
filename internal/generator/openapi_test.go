package generator

import (
	"encoding/json"
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

func TestBuildSupportsMultipartRequestAndStreamResponse(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)
	writeFile(t, filepath.Join(dir, "go.mod"), "module github.com/acme/demo\n\ngo 1.23.4\n")
	writeFile(t, filepath.Join(dir, "upload", "types.go"), `package upload

import "mime/multipart"

type ImportEnergyConsumption struct {
	Token string `+"`form:\"token\" validate:\"required\"`"+`
	File  *multipart.FileHeader `+"`form:\"file\" validate:\"required\"`"+`
}
`)
	writeFile(t, filepath.Join(dir, "pkg", "streamx", "stream.go"), `package streamx

type Stream struct{}
`)

	mod, err := loader.Load([]string{"./..."})
	if err != nil {
		t.Fatal(err)
	}
	docBytes, err := Build(mod, []analyzer.Operation{
		{
			Method:   "post",
			Path:     "/import",
			Request:  &analyzer.TypeRef{Kind: analyzer.TypeNamed, Package: "github.com/acme/demo/upload", Name: "ImportEnergyConsumption"},
			Response: &analyzer.TypeRef{Kind: analyzer.TypeNamed, Package: "github.com/acme/demo/pkg/streamx", Name: "Stream"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]interface{}
	if err := json.Unmarshal(docBytes, &doc); err != nil {
		t.Fatal(err)
	}
	paths := doc["paths"].(map[string]interface{})
	importPath := paths["/import"].(map[string]interface{})
	post := importPath["post"].(map[string]interface{})
	requestBody := post["requestBody"].(map[string]interface{})
	content := requestBody["content"].(map[string]interface{})
	if _, ok := content["multipart/form-data"]; !ok {
		t.Fatalf("expected multipart/form-data content: %+v", content)
	}
	responses := post["responses"].(map[string]interface{})
	okResp := responses["200"].(map[string]interface{})
	respContent := okResp["content"].(map[string]interface{})
	octet := respContent["application/octet-stream"].(map[string]interface{})
	schema := octet["schema"].(map[string]interface{})
	if schema["type"] != "string" || schema["format"] != "binary" {
		t.Fatalf("expected binary stream schema: %+v", schema)
	}
}
