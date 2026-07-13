package dbschema_test

import (
	"testing"

	"github.com/prongbang/codegen/pkg/dbschema"
)

func TestGoType(t *testing.T) {
	cases := []struct {
		name     string
		column   dbschema.Column
		expected string
	}{
		{"bigint", dbschema.Column{DataType: "bigint", ColumnType: "bigint(20)"}, "int64"},
		{"int", dbschema.Column{DataType: "int", ColumnType: "int(11)"}, "int64"},
		{"tinyint flag", dbschema.Column{DataType: "tinyint", ColumnType: "tinyint(1)"}, "*bool"},
		{"tinyint number", dbschema.Column{DataType: "tinyint", ColumnType: "tinyint(4)"}, "int64"},
		{"decimal", dbschema.Column{DataType: "decimal", ColumnType: "decimal(10,2)"}, "float64"},
		{"double", dbschema.Column{DataType: "double", ColumnType: "double"}, "float64"},
		{"varchar", dbschema.Column{DataType: "varchar", ColumnType: "varchar(255)"}, "string"},
		{"text", dbschema.Column{DataType: "text", ColumnType: "text"}, "string"},
		{"json", dbschema.Column{DataType: "json", ColumnType: "json"}, "string"},
		{"enum", dbschema.Column{DataType: "enum", ColumnType: "enum('a','b')"}, "string"},
		{"datetime", dbschema.Column{DataType: "datetime", ColumnType: "datetime"}, "*time.Time"},
		{"timestamp", dbschema.Column{DataType: "timestamp", ColumnType: "timestamp"}, "*time.Time"},
		{"date", dbschema.Column{DataType: "date", ColumnType: "date"}, "*time.Time"},
		{"boolean", dbschema.Column{DataType: "boolean", ColumnType: "boolean"}, "*bool"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if actual := dbschema.GoType(c.column); actual != c.expected {
				t.Fatalf("GoType(%s) = %s, expected %s", c.column.DataType, actual, c.expected)
			}
		})
	}
}

func TestNewMySQLInvalidDsn(t *testing.T) {
	if _, err := dbschema.NewMySQL("invalid-dsn"); err == nil {
		t.Fatal("expected error for invalid DSN")
	}
}
