package tools

import (
	"github.com/prongbang/codegen/pkg/command"
	"github.com/pterm/pterm"
)

type wireInstaller struct {
	Cmd command.Command
}

// Install implements WireInstall.
func (w *wireInstaller) Install() error {
	spinnerWire, _ := pterm.DefaultSpinner.Start("Install wire")
	_, err := w.Cmd.Run("wire", "help")
	if err != nil {
		_, err = w.Cmd.RunAsync("go", "install", "github.com/google/wire/cmd/wire@latest")
		if err != nil {
			spinnerWire.Fail()
		} else {
			spinnerWire.Success()
		}
	} else {
		spinnerWire.Success()
	}
	return err
}

func NewWireInstaller(cmd command.Command) Installer {
	return &wireInstaller{
		Cmd: cmd,
	}
}
