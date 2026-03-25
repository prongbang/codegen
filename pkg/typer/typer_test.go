package typer

import (
	"encoding/json"
	"testing"
)

func TestGet(t *testing.T) {
	tests := []struct {
		name  string
		value any
		want  string
	}{
		{name: "nil", value: nil, want: "any"},
		{name: "string", value: "Alice", want: "string"},
		{name: "time", value: "2024-10-15T14:30:00Z", want: "*time.Time"},
		{name: "int", value: 30, want: "int64"},
		{name: "json-number-int", value: json.Number("42"), want: "int64"},
		{name: "json-number-float", value: json.Number("5.7"), want: "float64"},
		{name: "bool", value: true, want: "*bool"},
		{name: "float", value: 5.7, want: "float64"},
		{name: "fallback", value: []string{"x"}, want: "any"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Get(tt.value); got != tt.want {
				t.Fatalf("Get(%v) = %s, want %s", tt.value, got, tt.want)
			}
		})
	}
}

func TestValue(t *testing.T) {
	tests := map[string]string{
		"any":        "nil",
		"*time.Time": "nil",
		"*bool":      "nil",
		"string":     `""`,
		"int64":      "0",
		"float64":    "0.0",
		"unknown":    "",
	}

	for typ, want := range tests {
		if got := Value(typ); got != want {
			t.Fatalf("Value(%s) = %s, want %s", typ, got, want)
		}
	}
}

func TestOperate(t *testing.T) {
	tests := map[string]string{
		"any":        "!=",
		"string":     "!=",
		"*time.Time": "!=",
		"int64":      ">",
		"float64":    ">",
		"other":      "!=",
	}

	for typ, want := range tests {
		if got := Operate(typ); got != want {
			t.Fatalf("Operate(%s) = %s, want %s", typ, got, want)
		}
	}
}

func TestTypeHelpers(t *testing.T) {
	if got := Type(1); got != "int64" {
		t.Fatalf("unexpected type: %s", got)
	}
	if got := Type(1.2); got != "float64" {
		t.Fatalf("unexpected type: %s", got)
	}
	if got := Type(true); got != "boolean" {
		t.Fatalf("unexpected type: %s", got)
	}
	if got := Type("x"); got != "string" {
		t.Fatalf("unexpected type: %s", got)
	}
	if got := Type([]string{"x"}); got != "interface{}" {
		t.Fatalf("unexpected fallback type: %s", got)
	}

	if got := SprintOf(1); got != "int" {
		t.Fatalf("unexpected sprint type: %s", got)
	}
	if got := TypeOf("x"); got != "string" {
		t.Fatalf("unexpected reflect type: %s", got)
	}
	if got := ValueOf("x"); got != "string" {
		t.Fatalf("unexpected reflect kind: %s", got)
	}
}
