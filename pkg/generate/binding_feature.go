package generate

import (
	"fmt"
	"github.com/prongbang/codegen/pkg/common"
	"github.com/prongbang/codegen/pkg/config"
	"github.com/prongbang/codegen/pkg/filex"
	"github.com/prongbang/codegen/pkg/option"
	"github.com/pterm/pterm"
	"strings"

	"github.com/ettle/strcase"
)

type bindingFeature struct {
	FileX filex.FileX
}

func (b *bindingFeature) Bind(pkg option.Package) error {
	changeToRoot := "../../../"
	pwd, err := b.FileX.Getwd()
	if err != nil {
		return err
	}

	wirePath := ""
	if pkg.Module.AppPath == config.AppPath {
		// Binding wire
		// Change to root directory
		_ = b.FileX.Chdir(changeToRoot)
		pwdRoot, err := b.FileX.Getwd()
		if err != nil {
			return err
		}

		wirePath = "/" + pwdRoot + "/wire.go"
	} else {
		// Reset root path
		changeToRoot = ""
		wirePath = "/" + pwd + "/wire.go"
	}

	wireB := b.FileX.ReadFile(wirePath)
	wireText := wireB
	wireImpPat1 := "//+codegen:import wire:package"
	wireImpPat2 := "// +codegen:import wire:package"
	wireImp := fmt.Sprintf(
		`"%s/%s/api/%s"
	%s`, pkg.Module.Module, pkg.Module.AppPath, common.ToLower(pkg.Name), wireImpPat1,
	)
	wireText = strings.Replace(wireText, wireImpPat1, wireImp, 1)
	wireText = strings.Replace(wireText, wireImpPat2, wireImp, 1)

	wireBuildPat1 := "//+codegen:func wire:build"
	wireBuildPat2 := "// +codegen:func wire:build"
	wireBuild := fmt.Sprintf(
		`%s.ProviderSet,
		%s`, common.ToLower(pkg.Name), wireBuildPat1,
	)
	wireText = strings.Replace(wireText, wireBuildPat1, wireBuild, 1)
	wireText = strings.Replace(wireText, wireBuildPat2, wireBuild, 1)

	spinnerBindWire, _ := pterm.DefaultSpinner.Start("Binding file wire.go")
	if err := b.FileX.WriteFile(wirePath, []byte(wireText)); err == nil {
		spinnerBindWire.Success()
	} else {
		spinnerBindWire.Fail()
	}

	// Binding routers
	// Change to api directory
	_ = b.FileX.Chdir(pwd)
	routerPath := "/" + pwd + "/routers.go"
	routerB := b.FileX.ReadFile(routerPath)
	routerText := routerB
	routerImpPat1 := "//+codegen:import routers:package"
	routerImpPat2 := "// +codegen:import routers:package"
	routerImp := fmt.Sprintf(
		`"%s/%s/api/%s"
	%s`, pkg.Module.Module, pkg.Module.AppPath, common.ToLower(pkg.Name), routerImpPat1,
	)
	routerText = strings.Replace(routerText, routerImpPat1, routerImp, 1)
	routerText = strings.Replace(routerText, routerImpPat2, routerImp, 1)

	routerStructPat1 := "//+codegen:struct routers"
	routerStructPat2 := "// +codegen:struct routers"
	routerStruct := fmt.Sprintf(
		`%sRoute %s.Router
	%s`, common.UpperCamelName(pkg.Name), common.ToLower(pkg.Name), routerStructPat1,
	)
	routerText = strings.Replace(routerText, routerStructPat1, routerStruct, 1)
	routerText = strings.Replace(routerText, routerStructPat2, routerStruct, 1)

	routerInitPat1 := "//+codegen:func initials"
	routerInitPat2 := "// +codegen:func initials"
	routerInit := fmt.Sprintf(
		`r.%sRoute.Initial(app)
	%s`, common.UpperCamelName(pkg.Name), routerInitPat1,
	)
	routerText = strings.Replace(routerText, routerInitPat1, routerInit, 1)
	routerText = strings.Replace(routerText, routerInitPat2, routerInit, 1)

	routerNewPat1 := "//+codegen:func new:routers"
	routerNewPat2 := "// +codegen:func new:routers"
	routerNew := fmt.Sprintf(
		`	%sRoute %s.Router,
	%s`, strcase.ToCamel(pkg.Name), common.ToLower(pkg.Name), routerNewPat1,
	)
	routerText = strings.Replace(routerText, routerNewPat1, routerNew, 1)
	routerText = strings.Replace(routerText, routerNewPat2, routerNew, 1)

	routerBindPat1 := "//+codegen:return &routers"
	routerBindPat2 := "// +codegen:return &routers"
	routerBind := fmt.Sprintf(
		`%sRoute: %sRoute,
		%s`, common.UpperCamelName(pkg.Name), strcase.ToCamel(pkg.Name), routerBindPat1,
	)
	routerText = strings.Replace(routerText, routerBindPat1, routerBind, 1)
	routerText = strings.Replace(routerText, routerBindPat2, routerBind, 1)

	spinnerBindRouter, _ := pterm.DefaultSpinner.Start("Binding file routers.go")
	if err := b.FileX.WriteFile(routerPath, []byte(routerText)); err == nil {
		spinnerBindRouter.Success()
	} else {
		spinnerBindRouter.Fail()
	}

	// Change to root directory
	if changeToRoot != "" {
		return b.FileX.Chdir(changeToRoot)
	}
	return nil
}

func NewFeatureBinding(fileX filex.FileX) Binding {
	return &bindingFeature{
		FileX: fileX,
	}
}
