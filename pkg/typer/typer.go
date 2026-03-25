package typer

import (
	"fmt"
	"reflect"
	"strings"
	"time"
)

func Get(value any) string {
	if value == nil {
		return "any"
	}

	switch v := value.(type) {
	case string:
		t, err := time.Parse("2006-01-02T15:04:05Z", v)
		if err == nil && t.Day() > 0 {
			return "*time.Time"
		}
		return "string"
	case int, int8, int16, int32, int64:
		return "int64"
	case bool:
		return "*bool"
	case float32, float64:
		if strings.Index(fmt.Sprintf("%v", v), ".") > -1 {
			return "float64"
		}
		return "int64"
	default:
		return "any"
	}
}

func Value(typ string) string {
	if typ == "any" || typ == "*time.Time" || typ == "*bool" {
		return "nil"
	} else if typ == "string" {
		return `""`
	} else if typ == "int64" {
		return "0"
	} else if typ == "float64" {
		return "0.0"
	}
	return ""
}

func Operate(typ string) string {
	if typ == "any" || typ == "string" || typ == "*time.Time" {
		return `!=`
	} else if typ == "int64" || typ == "float64" {
		return ">"
	}
	return "!="
}

func Type(variable interface{}) string {
	switch variable.(type) {
	case int, int16, int32, int64:
		return "int64"
	case float32, float64:
		return "float64"
	case bool:
		return "boolean"
	case string:
		return "string"
	}
	return "interface{}"
}

func SprintOf(variable interface{}) string {
	return fmt.Sprintf("%T", variable)
}

func TypeOf(variable interface{}) string {
	return reflect.TypeOf(variable).String()
}

func ValueOf(variable interface{}) string {
	return reflect.ValueOf(variable).Kind().String()
}
