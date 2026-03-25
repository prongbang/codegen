package config

import "testing"

func TestConstants(t *testing.T) {
	if InternalPath != "internal" {
		t.Fatalf("unexpected InternalPath: %s", InternalPath)
	}
	if AppPath != "internal/app" {
		t.Fatalf("unexpected AppPath: %s", AppPath)
	}
}
