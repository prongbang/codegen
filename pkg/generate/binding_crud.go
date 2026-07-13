package generate

import (
	"fmt"
	"strings"

	"github.com/prongbang/codegen/pkg/option"
	"github.com/pterm/pterm"
)

// ensureCrudWireProviders binds database.NewDB (provides dbre.AppIDB for
// generated datasources) to wire.go if it is not already bound. Only applied
// when the package was generated from a spec (CRUD); prototype packages do
// not need it, and wire fails on unused providers.
//
// The middleware.OnRequest dependency required by feature routers is NOT
// bound automatically — projects provide their own (e.g. middleware.NewOnRequest
// or middleware.NewOnRequestGuard); a hint is printed when none is found.
func ensureCrudWireProviders(wireText string, pkg option.Package, withOnRequest bool) string {
	if len(pkg.Spec.Fields) == 0 {
		return wireText
	}

	if !strings.Contains(wireText, "database.NewDB") {
		wireText = replaceFirstMarker(wireText, wireBuildMarkers(), func(marker string) string {
			return fmt.Sprintf(`database.NewDB,
		%s`, marker)
		})
		pterm.Info.Println("Added database.NewDB to wire.go (required by the CRUD datasource)")
	}

	if withOnRequest && !strings.Contains(wireText, "middleware.New") {
		pterm.Warning.Println("wire.go has no provider for middleware.OnRequest (required by the CRUD router); add e.g. middleware.NewOnRequestGuard to wire.Build")
	}

	return wireText
}
