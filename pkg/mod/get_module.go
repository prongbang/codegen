package mod

import (
	"fmt"
	"github.com/prongbang/codegen/pkg/config"
	"strings"

	"github.com/prongbang/codegen/pkg/filex"
)

func GetModule(fx filex.FileX) Mod {
	// Get current path
	pwd, _ := fx.Getwd()

	// Get root project path
	changeToRootProject := "."
	if !fx.IsExist(fmt.Sprintf("%s/go.mod", pwd)) {
		changeToRootProject = "../../../"
	}
	// Change current directory
	_ = fx.Chdir(changeToRootProject)

	root, _ := fx.Getwd()
	if bt := fx.ReadFile(root + "/go.mod"); bt != "" {
		// Find module
		text := bt
		m := "module "
		s := strings.Index(text, m)
		e := strings.Index(text, "\n")
		if s < 0 && e < 0 {
			return Mod{}
		}
		module := text[s+len(m) : e]

		mds := strings.Split(module, "/")
		mdl := len(mds)
		if mdl <= 0 {
			return Mod{}
		}
		name := mds[mdl-1]

		// Find internal/project-name
		appPath := fmt.Sprintf("internal/%s", name)
		if !fx.IsDirExist(fmt.Sprintf("%s/%s", root, appPath)) {
			appPath = config.AppPath
		}

		// Change current directory to api path
		apiPath := fmt.Sprintf("./%s/api", appPath)
		_ = fx.Chdir(apiPath)

		return Mod{
			Module:  module,
			AppPath: appPath,
			Name:    name,
		}
	}
	return Mod{}
}
