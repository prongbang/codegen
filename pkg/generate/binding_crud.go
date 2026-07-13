package generate

import (
	"fmt"
	"strings"

	"github.com/prongbang/codegen/pkg/option"
)

// ensureCrudWireProviders binds the providers required by generated CRUD code
// to wire.go if they are not already bound: database.NewDB provides
// dbre.AppIDB for datasources, and middleware.NewOnRequestGuard provides
// middleware.OnRequest for feature routers. Only applied when the package was
// generated from a spec (CRUD); prototype packages do not need them, and wire
// fails on unused providers.
func ensureCrudWireProviders(wireText string, pkg option.Package, withOnRequest bool) string {
	if len(pkg.Spec.Fields) == 0 {
		return wireText
	}

	if !strings.Contains(wireText, "database.NewDB") {
		wireText = replaceFirstMarker(wireText, wireBuildMarkers(), func(marker string) string {
			return fmt.Sprintf(`database.NewDB,
		%s`, marker)
		})
	}

	if withOnRequest {
		middlewareImport := fmt.Sprintf(`"%s/internal/middleware"`, pkg.Module.Module)
		if !strings.Contains(wireText, middlewareImport) {
			wireText = replaceFirstMarker(wireText, wireImportMarkers(), func(marker string) string {
				return fmt.Sprintf(`%s
	%s`, middlewareImport, marker)
			})
		}
		if !strings.Contains(wireText, "middleware.NewOnRequestGuard") {
			wireText = replaceFirstMarker(wireText, wireBuildMarkers(), func(marker string) string {
				return fmt.Sprintf(`middleware.NewOnRequestGuard,
		%s`, marker)
			})
		}
	}

	return wireText
}
