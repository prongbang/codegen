# Codegen 🚀

[![Coverage](https://codecov.io/gh/prongbang/codegen/graph/badge.svg)](https://codecov.io/gh/prongbang/codegen)
[![Go Report Card](https://goreportcard.com/badge/github.com/prongbang/codegen)](https://goreportcard.com/report/github.com/prongbang/codegen)
[![Go Reference](https://pkg.go.dev/badge/github.com/prongbang/codegen.svg)](https://pkg.go.dev/github.com/prongbang/codegen)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Version](https://img.shields.io/github/v/release/prongbang/codegen)](https://github.com/prongbang/codegen/releases)

> Generate Clean Architecture for REST API with support for Fiber Web Framework and gRPC in Golang. Speed up your development with automatic code generation.


## ✨ Features

- 🏗️ **Clean Architecture** - Automatically generates layered architecture structure
- 🔌 **Fiber Framework Support** - Optimized for the Fiber web framework
- 🌐 **gRPC Support** - Built-in support for gRPC services
- 🔄 **CRUD Generation** - Generate CRUD operations from JSON specifications
- 🗄️ **Database Schema Generation** - Generate CRUD directly from a live MySQL/MariaDB schema
- 🛠️ **Open API Generation** - Generate Open API without configuration
- 🧩 **Modular Design** - Feature-based modules for better organization
- 🔧 **Wire Integration** - Dependency injection with Google Wire
- ⚡ **Fast Development** - Speed up your development workflow

## 📦 Installation

Latest version:
```shell
go install github.com/prongbang/codegen@v1.6.1
```

## 🚩 CLI Flags

| Flag | Alias | Description |
|---|---|---|
| `-new` | `-n` | Project name (create a new project) |
| `-mod` | `-m` | Module name, e.g. `github.com/prongbang` |
| `-template` | `-t` | Project template (currently supported: `mqtt`) |
| `-feature` | `-f` | Feature name (generate a feature module) |
| `-shared` | `-sh` | Shared module name |
| `-package` | `-p` | Package prototype name |
| `-spec` | `-s` | JSON spec file for CRUD generation |
| `-dsn` | | Database connection string — generate CRUD from a live database schema |
| `-table` | `-tb` | Table name (optional, defaults to the feature name) |
| `-driver` | `-d` | Database driver, e.g. `mariadb`, `mysql` |
| `-orm` | | ORM for generated data source: `bun` or `sqlbuilder` |

## 🚀 Quick Start

Generate OpenAPI spec from a Fiber codebase:

```shell
codegen openapi -framework fiber ./... > docs/openapi.json
```

### 1. Create a New Project - API

Generate a new project with module structure:

```shell
codegen -new test_project -mod github.com/prongbang
```

Parameters:
- `-new`: Project name
- `-mod`: Module name
- `-d`, `-driver`: *(optional)* databases to scaffold — `mariadb`, `mongodb`, `influxdb3`.
  Separate several with a comma. **Omit it and the project is generated with no
  database code at all**; you can add one later with `codegen database init`.

```shell
# no database code
codegen -new test_project -mod github.com/prongbang

# with one or more databases
codegen -new test_project -mod github.com/prongbang -d mariadb
codegen -new test_project -mod github.com/prongbang -d mariadb,influxdb3
```

This creates the following structure:

```
.
├── Makefile
├── cmd
│     └── api
│         └── main.go
├── configuration
│     ├── configuration.go
│     ├── development.yml
│     ├── environment.go
│     └── production.yml
├── deployments
│     ├── Dockerfile
│     └── api-prod.yml
├── docs
│     └── apispec
│         ├── docs.go
│         ├── swagger.json
│         └── swagger.yaml
├── go.mod
├── go.sum
├── internal
│     ├── app
│     │     ├── api
│     │     │     ├── api.go
│     │     │     ├── health
│     │     │     │     ├── health.go
│     │     │     │     ├── handler.go
│     │     │     │     ├── provider.go
│     │     │     │     ├── router.go
│     │     │     │     └── usecase.go
│     │     │     └── routers.go
│     │     └── app.go
│     ├── database          # only with -d; one file per selected database
│     │     ├── db.go       # -d mariadb
│     │     ├── drivers.go
│     │     ├── mariadb.go  # -d mariadb
│     │     ├── mongodb.go  # -d mongodb
│     │     ├── influxdb3.go # -d influxdb3
│     │     ├── wire.go
│     │     └── wire_gen.go
│     ├── middleware
│     │     ├── api_key.go
│     │     ├── jwt.go
│     │     └── on_request.go
│     ├── pkg
│           ├── casbinx
│           │     └── casbinx.go
│           ├── response
│           │     └── response.go
│           └── validator
│               └── validator.go
│
├── pkg
│     ├── core
│     │     ├── common.go
│     │     ├── flag.go
│     │     ├── handler.go
│     │     ├── header.go
│     │     ├── jwt.go
│     │     ├── paging.go
│     │     ├── params.go
│     │     ├── request.go
│     │     ├── response.go
│     │     ├── router.go
│     │     └── sorting.go
│     ├── multipartx
│     │     └── multipartx.go
│     ├── requestx
│     │     └── request.go
│     ├── schema
│     │     └── sql.go
│     ├── streamx
│     │     └── streamx.go
│     ├── structx
│     │     └── structx.go
│     └── typex
│         └── typex.go
├── policy
│     ├── model.conf
│     └── policy.csv
├── spec
│     └── promotion.json
├── wire.go
└── wire_gen.go
```

### Create a New Project - MQTT

Generate a project from a predefined template using the `-template` (`-t`) flag:

```shell
codegen -new test_project -mod github.com/prongbang -template mqtt
```

Parameters:
- `-new`: Project name
- `-mod`: Module name
- `-template`: Template name (currently supported: `mqtt`)

The `mqtt` template scaffolds an MQTT forward service that subscribes to an
upstream MQTT broker and forwards messages to a built-in (mochi) MQTT server:

```
.
├── Makefile
├── cmd
│     └── api
│         └── main.go
├── configuration
│     ├── configuration.go
│     ├── development.yml
│     ├── environment.go
│     └── production.yml
├── go.mod
└── internal
      └── app
          └── mqtt
              ├── broker
              │     └── mqtt_broker.go
              ├── forward
              │     ├── forward.go
              │     ├── mqtt_forward.go
              │     └── payload.go
              └── subscribe
                    └── mqtt_subscribe.go
```

After generating, run `go mod tidy` inside the project, then start it with
`make run` (or `go run cmd/api/main.go -env development`).

### Add a Database Later

A project created without `-d` has no database code. Add one at any time:

```shell
codegen database init -d mariadb
codegen database init -d influxdb3
codegen database init -d mariadb,mongodb,influxdb3
```

Supported: `mariadb`, `mongodb`, `influxdb3`.

This generates `internal/database` for the selected databases and wires them in:

- adds the driver files and regenerates `internal/database/drivers.go`
- gives `CreateApp` its `database.Drivers` parameter in `wire.go` / `wire_gen.go`
- makes `cmd/api/main.go` build and close the driver
- appends the config struct and the `development.yml` / `production.yml` sections
- resolves the dependencies and runs `go mod tidy` + `wire`

Running it again for a database that is already set up does nothing, and adding a
second database keeps the existing one:

```shell
codegen database init -d mongodb     # project now has mongodb
codegen database init -d influxdb3   # project now has mongodb + influxdb3
```

### 1.1 Initial gRPC

```sh
codegen grpc init
```

This creates the initial gRPC scaffold under `internal/app/grpc` and generates the default `health` service.

### 1.2 Generate gRPC Server

```sh
codegen grpc server --new device
```

This creates:

```
internal/app/grpc/device/v1
├── device.proto
├── provider.go
└── server.go
```

It also updates:
- `internal/app/grpc/servers.go`
- `wire.go`

Then it runs:

```sh
make gen service=device version=v1
wire
```

### 1.3 Generate gRPC Client

Use the format `<thirdparty>/<service>`:

```sh
codegen grpc client --new core/device
```

This creates:

```
internal/thirdparty
├── Makefile
└── core
    ├── clients.go
    └── device
        └── v1
            ├── client.go
            └── device.proto
```

It also updates:
- `internal/thirdparty/core/clients.go`
- `internal/thirdparty/Makefile`

Then it runs:

```sh
make gen service=device version=v1 thirdparty=core
```

### 2. Generate Features Prototype

Generate a new feature module (run inside `internal/app/api` of your project):

```shell
codegen -f promotion
```

This creates:
```
test-project/internal/app/api/promotion
├── datasource.go
├── handler.go
├── permission.go
├── promotion.go
├── provider.go
├── repository.go
├── router.go
└── usecase.go
```

### 3. Generate Features CRUD and Swagger from JSON Spec

Generate CRUD operations from JSON specifications:

#### 3.1 Define Spec File

Create `spec/auth.json`:
```json
{
    "accessToken": "JWT",
    "expired": 1234567,
    "date": "2024-10-15T14:30:00Z"
}
```

#### 3.2 Generate CRUD

- SQL Builder

```shell
codegen -f auth -s spec/auth.json -d mariadb -orm sqlbuilder
```

- Bun

```shell
codegen -f auth -s spec/auth.json -d mariadb -orm bun
```

This generates complete CRUD operations based on your JSON structure.

```
test-project/internal/app/api/auth
├── auth.go
├── datasource.go
├── handler.go
├── permission.go
├── provider.go
├── repository.go
├── router.go
└── usecase.go
```

### 4. Generate Features CRUD from Database Schema

Instead of writing a JSON spec, point codegen at a live MySQL/MariaDB database
with `-dsn` and it reads the table schema from `information_schema` to generate
the model and CRUD code:

```shell
codegen -f brand -dsn "user:password@tcp(127.0.0.1:3306)/dbname" -orm bun
```

By default the table name is the snake_case of the feature name. If the table
name differs, override it with `-table`:

```shell
codegen -f company -dsn "user:password@tcp(127.0.0.1:3306)/dbname" -table company_group -orm bun
```

Parameters:
- `-dsn`: Database connection string in [go-sql-driver DSN format](https://github.com/go-sql-driver/mysql#dsn-data-source-name)
- `-table` (`-tb`): Table name (optional, defaults to the feature name)
- `-orm`: `bun` or `sqlbuilder`
- `-d`: Driver (optional, defaults to `mysql` when `-dsn` is set)

Column types are mapped automatically:

| MySQL type | Go type |
|---|---|
| `int`, `bigint`, `smallint`, `tinyint`, `year`, `bit` | `int64` |
| `tinyint(1)`, `boolean` | `*bool` |
| `decimal`, `float`, `double` | `float64` |
| `date`, `datetime`, `timestamp`, `time` | `*time.Time` |
| `varchar`, `text`, `enum`, `json`, others | `string` |

The primary key is detected from the table's `PRIMARY KEY` definition, and
column names are used as-is for the generated SQL and `db`/`bun` tags. For
example, a `brand` table generates:

```go
type Brand struct {
    bun.BaseModel `bun:"table:brand,alias:b" json:"-" swaggerignore:"true"`
    Id        string     `bun:"id,pk" json:"id" db:"id"`
    Name      string     `bun:"name" json:"name" db:"name"`
    Active    int64      `bun:"active" json:"active" db:"active"`
    CreatedAt *time.Time `bun:"created_at" json:"createdAt" db:"created_at"`
    UpdatedAt *time.Time `bun:"updated_at" json:"updatedAt" db:"updated_at"`
}
```

This also works for shared modules with `-sh`:

```shell
codegen -sh brand -dsn "user:password@tcp(127.0.0.1:3306)/dbname" -orm bun
```

> [!NOTE]
> The DSN is only used at generation time to read the schema — it is never
> written into the generated code.

### 5. Generate Shared Prototype

```shell
codegen -sh promotion
```
This generates shared prototype

```
test-project/internal/shared/promotion
├── datasource.go
├── promotion.go
├── provider.go
└── repository.go
```

### 6. Generate Shared CRUD

- SQL Builder

```shell
codegen -sh promotion -s spec/promotion.json -d mariadb -orm sqlbuilder
```

- Bun

```shell
codegen -sh promotion -s spec/promotion.json -d mariadb -orm bun
```

- From a database schema

```shell
codegen -sh promotion -dsn "user:password@tcp(127.0.0.1:3306)/dbname" -orm bun
```

This generates shared CRUD operations based on your JSON structure or database schema.

```
test-project/internal/shared/promotion
├── datasource.go
├── promotion.go
├── provider.go
└── repository.go
```


## 🤝 Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 💖 Support

If you find this tool helpful, please consider buying me a coffee:

[!["Buy Me A Coffee"](https://www.buymeacoffee.com/assets/img/custom_images/orange_img.png)](https://www.buymeacoffee.com/prongbang)

## 🔗 Related Projects

- [Fiber](https://github.com/gofiber/fiber) - Express-inspired web framework
- [Wire](https://github.com/google/wire) - Compile-time dependency injection
- [gRPC](https://grpc.io/) - High performance RPC framework

---
