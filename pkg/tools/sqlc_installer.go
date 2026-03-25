package tools

import (
	"github.com/prongbang/codegen/pkg/arch"
	"github.com/prongbang/codegen/pkg/command"
	"github.com/pterm/pterm"
)

type sqlcInstaller struct {
	Cmd  command.Command
	Arch arch.Arch
}

// Install implements Tools.
func (s *sqlcInstaller) Install() error {
	spinnerSqlc, _ := pterm.DefaultSpinner.Start("Install sqlc")
	_, err := s.Cmd.Run("sqlc", "version")
	if err != nil {
		if s.Arch.IsDarwinArm64() {
			_, err = s.Cmd.RunAsync("arch", "-arm64", "brew", "install", "sqlc")
		} else {
			_, err = s.Cmd.Run("brew", "install", "sqlc")
		}
		if err != nil {
			spinnerSqlc.Fail()
		} else {
			spinnerSqlc.Success()
		}
	} else {
		spinnerSqlc.Success()
	}
	return err
}

func NewSqlcInstaller(cmd command.Command, arch arch.Arch) Installer {
	return &sqlcInstaller{
		Cmd:  cmd,
		Arch: arch,
	}
}
