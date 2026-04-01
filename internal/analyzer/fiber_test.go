package analyzer

import (
	"os"
	"path/filepath"
	"testing"

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

func TestAnalyzeFiberSupportsNestedGroupsAndDirectUsecaseCalls(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)
	writeFile(t, filepath.Join(dir, "go.mod"), "module github.com/acme/demo\n\ngo 1.23.4\n")
	writeFile(t, filepath.Join(dir, "internal", "app", "api", "example", "router.go"), `package example

import (
	"github.com/gofiber/fiber/v2"
	"github.com/acme/demo/internal/middleware"
)

type Handler interface {
	Create(c *fiber.Ctx) error
}

type Router struct {
	Handle    Handler
	OnRequest middleware.OnRequest
}

func (r *Router) Initial(app *fiber.App) {
	v1 := app.Group("/v1")
	example := v1.Group("/example")
	{
		example.Post("/create", r.OnRequest.Handler("perm"), r.Handle.Create)
	}
}
`)
	writeFile(t, filepath.Join(dir, "internal", "app", "api", "example", "handler.go"), `package example

import (
	"context"
	"github.com/gofiber/fiber/v2"
)

type handler struct {
	uc UseCase
}

func (h *handler) Create(c *fiber.Ctx) error {
	payload := &CreateRequest{}
	return h.uc.Create(context.Background(), payload)
}
`)
	writeFile(t, filepath.Join(dir, "internal", "app", "api", "example", "usecase.go"), `package example

import "context"

type UseCase interface {
	Create(ctx context.Context, req *CreateRequest) (*CreateResponse, error)
}
`)
	writeFile(t, filepath.Join(dir, "internal", "middleware", "on_request.go"), `package middleware

import "github.com/gofiber/fiber/v2"

type OnRequest interface {
	Handler(permissionId string, options ...func()) fiber.Handler
}
`)
	writeFile(t, filepath.Join(dir, "internal", "app", "api", "example", "model.go"), "package example\n\ntype CreateRequest struct { Name string `json:\"name\"` }\ntype CreateResponse struct { ID string `json:\"id\"` }\n")

	mod, err := loader.Load([]string{"./..."})
	if err != nil {
		t.Fatal(err)
	}
	ops, err := AnalyzeFiber(mod)
	if err != nil {
		t.Fatal(err)
	}
	if len(ops) != 1 {
		t.Fatalf("expected 1 operation, got %d", len(ops))
	}
	if ops[0].Path != "/v1/example/create" {
		t.Fatalf("unexpected path: %s", ops[0].Path)
	}
	if ops[0].Request == nil || ops[0].Request.Name != "CreateRequest" {
		t.Fatalf("unexpected request: %+v", ops[0].Request)
	}
	if ops[0].Response == nil || ops[0].Response.Name != "CreateResponse" {
		t.Fatalf("unexpected response: %+v", ops[0].Response)
	}
}

func TestAnalyzeFiberSupportsHandlerWithoutUsecase(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)
	writeFile(t, filepath.Join(dir, "go.mod"), "module github.com/acme/demo\n\ngo 1.23.4\n")
	writeFile(t, filepath.Join(dir, "internal", "app", "api", "health", "router.go"), `package health

import "github.com/gofiber/fiber/v2"

type Handler interface {
	HealthCheck(c *fiber.Ctx) error
}

type router struct {
	Handle Handler
}

func (r *router) Initial(app *fiber.App) {
	app.Get("/health", r.Handle.HealthCheck)
}
`)
	writeFile(t, filepath.Join(dir, "internal", "app", "api", "health", "handler.go"), "package health\n\nimport \"github.com/gofiber/fiber/v2\"\n\ntype HealthResponse struct { Status string `json:\"status\"` }\n\ntype handler struct{}\n\nfunc (h *handler) HealthCheck(c *fiber.Ctx) error {\n\treturn c.Status(fiber.StatusOK).JSON(&HealthResponse{Status: \"ok\"})\n}\n")

	mod, err := loader.Load([]string{"./..."})
	if err != nil {
		t.Fatal(err)
	}
	ops, err := AnalyzeFiber(mod)
	if err != nil {
		t.Fatal(err)
	}
	if len(ops) != 1 {
		t.Fatalf("expected 1 operation, got %d", len(ops))
	}
	if ops[0].Path != "/health" {
		t.Fatalf("unexpected path: %s", ops[0].Path)
	}
	if ops[0].Response == nil || ops[0].Response.Name != "HealthResponse" {
		t.Fatalf("unexpected response: %+v", ops[0].Response)
	}
}

func TestAnalyzeFiberSupportsAssignedUsecaseCall(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)
	writeFile(t, filepath.Join(dir, "go.mod"), "module github.com/acme/demo\n\ngo 1.23.4\n")
	writeFile(t, filepath.Join(dir, "internal", "app", "api", "device", "router.go"), `package device

import "github.com/gofiber/fiber/v2"

type Handler interface {
	FindList(c *fiber.Ctx) error
}

type router struct {
	Handle Handler
}

func (r *router) Initial(app *fiber.App) {
	app.Post("/device/list", r.Handle.FindList)
}
`)
	writeFile(t, filepath.Join(dir, "internal", "app", "api", "device", "handler.go"), `package device

import (
	"context"
	"github.com/gofiber/fiber/v2"
)

type handler struct {
	Uc UseCase
}

func (h *handler) FindList(c *fiber.Ctx) error {
	request := &ListDeviceRequest{}
	return do(c, request, func(ctx context.Context) (interface{}, error) {
		data, total, err := h.Uc.FindList(ctx, request)
		if err != nil {
			return nil, err
		}
		return NewPage(data, total), nil
	})
}

func do(c *fiber.Ctx, request interface{}, fn func(ctx context.Context) (interface{}, error)) error {
	return nil
}

func NewPage(data *[]Device, total int64) *PagedDevice {
	return &PagedDevice{}
}
`)
	writeFile(t, filepath.Join(dir, "internal", "app", "api", "device", "usecase.go"), `package device

import "context"

type UseCase interface {
	FindList(ctx context.Context, obj *ListDeviceRequest) (*[]Device, int64, error)
}
`)
	writeFile(t, filepath.Join(dir, "internal", "app", "api", "device", "model.go"), "package device\n\ntype ListDeviceRequest struct { Keyword string `json:\"keyword\"` }\ntype Device struct { ID string `json:\"id\"` }\ntype PagedDevice struct { List *[]Device `json:\"list\"` }\n")

	mod, err := loader.Load([]string{"./..."})
	if err != nil {
		t.Fatal(err)
	}
	ops, err := AnalyzeFiber(mod)
	if err != nil {
		t.Fatal(err)
	}
	if len(ops) != 1 {
		t.Fatalf("expected 1 operation, got %d", len(ops))
	}
	if ops[0].Request == nil || ops[0].Request.Name != "ListDeviceRequest" {
		t.Fatalf("unexpected request: %+v", ops[0].Request)
	}
	if ops[0].Response == nil || ops[0].Response.Kind != TypeArray || ops[0].Response.Elem == nil || ops[0].Response.Elem.Name != "Device" {
		t.Fatalf("unexpected response: %+v", ops[0].Response)
	}
}

func TestAnalyzeFiberFallsBackToRequestAndResponseInHandler(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)
	writeFile(t, filepath.Join(dir, "go.mod"), "module github.com/acme/demo\n\ngo 1.23.4\n")
	writeFile(t, filepath.Join(dir, "internal", "app", "api", "ping", "router.go"), `package ping

import "github.com/gofiber/fiber/v2"

type Handler interface {
	Ping(c *fiber.Ctx) error
}

type router struct {
	Handle Handler
}

func (r *router) Initial(app *fiber.App) {
	app.Post("/ping", r.Handle.Ping)
}
`)
	writeFile(t, filepath.Join(dir, "internal", "app", "api", "ping", "handler.go"), `package ping

import (
	"context"
	"github.com/gofiber/fiber/v2"
)

type handler struct{}

func (h *handler) Ping(c *fiber.Ctx) error {
	request := &PingRequest{}
	return do(c, request, func(ctx context.Context) (interface{}, error) {
		response := &PingResponse{Message: "pong"}
		return response, nil
	})
}

func do(c *fiber.Ctx, request interface{}, fn func(ctx context.Context) (interface{}, error)) error {
	return nil
}
`)
	writeFile(t, filepath.Join(dir, "internal", "app", "api", "ping", "model.go"), "package ping\n\ntype PingRequest struct { Message string `json:\"message\"` }\ntype PingResponse struct { Message string `json:\"message\"` }\n")

	mod, err := loader.Load([]string{"./..."})
	if err != nil {
		t.Fatal(err)
	}
	ops, err := AnalyzeFiber(mod)
	if err != nil {
		t.Fatal(err)
	}
	if len(ops) != 1 {
		t.Fatalf("expected 1 operation, got %d", len(ops))
	}
	if ops[0].Request == nil || ops[0].Request.Name != "PingRequest" {
		t.Fatalf("unexpected request: %+v", ops[0].Request)
	}
	if ops[0].Response == nil || ops[0].Response.Name != "PingResponse" {
		t.Fatalf("unexpected response: %+v", ops[0].Response)
	}
}

func TestAnalyzeFiberPrefersHandlerMethodWhenUsecaseSharesName(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)
	writeFile(t, filepath.Join(dir, "go.mod"), "module github.com/acme/demo\n\ngo 1.23.4\n")
	writeFile(t, filepath.Join(dir, "internal", "app", "api", "report", "router.go"), `package report

import "github.com/gofiber/fiber/v2"

type Handler interface {
	Update(c *fiber.Ctx) error
}

type router struct {
	Handle Handler
}

func (r *router) Initial(app *fiber.App) {
	app.Post("/report/update", r.Handle.Update)
}
`)
	writeFile(t, filepath.Join(dir, "internal", "app", "api", "report", "handler.go"), `package report

import (
	"context"
	"github.com/gofiber/fiber/v2"
)

type handler struct {
	UseCase UseCase
}

func (h *handler) Update(c *fiber.Ctx) error {
	request := &UpdateReportRequest{}
	return h.UseCase.Update(context.Background(), request)
}
`)
	writeFile(t, filepath.Join(dir, "internal", "app", "api", "report", "usecase.go"), `package report

import "context"

type UseCase interface {
	Update(ctx context.Context, obj *UpdateReportRequest) (*ReportResponse, error)
}

type useCase struct{}

func (uc *useCase) Update(ctx context.Context, obj *UpdateReportRequest) (*ReportResponse, error) {
	return &ReportResponse{}, nil
}
`)
	writeFile(t, filepath.Join(dir, "internal", "app", "api", "report", "model.go"), "package report\n\ntype UpdateReportRequest struct { ID string `json:\"id\"` }\ntype ReportResponse struct { ID string `json:\"id\"` }\n")

	mod, err := loader.Load([]string{"./..."})
	if err != nil {
		t.Fatal(err)
	}
	ops, err := AnalyzeFiber(mod)
	if err != nil {
		t.Fatal(err)
	}
	if len(ops) != 1 {
		t.Fatalf("expected 1 operation, got %d", len(ops))
	}
	if ops[0].Request == nil || ops[0].Request.Name != "UpdateReportRequest" {
		t.Fatalf("unexpected request: %+v", ops[0].Request)
	}
	if ops[0].Response == nil || ops[0].Response.Name != "ReportResponse" {
		t.Fatalf("unexpected response: %+v", ops[0].Response)
	}
}
