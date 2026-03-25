package tools

import (
	"github.com/prongbang/codegen/pkg/command"
	"github.com/pterm/pterm"
)

type wireRunner struct {
	Cmd command.Command
}

// Run implements Runner.
func (r *wireRunner) Run() error {
	spinnerWire, _ := pterm.DefaultSpinner.Start("Wire: Automated Initialization")
	_, err := r.Cmd.Run("wire")
	if err != nil {
		spinnerWire.Fail()
	} else {
		spinnerWire.Success()
	}
	return err
}

func NewWireRunner(cmd command.Command) Runner {
	return &wireRunner{
		Cmd: cmd,
	}
}
