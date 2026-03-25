package typer_test

import (
	"fmt"
	"github.com/prongbang/codegen/pkg/typer"
	"testing"
)

func TestGet(t *testing.T) {
	result := map[string]interface{}{
		"string":  "Alice",
		"int":     30,
		"null":    nil,
		"float":   5.7,
		"boolean": true,
	}

	for k, v := range result {
		ty := typer.Get(v)
		fmt.Println(k, ":", ty)
	}
}
