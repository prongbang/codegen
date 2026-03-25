package tools

import (
	"github.com/prongbang/codegen/pkg/arch"
	"github.com/prongbang/codegen/pkg/command"
	"github.com/pterm/pterm"
)

type dbmlInstaller struct {
	Cmd  command.Command
	Arch arch.Arch
}

// Install implements Tools.
func (d *dbmlInstaller) Install() error {
	spinnerDbml, _ := pterm.DefaultSpinner.Start("Install sql2dbml")
	_, err := d.Cmd.Run("sql2dbml", "--version")
	if err != nil {
		if d.Arch.IsDarwinArm64() {
			_, err = d.Cmd.RunAsync("arch", "-arm64", "brew", "install", "dbml-cli")
		} else {
			_, err = d.Cmd.Run("brew", "install", "dbml-cli")
		}
		if err != nil {
			spinnerDbml.Fail()
		} else {
			spinnerDbml.Success()
		}
	} else {
		spinnerDbml.Success()
	}
	return err
}

func NewDbmlInstaller(cmd command.Command, arch arch.Arch) Installer {
	return &dbmlInstaller{
		Cmd:  cmd,
		Arch: arch,
	}
}
