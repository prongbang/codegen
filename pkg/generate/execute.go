package generate

import (
	"github.com/prongbang/codegen/pkg/filex"
	"github.com/prongbang/codegen/template"
)

func WriteFile(fx filex.FileX, filename, tmpl string, data interface{}) error {
	buf, err := template.RenderText(tmpl, data)
	if err != nil {
		return err
	}
	return fx.WriteFile(filename, buf)
}
