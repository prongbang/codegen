package generate

import (
	"fmt"

	"github.com/prongbang/fibergen/pkg/config"
	"github.com/prongbang/fibergen/pkg/creator"
	"github.com/prongbang/fibergen/pkg/filex"
	"github.com/prongbang/fibergen/pkg/mod"
	"github.com/prongbang/fibergen/pkg/option"
	"github.com/prongbang/fibergen/pkg/tools"
	"github.com/prongbang/fibergen/template"
)

func sharedTemplates(pkg option.Package) map[string][]byte {
	dsTmpl, _ := template.RenderText(template.PrototypeDatasourceTemplate, template.Project{Name: pkg.Name, Module: pkg.Module.Module, Path: pkg.Module.NewAppPath()})
	pdTmpl, _ := template.RenderText(template.PrototypeSharedProviderTemplate, template.Project{Name: pkg.Name})
	rpTmpl, _ := template.RenderText(template.PrototypeRepositoryTemplate, template.Project{Name: pkg.Name})
	mdTmpl, _ := template.RenderText(template.PrototypeModelTemplate, template.Project{Name: pkg.Name, Module: pkg.Module.Module})

	return map[string][]byte{
		"datasource.go":                dsTmpl,
		"provider.go":                  pdTmpl,
		"repository.go":                rpTmpl,
		fmt.Sprintf("%s.go", pkg.Name): mdTmpl,
	}
}

func sharedCrudTemplates(pkg option.Package) map[string][]byte {
	appPath := pkg.Module.AppPath
	if appPath == config.AppPath {
		appPath = config.InternalPath
	}

	// Use template by ORM
	dataSourceTmpl := template.CrudDatasourceSqlBuilderTemplate
	modelTmpl := template.CrudModelTemplate
	repoTmpl := template.CrudRepositoryTemplate
	if pkg.Spec.Orm == "bun" {
		dataSourceTmpl = template.CrudDatasourceBunTemplate
		modelTmpl = template.CrudModelBunTemplate
		repoTmpl = template.CrudRepositoryBunTemplate
	}

	dsTmpl, _ := template.RenderText(dataSourceTmpl, template.Project{Name: pkg.Name, Alias: pkg.Spec.Alias, Fields: pkg.Spec.Fields, PrimaryField: pkg.Spec.PrimaryField, Module: pkg.Module.Module, Path: appPath, Driver: pkg.Spec.Driver})
	mdTmpl, _ := template.RenderText(modelTmpl, template.Project{Imports: pkg.Spec.Imports, Module: pkg.Module.Module, Fields: pkg.Spec.Fields, PrimaryField: pkg.Spec.PrimaryField, Name: pkg.Name})
	rpTmpl, _ := template.RenderText(repoTmpl, template.Project{Name: pkg.Name, PrimaryField: pkg.Spec.PrimaryField, Module: pkg.Module.Module})
	pdTmpl, _ := template.RenderText(template.CrudSharedProviderTemplate, template.Project{Name: pkg.Name})

	return map[string][]byte{
		"datasource.go":                dsTmpl,
		"provider.go":                  pdTmpl,
		"repository.go":                rpTmpl,
		fmt.Sprintf("%s.go", pkg.Name): mdTmpl,
	}
}

type sharedGenerator struct {
	FileX         filex.FileX
	Creator       creator.Creator
	Installer     tools.Installer
	WireInstaller tools.Installer
	WireRunner    tools.Runner
	SharedBinding Binding
}

func (f *sharedGenerator) Generate(opt option.Options) error {
	opt.Feature = opt.Shared
	if opt.Driver != "" {
		return f.generateSharedCrud(opt)
	}
	return f.generateSharedPrototype(opt)
}

func (f *sharedGenerator) generateSharedCrud(opt option.Options) error {
	spec, err := generateSpec(f.FileX, opt)
	if err != nil {
		return err
	}

	// Install library
	if err := f.Installer.Install(); err == nil {
		module := mod.GetModule(f.FileX)

		// Change current directory to the shared path
		sharedPath := "../../shared"
		_ = f.FileX.EnsureDir(sharedPath)
		_ = f.FileX.Chdir(sharedPath)

		pkg := option.Package{
			Name:   opt.Shared,
			Module: module,
			Spec:   spec,
		}
		for filename, tmpl := range sharedCrudTemplates(pkg) {
			_ = f.Creator.Create(creator.Config{Pkg: pkg, Filename: filename, Template: tmpl})
		}

		// Change current directory to api path
		apiPath := fmt.Sprintf("../../%s/api", module.AppPath)
		_ = f.FileX.Chdir(apiPath)

		_ = f.SharedBinding.Bind(pkg)
		return f.WireRunner.Run()
	}
	return nil
}

func (f *sharedGenerator) generateSharedPrototype(opt option.Options) error {
	_ = f.WireInstaller.Install()
	module := mod.GetModule(f.FileX)

	// Change current directory to the shared path
	sharedPath := "../../shared"
	_ = f.FileX.EnsureDir(sharedPath)
	_ = f.FileX.Chdir(sharedPath)

	pkg := option.Package{Name: opt.Shared, Module: module}
	for filename, tmpl := range sharedTemplates(pkg) {
		_ = f.Creator.Create(creator.Config{Pkg: pkg, Filename: filename, Template: tmpl})
	}

	// Change current directory to api path
	apiPath := fmt.Sprintf("../../%s/api", module.AppPath)
	_ = f.FileX.Chdir(apiPath)

	_ = f.SharedBinding.Bind(pkg)
	return f.WireRunner.Run()
}

func NewSharedGenerator(
	fx filex.FileX,
	creator creator.Creator,
	installer tools.Installer,
	wireInstaller tools.Installer,
	wireRunner tools.Runner,
	sharedBinding Binding,
) Generator {
	return &sharedGenerator{
		FileX:         fx,
		Creator:       creator,
		Installer:     installer,
		WireInstaller: wireInstaller,
		WireRunner:    wireRunner,
		SharedBinding: sharedBinding,
	}
}
