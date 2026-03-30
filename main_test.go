package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func writeTextFile(t *testing.T, name, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(name), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(name, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func captureStdout(t *testing.T, fn func() error) string {
	t.Helper()
	origStdout := os.Stdout
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = writer
	defer func() {
		os.Stdout = origStdout
	}()

	runErr := fn()
	_ = writer.Close()

	var buf bytes.Buffer
	if _, err := buf.ReadFrom(reader); err != nil {
		t.Fatal(err)
	}
	if runErr != nil {
		t.Fatal(runErr)
	}
	return buf.String()
}

func TestFlagsHelpers(t *testing.T) {
	flags := Flags{
		ProjectName: "Hello World",
		ModuleName:  "github.com/acme",
		FeatureName: "User Profile",
		SharedName:  "Auth Token",
	}

	if got := flags.Project(); got != "hello-world" {
		t.Fatalf("unexpected project: %s", got)
	}
	if got := flags.Module(); got != "github.com/acme/hello-world" {
		t.Fatalf("unexpected module: %s", got)
	}
	if got := flags.Feature(); got != "user_profile" {
		t.Fatalf("unexpected feature: %s", got)
	}
	if got := flags.Shared(); got != "auth_token" {
		t.Fatalf("unexpected shared: %s", got)
	}
}

func TestFlagsEmptyFeatureAndShared(t *testing.T) {
	flags := Flags{}
	if got := flags.Feature(); got != "" {
		t.Fatalf("expected empty feature, got %s", got)
	}
	if got := flags.Shared(); got != "" {
		t.Fatalf("expected empty shared, got %s", got)
	}
}

func TestMainHelpPaths(t *testing.T) {
	origArgs := os.Args
	t.Cleanup(func() { os.Args = origArgs })

	for _, args := range [][]string{
		{"codegen", "-h"},
		{"codegen", "openapi", "-h"},
		{"codegen", "grpc", "-h"},
		{"codegen", "grpc", "init", "-h"},
		{"codegen", "grpc", "server", "-h"},
		{"codegen", "grpc", "client", "-h"},
	} {
		os.Args = args
		main()
	}
}

func TestNewGRPCGeneratorFactory(t *testing.T) {
	if gen := newGRPCGenerator(); gen == nil {
		t.Fatal("expected grpc generator")
	}
}

func TestNewGeneratorFactory(t *testing.T) {
	if gen := newGenerator(); gen == nil {
		t.Fatal("expected generator")
	}
}

func TestOpenAPICommandGeneratesFiberSpec(t *testing.T) {
	dir := t.TempDir()
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

	writeTextFile(t, filepath.Join(dir, "go.mod"), "module github.com/acme/demo\n\ngo 1.23.4\n")
	writeTextFile(t, filepath.Join(dir, "internal", "app", "api", "example", "router.go"), `package example

import "github.com/gofiber/fiber/v2"

type Handler interface {
	Echo(c *fiber.Ctx) error
}

type router struct {
	Handle Handler
}

func (r *router) Initial(app *fiber.App) {
	v1 := app.Group("/v1")
	{
		v1.Post("/example/echo", r.Handle.Echo)
	}
}
`)
	writeTextFile(t, filepath.Join(dir, "internal", "app", "api", "example", "handler.go"), `package example

import (
	"context"
	"github.com/gofiber/fiber/v2"
)

type handler struct {
	Uc UseCase
}

func (h *handler) Echo(c *fiber.Ctx) error {
	request := &EchoRequest{}
	return do(c, request, func(ctx context.Context) (interface{}, error) {
		return h.Uc.Echo(ctx, request)
	})
}

func do(c *fiber.Ctx, request interface{}, fn func(ctx context.Context) (interface{}, error)) error {
	return nil
}
`)
	writeTextFile(t, filepath.Join(dir, "internal", "app", "api", "example", "usecase.go"), `package example

import "context"

type UseCase interface {
	Echo(ctx context.Context, obj *EchoRequest) (*Example, error)
}
`)
	writeTextFile(t, filepath.Join(dir, "internal", "app", "api", "example", "model.go"), `package example

type EchoRequest struct {
	Name string `+"`json:\"name\"`"+`
}

type Example struct {
	Name string `+"`json:\"name\"`"+`
	Meta Meta   `+"`json:\"meta\"`"+`
}

type Meta struct {
	Version int `+"`json:\"version\"`"+`
}
`)

	output := captureStdout(t, func() error {
		return newApp().Run([]string{"codegen", "openapi", "-framework", "fiber", "./..."})
	})

	var doc map[string]interface{}
	if err := json.Unmarshal([]byte(output), &doc); err != nil {
		t.Fatal(err)
	}

	paths, ok := doc["paths"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected paths in output: %s", output)
	}
	if _, ok := paths["/v1/example/echo"]; !ok {
		t.Fatalf("expected /v1/example/echo in output: %s", output)
	}

	components, ok := doc["components"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected components in output: %s", output)
	}
	schemas, ok := components["schemas"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected schemas in output: %s", output)
	}
	for _, key := range []string{"example_EchoRequest", "example_Example", "example_Meta"} {
		if _, ok := schemas[key]; !ok {
			t.Fatalf("expected schema %s in output: %s", key, output)
		}
	}
}
