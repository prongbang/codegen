package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/prongbang/codegen/pkg/creator"
	"github.com/prongbang/codegen/pkg/filex"
	"github.com/prongbang/codegen/pkg/generate"

	"github.com/ettle/strcase"
	"github.com/prongbang/codegen/pkg/arch"
	"github.com/prongbang/codegen/pkg/command"
	"github.com/prongbang/codegen/pkg/dbdriver"
	"github.com/prongbang/codegen/pkg/option"
	"github.com/prongbang/codegen/pkg/tools"
	"github.com/prongbang/codegen/template"
	"github.com/urfave/cli/v2"
)

type Flags struct {
	ProjectName string
	ModuleName  string
	FeatureName string
	PackageName string
	SharedName  string
	Crud        string
	Spec        string
	Dsn         string
	Table       string
	Driver      string
	Orm         string
	Framework   string
	Template    string
}

func (f Flags) Project() string {
	return strcase.ToKebab(strings.ReplaceAll(f.ProjectName, " ", "_"))
}

func (f Flags) Module() string {
	return fmt.Sprintf("%s/%s", f.ModuleName, strcase.ToKebab(f.Project()))
}

func (f Flags) Feature() string {
	if f.FeatureName != "" {
		return strcase.ToSnake(strings.ReplaceAll(f.FeatureName, " ", ""))
	}
	return ""
}

func (f Flags) Package() string {
	if f.PackageName != "" {
		return strcase.ToSnake(strings.ReplaceAll(f.PackageName, " ", ""))
	}
	return ""
}

func (f Flags) Shared() string {
	if f.SharedName != "" {
		return strcase.ToSnake(strings.ReplaceAll(f.SharedName, " ", ""))
	}
	return ""
}

func newGRPCGenerator() generate.GRPCGenerator {
	cmd := command.New()
	fileX := filex.NewFileX()
	grpcInstaller := tools.NewGRPCInstaller(cmd)
	wireInstaller := tools.NewWireInstaller(cmd)
	wireRunner := tools.NewWireRunner(cmd)
	return generate.NewGRPCGenerator(fileX, cmd, grpcInstaller, wireInstaller, wireRunner)
}

func newDatabaseGenerator() generate.DatabaseGenerator {
	cmd := command.New()
	fileX := filex.NewFileX()
	return generate.NewDatabaseGenerator(fileX, cmd, tools.NewWireInstaller(cmd), tools.NewWireRunner(cmd))
}

func newGenerator() generate.Generator {
	cmd := command.New()
	arc := arch.New()
	wireInstaller := tools.NewWireInstaller(cmd)
	wireRunner := tools.NewWireRunner(cmd)
	fileX := filex.NewFileX()
	creatorX := creator.New(fileX)
	installer := tools.New(
		wireInstaller,
		tools.NewSqlcInstaller(cmd, arc),
		tools.NewDbmlInstaller(cmd, arc),
	)
	featureBinding := generate.NewFeatureBinding(fileX)
	sharedBinding := generate.NewSharedBinding(fileX)
	projectGenerator := generate.NewProjectGenerator(fileX)
	featureGenerator := generate.NewFeatureGenerator(fileX, creatorX, installer, wireInstaller, wireRunner, featureBinding)
	sharedGenerator := generate.NewSharedGenerator(fileX, creatorX, installer, wireInstaller, wireRunner, sharedBinding)
	openAPIGenerator := generate.NewOpenAPIGenerator()
	mqttGenerator := generate.NewMqttGenerator(fileX)
	return generate.NewGenerator(projectGenerator, featureGenerator, sharedGenerator, openAPIGenerator, mqttGenerator)
}

func newApp() *cli.App {
	flags := Flags{}

	return &cli.App{
		Name:      "codegen",
		Usage:     "Generate a Clean Architecture for REST API with support for the Fiber Web Framework in Golang",
		Version:   "v1.7.0",
		Compiled:  time.Now(),
		Copyright: "(c) 2023 prongbang",
		Authors: []*cli.Author{
			{
				Name:  "prongbang",
				Email: "github.com/prongbang",
			},
		},
		Commands: []*cli.Command{
			{
				Name:  "grpc",
				Usage: "gRPC utilities",
				Action: func(c *cli.Context) error {
					return cli.ShowSubcommandHelp(c)
				},
				Subcommands: []*cli.Command{
					{
						Name:  "init",
						Usage: "Initialize gRPC scaffold under internal/app/grpc",
						Action: func(*cli.Context) error {
							return newGRPCGenerator().Init()
						},
					},
					{
						Name:  "server",
						Usage: "Generate gRPC server code",
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:  "new",
								Usage: "Generate gRPC server package scaffold, e.g. --new user",
							},
						},
						Action: func(c *cli.Context) error {
							name := c.String("new")
							if name == "" {
								return cli.ShowSubcommandHelp(c)
							}
							return newGRPCGenerator().New(name)
						},
					},
					{
						Name:  "client",
						Usage: "Generate gRPC client code",
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:  "new",
								Usage: "Generate gRPC client package scaffold, e.g. --new core/device",
							},
						},
						Action: func(c *cli.Context) error {
							name := c.String("new")
							if name == "" {
								return cli.ShowSubcommandHelp(c)
							}
							return newGRPCGenerator().NewClient(name)
						},
					},
				},
			},
			{
				Name:  "database",
				Usage: "Database utilities",
				Action: func(c *cli.Context) error {
					return cli.ShowSubcommandHelp(c)
				},
				Subcommands: []*cli.Command{
					{
						Name:  "init",
						Usage: "Add database code to an existing project, e.g. -d mariadb,influxdb3",
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:    "driver",
								Aliases: []string{"d"},
								Usage:   fmt.Sprintf("-d %s", strings.Join(template.SupportedDatabases(), ",")),
							},
						},
						Action: func(c *cli.Context) error {
							databases, unknown := generate.NormalizeDatabases(c.String("driver"))
							if len(unknown) > 0 {
								return fmt.Errorf("unsupported database %q, expected any of %s",
									strings.Join(unknown, ","), strings.Join(template.SupportedDatabases(), ", "))
							}
							if len(databases) == 0 {
								return cli.ShowSubcommandHelp(c)
							}
							return newDatabaseGenerator().Init(databases)
						},
					},
				},
			},
			{
				Name:  "openapi",
				Usage: "Generate an OpenAPI spec",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:        "framework",
						Aliases:     []string{"fw"},
						Usage:       "-framework fiber",
						Destination: &flags.Framework,
					},
				},
				Action: func(c *cli.Context) error {
					if flags.Framework == "" {
						return cli.ShowSubcommandHelp(c)
					}
					return newGenerator().Generate(option.Options{
						Framework: flags.Framework,
						OpenAPI:   true,
						Patterns:  c.Args().Slice(),
					})
				},
			},
		},
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "new",
				Aliases:     []string{"n"},
				Usage:       "-n project-name",
				Destination: &flags.ProjectName,
			},
			&cli.StringFlag{
				Name:        "mod",
				Aliases:     []string{"m"},
				Usage:       "-m github.com/prongbang/module-name",
				Destination: &flags.ModuleName,
			},
			&cli.StringFlag{
				Name:        "package",
				Aliases:     []string{"p"},
				Usage:       "-p auth",
				Destination: &flags.PackageName,
			},
			&cli.StringFlag{
				Name:        "feature",
				Aliases:     []string{"f"},
				Usage:       "-f auth",
				Destination: &flags.FeatureName,
			},
			&cli.StringFlag{
				Name:        "shared",
				Aliases:     []string{"sh"},
				Usage:       "-sh auth",
				Destination: &flags.SharedName,
			},
			&cli.StringFlag{
				Name:        "spec",
				Aliases:     []string{"s"},
				Usage:       "-s auth.json",
				Destination: &flags.Spec,
			},
			&cli.StringFlag{
				Name:        "dsn",
				Usage:       `-dsn "user:password@tcp(127.0.0.1:3306)/dbname" (generate CRUD from database schema)`,
				Destination: &flags.Dsn,
			},
			&cli.StringFlag{
				Name:        "table",
				Aliases:     []string{"tb"},
				Usage:       "-table users (table name, defaults to feature name)",
				Destination: &flags.Table,
			},
			&cli.StringFlag{
				Name:        "driver",
				Aliases:     []string{"d"},
				Usage:       "-d mariadb (with -new: mariadb,mongodb,influxdb3; omit for a project without database code)",
				Destination: &flags.Driver,
			},
			&cli.StringFlag{
				Name:        "orm",
				Usage:       "-orm bun,sqlbuilder",
				Destination: &flags.Orm,
			},
			&cli.StringFlag{
				Name:        "template",
				Aliases:     []string{"t"},
				Usage:       "-template mqtt",
				Destination: &flags.Template,
			},
		},
		Action: func(*cli.Context) error {
			driver := flags.Driver
			if driver == "" && flags.Dsn != "" {
				driver = dbdriver.DriverMysql
			}
			// When creating a project, -driver selects which database packages to
			// scaffold; omitting it produces a project without any database code.
			databases, unknown := generate.NormalizeDatabases(flags.Driver)
			if flags.ProjectName != "" && len(unknown) > 0 {
				return fmt.Errorf("unsupported database %q, expected any of %s",
					strings.Join(unknown, ","), strings.Join(template.SupportedDatabases(), ", "))
			}

			opt := option.Options{
				Project:   flags.Project(),
				Module:    flags.Module(),
				Package:   flags.Package(),
				Feature:   flags.Feature(),
				Shared:    flags.Shared(),
				Spec:      flags.Spec,
				Dsn:       flags.Dsn,
				Table:     flags.Table,
				Driver:    driver,
				Orm:       flags.Orm,
				Template:  flags.Template,
				Databases: databases,
			}
			return newGenerator().Generate(opt)
		},
	}
}

func main() {
	if err := newApp().Run(os.Args); err != nil {
		fmt.Println("[codegen]", err.Error())
	}
}
