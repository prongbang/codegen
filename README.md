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
- 🔌 **Optional Database** - Projects are generated without database code unless you ask for it; MariaDB, MongoDB and InfluxDB 3 can be added at any time
- 🧬 **SQL Migrations** - MariaDB projects ship a bun-backed migration runner that applies embedded SQL files on boot
- 📡 **MQTT Template** - Scaffold an MQTT forward service instead of a REST API
- 🛠️ **OpenAPI + Scalar** - Generate the OpenAPI spec from the routes and browse it with [Scalar](https://github.com/prongbang/goscalar) at `/docs`
- 🧩 **Modular Design** - Feature-based modules for better organization
- 🔧 **Wire Integration** - Dependency injection with Google Wire
- ⚡ **Fast Development** - Speed up your development workflow

## 📦 Installation

Latest version:
```shell
go install github.com/prongbang/codegen@v1.8.0
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
| `-driver` | `-d` | See below — the meaning depends on what you are generating |
| `-orm` | | ORM for generated data source: `bun` or `sqlbuilder` |

`-driver` does two different jobs:

| Used with | Meaning | Example |
|---|---|---|
| `-new` | Which databases to scaffold. Comma-separated. **Omit it for a project with no database code.** | `-d mariadb,influxdb3` |
| `-f` / `-sh` | Which database the generated CRUD data source targets | `-d mariadb` |

## 🧭 Commands

| Command | Description |
|---|---|
| `codegen -new <name> -mod <module>` | Create a new project |
| `codegen database init -d <databases>` | Add database code to an existing project |
| `codegen migration init` | Add the startup migration runner to an existing MariaDB project |
| `codegen migration new <name>` | Create the next migration file under `migrations/sql` |
| `codegen grpc init` | Add the gRPC scaffold |
| `codegen grpc server --new <service>` | Generate a gRPC server |
| `codegen grpc client --new <thirdparty>/<service>` | Generate a gRPC client |
| `codegen openapi -framework fiber ./...` | Print an OpenAPI spec for the current project |

Run `codegen <command> --help` for the flags of any command.

## 🚀 Quick Start

A typical run, from nothing to a working CRUD endpoint:

```shell
# 1. Create the project (add -d here if you already know you need a database)
codegen -new test_project -mod github.com/prongbang
cd test-project

# 2. Add a database whenever you need one
codegen database init -d mariadb

# 3. Generate a CRUD feature from a JSON spec (the spec must include the primary key)
mkdir -p spec && echo '{"id": "uuid", "name": "text", "active": 1}' > spec/brand.json
codegen -f brand -s spec/brand.json -d mariadb -orm bun

#    ...or straight from an existing table
codegen -f brand -dsn "user:password@tcp(127.0.0.1:3306)/dbname" -orm bun

# 4. Run it
go run cmd/api/main.go -env development
```

Generate an OpenAPI spec from the finished project (run inside the project):

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
│     └── openapi.json      # served by Scalar at /docs; regenerate with `make openapi-gen`
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
│     └── requestx
│         └── request.go
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

A project created without `-d` has no database code at all. Add one at any time —
run this **from the project root** (where `go.mod` is):

```shell
codegen database init -d mariadb
codegen database init -d influxdb3
codegen database init -d mariadb,mongodb,influxdb3
```

Supported: `mariadb`, `mongodb`, `influxdb3`.

#### What it changes

| File | Change |
|---|---|
| `internal/database/*.go` | Driver for each selected database, plus `drivers.go`, `wire.go`, `wire_gen.go` |
| `wire.go`, `wire_gen.go` | `CreateApp` gains its `database.Drivers` parameter |
| `cmd/api/main.go` | Builds the driver, closes it on shutdown, and runs migrations (MariaDB) |
| `migrations/`, `internal/database/migrations.go` | Migration runner and its SQL directory (MariaDB only) |
| `configuration/configuration.go` | Config struct fields for each database |
| `configuration/development.yml`, `production.yml` | Connection settings to fill in |
| `go.mod` | Database dependencies, then `go mod tidy` + `wire` |

So `cmd/api/main.go` goes from:

```go
apps := myproject.CreateApp()
apps.StartAPI()
```

to:

```go
dbDriver := database.NewDatabaseDriver()
defer dbDriver.Close()

apps := myproject.CreateApp(dbDriver)
apps.StartAPI()
```

#### Adding databases one at a time

It merges with what the project already has, so the existing databases are kept,
and re-running it for one that is already set up does nothing:

```shell
codegen database init -d mongodb     # project has mongodb
codegen database init -d influxdb3   # project has mongodb + influxdb3
codegen database init -d influxdb3   # "Nothing to do: influxdb3 already set up"
```

#### Fill in the connection settings

`database init` writes the settings with placeholder values — update
`configuration/development.yml` (and `production.yml`) before running:

```yaml
mariadb:
  host: "localhost"
  port: 3306
  database: "mariaDB"
  user: "root"
  pass: "password"

mongodb:
  host: "localhost"
  port: 27017
  database: "mongoDB"
  user: "root"
  pass: "password"

influxdb3:
  host: "http://localhost:8181"
  token: ""
  organization: ""
  database: "influxDB"
```

#### Using the driver

Inject `database.Drivers` into any provider and take the connection you need:

```go
func NewDataSource(driver database.Drivers) DataSource {
	return &dataSource{Driver: driver}
}

driver.GetMariaDB()   // *bun.DB          (-d mariadb)
driver.GetMongoDB()   // *mongo.Database  (-d mongodb)
driver.GetInfluxDB3() // *influxdb3.Client (-d influxdb3)
```

> [!NOTE]
> Only the databases you selected have a getter — `drivers.go` is generated from
> the selected set, so asking for one you did not add is a compile error rather
> than a nil connection at runtime.

### Database Migrations

A project with MariaDB ships a migration runner. SQL files live in
`migrations/sql`, are embedded into the binary, and are applied at boot by
`database.StartupMigrations(dbDriver)` in `cmd/api/main.go`.

```
migrations/
├── embed.go                  //go:embed sql/*.sql
├── README.md                 the conventions, generated with the project
└── sql/
    └── 0001_init.up.sql      no-op baseline, keeps the embed non-empty
internal/database/migrations.go   the runner (bun/migrate)
```

Create the next migration — run this **from the project root**:

```shell
codegen migration new master_room_setup
# → migrations/sql/0002_master_room_setup.up.sql
```

It picks the next free version, matches the zero-padding already in use, and
writes a skeleton with the rules in its header. Fill it in and rebuild — the
runner only sees files compiled into the binary.

Projects that already have MariaDB but predate the runner get it with:

```shell
codegen migration init
```

That writes the four files above and adds the `StartupMigrations` call to
`cmd/api/main.go`. It never overwrites a file that already exists, so it is safe
to re-run.

#### How it behaves at runtime

- **Every boot.** `RUN_MIGRATIONS=false` (or `0`, `f`) opts out — for a replica
  that must never touch the schema.
- **Fail-fast.** Any error exits the process before the API starts, bounded by a
  5-minute timeout.
- **Locked.** bun holds a lock in `migrations_locks`, so concurrent replicas
  cannot race. Applied versions are recorded in `migrations`.
- **Retried on failure.** A migration is recorded only *after* its SQL succeeds,
  so a failed one runs again next boot. That is why every file must be
  idempotent and survive being re-run from its first statement.

#### Writing one

Two rules the generated `migrations/README.md` covers in full, and that are not
style preferences:

- **Separate every statement with a line containing exactly `--bun:split`.** Each
  chunk is sent as one query; two statements in a chunk fail with errno 1064.
- **`ADD COLUMN IF NOT EXISTS` is MariaDB-only** and a syntax error on MySQL.
  Guard through `information_schema` plus `PREPARE`/`EXECUTE` instead.

Append `.tx` before the suffix — `0002_seed.tx.up.sql` — to run a file inside a
transaction. Useful for DML; DDL auto-commits regardless.

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

### 3. Generate Features CRUD from JSON Spec

Generate CRUD operations from JSON specifications:

> [!IMPORTANT]
> CRUD data sources talk to `internal/database`, so the project needs a database
> first. On a project created without `-d` the generated code will not compile
> (`no required module provides package .../internal/database`). Add one with
> `codegen database init -d mariadb` before generating CRUD.

#### 3.1 Define Spec File

Create `spec/auth.json`. The field named `id` (case-insensitive) becomes the
primary key, and the spec must contain one — without it the generated code will
not compile:

```json
{
    "id": "uuid",
    "accessToken": "JWT",
    "expired": 1234567,
    "date": "2024-10-15T14:30:00Z"
}
```

Field values are samples that decide the Go type:

| Sample value | Go type |
|---|---|
| `"uuid"`, `"text"`, any string | `string` |
| `1234567` | `int64` |
| `1.5` | `float64` |
| `true` | `bool` |
| `"2024-10-15T14:30:00Z"` | `*time.Time` |

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
