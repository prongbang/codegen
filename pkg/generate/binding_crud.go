package generate

import (
	"fmt"
	"strings"

	"github.com/prongbang/codegen/pkg/option"
	"github.com/pterm/pterm"
)

// ensureCrudWireProviders binds the providers required by generated CRUD code to
// wire.go if they are not already bound. Only applied when the package was
// generated from a spec (CRUD); prototype packages need neither, and wire fails
// on unused providers.
//
// Both bindings are gated on the bun ORM, because only the bun templates consume
// them: the bun datasource takes dbre.AppIDB (from database.NewDB) and the bun
// router takes middleware.OnRequest. The sqlbuilder datasource takes
// database.Drivers (already provided by the CreateApp parameter) and its router
// takes no middleware, so binding either provider there would leave it unused and
// wire would fail the whole build.
func ensureCrudWireProviders(wireText string, pkg option.Package, withOnRequest bool) string {
	if len(pkg.Spec.Fields) == 0 {
		return wireText
	}

	isBun := pkg.Spec.Orm == "bun"

	if isBun && !strings.Contains(wireText, "database.NewDB") {
		wireText = replaceFirstMarker(wireText, wireBuildMarkers(), func(marker string) string {
			return fmt.Sprintf(`database.NewDB,
		%s`, marker)
		})
		pterm.Info.Println("Added database.NewDB to wire.go (required by the CRUD datasource)")
	}

	// Skip when any middleware provider is already bound (e.g. the project binds
	// middleware.NewOnRequest with its own options); a second provider for
	// middleware.OnRequest also breaks wire.
	if withOnRequest && isBun && !strings.Contains(wireText, "middleware.New") {
		middlewareImport := fmt.Sprintf(`"%s/internal/middleware"`, pkg.Module.Module)
		if !strings.Contains(wireText, middlewareImport) {
			wireText = replaceFirstMarker(wireText, wireImportMarkers(), func(marker string) string {
				return fmt.Sprintf(`%s
	%s`, middlewareImport, marker)
			})
		}
		wireText = replaceFirstMarker(wireText, wireBuildMarkers(), func(marker string) string {
			return fmt.Sprintf(`middleware.NewOnRequest,
		middleware.NewOnRequestOptions,
		%s`, marker)
		})
		pterm.Info.Println("Added middleware.NewOnRequest and middleware.NewOnRequestOptions to wire.go (required by the CRUD router); customize OnRequestOptions to enable audit/permission handling")
	}

	return wireText
}
