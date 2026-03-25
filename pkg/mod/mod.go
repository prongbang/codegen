package mod

import "github.com/prongbang/codegen/pkg/config"

// Mod is struct
type Mod struct {
	Module  string
	AppPath string
	Name    string
}

func (m Mod) NewAppPath() string {
	appPath := m.AppPath
	if appPath == config.AppPath {
		appPath = config.InternalPath
	}
	return appPath
}
