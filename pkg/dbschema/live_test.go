package dbschema_test

import (
	"os"
	"testing"

	"github.com/prongbang/codegen/pkg/dbschema"
)

func TestLiveIntrospection(t *testing.T) {
	dsn := os.Getenv("CODEGEN_TEST_DSN")
	if dsn == "" {
		t.Skip("CODEGEN_TEST_DSN not set")
	}
	in, err := dbschema.NewMySQL(dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer func() { _ = in.Close() }()

	tables, err := in.Tables()
	if err != nil {
		t.Fatalf("tables: %v", err)
	}
	t.Logf("tables: %v", tables)

	for _, table := range tables {
		cols, err := in.Columns(table)
		if err != nil {
			t.Fatalf("columns(%s): %v", table, err)
		}
		for _, c := range cols {
			t.Logf("%s.%s %s (%s) pk=%v null=%v -> %s", table, c.Name, c.DataType, c.ColumnType, c.PrimaryKey, c.Nullable, dbschema.GoType(c))
		}
	}
}
