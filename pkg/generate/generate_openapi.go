package generate

import (
	"fmt"
	"os"
	"strings"

	"github.com/prongbang/codegen/internal/analyzer"
	openapigen "github.com/prongbang/codegen/internal/generator"
	"github.com/prongbang/codegen/internal/loader"
	"github.com/prongbang/codegen/pkg/option"
)

type openAPIGenerator struct{}

func (g *openAPIGenerator) Generate(opt option.Options) error {
	framework := strings.TrimSpace(strings.ToLower(opt.Framework))
	switch framework {
	case "fiber":
		mod, err := loader.Load(opt.Patterns)
		if err != nil {
			return err
		}
		ops, err := analyzer.AnalyzeFiber(mod)
		if err != nil {
			return err
		}
		if len(ops) == 0 {
			return fmt.Errorf("no fiber routes found")
		}
		spec, err := openapigen.Build(mod, ops)
		if err != nil {
			return err
		}
		_, err = os.Stdout.Write(spec)
		if err == nil {
			_, err = os.Stdout.Write([]byte("\n"))
		}
		return err
	default:
		return fmt.Errorf("unsupported openapi framework: %s", opt.Framework)
	}
}

func NewOpenAPIGenerator() Generator {
	return &openAPIGenerator{}
}
