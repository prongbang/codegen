package generate

import (
	"fmt"

	"github.com/prongbang/codegen/pkg/common"
	"github.com/prongbang/codegen/pkg/config"
	"github.com/prongbang/codegen/pkg/filex"
	"github.com/prongbang/codegen/pkg/option"
	"github.com/pterm/pterm"
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
	wireText = replaceFirstMarker(wireText, wireImportMarkers(), func(marker string) string {
		return fmt.Sprintf(
			`shared%s "%s/%s/shared/%s"
	%s`, common.ToLower(pkg.Name), pkg.Module.Module, appPath, common.ToLower(pkg.Name), marker,
		)
	})

	wireText = replaceFirstMarker(wireText, wireBuildMarkers(), func(marker string) string {
		return fmt.Sprintf(
			`shared%s.ProviderSet,
		%s`, common.ToLower(pkg.Name), marker,
		)
	})

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
