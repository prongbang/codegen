package generate

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ettle/strcase"
	"github.com/prongbang/codegen/pkg/common"
	"github.com/prongbang/codegen/pkg/config"
	"github.com/prongbang/codegen/pkg/creator"
	"github.com/prongbang/codegen/pkg/filex"
	"github.com/prongbang/codegen/pkg/mod"
	"github.com/prongbang/codegen/pkg/option"
	"github.com/prongbang/codegen/pkg/tools"
	"github.com/prongbang/codegen/pkg/typer"
	"github.com/prongbang/codegen/template"
)

func featureTemplates(pkg option.Package) map[string][]byte {
	dsTmpl, _ := template.RenderText(template.PrototypeDatasourceTemplate, template.Project{Name: pkg.Name, Module: pkg.Module.Module, Path: pkg.Module.NewAppPath()})
	hdTmpl, _ := template.RenderText(template.PrototypeHandlerTemplate, template.Project{Name: pkg.Name, Module: pkg.Module.Module})
	pdTmpl, _ := template.RenderText(template.PrototypeProviderTemplate, template.Project{Name: pkg.Name})
	rpTmpl, _ := template.RenderText(template.PrototypeRepositoryTemplate, template.Project{Name: pkg.Name})
	rtTmpl, _ := template.RenderText(template.PrototypeRouterTemplate, template.Project{Name: pkg.Name, Module: pkg.Module.Module})
	ucTmpl, _ := template.RenderText(template.PrototypeUseCaseTemplate, template.Project{Name: pkg.Name})
	mdTmpl, _ := template.RenderText(template.PrototypeModelTemplate, template.Project{Name: pkg.Name, Module: pkg.Module.Module})

	return map[string][]byte{
		"datasource.go":                dsTmpl,
		"handler.go":                   hdTmpl,
		"provider.go":                  pdTmpl,
		"repository.go":                rpTmpl,
		"router.go":                    rtTmpl,
		"usecase.go":                   ucTmpl,
		fmt.Sprintf("%s.go", pkg.Name): mdTmpl,
	}
}

func featureCrudTemplates(pkg option.Package) map[string][]byte {
	appPath := pkg.Module.AppPath
	if appPath == config.AppPath {
		appPath = config.InternalPath
	}

	// Use template by ORM
	dataSourceTmpl := template.CrudDatasourceSqlBuilderTemplate
	modelTmpl := template.CrudModelTemplate
	usecaseTmpl := template.CrudUseCaseTemplate
	repoTmpl := template.CrudRepositoryTemplate
	routeTmpl := template.CrudRouterTemplate
	if pkg.Spec.Orm == "bun" {
		dataSourceTmpl = template.CrudDatasourceBunTemplate
		modelTmpl = template.CrudModelBunTemplate
		usecaseTmpl = template.CrudUseCaseBunTemplate
		repoTmpl = template.CrudRepositoryBunTemplate
		routeTmpl = template.CrudRouterBunTemplate
	}

	// Render
	dsTmpl, _ := template.RenderText(dataSourceTmpl, template.Project{Name: pkg.Name, Alias: pkg.Spec.Alias, Fields: pkg.Spec.Fields, PrimaryField: pkg.Spec.PrimaryField, Module: pkg.Module.Module, Path: appPath, Driver: pkg.Spec.Driver})
	rpTmpl, _ := template.RenderText(repoTmpl, template.Project{Name: pkg.Name, PrimaryField: pkg.Spec.PrimaryField, Module: pkg.Module.Module})
	ucTmpl, _ := template.RenderText(usecaseTmpl, template.Project{Name: pkg.Name, Module: pkg.Module.Module, Fields: pkg.Spec.Fields})
	mdTmpl, _ := template.RenderText(modelTmpl, template.Project{Imports: pkg.Spec.Imports, Module: pkg.Module.Module, Fields: pkg.Spec.Fields, PrimaryField: pkg.Spec.PrimaryField, Name: pkg.Name})
	rtTmpl, _ := template.RenderText(routeTmpl, template.Project{Name: pkg.Name, Module: pkg.Module.Module})
	hdTmpl, _ := template.RenderText(template.CrudHandlerTemplate, template.Project{Name: pkg.Name, Module: pkg.Module.Module})
	pdTmpl, _ := template.RenderText(template.CrudProviderTemplate, template.Project{Name: pkg.Name})
	pmTmpl, _ := template.RenderText(template.CrudPermissionTemplate, template.Project{Name: pkg.Name})

	return map[string][]byte{
		"datasource.go":                dsTmpl,
		"handler.go":                   hdTmpl,
		"provider.go":                  pdTmpl,
		"permission.go":                pmTmpl,
		"repository.go":                rpTmpl,
		"router.go":                    rtTmpl,
		"usecase.go":                   ucTmpl,
		fmt.Sprintf("%s.go", pkg.Name): mdTmpl,
	}
}

type featureGenerator struct {
	FileX          filex.FileX
	Creator        creator.Creator
	Installer      tools.Installer
	WireInstaller  tools.Installer
	WireRunner     tools.Runner
	FeatureBinding Binding
}

func (f *featureGenerator) Generate(opt option.Options) error {
	if opt.Driver != "" {
		return f.generateFeatureCrud(opt)
	}
	return f.generateFeaturePrototype(opt)
}

func (f *featureGenerator) generateFeaturePrototype(opt option.Options) error {
	_ = f.WireInstaller.Install()
	pkg := option.Package{Name: opt.Feature, Module: mod.GetModule(f.FileX)}
	for filename, tmpl := range featureTemplates(pkg) {
		_ = f.Creator.Create(creator.Config{Pkg: pkg, Filename: filename, Template: tmpl})
	}
	_ = f.FeatureBinding.Bind(pkg)
	return f.WireRunner.Run()
}

func (f *featureGenerator) generateFeatureCrud(opt option.Options) error {
	spec, err := generateSpec(f.FileX, opt)
	if err != nil {
		return err
	}

	// Install library
	if err := f.Installer.Install(); err == nil {
		module := mod.GetModule(f.FileX)
		pkg := option.Package{
			Name:   opt.Feature,
			Module: module,
			Spec:   spec,
		}
		for filename, tmpl := range featureCrudTemplates(pkg) {
			_ = f.Creator.Create(creator.Config{Pkg: pkg, Filename: filename, Template: tmpl})
		}
		_ = f.FeatureBinding.Bind(pkg)
		return f.WireRunner.Run()
	}
	return nil
}

func generateSpec(fileX filex.FileX, opt option.Options) (option.Spec, error) {
	// Load spec from JSON file
	jsonSpec := fileX.ReadFile(opt.Spec)
	var result map[string]interface{}
	decoder := json.NewDecoder(bytes.NewBufferString(jsonSpec))
	decoder.UseNumber()
	err := decoder.Decode(&result)
	if err != nil {
		return option.Spec{}, fmt.Errorf("JSON format invalid: %s", err.Error())
	}

	imports := []string{}
	spec := option.Spec{
		Driver: opt.Driver,
		Orm:    opt.Orm,
	}
	alias := common.Abbrev(opt.Feature)
	fields := []template.Field{}
	for key, value := range result {
		snakeTag := strcase.ToSnake(key)
		camelTag := strcase.ToCamel(key)
		vars := strcase.ToPascal(key)
		typeValue := typer.Get(value)
		isPrimaryKey := strings.ToUpper(key) == "ID"

		// Imports
		if strings.Contains(typeValue, "time.Time") {
			if len(imports) == 0 {
				imports = append(imports, "time")
			} else {
				for _, imp := range imports {
					if imp != "time" {
						imports = append(imports, "time")
					}
				}
			}
		}

		// Fields
		fields = append(fields, template.Field{PrimaryKey: isPrimaryKey, Alias: alias, CamelCase: camelTag, SnakeCase: snakeTag, PascalCase: vars, Name: vars, Type: typeValue, JsonTag: camelTag, DbTag: snakeTag, Update: true, Create: true})

		// Pk
		if isPrimaryKey {
			spec.PrimaryField = template.PrimaryField{
				Name:    "Id",
				Type:    typeValue,
				JsonTag: "id",
			}
		}
	}

	spec.Alias = alias
	spec.Fields = fields
	spec.Imports = imports
	return spec, nil
}

func NewFeatureGenerator(
	fx filex.FileX,
	creator creator.Creator,
	installer tools.Installer,
	wireInstaller tools.Installer,
	wireRunner tools.Runner,
	featureBinding Binding,
) Generator {
	return &featureGenerator{
		FileX:          fx,
		Creator:        creator,
		Installer:      installer,
		WireInstaller:  wireInstaller,
		WireRunner:     wireRunner,
		FeatureBinding: featureBinding,
	}
}
