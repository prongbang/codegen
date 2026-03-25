package generate

import (
	"fmt"

	"github.com/prongbang/codegen/pkg/option"
)

// Generator is the interface
type Generator interface {
	Generate(opt option.Options) error
}

type generator struct {
	ProjectGenerator Generator
	FeatureGenerator Generator
	SharedGenerator  Generator
}

func (f *generator) Generate(opt option.Options) error {
	switch {
	case opt.Project != "" && opt.Module != "":
		return f.ProjectGenerator.Generate(opt)
	case opt.Feature != "":
		return f.FeatureGenerator.Generate(opt)
	case opt.Shared != "":
		return f.SharedGenerator.Generate(opt)
	default:
		return fmt.Errorf("unsupported option combination: %+v", opt)
	}
}

// NewGenerator is new instance with func
func NewGenerator(projectGenerator Generator, featureGenerator Generator, sharedGenerator Generator) Generator {
	return &generator{
		ProjectGenerator: projectGenerator,
		FeatureGenerator: featureGenerator,
		SharedGenerator:  sharedGenerator,
	}
}
