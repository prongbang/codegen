package generate

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/ettle/strcase"
	"github.com/prongbang/codegen/pkg/command"
	"github.com/prongbang/codegen/pkg/filex"
	"github.com/prongbang/codegen/pkg/tools"
	"github.com/prongbang/codegen/template"
	"github.com/pterm/pterm"
)

type GRPCGenerator interface {
	Init() error
	New(name string) error
	NewClient(name string) error
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

func getGRPCServiceConfig(rootDir, module, name string) []FileConfig {
	grpcRoot := filepath.Join(rootDir, "internal", "app", "grpc")
	data := template.Project{Module: module, Name: name}
	serviceDir := filepath.Join(grpcRoot, data.GRPCPackageName(), "v1")

	return []FileConfig{
		{
			Path:     filepath.Join(serviceDir, fmt.Sprintf("%s.proto", data.GRPCPackageName())),
			Template: template.InternalGRPCProtoTemplate,
			Data:     data,
		},
		{
			Path:     filepath.Join(serviceDir, "provider.go"),
			Template: template.InternalGRPCProviderTemplate,
			Data:     data,
		},
		{
			Path:     filepath.Join(serviceDir, "server.go"),
			Template: template.InternalGRPCServerTemplate,
			Data:     data,
		},
	}
}

func getGRPCClientConfig(rootDir, module, thirdparty, name string) []FileConfig {
	thirdPartyRoot := filepath.Join(rootDir, "internal", "thirdparty")
	data := template.Project{Module: module, ThirdParty: thirdparty, Name: name}
	thirdPartyDir := filepath.Join(thirdPartyRoot, data.ThirdPartyName())
	serviceDir := filepath.Join(thirdPartyDir, data.GRPCPackageName(), "v1")

	return []FileConfig{
		{
			Path:     filepath.Join(thirdPartyRoot, "Makefile"),
			Template: template.InternalGRPCClientMakefileTemplate,
		},
		{
			Path:     filepath.Join(thirdPartyDir, "clients.go"),
			Template: template.InternalGRPCClientsTemplate,
			Data:     data,
		},
		{
			Path:     filepath.Join(serviceDir, "client.go"),
			Template: template.InternalGRPCClientTemplate,
			Data:     data,
		},
		{
			Path:     filepath.Join(serviceDir, fmt.Sprintf("%s.proto", data.GRPCPackageName())),
			Template: template.InternalGRPCClientProtoTemplate,
			Data:     data,
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

func (g *grpcGenerator) New(name string) error {
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

	serviceName := normalizeGRPCName(name)
	if serviceName == "" {
		return fmt.Errorf("grpc package name is required")
	}

	for _, config := range getGRPCServiceConfig(rootDir, module, serviceName) {
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

	if err := bindGRPCServers(g.FileX, rootDir, module, serviceName); err != nil {
		return err
	}

	if err := bindGRPCServiceWire(g.FileX, rootDir, module, serviceName); err != nil {
		return err
	}

	if err := bindGRPCMakefile(g.FileX, rootDir, serviceName); err != nil {
		return err
	}

	if err := g.generateProto(rootDir, serviceName); err != nil {
		return err
	}

	if err := g.runWire(rootDir); err != nil {
		return err
	}

	pterm.Success.Println("Generate gRPC package", serviceName)
	return nil
}

func (g *grpcGenerator) NewClient(name string) error {
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

	thirdpartyName, serviceName, err := parseGRPCClientName(name)
	if err != nil {
		return err
	}

	if err := g.Installer.Install(); err != nil {
		return err
	}

	for _, config := range getGRPCClientConfig(rootDir, module, thirdpartyName, serviceName) {
		dir := filepath.Dir(config.Path)
		if err := g.FileX.EnsureDir(dir); err != nil {
			return err
		}

		if config.Data == nil {
			config.Data = template.Any{}
		}

		if shouldSkipExistingGRPCClientFile(g.FileX, config.Path) {
			continue
		}

		if err := WriteFile(g.FileX, config.Path, config.Template, config.Data); err != nil {
			return err
		}
	}

	if err := bindGRPCClients(g.FileX, rootDir, module, thirdpartyName, serviceName); err != nil {
		return err
	}

	if err := bindGRPCThirdPartyMakefile(g.FileX, rootDir, thirdpartyName, serviceName); err != nil {
		return err
	}

	if err := g.generateClientProto(rootDir, thirdpartyName, serviceName); err != nil {
		return err
	}

	pterm.Success.Println("Generate gRPC client", thirdpartyName+"/"+serviceName)
	return nil
}

func (g *grpcGenerator) generateHealthProto(rootDir string) error {
	return g.generateProto(rootDir, "health")
}

func (g *grpcGenerator) generateProto(rootDir, service string) error {
	grpcDir := filepath.Join(rootDir, "internal", "app", "grpc")
	if err := g.FileX.Chdir(grpcDir); err != nil {
		return err
	}

	_, err := g.Cmd.RunAsync("make", "gen", fmt.Sprintf("service=%s", service), "version=v1")
	return err
}

func (g *grpcGenerator) generateClientProto(rootDir, thirdparty, service string) error {
	thirdPartyDir := filepath.Join(rootDir, "internal", "thirdparty")
	if err := g.FileX.Chdir(thirdPartyDir); err != nil {
		return err
	}

	_, err := g.Cmd.RunAsync("make", "gen", fmt.Sprintf("service=%s", service), "version=v1", fmt.Sprintf("thirdparty=%s", thirdparty))
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

func bindGRPCServers(fx filex.FileX, rootDir, module, name string) error {
	serversPath := filepath.Join(rootDir, "internal", "app", "grpc", "servers.go")
	if !fx.IsExist(serversPath) {
		return fmt.Errorf("grpc servers.go not found, run `codegen grpc init` first")
	}

	serviceName := strcase.ToPascal(name)
	serviceAlias := grpcServiceAlias(name)
	serviceImport := fmt.Sprintf("%s %q", serviceAlias, module+"/internal/app/grpc/"+normalizeGRPCName(name)+"/v1")
	serviceField := fmt.Sprintf("%sServiceServer %s.%sServiceServer", serviceName, serviceAlias, serviceName)
	serviceRegister := fmt.Sprintf("%s.Register%sServiceServer(svr, s.%sServiceServer)", serviceAlias, serviceName, serviceName)
	serviceArg := fmt.Sprintf("%sServiceServer %s.%sServiceServer,", strcase.ToCamel(normalizeGRPCName(name)), serviceAlias, serviceName)
	serviceReturn := fmt.Sprintf("%sServiceServer: %sServiceServer,", serviceName, strcase.ToCamel(normalizeGRPCName(name)))

	serversText := normalizeGRPCServersText(fx.ReadFile(serversPath))
	if !strings.Contains(serversText, serviceImport) {
		serversText = replaceFirstMarker(serversText, grpcImportMarkers(), func(marker string) string {
			return fmt.Sprintf("%s\n\t%s", serviceImport, marker)
		})
	}
	if !strings.Contains(serversText, serviceField) {
		serversText = replaceFirstMarker(serversText, grpcStructMarkers(), func(marker string) string {
			return fmt.Sprintf("%s\n\t%s", serviceField, marker)
		})
	}
	if !strings.Contains(serversText, serviceRegister) {
		serversText = replaceFirstMarker(serversText, grpcInitialsMarkers(), func(marker string) string {
			return fmt.Sprintf("%s\n\t%s", serviceRegister, marker)
		})
	}
	if !strings.Contains(serversText, serviceArg) {
		serversText = replaceFirstMarker(serversText, grpcNewMarkers(), func(marker string) string {
			return fmt.Sprintf("%s\n\t%s", serviceArg, marker)
		})
	}
	if !strings.Contains(serversText, serviceReturn) {
		serversText = replaceFirstMarker(serversText, grpcReturnMarkers(), func(marker string) string {
			return fmt.Sprintf("%s\n\t\t%s", serviceReturn, marker)
		})
	}

	return fx.WriteFile(serversPath, []byte(serversText))
}

func bindGRPCServiceWire(fx filex.FileX, rootDir, module, name string) error {
	wirePath := filepath.Join(rootDir, "wire.go")
	if !fx.IsExist(wirePath) {
		return nil
	}

	serviceAlias := grpcServiceAlias(name)
	serviceImport := fmt.Sprintf("%s %q", serviceAlias, module+"/internal/app/grpc/"+normalizeGRPCName(name)+"/v1")
	serviceProvider := fmt.Sprintf("%s.ProviderSet,", serviceAlias)

	wireText := fx.ReadFile(wirePath)
	if strings.Contains(wireText, serviceImport) {
		return nil
	}

	wireText = replaceFirstMarker(wireText, wireImportMarkers(), func(marker string) string {
		return fmt.Sprintf("%s\n\t%s", serviceImport, marker)
	})
	wireText = replaceFirstMarker(wireText, wireBuildMarkers(), func(marker string) string {
		return fmt.Sprintf("%s\n\t\t%s", serviceProvider, marker)
	})

	return fx.WriteFile(wirePath, []byte(wireText))
}

func bindGRPCMakefile(fx filex.FileX, rootDir, name string) error {
	makefilePath := filepath.Join(rootDir, "internal", "app", "grpc", "Makefile")
	if !fx.IsExist(makefilePath) {
		return nil
	}

	targetName := fmt.Sprintf("gen_%s:", normalizeGRPCName(name))
	makeCmd := fmt.Sprintf("\tmake gen service=%s version=v1", normalizeGRPCName(name))

	makefileText := normalizeGRPCThirdPartyMakefileText(fx.ReadFile(makefilePath))
	if !strings.Contains(makefileText, targetName) {
		makefileText = replaceFirstMarker(makefileText, thirdPartyGenMarkers(), func(marker string) string {
			return fmt.Sprintf("%s\n%s\n\n%s", targetName, makeCmd, marker)
		})
	}

	return fx.WriteFile(makefilePath, []byte(makefileText))
}

func bindGRPCClients(fx filex.FileX, rootDir, module, thirdparty, name string) error {
	clientsPath := filepath.Join(rootDir, "internal", "thirdparty", normalizeGRPCName(thirdparty), "clients.go")
	if !fx.IsExist(clientsPath) {
		return fmt.Errorf("grpc clients.go not found, run `codegen grpc client --new %s/%s` first", normalizeGRPCName(thirdparty), normalizeGRPCName(name))
	}

	serviceName := strcase.ToPascal(normalizeGRPCName(name))
	serviceAlias := grpcServiceAlias(name)
	serviceImport := fmt.Sprintf("%s %q", serviceAlias, module+"/internal/thirdparty/"+normalizeGRPCName(thirdparty)+"/"+normalizeGRPCName(name)+"/v1")
	serviceField := fmt.Sprintf("%sClient %s.%sServiceClient", serviceName, serviceAlias, serviceName)
	serviceVar := strcase.ToCamel(normalizeGRPCName(name))
	serviceConn := fmt.Sprintf("%sCon", serviceVar)
	serviceNew := fmt.Sprintf("%s, %sClient := %s.NewClient()", serviceConn, serviceVar, serviceAlias)
	serviceAppend := fmt.Sprintf("conns = append(conns, %s)", serviceConn)
	serviceReturn := fmt.Sprintf("%sClient: %sClient,", serviceName, serviceVar)

	clientsText := normalizeGRPCClientsText(fx.ReadFile(clientsPath))
	if !strings.Contains(clientsText, serviceImport) {
		clientsText = replaceFirstMarker(clientsText, clientImportMarkers(), func(marker string) string {
			return fmt.Sprintf("%s\n\t%s", serviceImport, marker)
		})
	}
	if !strings.Contains(clientsText, serviceField) {
		clientsText = replaceFirstMarker(clientsText, clientStructMarkers(), func(marker string) string {
			return fmt.Sprintf("%s\n\t%s", serviceField, marker)
		})
	}
	if !strings.Contains(clientsText, serviceNew) {
		clientsText = replaceFirstMarker(clientsText, clientNewMarkers(), func(marker string) string {
			return fmt.Sprintf("%s\n\t%s", serviceNew, marker)
		})
	}
	if !strings.Contains(clientsText, serviceAppend) {
		clientsText = replaceFirstMarker(clientsText, clientAppendMarkers(), func(marker string) string {
			return fmt.Sprintf("%s\n\t%s", serviceAppend, marker)
		})
	}
	if !strings.Contains(clientsText, serviceReturn) {
		clientsText = replaceFirstMarker(clientsText, clientReturnMarkers(), func(marker string) string {
			return fmt.Sprintf("%s\n\t\t%s", serviceReturn, marker)
		})
	}

	return fx.WriteFile(clientsPath, []byte(clientsText))
}

func bindGRPCThirdPartyMakefile(fx filex.FileX, rootDir, thirdparty, name string) error {
	makefilePath := filepath.Join(rootDir, "internal", "thirdparty", "Makefile")
	if !fx.IsExist(makefilePath) {
		return fmt.Errorf("thirdparty Makefile not found, run `codegen grpc client --new %s/%s` first", normalizeGRPCName(thirdparty), normalizeGRPCName(name))
	}

	targetName := fmt.Sprintf("gen_%s_%s:", normalizeGRPCName(thirdparty), normalizeGRPCName(name))
	makeCmd := fmt.Sprintf("\tmake gen service=%s version=v1 thirdparty=%s", normalizeGRPCName(name), normalizeGRPCName(thirdparty))

	makefileText := normalizeGRPCThirdPartyMakefileText(fx.ReadFile(makefilePath))
	if !strings.Contains(makefileText, targetName) {
		makefileText = replaceFirstMarker(makefileText, thirdPartyGenMarkers(), func(marker string) string {
			return fmt.Sprintf("%s\n%s\n\n%s", targetName, makeCmd, marker)
		})
	}

	return fx.WriteFile(makefilePath, []byte(makefileText))
}

func replaceFirstMarker(text string, markers []string, replacement func(marker string) string) string {
	for _, marker := range markers {
		if strings.Contains(text, marker) {
			return strings.Replace(text, marker, replacement(marker), 1)
		}
	}
	return text
}

func replaceMarkerOrAnchor(text string, markers []string, anchor string, markerReplacement func(marker string) string, anchorReplacement func(anchor string) string) string {
	updated := replaceFirstMarker(text, markers, markerReplacement)
	if updated != text {
		return updated
	}
	if strings.Contains(text, anchor) {
		return strings.Replace(text, anchor, anchorReplacement(anchor), 1)
	}
	return text
}

func normalizeGRPCServersText(text string) string {
	text = normalizeSectionMarker(text, "import (\n", "\n)\n\ntype Servers interface {", grpcImportMarkers()[0], false)
	text = normalizeSectionMarker(text, "type servers struct {\n", "\n}\n\nfunc (s *servers) Initials(svr *grpc.Server) {", grpcStructMarkers()[0], false)
	text = normalizeSectionMarker(text, "func (s *servers) Initials(svr *grpc.Server) {\n", "\n}\n\nfunc NewServers(", grpcInitialsMarkers()[0], false)
	text = normalizeSectionMarker(text, "func NewServers(\n", "\n) Servers {", grpcNewMarkers()[0], true)
	text = normalizeSectionMarker(text, "return &servers{\n", "\n\t}\n}", grpcReturnMarkers()[0], true)
	return text
}

func normalizeGRPCClientsText(text string) string {
	text = normalizeSectionMarker(text, "import (\n", "\n)\n\ntype Clients struct {", clientImportMarkers()[0], false)
	text = normalizeSectionMarker(text, "type Clients struct {\n", "\n\tconns []*grpc.ClientConn\n}\n\nfunc (c *Clients) Close() {", clientStructMarkers()[0], false)
	text = normalizeSectionMarker(text, "\t// New connection\n", "\n\n\t// Temp conection\n", clientNewMarkers()[0], false)
	text = normalizeSectionMarker(text, "\t// Temp conection\n", "\n\n\treturn &Clients{\n", clientAppendMarkers()[0], false)
	text = normalizeSectionMarker(text, "\treturn &Clients{\n", "\n\t\tconns: conns,\n\t}\n}", clientReturnMarkers()[0], true)
	return text
}

func normalizeGRPCThirdPartyMakefileText(text string) string {
	lines := strings.Split(strings.TrimRight(text, "\n"), "\n")
	filtered := make([]string, 0, len(lines))
	for _, line := range lines {
		if isMarkerLine(strings.TrimSpace(line)) {
			continue
		}
		filtered = append(filtered, line)
	}

	for len(filtered) > 0 && strings.TrimSpace(filtered[len(filtered)-1]) == "" {
		filtered = filtered[:len(filtered)-1]
	}

	if len(filtered) == 0 {
		return thirdPartyGenMarkers()[0]
	}

	return strings.Join(filtered, "\n") + "\n\n" + thirdPartyGenMarkers()[0]
}

func normalizeSectionMarker(text, start, end, marker string, ensureTrailingComma bool) string {
	startIdx := strings.Index(text, start)
	if startIdx < 0 {
		return text
	}
	contentStart := startIdx + len(start)
	endIdx := strings.Index(text[contentStart:], end)
	if endIdx < 0 {
		return text
	}
	contentEnd := contentStart + endIdx

	lines := strings.Split(text[contentStart:contentEnd], "\n")
	filtered := make([]string, 0, len(lines))
	lastContent := -1
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if isMarkerLine(trimmed) {
			continue
		}
		filtered = append(filtered, line)
		if trimmed != "" {
			lastContent = len(filtered) - 1
		}
	}

	if ensureTrailingComma && lastContent >= 0 {
		trimmed := strings.TrimSpace(filtered[lastContent])
		if trimmed != "" && !strings.HasSuffix(trimmed, ",") {
			filtered[lastContent] += ","
		}
	}

	for len(filtered) > 0 && strings.TrimSpace(filtered[len(filtered)-1]) == "" {
		filtered = filtered[:len(filtered)-1]
	}

	indent := detectMarkerIndent(start)
	filtered = append(filtered, indent+marker)

	section := strings.Join(filtered, "\n")
	return text[:contentStart] + section + text[contentEnd:]
}

func isMarkerLine(line string) bool {
	markers := append([]string{}, grpcImportMarkers()...)
	markers = append(markers, grpcStructMarkers()...)
	markers = append(markers, grpcInitialsMarkers()...)
	markers = append(markers, grpcNewMarkers()...)
	markers = append(markers, grpcReturnMarkers()...)
	markers = append(markers, clientImportMarkers()...)
	markers = append(markers, clientStructMarkers()...)
	markers = append(markers, clientNewMarkers()...)
	markers = append(markers, clientAppendMarkers()...)
	markers = append(markers, clientReturnMarkers()...)
	markers = append(markers, thirdPartyGenMarkers()...)
	for _, marker := range markers {
		if line == marker {
			return true
		}
	}
	return false
}

func detectMarkerIndent(sectionStart string) string {
	switch sectionStart {
	case "import (\n":
		return "\t"
	case "type servers struct {\n":
		return "\t"
	case "func (s *servers) Initials(svr *grpc.Server) {\n":
		return "\t"
	case "func NewServers(\n":
		return "\t"
	case "return &servers{\n":
		return "\t\t"
	case "type Clients struct {\n":
		return "\t"
	case "\t// New connection\n":
		return "\t"
	case "\t// Temp conection\n":
		return "\t"
	case "\treturn &Clients{\n":
		return "\t\t"
	default:
		return "\t"
	}
}

func normalizeGRPCName(name string) string {
	return strcase.ToSnake(strings.ReplaceAll(strings.TrimSpace(name), " ", ""))
}

func parseGRPCClientName(name string) (string, string, error) {
	target := strings.TrimSpace(name)
	if target == "" {
		return "", "", fmt.Errorf("grpc client package name is required")
	}

	target = strings.ReplaceAll(target, "\\", "/")
	target = strings.ReplaceAll(target, ":", "/")
	parts := strings.FieldsFunc(target, func(r rune) bool {
		return r == '/'
	})
	if len(parts) != 2 {
		return "", "", fmt.Errorf("grpc client package must be <thirdparty>/<service>")
	}

	thirdparty := normalizeGRPCName(parts[0])
	service := normalizeGRPCName(parts[1])
	if thirdparty == "" || service == "" {
		return "", "", fmt.Errorf("grpc client package must be <thirdparty>/<service>")
	}

	return thirdparty, service, nil
}

func shouldSkipExistingGRPCClientFile(fx filex.FileX, path string) bool {
	base := filepath.Base(path)
	return fx.IsExist(path) && (base == "Makefile" || base == "clients.go")
}

func grpcServiceAlias(name string) string {
	return fmt.Sprintf("%sv1", strcase.ToCamel(normalizeGRPCName(name)))
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

func grpcImportMarkers() []string {
	return []string{
		"//+codegen:import grpc:package",
		"// +codegen:import grpc:package",
		"//+fibergen:import grpc:package",
		"// +fibergen:import grpc:package",
	}
}

func grpcStructMarkers() []string {
	return []string{
		"//+codegen:struct grpc",
		"// +codegen:struct grpc",
		"//+fibergen:struct grpc",
		"// +fibergen:struct grpc",
	}
}

func grpcInitialsMarkers() []string {
	return []string{
		"//+codegen:func grpc:initials",
		"// +codegen:func grpc:initials",
		"//+fibergen:func grpc:initials",
		"// +fibergen:func grpc:initials",
	}
}

func grpcNewMarkers() []string {
	return []string{
		"//+codegen:func grpc:new",
		"// +codegen:func grpc:new",
		"//+fibergen:func grpc:new",
		"// +fibergen:func grpc:new",
	}
}

func grpcReturnMarkers() []string {
	return []string{
		"//+codegen:return grpc",
		"// +codegen:return grpc",
		"//+fibergen:return grpc",
		"// +fibergen:return grpc",
	}
}

func clientImportMarkers() []string {
	return []string{
		"//+codegen:import client:package",
		"// +codegen:import client:package",
		"//+fibergen:import client:package",
		"// +fibergen:import client:package",
	}
}

func clientStructMarkers() []string {
	return []string{
		"//+codegen:struct client",
		"// +codegen:struct client",
		"//+fibergen:struct client",
		"// +fibergen:struct client",
	}
}

func clientNewMarkers() []string {
	return []string{
		"//+codegen:func client:new",
		"// +codegen:func client:new",
		"//+fibergen:func client:new",
		"// +fibergen:func client:new",
	}
}

func clientAppendMarkers() []string {
	return []string{
		"//+codegen:func client:append",
		"// +codegen:func client:append",
		"//+fibergen:func client:append",
		"// +fibergen:func client:append",
	}
}

func clientReturnMarkers() []string {
	return []string{
		"//+codegen:return client",
		"// +codegen:return client",
		"//+fibergen:return client",
		"// +fibergen:return client",
	}
}

func thirdPartyGenMarkers() []string {
	return []string{
		"# //+codegen:func thirdparty:gen",
		"# // +codegen:func thirdparty:gen",
		"# //+fibergen:func thirdparty:gen",
		"# // +fibergen:func thirdparty:gen",
		"//+codegen:func thirdparty:gen",
		"// +codegen:func thirdparty:gen",
		"//+fibergen:func thirdparty:gen",
		"// +fibergen:func thirdparty:gen",
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
