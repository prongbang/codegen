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
- 🛠️ **Open API Generation** - Generate Open API without configuration
- 🧩 **Modular Design** - Feature-based modules for better organization
- 🔧 **Wire Integration** - Dependency injection with Google Wire
- ⚡ **Fast Development** - Speed up your development workflow

## 📦 Installation

Latest version:
```shell
go install github.com/prongbang/codegen@v1.5.4
```

## 🚀 Quick Start

Generate OpenAPI spec from a Fiber codebase:

```shell
codegen openapi -framework fiber ./... > docs/openapi.json
```

### 1. Create a New Project

Generate a new project with module structure:

```shell
codegen -new test_project -mod github.com/prongbang
```

Parameters:
- `-new`: Project name
- `-mod`: Module name

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
│     │     │     ├── example
│     │     │     │     ├── datasource.go
│     │     │     │     ├── example.go
│     │     │     │     ├── handler.go
│     │     │     │     ├── permission.go
│     │     │     │     ├── provider.go
│     │     │     │     ├── repository.go
│     │     │     │     ├── router.go
│     │     │     │     └── usecase.go
│     │     │     └── routers.go
│     │     └── app.go
│     ├── database
│     │     ├── db.go
│     │     ├── drivers.go
│     │     ├── mariadb.go
│     │     ├── mongodb.go
│     │     ├── wire.go
│     │     └── wire_gen.go
│     ├── middleware
│     │     ├── api_key.go
│     │     ├── jwt.go
│     │     └── on_request.go
│     ├── pkg
│     │     ├── casbinx
│     │     │     └── casbinx.go
│     │     ├── response
│     │     │     └── response.go
│     │     └── validator
│     │         └── validator.go
│     └── shared
│           └── example
│               ├── datasource.go
│               ├── example.go
│               ├── provider.go
│               └── repository.go
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

Generate a new feature module:

```shell
codegen -f user
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

### 3. Generate Features CRUD and Swagger

Generate CRUD operations from JSON specifications:

### 1. Define Spec File

Create `spec/auth.json`:
```json
{
    "accessToken": "JWT",
    "expired": 1234567,
    "date": "2024-10-15T14:30:00Z"
}
```

### 2. Generate CRUD

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

### 4. Generate Shared Prototype

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

### 5. Generate Shared CRUD

- SQL Builder

```shell
codegen -sh promotion -s spec/promotion.json -d maridb -orm sqlbuilder
```

- Bun

```shell
codegen -sh promotion -s spec/promotion.json -d maridb -orm bun
```

This generates shared CRUD operations based on your JSON structure.

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
