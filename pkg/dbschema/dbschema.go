package dbschema

import (
	"database/sql"
	"fmt"
	"strings"

	_ "github.com/go-sql-driver/mysql"
)

// Column describes a table column read from the database schema.
type Column struct {
	Name       string
	DataType   string
	ColumnType string
	Nullable   bool
	PrimaryKey bool
}

// Introspector reads table schemas from a live database.
type Introspector interface {
	Columns(table string) ([]Column, error)
	Tables() ([]string, error)
	Close() error
}

type mysqlIntrospector struct {
	db *sql.DB
}

// NewMySQL connects to a MySQL/MariaDB database using a go-sql-driver DSN,
// e.g. user:password@tcp(127.0.0.1:3306)/dbname
func NewMySQL(dsn string) (Introspector, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("connect database: %s", err.Error())
	}
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("connect database: %s", err.Error())
	}
	return &mysqlIntrospector{db: db}, nil
}

func (m *mysqlIntrospector) Columns(table string) ([]Column, error) {
	rows, err := m.db.Query(`
		SELECT COLUMN_NAME, DATA_TYPE, COLUMN_TYPE, IS_NULLABLE, COLUMN_KEY
		FROM information_schema.COLUMNS
		WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = ?
		ORDER BY ORDINAL_POSITION`, table)
	if err != nil {
		return nil, fmt.Errorf("read schema: %s", err.Error())
	}
	defer func() { _ = rows.Close() }()

	columns := []Column{}
	for rows.Next() {
		var name, dataType, columnType, nullable, columnKey string
		if err := rows.Scan(&name, &dataType, &columnType, &nullable, &columnKey); err != nil {
			return nil, fmt.Errorf("read schema: %s", err.Error())
		}
		columns = append(columns, Column{
			Name:       name,
			DataType:   dataType,
			ColumnType: columnType,
			Nullable:   strings.EqualFold(nullable, "YES"),
			PrimaryKey: strings.EqualFold(columnKey, "PRI"),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read schema: %s", err.Error())
	}
	if len(columns) == 0 {
		return nil, fmt.Errorf("table %q not found in database", table)
	}
	return columns, nil
}

func (m *mysqlIntrospector) Tables() ([]string, error) {
	rows, err := m.db.Query(`
		SELECT TABLE_NAME
		FROM information_schema.TABLES
		WHERE TABLE_SCHEMA = DATABASE() AND TABLE_TYPE = 'BASE TABLE'
		ORDER BY TABLE_NAME`)
	if err != nil {
		return nil, fmt.Errorf("read schema: %s", err.Error())
	}
	defer func() { _ = rows.Close() }()

	tables := []string{}
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("read schema: %s", err.Error())
		}
		tables = append(tables, name)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read schema: %s", err.Error())
	}
	return tables, nil
}

func (m *mysqlIntrospector) Close() error {
	return m.db.Close()
}

// GoType maps a MySQL column type to the Go type used in generated models.
func GoType(c Column) string {
	dataType := strings.ToLower(c.DataType)
	if dataType == "tinyint" && strings.HasPrefix(strings.ToLower(c.ColumnType), "tinyint(1)") {
		return "*bool"
	}
	switch dataType {
	case "tinyint", "smallint", "mediumint", "int", "integer", "bigint", "year", "bit":
		return "int64"
	case "decimal", "numeric", "float", "double", "real":
		return "float64"
	case "date", "datetime", "timestamp", "time":
		return "*time.Time"
	case "bool", "boolean":
		return "*bool"
	default:
		// char, varchar, text, enum, set, json, uuid, binary, blob, ...
		return "string"
	}
}
