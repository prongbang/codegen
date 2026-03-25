package main

import (
	"os"
	"testing"
)

func TestFlagsHelpers(t *testing.T) {
	flags := Flags{
		ProjectName: "Hello World",
		ModuleName:  "github.com/acme",
		FeatureName: "User Profile",
		SharedName:  "Auth Token",
	}

	if got := flags.Project(); got != "hello-world" {
		t.Fatalf("unexpected project: %s", got)
	}
	if got := flags.Module(); got != "github.com/acme/hello-world" {
		t.Fatalf("unexpected module: %s", got)
	}
	if got := flags.Feature(); got != "user_profile" {
		t.Fatalf("unexpected feature: %s", got)
	}
	if got := flags.Shared(); got != "auth_token" {
		t.Fatalf("unexpected shared: %s", got)
	}
}

func TestFlagsEmptyFeatureAndShared(t *testing.T) {
	flags := Flags{}
	if got := flags.Feature(); got != "" {
		t.Fatalf("expected empty feature, got %s", got)
	}
	if got := flags.Shared(); got != "" {
		t.Fatalf("expected empty shared, got %s", got)
	}
}

func TestMainHelpPaths(t *testing.T) {
	origArgs := os.Args
	t.Cleanup(func() { os.Args = origArgs })

	for _, args := range [][]string{
		{"codegen", "-h"},
		{"codegen", "grpc", "-h"},
		{"codegen", "grpc", "init", "-h"},
		{"codegen", "grpc", "server", "-h"},
		{"codegen", "grpc", "client", "-h"},
	} {
		os.Args = args
		main()
	}
}

func TestNewGRPCGeneratorFactory(t *testing.T) {
	if gen := newGRPCGenerator(); gen == nil {
		t.Fatal("expected grpc generator")
	}
}
