package generate

import (
	"fmt"
	"strings"

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
	// Bind runs from <root>/internal/app/api, so the project root is three
	// levels up — the same depth the feature binding uses.
	changeToRoot := "../../../"
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

		// Restore the working directory so the trailing Chdir(changeToRoot)
		// below lands on the project root rather than above it.
		_ = b.FileX.Chdir(pwd)
	} else {
		// Reset root path
		changeToRoot = ""
		wirePath = "/" + pwd + "/wire.go"
	}

	wireB := b.FileX.ReadFile(wirePath)
	wireText := wireB
	sharedImport := fmt.Sprintf(`shared%s "%s/%s/shared/%s"`, common.ToLower(pkg.Name), pkg.Module.Module, appPath, common.ToLower(pkg.Name))
	if !strings.Contains(wireText, sharedImport) {
		wireText = replaceFirstMarker(wireText, wireImportMarkers(), func(marker string) string {
			return fmt.Sprintf(
				`%s
	%s`, sharedImport, marker,
			)
		})
	}

	sharedProvider := fmt.Sprintf(`shared%s.ProviderSet,`, common.ToLower(pkg.Name))
	if !strings.Contains(wireText, sharedProvider) {
		wireText = replaceFirstMarker(wireText, wireBuildMarkers(), func(marker string) string {
			return fmt.Sprintf(
				`%s
		%s`, sharedProvider, marker,
			)
		})
	}

	wireText = ensureCrudWireProviders(wireText, pkg, false)

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
