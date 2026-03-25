package creator

import (
	"fmt"
	"github.com/prongbang/codegen/pkg/common"
	"github.com/prongbang/codegen/pkg/filex"
	"github.com/prongbang/codegen/pkg/option"
	"github.com/pterm/pterm"
)

type Config struct {
	Pkg      option.Package
	Filename string
	Template []byte
}

type Creator interface {
	Create(config Config) error
}

type creator struct {
	FileX filex.FileX
}

func (f *creator) Create(config Config) error {
	spinnerGenFile, _ := pterm.DefaultSpinner.Start(fmt.Sprintf("Generate file %s", config.Filename))
	currentDir, err := f.FileX.Getwd()
	if err != nil {
		spinnerGenFile.Fail(err)
		return err
	}
	currentDir = fmt.Sprintf("%s/%s", currentDir, common.ToLower(config.Pkg.Name))
	err = f.FileX.EnsureDir(currentDir)
	if err != nil {
		spinnerGenFile.Fail(err)
		return err
	}
	target := fmt.Sprintf("%s/%s", currentDir, config.Filename)
	if err = f.FileX.WriteFile(target, config.Template); err != nil {
		spinnerGenFile.Fail(err)
		return err
	}
	spinnerGenFile.Success()
	return nil
}

func New(fileX filex.FileX) Creator {
	return &creator{
		FileX: fileX,
	}
}
