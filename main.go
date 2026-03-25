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
	"github.com/prongbang/codegen/pkg/option"
	"github.com/prongbang/codegen/pkg/tools"
	"github.com/urfave/cli/v2"
)

type Flags struct {
	ProjectName string
	ModuleName  string
	FeatureName string
	SharedName  string
	Crud        string
	Spec        string
	Driver      string
	Orm         string
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

func (f Flags) Shared() string {
	if f.SharedName != "" {
		return strcase.ToSnake(strings.ReplaceAll(f.SharedName, " ", ""))
	}
	return ""
}

func main() {
	flags := Flags{}

	app := &cli.App{
		Name:      "codegen",
		Usage:     "Generate a Clean Architecture for REST API with support for the Fiber Web Framework in Golang",
		Version:   "v1.4.5",
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
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "new",
						Usage: "Generate gRPC package scaffold, e.g. --new user",
					},
				},
				Action: func(c *cli.Context) error {
					name := c.String("new")
					if name == "" {
						return cli.ShowSubcommandHelp(c)
					}
					cmd := command.New()
					fileX := filex.NewFileX()
					grpcInstaller := tools.NewGRPCInstaller(cmd)
					wireInstaller := tools.NewWireInstaller(cmd)
					wireRunner := tools.NewWireRunner(cmd)
					grpcGenerator := generate.NewGRPCGenerator(fileX, cmd, grpcInstaller, wireInstaller, wireRunner)
					return grpcGenerator.New(name)
				},
				Subcommands: []*cli.Command{
					{
						Name:  "init",
						Usage: "Initialize gRPC scaffold under internal/app/grpc",
						Action: func(*cli.Context) error {
							cmd := command.New()
							fileX := filex.NewFileX()
							grpcInstaller := tools.NewGRPCInstaller(cmd)
							wireInstaller := tools.NewWireInstaller(cmd)
							wireRunner := tools.NewWireRunner(cmd)
							grpcGenerator := generate.NewGRPCGenerator(fileX, cmd, grpcInstaller, wireInstaller, wireRunner)
							return grpcGenerator.Init()
						},
					},
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
				Name:        "driver",
				Aliases:     []string{"d"},
				Usage:       "-d mariadb",
				Destination: &flags.Driver,
			},
			&cli.StringFlag{
				Name:        "orm",
				Usage:       "-orm bun,sqlbuilder",
				Destination: &flags.Orm,
			},
		},
		Action: func(*cli.Context) error {
			opt := option.Options{
				Project: flags.Project(),
				Module:  flags.Module(),
				Feature: flags.Feature(),
				Shared:  flags.Shared(),
				Spec:    flags.Spec,
				Driver:  flags.Driver,
				Orm:     flags.Orm,
			}
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
			gen := generate.NewGenerator(projectGenerator, featureGenerator, sharedGenerator)
			return gen.Generate(opt)
		},
	}

	if err := app.Run(os.Args); err != nil {
		fmt.Println("[codegen]", err.Error())
	}
}
