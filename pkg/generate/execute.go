package generate

import (
	"github.com/prongbang/fibergen/pkg/filex"
	"github.com/prongbang/fibergen/template"
)

func WriteFile(fx filex.FileX, filename, tmpl string, data interface{}) error {
	buf, err := template.RenderText(tmpl, data)
	if err != nil {
		return err
	}
	return fx.WriteFile(filename, buf)
}
