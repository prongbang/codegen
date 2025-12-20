package template

import (
	"bytes"
	_ "embed"
	"strings"
	"text/template"

	"github.com/ettle/strcase"
	"github.com/prongbang/fibergen/pkg/common"
)

// API

//go:embed api_routers_template.tmpl
var ApiRoutersTemplate string

//go:embed api_template.tmpl
var ApiTemplate string

// APP

//go:embed app_template.tmpl
var AppTemplate string

// Casbin

//go:embed casbin_model_template.tmpl
var CasbinModelTemplate string

//go:embed casbin_policy_template.tmpl
var CasbinPolicyTemplate string

// CMD

//go:embed cmd_main_template.tmpl
var CmdMainTemplate string

// Configuration

//go:embed configuration_development_template.tmpl
var ConfigurationDevelopmentTemplate string

//go:embed configuration_environment_template.tmpl
var ConfigurationEnvironmentTemplate string

//go:embed configuration_production_template.tmpl
var ConfigurationProductionTemplate string

//go:embed configuration_template.tmpl
var ConfigurationTemplate string

// Core

//go:embed pkg_core_flag_template.tmpl
var CoreFlagTemplate string

//go:embed pkg_core_handler_template.tmpl
var CoreHandlerTemplate string

//go:embed pkg_core_sorting_template.tmpl
var CoreSortingTemplate string

//go:embed pkg_core_header_template.tmpl
var CoreHeaderTemplate string

//go:embed pkg_core_common_template.tmpl
var CoreCommonTemplate string

//go:embed pkg_core_jwt_template.tmpl
var CoreJWTTemplate string

//go:embed pkg_core_paging_template.tmpl
var CorePagingTemplate string

//go:embed pkg_core_params_template.tmpl
var CoreParamsTemplate string

//go:embed pkg_core_request_template.tmpl
var CoreRequestTemplate string

//go:embed pkg_core_response_template.tmpl
var CoreResponseTemplate string

//go:embed pkg_core_router_template.tmpl
var CoreRouterTemplate string

// MultipartX

//go:embed pkg_multipartx_template.tmpl
var MultipartXTemplate string

// CRUD

//go:embed crud_datasource_bun_template.tmpl
var CrudDatasourceBunTemplate string

//go:embed crud_router_bun_template.tmpl
var CrudRouterBunTemplate string

//go:embed crud_model_bun_template.tmpl
var CrudModelBunTemplate string

//go:embed crud_repository_bun_template.tmpl
var CrudRepositoryBunTemplate string

//go:embed crud_usecase_bun_template.tmpl
var CrudUseCaseBunTemplate string

//go:embed crud_datasource_sqlbuilder_template.tmpl
var CrudDatasourceSqlBuilderTemplate string

//go:embed crud_handler_template.tmpl
var CrudHandlerTemplate string

//go:embed crud_model_template.tmpl
var CrudModelTemplate string

//go:embed crud_provider_template.tmpl
var CrudProviderTemplate string

//go:embed crud_shared_provider_template.tmpl
var CrudSharedProviderTemplate string

//go:embed crud_repository_template.tmpl
var CrudRepositoryTemplate string

//go:embed crud_router_template.tmpl
var CrudRouterTemplate string

//go:embed crud_usecase_template.tmpl
var CrudUseCaseTemplate string

//go:embed crud_permission_template.tmpl
var CrudPermissionTemplate string

// Database

//go:embed database_db_template.tmpl
var DatabaseDbTemplate string

//go:embed database_drivers_template.tmpl
var DatabaseDriversTemplate string

//go:embed database_mongodb_template.tmpl
var DatabaseMongoDBTemplate string

//go:embed database_mariadb_template.tmpl
var DatabaseMariaDBTemplate string

//go:embed database_wire_gen_template.tmpl
var DatabaseWireGenTemplate string

//go:embed database_wire_template.tmpl
var DatabaseWireTemplate string

// Deployments

//go:embed deployments_api_compose_template.tmpl
var DeploymentsAPIComposeTemplate string

//go:embed deployments_dockerfile_template.tmpl
var DeploymentsDockerfileTemplate string

// Documentation

//go:embed docs_swagger_json_template.tmpl
var DocsSwaggerJSONTemplate string

//go:embed docs_swagger_yaml_template.tmpl
var DocsSwaggerYAMLTemplate string

//go:embed docs_template.tmpl
var DocsTemplate string

// Internal Packages

//go:embed internal_pkg_casbinx_template.tmpl
var InternalPkgCasbinxTemplate string

//go:embed internal_pkg_response_template.tmpl
var InternalPkgResponseTemplate string

//go:embed internal_pkg_validator_template.tmpl
var InternalPkgValidatorTemplate string

//go:embed internal_pkg_streamx_template.tmpl
var InternalPkgStreamXTemplate string

//go:embed internal_pkg_typex_template.tmpl
var InternalPkgTypeXTemplate string

// Internal Middleware

//go:embed internal_middleware_jwt_template.tmpl
var InternalMiddlewareJwtTemplate string

//go:embed internal_middleware_api_key_template.tmpl
var InternalMiddlewareApiKeyTemplate string

//go:embed internal_middleware_on_request_template.tmpl
var InternalMiddlewareOnRequestTemplate string

// Makefile

//go:embed makefile_template.tmpl
var MakefileTemplate string

// Mod

//go:embed mod_template.tmpl
var ModTemplate string

// Prototype

//go:embed prototype_datasource_template.tmpl
var PrototypeDatasourceTemplate string

//go:embed prototype_handler_template.tmpl
var PrototypeHandlerTemplate string

//go:embed prototype_model_template.tmpl
var PrototypeModelTemplate string

//go:embed prototype_provider_template.tmpl
var PrototypeProviderTemplate string

//go:embed prototype_shared_provider_template.tmpl
var PrototypeSharedProviderTemplate string

//go:embed prototype_repository_template.tmpl
var PrototypeRepositoryTemplate string

//go:embed prototype_router_template.tmpl
var PrototypeRouterTemplate string

//go:embed prototype_usecase_template.tmpl
var PrototypeUseCaseTemplate string

// Example

//go:embed example_datasource_template.tmpl
var ExampleDatasourceTemplate string

//go:embed example_handler_template.tmpl
var ExampleHandlerTemplate string

//go:embed example_model_template.tmpl
var ExampleModelTemplate string

//go:embed example_provider_template.tmpl
var ExampleProviderTemplate string

//go:embed example_repository_template.tmpl
var ExampleRepositoryTemplate string

//go:embed example_router_template.tmpl
var ExampleRouterTemplate string

//go:embed example_usecase_template.tmpl
var ExampleUseCaseTemplate string

//go:embed example_permission_template.tmpl
var ExamplePermissionTemplate string

// RequestX

//go:embed pkg_requestx_request_template.tmpl
var RequestXRequestTemplate string

// Schema

//go:embed pkg_schema_sql_template.tmpl
var SchemaSQLTemplate string

// StructX

//go:embed pkg_structx_template.tmpl
var StructXTemplate string

// Wire

//go:embed wire_gen_template.tmpl
var WireGenTemplate string

//go:embed wire_template.tmpl
var WireTemplate string

type Any map[string]interface{}

func RenderText[T any](tmpl string, data T) ([]byte, error) {
	funcs := template.FuncMap{
		"split":    strings.Split,
		"contains": strings.Contains,
		"sub": func(a, b int) int {
			return a - b
		},
	}

	t := template.Must(template.New("message").Funcs(funcs).Parse(tmpl))

	var buf bytes.Buffer
	err := t.Execute(&buf, data)
	if err != nil {
		panic(err)
	}

	return buf.Bytes(), nil
}

type Field struct {
	PrimaryKey bool
	Alias      string
	Name       string
	Type       string
	JsonTag    string
	DbTag      string
	Update     bool
	Create     bool
	CamelCase  string
	SnakeCase  string
	PascalCase string
}

type PrimaryField struct {
	Name    string
	Type    string
	JsonTag string
}

type Project struct {
	Imports      []string
	Fields       []Field
	Module       string
	Name         string
	Path         string
	ListQuery    string
	Alias        string
	SortFields   map[string]string
	PrimaryField PrimaryField
	Driver       string
}

func (w Project) DriverName() string {
	if w.Driver == "mariadb" {
		return "GetMariaDB"
	}
	return "GetUnknownDB"
}

func (w Project) PackageName() string {
	return common.ToLower(w.Name)
}

func (w Project) ProjectName() string {
	return strcase.ToPascal(w.Name)
}

func (w Project) ModelName() string {
	return strcase.ToPascal(w.Name)
}

func (w Project) RouteName() string {
	return strcase.ToKebab(w.Name)
}

func (w Project) KebabName() string {
	return strcase.ToKebab(w.Name)
}

func (w Project) TagsName() string {
	return strcase.ToSnake(w.Name)
}

func (w Project) TableName() string {
	return strcase.ToSnake(w.Name)
}
