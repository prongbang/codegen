package generate

import (
	"github.com/ettle/strcase"
	"github.com/prongbang/codegen/pkg/common"
	"github.com/prongbang/codegen/pkg/dbschema"
	"github.com/prongbang/codegen/pkg/option"
	"github.com/prongbang/codegen/template"
)

func generateSpecFromDatabase(opt option.Options) (option.Spec, error) {
	introspector, err := dbschema.NewMySQL(opt.Dsn)
	if err != nil {
		return option.Spec{}, err
	}
	defer func() { _ = introspector.Close() }()

	table := opt.Table
	if table == "" {
		table = strcase.ToSnake(opt.Feature)
	}
	columns, err := introspector.Columns(table)
	if err != nil {
		return option.Spec{}, err
	}
	return specFromColumns(opt, table, columns), nil
}

func specFromColumns(opt option.Options, table string, columns []dbschema.Column) option.Spec {
	spec := option.Spec{
		Driver: opt.Driver,
		Orm:    opt.Orm,
		Table:  table,
	}
	alias := common.Abbrev(opt.Feature)
	imports := []string{}
	fields := []template.Field{}
	for _, col := range columns {
		// Keep the real column name for SQL/db tags; templates use SnakeCase
		// and DbTag as the column name in generated queries.
		snakeTag := col.Name
		camelTag := strcase.ToCamel(col.Name)
		vars := strcase.ToPascal(col.Name)
		typeValue := dbschema.GoType(col)

		if typeValue == "*time.Time" && !contains(imports, "time") {
			imports = append(imports, "time")
		}

		// Audit fields are set from UserRequestInfo by the usecase template,
		// so exclude them from the create/update copy loops.
		isAudit := vars == "CreatedBy" || vars == "UpdatedBy"

		fields = append(fields, template.Field{PrimaryKey: col.PrimaryKey, Alias: alias, CamelCase: camelTag, SnakeCase: snakeTag, PascalCase: vars, Name: vars, Type: typeValue, JsonTag: camelTag, DbTag: col.Name, Update: !isAudit, Create: !isAudit})

		if col.PrimaryKey && spec.PrimaryField.Name == "" {
			spec.PrimaryField = template.PrimaryField{
				Name:    vars,
				Type:    typeValue,
				JsonTag: camelTag,
			}
		}
	}
	spec.Alias = alias
	spec.Fields = fields
	spec.Imports = imports
	return spec
}

func contains(items []string, target string) bool {
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
}
