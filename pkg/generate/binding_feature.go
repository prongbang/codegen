package generate

import (
	"fmt"

	"github.com/ettle/strcase"
	"github.com/prongbang/codegen/pkg/common"
	"github.com/prongbang/codegen/pkg/config"
	"github.com/prongbang/codegen/pkg/filex"
	"github.com/prongbang/codegen/pkg/option"
	"github.com/pterm/pterm"
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
	wireText = replaceFirstMarker(wireText, wireImportMarkers(), func(marker string) string {
		return fmt.Sprintf(
			`"%s/%s/api/%s"
	%s`, pkg.Module.Module, pkg.Module.AppPath, common.ToLower(pkg.Name), marker,
		)
	})

	wireText = replaceFirstMarker(wireText, wireBuildMarkers(), func(marker string) string {
		return fmt.Sprintf(
			`%s.ProviderSet,
		%s`, common.ToLower(pkg.Name), marker,
		)
	})

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
	routerText = replaceFirstMarker(routerText, routerImportMarkers(), func(marker string) string {
		return fmt.Sprintf(
			`"%s/%s/api/%s"
	%s`, pkg.Module.Module, pkg.Module.AppPath, common.ToLower(pkg.Name), marker,
		)
	})

	routerText = replaceFirstMarker(routerText, routerStructMarkers(), func(marker string) string {
		return fmt.Sprintf(
			`%sRoute %s.Router
	%s`, common.UpperCamelName(pkg.Name), common.ToLower(pkg.Name), marker,
		)
	})

	routerText = replaceFirstMarker(routerText, routerInitialsMarkers(), func(marker string) string {
		return fmt.Sprintf(
			`r.%sRoute.Initial(app)
	%s`, common.UpperCamelName(pkg.Name), marker,
		)
	})

	routerText = replaceFirstMarker(routerText, routerNewMarkers(), func(marker string) string {
		return fmt.Sprintf(
			`	%sRoute %s.Router,
	%s`, strcase.ToCamel(pkg.Name), common.ToLower(pkg.Name), marker,
		)
	})

	routerText = replaceFirstMarker(routerText, routerReturnMarkers(), func(marker string) string {
		return fmt.Sprintf(
			`%sRoute: %sRoute,
		%s`, common.UpperCamelName(pkg.Name), strcase.ToCamel(pkg.Name), marker,
		)
	})

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

func routerImportMarkers() []string {
	return []string{
		"//+codegen:import routers:package",
		"// +codegen:import routers:package",
		"//+fibergen:import routers:package",
		"// +fibergen:import routers:package",
	}
}

func routerStructMarkers() []string {
	return []string{
		"//+codegen:struct routers",
		"// +codegen:struct routers",
		"//+fibergen:struct routers",
		"// +fibergen:struct routers",
	}
}

func routerInitialsMarkers() []string {
	return []string{
		"//+codegen:func initials",
		"// +codegen:func initials",
		"//+fibergen:func initials",
		"// +fibergen:func initials",
	}
}

func routerNewMarkers() []string {
	return []string{
		"//+codegen:func new:routers",
		"// +codegen:func new:routers",
		"//+fibergen:func new:routers",
		"// +fibergen:func new:routers",
	}
}

func routerReturnMarkers() []string {
	return []string{
		"//+codegen:return &routers",
		"// +codegen:return &routers",
		"//+fibergen:return &routers",
		"// +fibergen:return &routers",
	}
}
