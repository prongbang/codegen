package generate

import (
	"fmt"
	"github.com/prongbang/codegen/pkg/common"
	"github.com/prongbang/codegen/pkg/config"
	"github.com/prongbang/codegen/pkg/filex"
	"github.com/prongbang/codegen/pkg/option"
	"github.com/pterm/pterm"
	"strings"
)

type sharedBinding struct {
	FileX filex.FileX
}

func (b *sharedBinding) Bind(pkg option.Package) error {
	changeToRoot := "../../"
	pwd, err := b.FileX.Getwd()
	if err != nil {
		return err
	}

	wirePath := ""
	appPath := pkg.Module.AppPath
	if pkg.Module.AppPath == config.AppPath {
		appPath = config.InternalPath
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
		`shared%s "%s/%s/shared/%s"
	%s`, common.ToLower(pkg.Name), pkg.Module.Module, appPath, common.ToLower(pkg.Name), wireImpPat1,
	)
	wireText = strings.Replace(wireText, wireImpPat1, wireImp, 1)
	wireText = strings.Replace(wireText, wireImpPat2, wireImp, 1)

	wireBuildPat1 := "//+codegen:func wire:build"
	wireBuildPat2 := "// +codegen:func wire:build"
	wireBuild := fmt.Sprintf(
		`shared%s.ProviderSet,
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

	// Change to root directory
	if changeToRoot != "" {
		return b.FileX.Chdir(changeToRoot)
	}

	return nil
}

func NewSharedBinding(fileX filex.FileX) Binding {
	return &sharedBinding{
		FileX: fileX,
	}
}
