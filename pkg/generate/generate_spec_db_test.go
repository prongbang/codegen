package generate

import (
	"testing"

	"github.com/prongbang/codegen/pkg/dbschema"
	"github.com/prongbang/codegen/pkg/option"
)

func TestSpecFromColumns(t *testing.T) {
	opt := option.Options{Feature: "user", Driver: "mysql", Orm: "bun"}
	columns := []dbschema.Column{
		{Name: "id", DataType: "bigint", ColumnType: "bigint(20)", PrimaryKey: true},
		{Name: "email", DataType: "varchar", ColumnType: "varchar(255)"},
		{Name: "is_active", DataType: "tinyint", ColumnType: "tinyint(1)", Nullable: true},
		{Name: "created_at", DataType: "datetime", ColumnType: "datetime", Nullable: true},
		{Name: "created_by", DataType: "varchar", ColumnType: "varchar(50)", Nullable: true},
	}

	spec := specFromColumns(opt, "users", columns)

	if spec.Table != "users" {
		t.Fatalf("expected table users, got %s", spec.Table)
	}
	if spec.Driver != "mysql" || spec.Orm != "bun" {
		t.Fatalf("unexpected driver/orm: %s/%s", spec.Driver, spec.Orm)
	}
	if len(spec.Fields) != 5 {
		t.Fatalf("expected 5 fields, got %d", len(spec.Fields))
	}
	if spec.PrimaryField.Name != "Id" || spec.PrimaryField.Type != "int64" || spec.PrimaryField.JsonTag != "id" {
		t.Fatalf("unexpected primary field: %+v", spec.PrimaryField)
	}

	byName := map[string]string{}
	for _, f := range spec.Fields {
		byName[f.Name] = f.Type
	}
	if byName["Email"] != "string" {
		t.Fatalf("expected Email string, got %s", byName["Email"])
	}
	if byName["IsActive"] != "*bool" {
		t.Fatalf("expected IsActive *bool, got %s", byName["IsActive"])
	}
	if byName["CreatedAt"] != "*time.Time" {
		t.Fatalf("expected CreatedAt *time.Time, got %s", byName["CreatedAt"])
	}

	if len(spec.Imports) != 1 || spec.Imports[0] != "time" {
		t.Fatalf("expected imports [time], got %v", spec.Imports)
	}

	if spec.Fields[0].DbTag != "id" || spec.Fields[3].DbTag != "created_at" {
		t.Fatalf("expected db tags to keep column names, got %s/%s", spec.Fields[0].DbTag, spec.Fields[3].DbTag)
	}
	if spec.Fields[3].JsonTag != "createdAt" {
		t.Fatalf("expected json tag createdAt, got %s", spec.Fields[3].JsonTag)
	}

	// Audit fields must be excluded from create/update copy loops
	// (the usecase template sets them from UserRequestInfo).
	if spec.Fields[4].Name != "CreatedBy" || spec.Fields[4].Update || spec.Fields[4].Create {
		t.Fatalf("expected CreatedBy with Update/Create=false, got %+v", spec.Fields[4])
	}
	if !spec.Fields[3].Update || !spec.Fields[3].Create {
		t.Fatalf("expected CreatedAt to keep Update/Create=true, got %+v", spec.Fields[3])
	}
}

func TestSpecFromColumnsNoPrimaryKey(t *testing.T) {
	opt := option.Options{Feature: "log", Driver: "mysql"}
	columns := []dbschema.Column{
		{Name: "message", DataType: "text", ColumnType: "text"},
	}
	spec := specFromColumns(opt, "log", columns)
	if spec.PrimaryField.Name != "" {
		t.Fatalf("expected empty primary field, got %+v", spec.PrimaryField)
	}
}

func TestLoadSpecFromDatabaseInvalidDsn(t *testing.T) {
	opt := option.Options{Feature: "user", Dsn: "invalid-dsn"}
	if _, err := loadSpec(nil, opt); err == nil {
		t.Fatal("expected error for invalid DSN")
	}
}
