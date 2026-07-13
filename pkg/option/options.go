package option

import (
	"github.com/prongbang/codegen/template"
)

type Options struct {
	Project   string
	Module    string
	Package   string
	Feature   string
	Shared    string
	Spec      string
	Dsn       string
	Table     string
	Driver    string
	Orm       string
	Framework string
	Template  string
	OpenAPI   bool
	Patterns  []string
}

type Spec struct {
	Imports      []string
	Driver       string
	Orm          string
	Alias        string
	Table        string
	Fields       []template.Field
	PrimaryField template.PrimaryField
}
