package generate

import "github.com/prongbang/codegen/pkg/option"

type Binding interface {
	Bind(pkg option.Package) error
}
