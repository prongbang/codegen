package tools

import (
	"errors"
	"strings"
	"testing"
)

type mockCommand struct {
	runCalls   []string
	asyncCalls []string
	runErr     map[string]error
	asyncErr   map[string]error
}

func (m *mockCommand) Run(name string, args ...string) (string, error) {
	call := strings.Join(append([]string{name}, args...), " ")
	m.runCalls = append(m.runCalls, call)
	if err := m.runErr[call]; err != nil {
		return "", err
	}
	return "", nil
}

func (m *mockCommand) RunAsync(name string, args ...string) (string, error) {
	call := strings.Join(append([]string{name}, args...), " ")
	m.asyncCalls = append(m.asyncCalls, call)
	if err := m.asyncErr[call]; err != nil {
		return "", err
	}
	return "", nil
}

type stubInstaller struct {
	called int
	err    error
}

func (s *stubInstaller) Install() error {
	s.called++
	return s.err
}

func TestWireInstallerInstallsWhenMissing(t *testing.T) {
	cmd := &mockCommand{runErr: map[string]error{"wire help": errors.New("missing")}, asyncErr: map[string]error{}}
	if err := NewWireInstaller(cmd).Install(); err != nil {
		t.Fatal(err)
	}
	if len(cmd.asyncCalls) != 1 || cmd.asyncCalls[0] != "go install github.com/google/wire/cmd/wire@latest" {
		t.Fatalf("unexpected async calls: %+v", cmd.asyncCalls)
	}
}

func TestWireInstallerNoopWhenInstalled(t *testing.T) {
	cmd := &mockCommand{runErr: map[string]error{}, asyncErr: map[string]error{}}
	if err := NewWireInstaller(cmd).Install(); err != nil {
		t.Fatal(err)
	}
	if len(cmd.asyncCalls) != 0 {
		t.Fatalf("expected no async calls, got %+v", cmd.asyncCalls)
	}
}

func TestWireRunnerRunsWire(t *testing.T) {
	cmd := &mockCommand{runErr: map[string]error{}, asyncErr: map[string]error{}}
	if err := NewWireRunner(cmd).Run(); err != nil {
		t.Fatal(err)
	}
	if len(cmd.runCalls) != 2 || cmd.runCalls[0] != "go mod tidy" || cmd.runCalls[1] != "wire" {
		t.Fatalf("unexpected run calls: %+v", cmd.runCalls)
	}
}

func TestWireRunnerReturnsError(t *testing.T) {
	cmd := &mockCommand{runErr: map[string]error{"wire": errors.New("wire failed")}, asyncErr: map[string]error{}}
	if err := NewWireRunner(cmd).Run(); err == nil {
		t.Fatal("expected error")
	}
}

func TestGRPCInstallerRunsAllInstallers(t *testing.T) {
	a := &stubInstaller{}
	b := &stubInstaller{}
	c := &stubInstaller{}
	d := &stubInstaller{}
	inst := &grpcInstaller{
		ProtocGenGo:     a,
		ProtocGenGoGRPC: b,
		Zerobrew:        c,
		Protobuf:        d,
	}
	if err := inst.Install(); err != nil {
		t.Fatal(err)
	}
	if a.called != 1 || b.called != 1 || c.called != 1 || d.called != 1 {
		t.Fatalf("unexpected calls: %d %d %d %d", a.called, b.called, c.called, d.called)
	}
}

func TestGRPCInstallerStopsOnError(t *testing.T) {
	want := errors.New("boom")
	a := &stubInstaller{err: want}
	b := &stubInstaller{}
	inst := &grpcInstaller{
		ProtocGenGo:     a,
		ProtocGenGoGRPC: b,
		Zerobrew:        &stubInstaller{},
		Protobuf:        &stubInstaller{},
	}
	if err := inst.Install(); !errors.Is(err, want) {
		t.Fatalf("unexpected error: %v", err)
	}
	if b.called != 0 {
		t.Fatalf("expected second installer not to run, got %d", b.called)
	}
}

func TestInstallerNewUsesWireInstaller(t *testing.T) {
	wire := &stubInstaller{}
	sqlc := &stubInstaller{}
	dbml := &stubInstaller{}
	if err := New(wire, sqlc, dbml).Install(); err != nil {
		t.Fatal(err)
	}
	if wire.called != 1 || sqlc.called != 0 || dbml.called != 0 {
		t.Fatalf("unexpected install calls: wire=%d sqlc=%d dbml=%d", wire.called, sqlc.called, dbml.called)
	}
}

func TestIndividualGRPCInstallers(t *testing.T) {
	tests := []struct {
		name      string
		installer Installer
		runErr    map[string]error
		wantAsync string
	}{
		{
			name:      "protoc-gen-go",
			installer: NewProtocGenGoInstaller(&mockCommand{}),
			runErr:    map[string]error{"protoc-gen-go --version": errors.New("missing")},
			wantAsync: "go install google.golang.org/protobuf/cmd/protoc-gen-go@latest",
		},
		{
			name:      "protoc-gen-go-grpc",
			installer: NewProtocGenGoGRPCInstaller(&mockCommand{}),
			runErr:    map[string]error{"protoc-gen-go-grpc --version": errors.New("missing")},
			wantAsync: "go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest",
		},
		{
			name:      "zerobrew",
			installer: NewZerobrewInstaller(&mockCommand{}),
			runErr:    map[string]error{"zb --version": errors.New("missing")},
			wantAsync: "sh -c curl -fsSL https://zerobrew.rs/install | bash",
		},
		{
			name:      "protobuf",
			installer: NewProtobufInstaller(&mockCommand{}),
			runErr:    map[string]error{"protoc --version": errors.New("missing")},
			wantAsync: "sh -c export PATH=\"$HOME/.zerobrew/bin:$HOME/.local/bin:$PATH\"; zb install protobuf",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &mockCommand{runErr: tt.runErr, asyncErr: map[string]error{}}
			switch ins := tt.installer.(type) {
			case *protocGenGoInstaller:
				ins.Cmd = cmd
			case *protocGenGoGRPCInstaller:
				ins.Cmd = cmd
			case *zerobrewInstaller:
				ins.Cmd = cmd
			case *protobufInstaller:
				ins.Cmd = cmd
			}
			if err := tt.installer.Install(); err != nil {
				t.Fatal(err)
			}
			if len(cmd.asyncCalls) != 1 || cmd.asyncCalls[0] != tt.wantAsync {
				t.Fatalf("unexpected async calls: %+v", cmd.asyncCalls)
			}
		})
	}
}

func TestIndividualGRPCInstallersNoopAndError(t *testing.T) {
	tests := []struct {
		name      string
		installer Installer
		runKey    string
		asyncKey  string
	}{
		{
			name:      "protoc-gen-go-error",
			installer: NewProtocGenGoInstaller(&mockCommand{}),
			runKey:    "protoc-gen-go --version",
			asyncKey:  "go install google.golang.org/protobuf/cmd/protoc-gen-go@latest",
		},
		{
			name:      "protoc-gen-go-grpc-error",
			installer: NewProtocGenGoGRPCInstaller(&mockCommand{}),
			runKey:    "protoc-gen-go-grpc --version",
			asyncKey:  "go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest",
		},
		{
			name:      "zerobrew-error",
			installer: NewZerobrewInstaller(&mockCommand{}),
			runKey:    "zb --version",
			asyncKey:  "sh -c curl -fsSL https://zerobrew.rs/install | bash",
		},
		{
			name:      "protobuf-error",
			installer: NewProtobufInstaller(&mockCommand{}),
			runKey:    "protoc --version",
			asyncKey:  "sh -c export PATH=\"$HOME/.zerobrew/bin:$HOME/.local/bin:$PATH\"; zb install protobuf",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &mockCommand{
				runErr:   map[string]error{tt.runKey: errors.New("missing")},
				asyncErr: map[string]error{tt.asyncKey: errors.New("install failed")},
			}
			switch ins := tt.installer.(type) {
			case *protocGenGoInstaller:
				ins.Cmd = cmd
			case *protocGenGoGRPCInstaller:
				ins.Cmd = cmd
			case *zerobrewInstaller:
				ins.Cmd = cmd
			case *protobufInstaller:
				ins.Cmd = cmd
			}
			if err := tt.installer.Install(); err == nil {
				t.Fatal("expected error")
			}
		})

		t.Run(strings.TrimSuffix(tt.name, "-error")+"-noop", func(t *testing.T) {
			cmd := &mockCommand{runErr: map[string]error{}, asyncErr: map[string]error{}}
			switch ins := tt.installer.(type) {
			case *protocGenGoInstaller:
				ins.Cmd = cmd
			case *protocGenGoGRPCInstaller:
				ins.Cmd = cmd
			case *zerobrewInstaller:
				ins.Cmd = cmd
			case *protobufInstaller:
				ins.Cmd = cmd
			}
			if err := tt.installer.Install(); err != nil {
				t.Fatal(err)
			}
			if len(cmd.asyncCalls) != 0 {
				t.Fatalf("expected no async calls, got %+v", cmd.asyncCalls)
			}
		})
	}
}
