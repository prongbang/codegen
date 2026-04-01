package generate

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/prongbang/codegen/pkg/command"
	creatorpkg "github.com/prongbang/codegen/pkg/creator"
	"github.com/prongbang/codegen/pkg/filex"
	"github.com/prongbang/codegen/pkg/mod"
	"github.com/prongbang/codegen/pkg/option"
	"github.com/prongbang/codegen/template"
)

type stubGenerator struct {
	called bool
	opt    option.Options
	err    error
}

func (s *stubGenerator) Generate(opt option.Options) error {
	s.called = true
	s.opt = opt
	return s.err
}

type recordFileX struct {
	writes map[string]string
}

func (f *recordFileX) EnsureDir(string) error { return nil }
func (f *recordFileX) ReadFile(string) string { return "" }
func (f *recordFileX) Getwd() (string, error) { return "", nil }
func (f *recordFileX) Chdir(string) error     { return nil }
func (f *recordFileX) IsExist(string) bool    { return false }
func (f *recordFileX) IsDirExist(string) bool { return false }
func (f *recordFileX) WriteFile(name string, data []byte) error {
	if f.writes == nil {
		f.writes = map[string]string{}
	}
	f.writes[name] = string(data)
	return nil
}

type readFileX struct {
	content map[string]string
}

func (f *readFileX) EnsureDir(string) error { return nil }
func (f *readFileX) WriteFile(string, []byte) error {
	return nil
}
func (f *readFileX) ReadFile(filename string) string { return f.content[filename] }
func (f *readFileX) Getwd() (string, error)          { return "", nil }
func (f *readFileX) Chdir(string) error              { return nil }
func (f *readFileX) IsExist(string) bool             { return true }
func (f *readFileX) IsDirExist(string) bool          { return false }

type mockCommand struct {
	asyncCalls []string
	runCalls   []string
	runErr     map[string]error
	asyncErr   map[string]error
}

func (m *mockCommand) Run(name string, args ...string) (string, error) {
	call := strings.Join(append([]string{name}, args...), " ")
	m.runCalls = append(m.runCalls, call)
	if err := m.runErr[call]; err != nil {
		return "", err
	}
	return "", nil
}

func (m *mockCommand) RunAsync(name string, args ...string) (string, error) {
	call := strings.Join(append([]string{name}, args...), " ")
	m.asyncCalls = append(m.asyncCalls, call)
	if err := m.asyncErr[call]; err != nil {
		return "", err
	}
	return "", nil
}

type mockInstaller struct {
	called int
	err    error
}

func (m *mockInstaller) Install() error {
	m.called++
	return m.err
}

type mockRunner struct {
	called int
	err    error
}

func (m *mockRunner) Run() error {
	m.called++
	return m.err
}

type mockCreator struct {
	configs []creatorpkg.Config
	err     error
}

func (m *mockCreator) Create(config creatorpkg.Config) error {
	m.configs = append(m.configs, config)
	return m.err
}

type mockBinding struct {
	called int
	pkg    option.Package
	err    error
}

func (m *mockBinding) Bind(pkg option.Package) error {
	m.called++
	m.pkg = pkg
	return m.err
}

type errFileX struct {
	getwd     string
	ensureErr error
	writeErr  error
}

func (e *errFileX) EnsureDir(string) error         { return e.ensureErr }
func (e *errFileX) WriteFile(string, []byte) error { return e.writeErr }
func (e *errFileX) ReadFile(string) string         { return "" }
func (e *errFileX) Getwd() (string, error)         { return e.getwd, nil }
func (e *errFileX) Chdir(string) error             { return nil }
func (e *errFileX) IsExist(string) bool            { return false }
func (e *errFileX) IsDirExist(string) bool         { return false }

func writeFile(t *testing.T, name, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(name), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(name, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func readFile(t *testing.T, name string) string {
	t.Helper()
	data, err := os.ReadFile(name)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
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

func TestGeneratorDispatchesByOption(t *testing.T) {
	project := &stubGenerator{}
	feature := &stubGenerator{}
	shared := &stubGenerator{}
	openAPI := &stubGenerator{}
	gen := NewGenerator(project, feature, shared, openAPI)

	if err := gen.Generate(option.Options{Project: "demo", Module: "github.com/acme/demo"}); err != nil {
		t.Fatal(err)
	}
	if !project.called || feature.called || shared.called || openAPI.called {
		t.Fatalf("unexpected dispatch: project=%v feature=%v shared=%v openapi=%v", project.called, feature.called, shared.called, openAPI.called)
	}

	project.called = false
	if err := gen.Generate(option.Options{Feature: "user"}); err != nil {
		t.Fatal(err)
	}
	if !feature.called {
		t.Fatal("expected feature generator to be called")
	}

	feature.called = false
	if err := gen.Generate(option.Options{Shared: "cache"}); err != nil {
		t.Fatal(err)
	}
	if !shared.called {
		t.Fatal("expected shared generator to be called")
	}

	shared.called = false
	if err := gen.Generate(option.Options{OpenAPI: true, Framework: "fiber"}); err != nil {
		t.Fatal(err)
	}
	if !openAPI.called {
		t.Fatal("expected openapi generator to be called")
	}
}

func TestGeneratorReturnsUnsupportedError(t *testing.T) {
	gen := NewGenerator(&stubGenerator{}, &stubGenerator{}, &stubGenerator{}, &stubGenerator{})
	if err := gen.Generate(option.Options{}); err == nil {
		t.Fatal("expected unsupported option error")
	}
}

func TestOpenAPIGeneratorRejectsUnsupportedFramework(t *testing.T) {
	gen := NewOpenAPIGenerator()
	if err := gen.Generate(option.Options{OpenAPI: true, Framework: "gin"}); err == nil {
		t.Fatal("expected unsupported framework error")
	}
}

func TestOpenAPIGeneratorBuildsSpec(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)
	writeFile(t, filepath.Join(dir, "go.mod"), "module github.com/acme/demo\n\ngo 1.23.4\n")
	writeFile(t, filepath.Join(dir, "internal", "app", "api", "example", "router.go"), `package example

import "github.com/gofiber/fiber/v2"

type Handler interface {
	Echo(c *fiber.Ctx) error
}

type router struct {
	Handle Handler
}

func (r *router) Initial(app *fiber.App) {
	v1 := app.Group("/v1")
	v1.Post("/example/echo", r.Handle.Echo)
}
`)
	writeFile(t, filepath.Join(dir, "internal", "app", "api", "example", "handler.go"), `package example

import (
	"context"
	"github.com/gofiber/fiber/v2"
)

type handler struct {
	Uc UseCase
}

func (h *handler) Echo(c *fiber.Ctx) error {
	request := &EchoRequest{}
	return wrap(func(ctx context.Context) (interface{}, error) {
		return h.Uc.Echo(ctx, request)
	})
}

func wrap(fn func(ctx context.Context) (interface{}, error)) error {
	return nil
}
`)
	writeFile(t, filepath.Join(dir, "internal", "app", "api", "example", "usecase.go"), `package example

import "context"

type UseCase interface {
	Echo(ctx context.Context, obj *EchoRequest) (*Example, error)
}
`)
	writeFile(t, filepath.Join(dir, "internal", "app", "api", "example", "model.go"), "package example\n\ntype EchoRequest struct { Name string `json:\"name\"` }\ntype Example struct { Name string `json:\"name\"` }\n")

	origStdout := os.Stdout
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = writer
	defer func() { os.Stdout = origStdout }()

	err = NewOpenAPIGenerator().Generate(option.Options{OpenAPI: true, Framework: "fiber", Patterns: []string{"./..."}})
	_ = writer.Close()
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if _, err := buf.ReadFrom(reader); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "\"/v1/example/echo\"") {
		t.Fatalf("expected route in output: %s", buf.String())
	}
}

func TestWriteFileRendersTemplate(t *testing.T) {
	fx := &recordFileX{}
	err := WriteFile(fx, "hello.txt", "hello {{.Name}}", map[string]string{"Name": "world"})
	if err != nil {
		t.Fatal(err)
	}
	if got := fx.writes["hello.txt"]; got != "hello world" {
		t.Fatalf("unexpected output: %q", got)
	}
}

func TestGenerateSpecParsesJSON(t *testing.T) {
	fx := &readFileX{
		content: map[string]string{
			"spec.json": `{"id":1,"createdAt":"2024-10-15T14:30:00Z","name":"demo"}`,
		},
	}

	spec, err := generateSpec(fx, option.Options{
		Feature: "device",
		Spec:    "spec.json",
		Driver:  "mariadb",
		Orm:     "bun",
	})
	if err != nil {
		t.Fatal(err)
	}

	if spec.Alias != "d" {
		t.Fatalf("unexpected alias: %s", spec.Alias)
	}
	if spec.PrimaryField.Name != "Id" {
		t.Fatalf("unexpected primary field: %+v", spec.PrimaryField)
	}
	if spec.Driver != "mariadb" || spec.Orm != "bun" {
		t.Fatalf("unexpected driver/orm: %+v", spec)
	}
	if len(spec.Fields) != 3 {
		t.Fatalf("expected 3 fields, got %d", len(spec.Fields))
	}
	if len(spec.Imports) != 1 || spec.Imports[0] != "time" {
		t.Fatalf("expected time import, got %+v", spec.Imports)
	}
}

func TestGenerateSpecRejectsInvalidJSON(t *testing.T) {
	fx := &readFileX{
		content: map[string]string{"spec.json": `{invalid}`},
	}
	if _, err := generateSpec(fx, option.Options{Feature: "device", Spec: "spec.json"}); err == nil {
		t.Fatal("expected invalid json error")
	}
}

func TestGenerateSpecPreservesIntAndFloatNumbers(t *testing.T) {
	fx := &readFileX{
		content: map[string]string{
			"spec.json": `{"id":1,"count":2,"price":10.5}`,
		},
	}

	spec, err := generateSpec(fx, option.Options{
		Feature: "invoice",
		Spec:    "spec.json",
		Driver:  "mariadb",
		Orm:     "bun",
	})
	if err != nil {
		t.Fatal(err)
	}

	gotTypes := map[string]string{}
	for _, field := range spec.Fields {
		gotTypes[field.Name] = field.Type
	}

	if gotTypes["Id"] != "int64" {
		t.Fatalf("unexpected Id type: %s", gotTypes["Id"])
	}
	if gotTypes["Count"] != "int64" {
		t.Fatalf("unexpected Count type: %s", gotTypes["Count"])
	}
	if gotTypes["Price"] != "float64" {
		t.Fatalf("unexpected Price type: %s", gotTypes["Price"])
	}
}

func TestCrudUseCaseBunTemplateSkipsAuditAssignmentsWhenFieldsMissing(t *testing.T) {
	buf := mustRender(t, template.CrudUseCaseBunTemplate, template.Project{
		Name:   "auth",
		Module: "github.com/acme/demo",
		Fields: []template.Field{
			{Name: "Id", Type: "int64", Update: true},
			{Name: "AccessToken", Type: "string", Update: true},
		},
	})

	got := string(buf)
	if strings.Contains(got, "CreatedBy: obj.UserRequestInfo.Id") {
		t.Fatalf("unexpected create audit assignment:\n%s", got)
	}
	if strings.Contains(got, "UpdatedBy: obj.UserRequestInfo.Id") || strings.Contains(got, "data.UpdatedBy = obj.UserRequestInfo.Id") {
		t.Fatalf("unexpected update audit assignment:\n%s", got)
	}
}

func TestFeatureAndSharedTemplates(t *testing.T) {
	pkg := option.Package{
		Name: "device",
		Module: mod.Mod{
			Module:  "github.com/acme/demo",
			AppPath: "internal/app",
		},
	}

	if got := featureTemplates(pkg); len(got) != 7 {
		t.Fatalf("unexpected feature template count: %d", len(got))
	}
	if got := sharedTemplates(pkg); len(got) != 4 {
		t.Fatalf("unexpected shared template count: %d", len(got))
	}
}

func TestCrudTemplates(t *testing.T) {
	pkg := option.Package{
		Name: "device",
		Module: mod.Mod{
			Module:  "github.com/acme/demo",
			AppPath: "internal/app",
		},
		Spec: option.Spec{
			Alias: "d",
			Orm:   "bun",
			Fields: []template.Field{
				{Name: "Id", Type: "int64"},
			},
			PrimaryField: template.PrimaryField{Name: "Id", Type: "int64"},
		},
	}

	if got := featureCrudTemplates(pkg); len(got) != 8 {
		t.Fatalf("unexpected feature crud template count: %d", len(got))
	}
	if got := sharedCrudTemplates(pkg); len(got) != 4 {
		t.Fatalf("unexpected shared crud template count: %d", len(got))
	}
}

func TestNormalizeGRPCServersTextMovesMarkersToBottom(t *testing.T) {
	input := `package grpc

import (
	"google.golang.org/grpc"
	//+codegen:import grpc:package
	healthv1 "github.com/acme/demo/internal/app/grpc/health/v1"
)

type Servers interface {
	Initials(svr *grpc.Server)
}

type servers struct {
	//+codegen:struct grpc
	healthServiceServer healthv1.HealthServiceServer
}

func (s *servers) Initials(svr *grpc.Server) {
	//+codegen:func grpc:initials
	healthv1.RegisterHealthServiceServer(svr, s.healthServiceServer)
}

func NewServers(
	healthServiceServer healthv1.HealthServiceServer
	//+codegen:func grpc:new
) Servers {
	return &servers{
		healthServiceServer: healthServiceServer
		//+codegen:return grpc
	}
}`

	got := normalizeGRPCServersText(input)

	if !strings.Contains(got, "healthServiceServer healthv1.HealthServiceServer\n\t//+codegen:struct grpc") {
		t.Fatalf("struct marker not moved to bottom:\n%s", got)
	}
	if !strings.Contains(got, "healthv1.RegisterHealthServiceServer(svr, s.healthServiceServer)\n\t//+codegen:func grpc:initials") {
		t.Fatalf("initials marker not moved to bottom:\n%s", got)
	}
	if !strings.Contains(got, "healthServiceServer healthv1.HealthServiceServer,\n\t//+codegen:func grpc:new") {
		t.Fatalf("new marker not normalized with comma:\n%s", got)
	}
	if !strings.Contains(got, "healthServiceServer: healthServiceServer,\n\t\t//+codegen:return grpc") {
		t.Fatalf("return marker not normalized:\n%s", got)
	}
}

func TestNormalizeGRPCClientsTextMovesMarkersToBottom(t *testing.T) {
	input := `package core

import (
	"google.golang.org/grpc"
	//+codegen:import client:package
	devicev1 "github.com/acme/demo/internal/thirdparty/core/device/v1"
)

type Clients struct {
	//+codegen:struct client
	DeviceClient devicev1.DeviceServiceClient
	conns []*grpc.ClientConn
}

func (c *Clients) Close() {
	for _, c := range c.conns {
		_ = c.Close()
	}
}

func NewClients() *Clients {
	var conns []*grpc.ClientConn

	// New connection
	//+codegen:func client:new
	deviceCon, deviceClient := devicev1.NewClient()

	// Temp conection
	//+codegen:func client:append
	conns = append(conns, deviceCon)

	return &Clients{
		//+codegen:return client
		DeviceClient: deviceClient
		conns: conns,
	}
}`

	got := normalizeGRPCClientsText(input)

	if !strings.Contains(got, "devicev1 \"github.com/acme/demo/internal/thirdparty/core/device/v1\"\n\t//+codegen:import client:package") {
		t.Fatalf("import marker not moved to bottom:\n%s", got)
	}
	if !strings.Contains(got, "DeviceClient devicev1.DeviceServiceClient\n\t//+codegen:struct client\n\tconns []*grpc.ClientConn") {
		t.Fatalf("struct marker not moved to bottom:\n%s", got)
	}
	if !strings.Contains(got, "deviceCon, deviceClient := devicev1.NewClient()\n\t//+codegen:func client:new") {
		t.Fatalf("new marker not moved to bottom:\n%s", got)
	}
	if !strings.Contains(got, "conns = append(conns, deviceCon)\n\t//+codegen:func client:append") {
		t.Fatalf("append marker not moved to bottom:\n%s", got)
	}
	if !strings.Contains(got, "DeviceClient: deviceClient,\n\t\t//+codegen:return client\n\t\tconns: conns,") {
		t.Fatalf("return marker not normalized:\n%s", got)
	}
}

func TestBindGRPCServersAddsServiceAndKeepsMarkers(t *testing.T) {
	root := t.TempDir()
	serversPath := filepath.Join(root, "internal", "app", "grpc", "servers.go")
	writeFile(t, serversPath, `package grpc

import (
	"google.golang.org/grpc"
	healthv1 "github.com/acme/demo/internal/app/grpc/health/v1"
	//+codegen:import grpc:package
)

type Servers interface {
	Initials(svr *grpc.Server)
}

type servers struct {
	healthServiceServer healthv1.HealthServiceServer
	//+codegen:struct grpc
}

func (s *servers) Initials(svr *grpc.Server) {
	healthv1.RegisterHealthServiceServer(svr, s.healthServiceServer)
	//+codegen:func grpc:initials
}

func NewServers(
	healthServiceServer healthv1.HealthServiceServer,
	//+codegen:func grpc:new
) Servers {
	return &servers{
		healthServiceServer: healthServiceServer,
		//+codegen:return grpc
	}
}`)

	fx := filex.NewFileX()
	if err := bindGRPCServers(fx, root, "github.com/acme/demo", "device"); err != nil {
		t.Fatal(err)
	}

	got := readFile(t, serversPath)
	if !strings.Contains(got, `devicev1 "github.com/acme/demo/internal/app/grpc/device/v1"`) {
		t.Fatalf("missing device import:\n%s", got)
	}
	if !strings.Contains(got, "DeviceServiceServer devicev1.DeviceServiceServer\n\t//+codegen:struct grpc") {
		t.Fatalf("missing device field above marker:\n%s", got)
	}
	if !strings.Contains(got, "devicev1.RegisterDeviceServiceServer(svr, s.DeviceServiceServer)\n\t//+codegen:func grpc:initials") {
		t.Fatalf("missing device register above marker:\n%s", got)
	}
	if !strings.Contains(got, "deviceServiceServer devicev1.DeviceServiceServer,\n\t//+codegen:func grpc:new") {
		t.Fatalf("missing device arg with trailing comma:\n%s", got)
	}
	if !strings.Contains(got, "DeviceServiceServer: deviceServiceServer,\n\t\t//+codegen:return grpc") {
		t.Fatalf("missing device return binding:\n%s", got)
	}
}

func TestBindGRPCClientsAddsServiceAndKeepsMarkers(t *testing.T) {
	root := t.TempDir()
	clientsPath := filepath.Join(root, "internal", "thirdparty", "core", "clients.go")
	writeFile(t, clientsPath, string(mustRender(t, template.InternalGRPCClientsTemplate, template.Project{ThirdParty: "core"})))

	fx := filex.NewFileX()
	if err := bindGRPCClients(fx, root, "github.com/acme/demo", "core", "device"); err != nil {
		t.Fatal(err)
	}

	got := readFile(t, clientsPath)
	if !strings.Contains(got, `devicev1 "github.com/acme/demo/internal/thirdparty/core/device/v1"`) {
		t.Fatalf("missing device import:\n%s", got)
	}
	if !strings.Contains(got, "DeviceClient devicev1.DeviceServiceClient\n\t//+codegen:struct client\n\tconns []*grpc.ClientConn") {
		t.Fatalf("missing device field above marker:\n%s", got)
	}
	if !strings.Contains(got, "deviceCon, deviceClient := devicev1.NewClient()\n\t//+codegen:func client:new") {
		t.Fatalf("missing new client binding above marker:\n%s", got)
	}
	if !strings.Contains(got, "conns = append(conns, deviceCon)\n\t//+codegen:func client:append") {
		t.Fatalf("missing append binding above marker:\n%s", got)
	}
	if !strings.Contains(got, "DeviceClient: deviceClient,\n\t\t//+codegen:return client\n\t\tconns: conns,") {
		t.Fatalf("missing return binding above marker:\n%s", got)
	}
}

func TestBindGRPCThirdPartyMakefileAddsTargetAndKeepsMarker(t *testing.T) {
	root := t.TempDir()
	makefilePath := filepath.Join(root, "internal", "thirdparty", "Makefile")
	writeFile(t, makefilePath, template.InternalGRPCClientMakefileTemplate)

	fx := filex.NewFileX()
	if err := bindGRPCThirdPartyMakefile(fx, root, "core", "device"); err != nil {
		t.Fatal(err)
	}

	got := readFile(t, makefilePath)
	if !strings.Contains(got, "gen_core_device:\n\tmake gen service=device version=v1 thirdparty=core\n\n# //+codegen:func thirdparty:gen") {
		t.Fatalf("missing make target or marker not kept at bottom:\n%s", got)
	}
}

func TestBindGRPCMakefileAddsTargetAndKeepsMarker(t *testing.T) {
	root := t.TempDir()
	makefilePath := filepath.Join(root, "internal", "app", "grpc", "Makefile")
	writeFile(t, makefilePath, template.InternalGRPCMakefileTemplate)

	fx := filex.NewFileX()
	if err := bindGRPCMakefile(fx, root, "device"); err != nil {
		t.Fatal(err)
	}

	got := readFile(t, makefilePath)
	if !strings.Contains(got, "gen_device:\n\tmake gen service=device version=v1\n\n# //+codegen:func thirdparty:gen") {
		t.Fatalf("missing make target or marker not kept at bottom:\n%s", got)
	}
}

func TestParseGRPCClientName(t *testing.T) {
	thirdparty, service, err := parseGRPCClientName("core/device")
	if err != nil {
		t.Fatal(err)
	}
	if thirdparty != "core" || service != "device" {
		t.Fatalf("unexpected target: %s/%s", thirdparty, service)
	}

	thirdparty, service, err = parseGRPCClientName("core:store")
	if err != nil {
		t.Fatal(err)
	}
	if thirdparty != "core" || service != "store" {
		t.Fatalf("unexpected target: %s/%s", thirdparty, service)
	}

	if _, _, err := parseGRPCClientName("device"); err == nil {
		t.Fatal("expected invalid client package error")
	}
}

func TestBindGRPCWireAndApp(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "wire.go"), `package demo

import (
	"github.com/google/wire"
	//+codegen:import wire:package
)

func CreateApp() {
	wire.Build(
		//+codegen:func wire:build
	)
}`)
	writeFile(t, filepath.Join(root, "internal", "app", "app.go"), `package app

import (
	"github.com/acme/demo/internal/app/api"
	//+codegen:import app:package
)

type App interface {
	StartAPI()
}

type app struct {
	API api.API
	//+codegen:struct app
}

func (a *app) StartAPI() {
	//+codegen:func app:start
	a.API.Register()
}

func NewApp(
	apis api.API,
	//+codegen:func app:new
) App {
	return &app{
		API: apis,
		//+codegen:func app:return
	}
}`)

	fx := filex.NewFileX()
	if err := bindGRPCWire(fx, root, "github.com/acme/demo"); err != nil {
		t.Fatal(err)
	}
	if err := bindGRPCApp(fx, root, "github.com/acme/demo"); err != nil {
		t.Fatal(err)
	}

	wire := readFile(t, filepath.Join(root, "wire.go"))
	if !strings.Contains(wire, `appgrpc "github.com/acme/demo/internal/app/grpc"`) {
		t.Fatalf("missing grpc import in wire.go:\n%s", wire)
	}
	if !strings.Contains(wire, "healthv1.ProviderSet,") || !strings.Contains(wire, "appgrpc.NewServers,") || !strings.Contains(wire, "appgrpc.New,") {
		t.Fatalf("missing grpc build entries:\n%s", wire)
	}

	app := readFile(t, filepath.Join(root, "internal", "app", "app.go"))
	if !strings.Contains(app, `grpc "github.com/acme/demo/internal/app/grpc"`) {
		t.Fatalf("missing grpc import in app.go:\n%s", app)
	}
	if !strings.Contains(app, "GRPC grpc.GRPC\n\t//+codegen:struct app") {
		t.Fatalf("missing grpc struct field:\n%s", app)
	}
	if !strings.Contains(app, "a.GRPC.Register()\n\t//+codegen:func app:start") {
		t.Fatalf("missing grpc start call:\n%s", app)
	}
	if !strings.Contains(app, "gRPC grpc.GRPC,\n\t//+codegen:func app:new") {
		t.Fatalf("missing grpc constructor arg:\n%s", app)
	}
	if !strings.Contains(app, "GRPC: gRPC,\n\t\t//+codegen:func app:return") {
		t.Fatalf("missing grpc return field:\n%s", app)
	}
}

func TestFeatureBindingBind(t *testing.T) {
	root := t.TempDir()
	apiDir := filepath.Join(root, "internal", "app", "api")
	writeFile(t, filepath.Join(root, "wire.go"), `package demo

import (
	//+codegen:import wire:package
)

func CreateApp() {
	wire.Build(
		//+codegen:func wire:build
	)
}`)
	writeFile(t, filepath.Join(apiDir, "routers.go"), `package api

import (
	//+codegen:import routers:package
)

type routers struct {
	//+codegen:struct routers
}

func (r *routers) Initials(app any) {
	//+codegen:func initials
}

func NewRouters(
	//+codegen:func new:routers
) any {
	return &routers{
		//+codegen:return &routers
	}
}`)
	chdir(t, apiDir)

	err := NewFeatureBinding(filex.NewFileX()).Bind(option.Package{
		Name: "device",
		Module: mod.Mod{
			Module:  "github.com/acme/demo",
			AppPath: "internal/app",
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	wire := readFile(t, filepath.Join(root, "wire.go"))
	if !strings.Contains(wire, `"github.com/acme/demo/internal/app/api/device"`) || !strings.Contains(wire, "device.ProviderSet,") {
		t.Fatalf("unexpected wire binding:\n%s", wire)
	}
	routers := readFile(t, filepath.Join(apiDir, "routers.go"))
	if !strings.Contains(routers, "DeviceRoute device.Router") || !strings.Contains(routers, "r.DeviceRoute.Initial(app)") {
		t.Fatalf("unexpected routers binding:\n%s", routers)
	}
}

func TestFeatureBindingBindSupportsFibergenMarkers(t *testing.T) {
	root := t.TempDir()
	apiDir := filepath.Join(root, "internal", "app", "api")
	writeFile(t, filepath.Join(root, "wire.go"), `package demo

import (
	//+fibergen:import wire:package
)

func CreateApp() {
	wire.Build(
		//+fibergen:func wire:build
	)
}`)
	writeFile(t, filepath.Join(apiDir, "routers.go"), `package api

import (
	//+fibergen:import routers:package
)

type routers struct {
	//+fibergen:struct routers
}

func (r *routers) Initials(app any) {
	//+fibergen:func initials
}

func NewRouters(
	//+fibergen:func new:routers
) any {
	return &routers{
		//+fibergen:return &routers
	}
}`)
	chdir(t, apiDir)

	err := NewFeatureBinding(filex.NewFileX()).Bind(option.Package{
		Name: "device",
		Module: mod.Mod{
			Module:  "github.com/acme/demo",
			AppPath: "internal/app",
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	wire := readFile(t, filepath.Join(root, "wire.go"))
	if !strings.Contains(wire, `"github.com/acme/demo/internal/app/api/device"`) || !strings.Contains(wire, "device.ProviderSet,") {
		t.Fatalf("unexpected fibergen wire binding:\n%s", wire)
	}
	if !strings.Contains(wire, "//+fibergen:import wire:package") || !strings.Contains(wire, "//+fibergen:func wire:build") {
		t.Fatalf("expected fibergen wire markers to be preserved:\n%s", wire)
	}
	routers := readFile(t, filepath.Join(apiDir, "routers.go"))
	if !strings.Contains(routers, "DeviceRoute device.Router") || !strings.Contains(routers, "r.DeviceRoute.Initial(app)") {
		t.Fatalf("unexpected fibergen routers binding:\n%s", routers)
	}
	if !strings.Contains(routers, "//+fibergen:import routers:package") || !strings.Contains(routers, "//+fibergen:struct routers") || !strings.Contains(routers, "//+fibergen:func initials") || !strings.Contains(routers, "//+fibergen:func new:routers") || !strings.Contains(routers, "//+fibergen:return &routers") {
		t.Fatalf("expected fibergen router markers to be preserved:\n%s", routers)
	}
}

func TestSharedBindingBind(t *testing.T) {
	root := t.TempDir()
	apiDir := filepath.Join(root, "internal", "app", "api")
	writeFile(t, filepath.Join(apiDir, ".keep"), "")
	writeFile(t, filepath.Join(root, "internal", "wire.go"), `package demo

import (
	//+codegen:import wire:package
)

func CreateApp() {
	wire.Build(
		//+codegen:func wire:build
	)
}`)
	chdir(t, apiDir)

	err := NewSharedBinding(filex.NewFileX()).Bind(option.Package{
		Name: "cache",
		Module: mod.Mod{
			Module:  "github.com/acme/demo",
			AppPath: "internal/app",
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	wire := readFile(t, filepath.Join(root, "internal", "wire.go"))
	if !strings.Contains(wire, `sharedcache "github.com/acme/demo/internal/shared/cache"`) || !strings.Contains(wire, "sharedcache.ProviderSet,") {
		t.Fatalf("unexpected shared wire binding:\n%s", wire)
	}
}

func TestSharedBindingBindSupportsFibergenMarkers(t *testing.T) {
	root := t.TempDir()
	apiDir := filepath.Join(root, "internal", "app", "api")
	writeFile(t, filepath.Join(apiDir, ".keep"), "")
	writeFile(t, filepath.Join(root, "internal", "wire.go"), `package demo

import (
	//+fibergen:import wire:package
)

func CreateApp() {
	wire.Build(
		//+fibergen:func wire:build
	)
}`)
	chdir(t, apiDir)

	err := NewSharedBinding(filex.NewFileX()).Bind(option.Package{
		Name: "cache",
		Module: mod.Mod{
			Module:  "github.com/acme/demo",
			AppPath: "internal/app",
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	wire := readFile(t, filepath.Join(root, "internal", "wire.go"))
	if !strings.Contains(wire, `sharedcache "github.com/acme/demo/internal/shared/cache"`) || !strings.Contains(wire, "sharedcache.ProviderSet,") {
		t.Fatalf("unexpected fibergen shared wire binding:\n%s", wire)
	}
	if !strings.Contains(wire, "//+fibergen:import wire:package") || !strings.Contains(wire, "//+fibergen:func wire:build") {
		t.Fatalf("expected fibergen shared markers to be preserved:\n%s", wire)
	}
}

func TestGRPCGeneratorNewCreatesServiceAndRunsMakeAndWire(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "go.mod"), "module github.com/acme/demo\n")
	writeFile(t, filepath.Join(root, "internal", "app", "grpc", "Makefile"), template.InternalGRPCMakefileTemplate)
	writeFile(t, filepath.Join(root, "wire.go"), `package demo

import (
	"github.com/google/wire"
	appgrpc "github.com/acme/demo/internal/app/grpc"
	healthv1 "github.com/acme/demo/internal/app/grpc/health/v1"
	//+codegen:import wire:package
)

func CreateApp() {
	wire.Build(
		healthv1.ProviderSet,
		appgrpc.NewServers,
		appgrpc.New,
		//+codegen:func wire:build
	)
}`)
	writeFile(t, filepath.Join(root, "internal", "app", "grpc", "servers.go"), string(mustRender(t, template.InternalGRPCServersTemplate, template.Project{Module: "github.com/acme/demo"})))
	chdir(t, root)

	cmd := &mockCommand{runErr: map[string]error{}, asyncErr: map[string]error{}}
	grpcInstaller := &mockInstaller{}
	wireInstaller := &mockInstaller{}
	runner := &mockRunner{}
	gen := NewGRPCGenerator(filex.NewFileX(), cmd, grpcInstaller, wireInstaller, runner)

	if err := gen.New("device"); err != nil {
		t.Fatal(err)
	}

	if grpcInstaller.called != 0 {
		t.Fatalf("grpc installer should not run for grpc --new, got %d", grpcInstaller.called)
	}
	if wireInstaller.called != 1 || runner.called != 1 {
		t.Fatalf("expected wire install/run once, got install=%d run=%d", wireInstaller.called, runner.called)
	}
	if len(cmd.asyncCalls) != 1 || cmd.asyncCalls[0] != "make gen service=device version=v1" {
		t.Fatalf("unexpected async calls: %+v", cmd.asyncCalls)
	}

	protoPath := filepath.Join(root, "internal", "app", "grpc", "device", "v1", "device.proto")
	if got := readFile(t, protoPath); !strings.Contains(got, "service DeviceService") {
		t.Fatalf("unexpected proto content:\n%s", got)
	}
	if got := readFile(t, filepath.Join(root, "internal", "app", "grpc", "Makefile")); !strings.Contains(got, "gen_device:\n\tmake gen service=device version=v1") {
		t.Fatalf("unexpected makefile content:\n%s", got)
	}
}

func TestGRPCGeneratorNewClientCreatesFilesAndRunsMake(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "go.mod"), "module github.com/acme/demo\n")
	chdir(t, root)

	cmd := &mockCommand{runErr: map[string]error{}, asyncErr: map[string]error{}}
	grpcInstaller := &mockInstaller{}
	wireInstaller := &mockInstaller{}
	runner := &mockRunner{}
	gen := NewGRPCGenerator(filex.NewFileX(), cmd, grpcInstaller, wireInstaller, runner)

	if err := gen.NewClient("core/device"); err != nil {
		t.Fatal(err)
	}

	if grpcInstaller.called != 1 {
		t.Fatalf("expected grpc installer once, got %d", grpcInstaller.called)
	}
	if wireInstaller.called != 0 || runner.called != 0 {
		t.Fatalf("wire should not run for grpc client generation, got install=%d run=%d", wireInstaller.called, runner.called)
	}
	if len(cmd.asyncCalls) != 1 || cmd.asyncCalls[0] != "make gen service=device version=v1 thirdparty=core" {
		t.Fatalf("unexpected async calls: %+v", cmd.asyncCalls)
	}

	makefilePath := filepath.Join(root, "internal", "thirdparty", "Makefile")
	if got := readFile(t, makefilePath); !strings.Contains(got, "gen_core_device:") {
		t.Fatalf("unexpected makefile content:\n%s", got)
	}

	clientsPath := filepath.Join(root, "internal", "thirdparty", "core", "clients.go")
	if got := readFile(t, clientsPath); !strings.Contains(got, "DeviceClient devicev1.DeviceServiceClient") {
		t.Fatalf("unexpected clients content:\n%s", got)
	}

	protoPath := filepath.Join(root, "internal", "thirdparty", "core", "device", "v1", "device.proto")
	if got := readFile(t, protoPath); !strings.Contains(got, "service DeviceService") {
		t.Fatalf("unexpected proto content:\n%s", got)
	}
}

func TestGRPCGeneratorInitRunsInstallMakeAndWire(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "go.mod"), "module github.com/acme/demo\n")
	writeFile(t, filepath.Join(root, "wire.go"), `package demo

import (
	"github.com/google/wire"
	//+codegen:import wire:package
)

func CreateApp() {
	wire.Build(
		//+codegen:func wire:build
	)
}`)
	writeFile(t, filepath.Join(root, "internal", "app", "app.go"), string(mustRender(t, template.AppTemplate, template.Project{Module: "github.com/acme/demo"})))
	chdir(t, root)

	cmd := &mockCommand{runErr: map[string]error{}, asyncErr: map[string]error{}}
	grpcInstaller := &mockInstaller{}
	wireInstaller := &mockInstaller{}
	runner := &mockRunner{}
	gen := NewGRPCGenerator(filex.NewFileX(), cmd, grpcInstaller, wireInstaller, runner)

	if err := gen.Init(); err != nil {
		t.Fatal(err)
	}

	if grpcInstaller.called != 1 {
		t.Fatalf("expected grpc installer once, got %d", grpcInstaller.called)
	}
	if wireInstaller.called != 1 || runner.called != 1 {
		t.Fatalf("expected wire install/run once, got install=%d run=%d", wireInstaller.called, runner.called)
	}
	if len(cmd.asyncCalls) != 1 || cmd.asyncCalls[0] != "make gen service=health version=v1" {
		t.Fatalf("unexpected async calls: %+v", cmd.asyncCalls)
	}

	if _, err := os.Stat(filepath.Join(root, "internal", "app", "grpc", "health", "v1", "health.proto")); err != nil {
		t.Fatal(err)
	}
}

func TestFeatureGeneratorPrototypeAndCrud(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "go.mod"), "module github.com/acme/demo\n")
	writeFile(t, filepath.Join(root, "internal", "app", "api", ".keep"), "")
	writeFile(t, filepath.Join(root, "spec.json"), `{"id":1}`)
	chdir(t, filepath.Join(root, "internal", "app", "api"))

	creator := &mockCreator{}
	installer := &mockInstaller{}
	wireInstaller := &mockInstaller{}
	runner := &mockRunner{}
	binding := &mockBinding{}
	gen := NewFeatureGenerator(filex.NewFileX(), creator, installer, wireInstaller, runner, binding)

	if err := gen.Generate(option.Options{Feature: "device"}); err != nil {
		t.Fatal(err)
	}
	if wireInstaller.called != 1 || installer.called != 0 || runner.called != 1 || binding.called != 1 || len(creator.configs) != 7 {
		t.Fatalf("unexpected prototype flow: wire=%d install=%d run=%d bind=%d create=%d", wireInstaller.called, installer.called, runner.called, binding.called, len(creator.configs))
	}

	creator.configs = nil
	binding.called = 0
	runner.called = 0
	if err := gen.Generate(option.Options{Feature: "device", Driver: "mariadb", Orm: "bun", Spec: filepath.Join(root, "spec.json")}); err != nil {
		t.Fatal(err)
	}
	if installer.called != 1 || runner.called != 1 || binding.called != 1 || len(creator.configs) != 8 {
		t.Fatalf("unexpected crud flow: install=%d run=%d bind=%d create=%d", installer.called, runner.called, binding.called, len(creator.configs))
	}
}

func TestSharedGeneratorPrototypeAndCrud(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "go.mod"), "module github.com/acme/demo\n")
	writeFile(t, filepath.Join(root, "internal", "app", "api", ".keep"), "")
	writeFile(t, filepath.Join(root, "spec.json"), `{"id":1}`)
	chdir(t, filepath.Join(root, "internal", "app", "api"))

	creator := &mockCreator{}
	installer := &mockInstaller{}
	wireInstaller := &mockInstaller{}
	runner := &mockRunner{}
	binding := &mockBinding{}
	gen := NewSharedGenerator(filex.NewFileX(), creator, installer, wireInstaller, runner, binding)

	if err := gen.Generate(option.Options{Shared: "cache"}); err != nil {
		t.Fatal(err)
	}
	if wireInstaller.called != 1 || installer.called != 0 || runner.called != 1 || binding.called != 1 || len(creator.configs) != 4 {
		t.Fatalf("unexpected prototype flow: wire=%d install=%d run=%d bind=%d create=%d", wireInstaller.called, installer.called, runner.called, binding.called, len(creator.configs))
	}

	creator.configs = nil
	binding.called = 0
	runner.called = 0
	if err := gen.Generate(option.Options{Shared: "cache", Driver: "mariadb", Orm: "bun", Spec: filepath.Join(root, "spec.json")}); err != nil {
		t.Fatal(err)
	}
	if installer.called != 1 || runner.called != 1 || binding.called != 1 || len(creator.configs) != 4 {
		t.Fatalf("unexpected crud flow: install=%d run=%d bind=%d create=%d", installer.called, runner.called, binding.called, len(creator.configs))
	}
}

func TestProjectGeneratorGenerate(t *testing.T) {
	root := t.TempDir()
	chdir(t, root)

	gen := NewProjectGenerator(filex.NewFileX())
	if err := gen.Generate(option.Options{Project: "Demo Project", Module: "github.com/acme/demo"}); err != nil {
		t.Fatal(err)
	}

	projectRoot := filepath.Join(root, "demo-project")
	if _, err := os.Stat(filepath.Join(projectRoot, "go.mod")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(projectRoot, "internal", "app", "api", "routers.go")); err != nil {
		t.Fatal(err)
	}
}

func TestProjectGeneratorGenerateErrors(t *testing.T) {
	gen := NewProjectGenerator(&errFileX{getwd: "/tmp", ensureErr: errors.New("ensure failed")})
	if err := gen.Generate(option.Options{Project: "demo", Module: "github.com/acme/demo"}); err == nil {
		t.Fatal("expected ensure dir error")
	}

	gen = NewProjectGenerator(&errFileX{getwd: "/tmp", writeErr: errors.New("write failed")})
	if err := gen.Generate(option.Options{Project: "demo", Module: "github.com/acme/demo"}); err == nil {
		t.Fatal("expected write file error")
	}
}

func TestGetProjectConfigContainsExpectedFiles(t *testing.T) {
	configs := getProjectConfig("/tmp/demo", option.Options{Project: "demo", Module: "github.com/acme/demo"})
	if len(configs) == 0 {
		t.Fatal("expected project configs")
	}

	paths := make([]string, 0, len(configs))
	for _, cfg := range configs {
		paths = append(paths, cfg.Path)
	}
	joined := strings.Join(paths, "\n")
	for _, want := range []string{
		"/tmp/demo/go.mod",
		"/tmp/demo/cmd/api/main.go",
		"/tmp/demo/internal/app/app.go",
		"/tmp/demo/internal/database/mariadb.go",
		"/tmp/demo/pkg/core/router.go",
	} {
		if !strings.Contains(joined, want) {
			t.Fatalf("missing config path %s", want)
		}
	}
}

func TestGetGRPCProjectContextFindsRootFromAPIDir(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "go.mod"), "module github.com/acme/demo\n")
	writeFile(t, filepath.Join(root, "internal", "app", "api", ".keep"), "")
	chdir(t, filepath.Join(root, "internal", "app", "api"))

	gotRoot, gotModule, err := getGRPCProjectContext(filex.NewFileX())
	if err != nil {
		t.Fatal(err)
	}
	resolvedRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatal(err)
	}
	if gotRoot != resolvedRoot || gotModule != "github.com/acme/demo" {
		t.Fatalf("unexpected context: root=%s module=%s", gotRoot, gotModule)
	}
}

func TestGetGRPCProjectContextErrorsWithoutGoMod(t *testing.T) {
	root := t.TempDir()
	chdir(t, root)

	_, _, err := getGRPCProjectContext(filex.NewFileX())
	if err == nil {
		t.Fatal("expected missing go.mod error")
	}
}

func mustRender(t *testing.T, tmpl string, data any) []byte {
	t.Helper()
	buf, err := template.RenderText(tmpl, data)
	if err != nil {
		t.Fatal(err)
	}
	return buf
}

func TestGRPCGeneratorRunWirePropagatesErrors(t *testing.T) {
	root := t.TempDir()
	chdir(t, root)

	gen := NewGRPCGenerator(
		filex.NewFileX(),
		&mockCommand{},
		&mockInstaller{},
		&mockInstaller{err: errors.New("wire install failed")},
		&mockRunner{},
	)

	err := gen.(*grpcGenerator).runWire(root)
	if err == nil || err.Error() != "wire install failed" {
		t.Fatalf("unexpected error: %v", err)
	}
}

var _ command.Command = (*mockCommand)(nil)
