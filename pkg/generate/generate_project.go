package generate

import (
	"fmt"
	"path/filepath"

	"github.com/ettle/strcase"
	"github.com/prongbang/codegen/pkg/filex"
	"github.com/prongbang/codegen/pkg/option"
	"github.com/prongbang/codegen/template"
	"github.com/pterm/pterm"
)

type FileConfig struct {
	Path     string
	Template string
	Data     interface{}
}

func getProjectConfig(currentDir string, opt option.Options) []FileConfig {
	data := template.Project{Module: opt.Module, Name: opt.Project, Databases: opt.Databases}

	configs := []FileConfig{
		// Root level files
		{
			Path:     fmt.Sprintf("%s/go.mod", currentDir),
			Template: template.ModTemplate,
			Data:     data,
		},
		{
			Path:     fmt.Sprintf("%s/wire.go", currentDir),
			Template: template.WireTemplate,
			Data:     data,
		},
		{
			Path:     fmt.Sprintf("%s/wire_gen.go", currentDir),
			Template: template.WireGenTemplate,
			Data:     data,
		},
		{
			Path:     fmt.Sprintf("%s/Makefile", currentDir),
			Template: template.MakefileTemplate,
		},

		// CMD files
		{
			Path:     fmt.Sprintf("%s/cmd/api/main.go", currentDir),
			Template: template.CmdMainTemplate,
			Data:     data,
		},

		// Docs files
		{
			Path:     fmt.Sprintf("%s/docs/openapi.json", currentDir),
			Template: template.DocsOpenAPIJSONTemplate,
			Data:     data,
		},

		// Middleware files
		{
			Path:     fmt.Sprintf("%s/internal/middleware/jwt.go", currentDir),
			Template: template.InternalMiddlewareJwtTemplate,
			Data:     template.Project{Module: opt.Module},
		},
		{
			Path:     fmt.Sprintf("%s/internal/middleware/api_key.go", currentDir),
			Template: template.InternalMiddlewareApiKeyTemplate,
		},
		{
			Path:     fmt.Sprintf("%s/internal/middleware/on_request.go", currentDir),
			Template: template.InternalMiddlewareOnRequestTemplate,
			Data:     template.Project{Module: opt.Module},
		},

		// App files
		{
			Path:     fmt.Sprintf("%s/internal/app/app.go", currentDir),
			Template: template.AppTemplate,
			Data:     template.Project{Module: opt.Module},
		},
		{
			Path:     fmt.Sprintf("%s/internal/app/api/api.go", currentDir),
			Template: template.ApiTemplate,
			Data:     template.Project{Module: opt.Module},
		},
		{
			Path:     fmt.Sprintf("%s/internal/app/api/routers.go", currentDir),
			Template: template.ApiRoutersTemplate,
			Data:     template.Project{Module: opt.Module},
		},

		// Health files
		{
			Path:     fmt.Sprintf("%s/internal/app/api/health/handler.go", currentDir),
			Template: template.HealthHandlerTemplate,
			Data:     template.Project{Module: opt.Module},
		},
		{
			Path:     fmt.Sprintf("%s/internal/app/api/health/health.go", currentDir),
			Template: template.HealthModelTemplate,
			Data:     template.Project{Module: opt.Module},
		},
		{
			Path:     fmt.Sprintf("%s/internal/app/api/health/provider.go", currentDir),
			Template: template.HealthProviderTemplate,
		},
		{
			Path:     fmt.Sprintf("%s/internal/app/api/health/router.go", currentDir),
			Template: template.HealthRouterTemplate,
			Data:     template.Project{Module: opt.Module},
		},
		{
			Path:     fmt.Sprintf("%s/internal/app/api/health/usecase.go", currentDir),
			Template: template.HealthUseCaseTemplate,
		},

		// Deployment files
		{
			Path:     fmt.Sprintf("%s/deployments/Dockerfile", currentDir),
			Template: template.DeploymentsDockerfileTemplate,
			Data:     template.Project{Module: opt.Module, Name: opt.Project},
		},
		{
			Path:     fmt.Sprintf("%s/deployments/api-prod.yml", currentDir),
			Template: template.DeploymentsAPIComposeTemplate,
			Data:     template.Project{Name: opt.Project},
		},

		// Other package files
		{
			Path:     fmt.Sprintf("%s/pkg/requestx/request.go", currentDir),
			Template: template.RequestXRequestTemplate,
			Data:     template.Project{Module: opt.Module},
		},

		// Casbin policy files
		{
			Path:     fmt.Sprintf("%s/policy/model.conf", currentDir),
			Template: template.CasbinModelTemplate,
		},
		{
			Path:     fmt.Sprintf("%s/policy/policy.csv", currentDir),
			Template: template.CasbinPolicyTemplate,
		},

		// Configuration files
		{
			Path:     fmt.Sprintf("%s/configuration/configuration.go", currentDir),
			Template: template.ConfigurationTemplate,
			Data:     data,
		},
		{
			Path:     fmt.Sprintf("%s/configuration/environment.go", currentDir),
			Template: template.ConfigurationEnvironmentTemplate,
		},
		{
			Path:     fmt.Sprintf("%s/configuration/development.yml", currentDir),
			Template: template.ConfigurationDevelopmentTemplate,
			Data:     data,
		},
		{
			Path:     fmt.Sprintf("%s/configuration/production.yml", currentDir),
			Template: template.ConfigurationProductionTemplate,
			Data:     data,
		},

		// Internal package files
		{
			Path:     fmt.Sprintf("%s/internal/pkg/casbinx/casbinx.go", currentDir),
			Template: template.InternalPkgCasbinxTemplate,
		},
		{
			Path:     fmt.Sprintf("%s/internal/pkg/response/response.go", currentDir),
			Template: template.InternalPkgResponseTemplate,
		},
		{
			Path:     fmt.Sprintf("%s/internal/pkg/validator/validator.go", currentDir),
			Template: template.InternalPkgValidatorTemplate,
		},
	}

	configs = append(configs, getDatabaseConfig(currentDir, data)...)
	configs = append(configs, getMigrationConfig(currentDir, data)...)

	return configs
}

type projectGenerator struct {
	FileX filex.FileX
}

func (p *projectGenerator) Generate(opt option.Options) error {
	opt.Project = strcase.ToKebab(opt.Project)

	spinnerGenProject, _ := pterm.DefaultSpinner.Start(fmt.Sprintf("Create project \"%s\"", opt.Project))

	currentDir, _ := p.FileX.Getwd()
	currentDir = fmt.Sprintf("%s/%s", currentDir, opt.Project)

	// Create project directory
	if err := p.FileX.EnsureDir(currentDir); err != nil {
		return fmt.Errorf("failed to create project directory: %w", err)
	}

	// Get file configurations
	configs := getProjectConfig(currentDir, opt)

	// Create directories and write files
	for _, config := range configs {
		// Ensure directory exists
		dir := filepath.Dir(config.Path)
		if err := p.FileX.EnsureDir(dir); err != nil {
			spinnerGenProject.Fail(fmt.Errorf("failed to create directory %s: %w", dir, err))
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}

		if config.Data == nil {
			config.Data = template.Any{}
		}

		// Write file
		if err := WriteFile(p.FileX, config.Path, config.Template, config.Data); err != nil {
			spinnerGenProject.Fail(fmt.Errorf("failed to write file %s: %w", config.Path, err))
			return fmt.Errorf("failed to write file %s: %w", config.Path, err)
		}
	}

	spinnerGenProject.Success()

	return nil
}

func NewProjectGenerator(fileX filex.FileX) Generator {
	return &projectGenerator{
		FileX: fileX,
	}
}
