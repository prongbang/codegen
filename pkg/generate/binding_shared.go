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
	name := common.ToLower(pkg.Name)
	sharedImport := fmt.Sprintf(`shared%s "%s/%s/shared/%s"`, name, pkg.Module.Module, appPath, name)
	sharedProvider := fmt.Sprintf(`shared%s.ProviderSet,`, name)

	// A shared package exists to be injected into a feature, and until one does
	// wire rejects the whole build with `unused provider set "ProviderSet"`,
	// leaving wire_gen.go stale so the next feature fails to compile. Bind it
	// only once something references the package, and point the way otherwise.
	if strings.Contains(wireText, sharedImport) || strings.Contains(wireText, sharedProvider) {
		wireText = ensureCrudWireProviders(wireText, pkg, false)
	} else {
		pterm.Info.Printfln("Add these to wire.go once a feature injects shared%s:\n\t%s\n\t\t%s",
			name, sharedImport, sharedProvider)
		return b.finish(changeToRoot)
	}

	spinnerBindWire, _ := pterm.DefaultSpinner.Start("Binding file wire.go")
	if err := b.FileX.WriteFile(wirePath, []byte(wireText)); err == nil {
		spinnerBindWire.Success()
	} else {
		spinnerBindWire.Fail()
	}

	return b.finish(changeToRoot)
}

// finish restores the working directory the caller expects.
func (b *sharedBinding) finish(changeToRoot string) error {
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
