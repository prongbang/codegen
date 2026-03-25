package tools

import (
	"github.com/prongbang/codegen/pkg/command"
)

type protocGenGoInstaller struct {
	Cmd command.Command
}

func (p *protocGenGoInstaller) Install() error {
	_, err := p.Cmd.Run("protoc-gen-go", "--version")
	if err != nil {
		_, err = p.Cmd.RunAsync("go", "install", "google.golang.org/protobuf/cmd/protoc-gen-go@latest")
		return err
	}
	return nil
}

type protocGenGoGRPCInstaller struct {
	Cmd command.Command
}

func (p *protocGenGoGRPCInstaller) Install() error {
	_, err := p.Cmd.Run("protoc-gen-go-grpc", "--version")
	if err != nil {
		_, err = p.Cmd.RunAsync("go", "install", "google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest")
		return err
	}
	return nil
}

type zerobrewInstaller struct {
	Cmd command.Command
}

func (z *zerobrewInstaller) Install() error {
	_, err := z.Cmd.Run("zb", "--version")
	if err != nil {
		_, err = z.Cmd.RunAsync("sh", "-c", "curl -fsSL https://zerobrew.rs/install | bash")
		return err
	}
	return nil
}

type protobufInstaller struct {
	Cmd command.Command
}

func (p *protobufInstaller) Install() error {
	_, err := p.Cmd.Run("protoc", "--version")
	if err != nil {
		_, err = p.Cmd.RunAsync("sh", "-c", "export PATH=\"$HOME/.zerobrew/bin:$HOME/.local/bin:$PATH\"; zb install protobuf")
		return err
	}
	return nil
}

type grpcInstaller struct {
	ProtocGenGo     Installer
	ProtocGenGoGRPC Installer
	Zerobrew        Installer
	Protobuf        Installer
}

func (g *grpcInstaller) Install() error {
	if err := g.ProtocGenGo.Install(); err != nil {
		return err
	}
	if err := g.ProtocGenGoGRPC.Install(); err != nil {
		return err
	}
	if err := g.Zerobrew.Install(); err != nil {
		return err
	}
	if err := g.Protobuf.Install(); err != nil {
		return err
	}
	return nil
}

func NewProtocGenGoInstaller(cmd command.Command) Installer {
	return &protocGenGoInstaller{Cmd: cmd}
}

func NewProtocGenGoGRPCInstaller(cmd command.Command) Installer {
	return &protocGenGoGRPCInstaller{Cmd: cmd}
}

func NewZerobrewInstaller(cmd command.Command) Installer {
	return &zerobrewInstaller{Cmd: cmd}
}

func NewProtobufInstaller(cmd command.Command) Installer {
	return &protobufInstaller{Cmd: cmd}
}

func NewGRPCInstaller(cmd command.Command) Installer {
	return &grpcInstaller{
		ProtocGenGo:     NewProtocGenGoInstaller(cmd),
		ProtocGenGoGRPC: NewProtocGenGoGRPCInstaller(cmd),
		Zerobrew:        NewZerobrewInstaller(cmd),
		Protobuf:        NewProtobufInstaller(cmd),
	}
}
