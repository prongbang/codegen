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

func getMqttConfig(currentDir string, opt option.Options) []FileConfig {
	data := template.Project{Module: opt.Module, Name: opt.Project}
	return []FileConfig{
		// Root level files
		{
			Path:     fmt.Sprintf("%s/go.mod", currentDir),
			Template: template.MqttModTemplate,
			Data:     data,
		},
		{
			Path:     fmt.Sprintf("%s/Makefile", currentDir),
			Template: template.MqttMakefileTemplate,
		},

		// CMD files
		{
			Path:     fmt.Sprintf("%s/cmd/api/main.go", currentDir),
			Template: template.MqttMainTemplate,
			Data:     data,
		},

		// Configuration files
		{
			Path:     fmt.Sprintf("%s/configuration/configuration.go", currentDir),
			Template: template.MqttConfigurationTemplate,
		},
		{
			Path:     fmt.Sprintf("%s/configuration/environment.go", currentDir),
			Template: template.ConfigurationEnvironmentTemplate,
		},
		{
			Path:     fmt.Sprintf("%s/configuration/development.yml", currentDir),
			Template: template.MqttConfigurationDevelopmentTemplate,
			Data:     data,
		},
		{
			Path:     fmt.Sprintf("%s/configuration/production.yml", currentDir),
			Template: template.MqttConfigurationProductionTemplate,
			Data:     data,
		},

		// MQTT app files
		{
			Path:     fmt.Sprintf("%s/internal/app/mqtt/broker/mqtt_broker.go", currentDir),
			Template: template.MqttBrokerTemplate,
		},
		{
			Path:     fmt.Sprintf("%s/internal/app/mqtt/subscribe/mqtt_subscribe.go", currentDir),
			Template: template.MqttSubscribeTemplate,
			Data:     data,
		},
		{
			Path:     fmt.Sprintf("%s/internal/app/mqtt/forward/forward.go", currentDir),
			Template: template.MqttForwardTemplate,
		},
		{
			Path:     fmt.Sprintf("%s/internal/app/mqtt/forward/mqtt_forward.go", currentDir),
			Template: template.MqttForwardMqttTemplate,
		},
		{
			Path:     fmt.Sprintf("%s/internal/app/mqtt/forward/payload.go", currentDir),
			Template: template.MqttPayloadTemplate,
		},
	}
}

type mqttGenerator struct {
	FileX filex.FileX
}

func (p *mqttGenerator) Generate(opt option.Options) error {
	opt.Project = strcase.ToKebab(opt.Project)

	spinnerGenProject, _ := pterm.DefaultSpinner.Start(fmt.Sprintf("Create mqtt project \"%s\"", opt.Project))

	currentDir, _ := p.FileX.Getwd()
	currentDir = fmt.Sprintf("%s/%s", currentDir, opt.Project)

	// Create project directory
	if err := p.FileX.EnsureDir(currentDir); err != nil {
		return fmt.Errorf("failed to create project directory: %w", err)
	}

	// Get file configurations
	configs := getMqttConfig(currentDir, opt)

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

func NewMqttGenerator(fileX filex.FileX) Generator {
	return &mqttGenerator{
		FileX: fileX,
	}
}
