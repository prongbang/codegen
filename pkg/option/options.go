package option

import (
	"github.com/prongbang/fibergen/template"
)

type Options struct {
	Project string
	Module  string
	Feature string
	Shared  string
	Spec    string
	Driver  string
	Orm     string
}

type Spec struct {
	Imports      []string
	Driver       string
	Orm          string
	Alias        string
	Fields       []template.Field
	PrimaryField template.PrimaryField
}
