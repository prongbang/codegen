package generate

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/prongbang/codegen/pkg/command"
	"github.com/prongbang/codegen/pkg/filex"
	"github.com/prongbang/codegen/pkg/tools"
	"github.com/prongbang/codegen/template"
	"github.com/pterm/pterm"
)

type GRPCGenerator interface {
	Init() error
}

type grpcGenerator struct {
	FileX     filex.FileX
	Cmd       command.Command
	Installer tools.Installer
	Wire      tools.Installer
	Runner    tools.Runner
}

func getGRPCConfig(rootDir, module string) []FileConfig {
	grpcRoot := filepath.Join(rootDir, "internal", "app", "grpc")
	data := template.Project{Module: module}

	return []FileConfig{
		{
			Path:     filepath.Join(grpcRoot, "Makefile"),
			Template: template.InternalGRPCMakefileTemplate,
		},
		{
			Path:     filepath.Join(grpcRoot, "grpc.go"),
			Template: template.InternalGRPCTemplate,
		},
		{
			Path:     filepath.Join(grpcRoot, "servers.go"),
			Template: template.InternalGRPCServersTemplate,
			Data:     data,
		},
		{
			Path:     filepath.Join(grpcRoot, "health", "v1", "health.proto"),
			Template: template.InternalGRPCHealthProtoTemplate,
			Data:     data,
		},
		{
			Path:     filepath.Join(grpcRoot, "health", "v1", "provider.go"),
			Template: template.InternalGRPCHealthProviderTemplate,
		},
		{
			Path:     filepath.Join(grpcRoot, "health", "v1", "server.go"),
			Template: template.InternalGRPCHealthServerTemplate,
		},
	}
}

func (g *grpcGenerator) Init() error {
	currentDir, err := g.FileX.Getwd()
	if err != nil {
		return err
	}

	rootDir, module, err := getGRPCProjectContext(g.FileX)
	if err != nil {
		return err
	}
	defer func() {
		_ = g.FileX.Chdir(currentDir)
	}()

	if err := g.Installer.Install(); err != nil {
		return err
	}

	for _, config := range getGRPCConfig(rootDir, module) {
		dir := filepath.Dir(config.Path)
		if err := g.FileX.EnsureDir(dir); err != nil {
			return err
		}

		if config.Data == nil {
			config.Data = template.Any{}
		}

		if err := WriteFile(g.FileX, config.Path, config.Template, config.Data); err != nil {
			return err
		}
	}

	if err := bindGRPCWire(g.FileX, rootDir, module); err != nil {
		return err
	}

	if err := bindGRPCApp(g.FileX, rootDir, module); err != nil {
		return err
	}

	if err := g.generateHealthProto(rootDir); err != nil {
		return err
	}

	if err := g.runWire(rootDir); err != nil {
		return err
	}

	pterm.Success.Println("Initialize gRPC project")
	return nil
}

func (g *grpcGenerator) generateHealthProto(rootDir string) error {
	grpcDir := filepath.Join(rootDir, "internal", "app", "grpc")
	if err := g.FileX.Chdir(grpcDir); err != nil {
		return err
	}

	_, err := g.Cmd.RunAsync("make", "gen", "service=health", "version=v1")
	return err
}

func (g *grpcGenerator) runWire(rootDir string) error {
	if err := g.FileX.Chdir(rootDir); err != nil {
		return err
	}
	if err := g.Wire.Install(); err != nil {
		return err
	}
	return g.Runner.Run()
}

func getGRPCProjectContext(fx filex.FileX) (string, string, error) {
	pwd, err := fx.Getwd()
	if err != nil {
		return "", "", err
	}

	candidates := []string{
		pwd,
		filepath.Clean(filepath.Join(pwd, "../../../")),
	}
	visited := map[string]bool{}

	for _, rootDir := range candidates {
		if visited[rootDir] {
			continue
		}
		visited[rootDir] = true

		goModPath := filepath.Join(rootDir, "go.mod")
		if !fx.IsExist(goModPath) {
			continue
		}

		module := parseModuleName(fx.ReadFile(goModPath))
		if module == "" {
			return "", "", fmt.Errorf("invalid go.mod: %s", goModPath)
		}

		return rootDir, module, nil
	}

	return "", "", fmt.Errorf("go.mod not found, run `codegen grpc init` from the project root or internal/app/api")
}

func parseModuleName(goMod string) string {
	const prefix = "module "

	for _, line := range strings.Split(goMod, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, prefix) {
			return strings.TrimSpace(strings.TrimPrefix(line, prefix))
		}
	}

	return ""
}

func bindGRPCWire(fx filex.FileX, rootDir, module string) error {
	wirePath := filepath.Join(rootDir, "wire.go")
	if !fx.IsExist(wirePath) {
		return nil
	}

	wireText := fx.ReadFile(wirePath)
	if wireText == "" {
		return nil
	}

	grpcImport := fmt.Sprintf("appgrpc %q", module+"/internal/app/grpc")
	healthImport := fmt.Sprintf("healthv1 %q", module+"/internal/app/grpc/health/v1")

	if !strings.Contains(wireText, grpcImport) {
		wireText = replaceFirstMarker(wireText, wireImportMarkers(), func(marker string) string {
			return fmt.Sprintf("%s\n\t%s\n\t%s", grpcImport, healthImport, marker)
		})
	}

	if !strings.Contains(wireText, "appgrpc.NewServers,") {
		wireText = replaceFirstMarker(wireText, wireBuildMarkers(), func(marker string) string {
			return fmt.Sprintf("healthv1.ProviderSet,\n\t\tappgrpc.NewServers,\n\t\tappgrpc.New,\n\t\t%s", marker)
		})
	}

	if err := fx.WriteFile(wirePath, []byte(wireText)); err != nil {
		return err
	}
	return nil
}

func bindGRPCApp(fx filex.FileX, rootDir, module string) error {
	appPath := filepath.Join(rootDir, "internal", "app", "app.go")
	if !fx.IsExist(appPath) {
		return nil
	}

	appText := fx.ReadFile(appPath)
	if appText == "" {
		return nil
	}

	grpcImportPath := module + "/internal/app/grpc"
	appText = strings.ReplaceAll(appText, fmt.Sprintf("appgrpc %q", grpcImportPath), fmt.Sprintf("grpc %q", grpcImportPath))
	grpcImport := fmt.Sprintf("grpc %q", grpcImportPath)
	if !strings.Contains(appText, grpcImport) {
		updated := replaceFirstMarker(appText, appImportMarkers(), func(marker string) string {
			return fmt.Sprintf("%s\n\t%s", grpcImport, marker)
		})
		if updated == appText {
			apiImport := fmt.Sprintf("%q", module+"/internal/app/api")
			withImport := fmt.Sprintf("%s\n\t%s", apiImport, grpcImport)
			appText = strings.Replace(appText, apiImport, withImport, 1)
		} else {
			appText = updated
		}
	}

	appText = strings.ReplaceAll(appText, "GRPC appgrpc.GRPC", "GRPC grpc.GRPC")
	if !strings.Contains(appText, "GRPC grpc.GRPC") {
		appText = replaceFirstMarker(appText, appStructMarkers(), func(marker string) string {
			return fmt.Sprintf("GRPC grpc.GRPC\n\t%s", marker)
		})
	}

	appText = strings.ReplaceAll(appText, "grpcApp := a.GRPC.Register()\n\t_ = grpcApp\n\t", "a.GRPC.Register()\n\t")
	appText = strings.ReplaceAll(appText, "grpcApp := a.GRPC.Register()\n    _ = grpcApp\n\t", "a.GRPC.Register()\n\t")
	appText = strings.ReplaceAll(appText, "grpcApp := a.GRPC.Register()", "a.GRPC.Register()")
	appText = strings.ReplaceAll(appText, "\n\t_ = grpcApp", "")
	appText = strings.ReplaceAll(appText, "\n    _ = grpcApp", "")
	if !strings.Contains(appText, "a.GRPC.Register()") {
		appText = replaceFirstMarker(appText, appStartMarkers(), func(marker string) string {
			return fmt.Sprintf("a.GRPC.Register()\n\t%s", marker)
		})
	}

	appText = strings.ReplaceAll(appText, "\ngRPC appgrpc.GRPC,", "\n\tgRPC grpc.GRPC,")
	appText = strings.ReplaceAll(appText, "\ngRPC grpc.GRPC,", "\n\tgRPC grpc.GRPC,")
	appText = strings.ReplaceAll(appText, "\n    gRPC appgrpc.GRPC,", "\n\tgRPC grpc.GRPC,")
	appText = strings.ReplaceAll(appText, "\n    gRPC grpc.GRPC,", "\n\tgRPC grpc.GRPC,")
	if !strings.Contains(appText, "gRPC grpc.GRPC,") {
		appText = replaceFirstMarker(appText, appNewMarkers(), func(marker string) string {
			return fmt.Sprintf("gRPC grpc.GRPC,\n\t%s", marker)
		})
	}

	if !strings.Contains(appText, "GRPC: gRPC,") {
		appText = replaceFirstMarker(appText, appReturnMarkers(), func(marker string) string {
			return fmt.Sprintf("GRPC: gRPC,\n\t\t%s", marker)
		})
	}

	if err := fx.WriteFile(appPath, []byte(appText)); err != nil {
		return err
	}
	return nil
}

func replaceFirstMarker(text string, markers []string, replacement func(marker string) string) string {
	for _, marker := range markers {
		if strings.Contains(text, marker) {
			return strings.Replace(text, marker, replacement(marker), 1)
		}
	}
	return text
}

func wireImportMarkers() []string {
	return []string{
		"//+codegen:import wire:package",
		"// +codegen:import wire:package",
		"//+fibergen:import wire:package",
		"// +fibergen:import wire:package",
	}
}

func wireBuildMarkers() []string {
	return []string{
		"//+codegen:func wire:build",
		"// +codegen:func wire:build",
		"//+fibergen:func wire:build",
		"// +fibergen:func wire:build",
	}
}

func appImportMarkers() []string {
	return []string{
		"//+codegen:import app:package",
		"// +codegen:import app:package",
		"//+fibergen:import app:package",
		"// +fibergen:import app:package",
	}
}

func appStructMarkers() []string {
	return []string{
		"//+codegen:struct app",
		"// +codegen:struct app",
		"//+fibergen:struct app",
		"// +fibergen:struct app",
	}
}

func appStartMarkers() []string {
	return []string{
		"//+codegen:func app:start",
		"// +codegen:func app:start",
		"//+fibergen:func app:start",
		"// +fibergen:func app:start",
	}
}

func appNewMarkers() []string {
	return []string{
		"//+codegen:func app:new",
		"// +codegen:func app:new",
		"//+fibergen:func app:new",
		"// +fibergen:func app:new",
	}
}

func appReturnMarkers() []string {
	return []string{
		"//+codegen:func app:return",
		"// +codegen:func app:return",
		"//+fibergen:func app:return",
		"// +fibergen:func app:return",
	}
}

func NewGRPCGenerator(fileX filex.FileX, cmd command.Command, installer tools.Installer, wire tools.Installer, runner tools.Runner) GRPCGenerator {
	return &grpcGenerator{
		FileX:     fileX,
		Cmd:       cmd,
		Installer: installer,
		Wire:      wire,
		Runner:    runner,
	}
}
