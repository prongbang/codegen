package option

import (
	"testing"

	"github.com/prongbang/codegen/template"
)

func TestOptionsAndSpec(t *testing.T) {
	opt := Options{
		Project: "demo",
		Module:  "github.com/acme/demo",
		Feature: "device",
		Shared:  "auth",
		Spec:    "spec/device.json",
		Driver:  "mariadb",
		Orm:     "bun",
	}

	if opt.Project != "demo" || opt.Module != "github.com/acme/demo" {
		t.Fatalf("unexpected options: %+v", opt)
	}

	spec := Spec{
		Imports: []string{"time"},
		Driver:  "mariadb",
		Orm:     "bun",
		Alias:   "dv",
		Fields: []template.Field{
			{Name: "Id", Type: "int64"},
		},
		PrimaryField: template.PrimaryField{
			Name: "Id",
			Type: "int64",
		},
	}

	if len(spec.Imports) != 1 || spec.Imports[0] != "time" {
		t.Fatalf("unexpected imports: %+v", spec.Imports)
	}
	if spec.PrimaryField.Name != "Id" || spec.Fields[0].Name != "Id" {
		t.Fatalf("unexpected spec: %+v", spec)
	}
}
