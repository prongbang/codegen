package command

import (
	"bufio"
	"fmt"
	"io"
	"os/exec"
)

type Command interface {
	Run(name string, args ...string) (string, error)
	RunAsync(name string, args ...string) (string, error)
}

type cmd struct{}

// RunAsync implements Command.
func (*cmd) RunAsync(name string, args ...string) (string, error) {
	command := exec.Command(name, args...)

	stdout, err := command.StdoutPipe()
	if err != nil {
		return "", fmt.Errorf("error creating StdoutPipe: %v", err)
	}

	stderr, err := command.StderrPipe()
	if err != nil {
		return "", fmt.Errorf("error creating StderrPipe: %v", err)
	}

	if err2 := command.Start(); err2 != nil {
		return "", fmt.Errorf("error starting command: %v", err2)
	}

	go printStream("stdout", stdout)
	go printStream("stderr", stderr)

	if err2 := command.Wait(); err2 != nil {
		return "", fmt.Errorf("command finished with error: %v", err2)
	}
	return "", nil
}

// Run implements Command.
func (*cmd) Run(name string, args ...string) (string, error) {
	command := exec.Command(name, args...)
	output, err := command.CombinedOutput()

	// Check if the command was successful
	if err != nil {
		// Print the combined output so the failure is visible, and return it so
		// callers can inspect the reason (go mod tidy, for example, reports which
		// modules clash).
		fmt.Printf("Error output: %s\n", output)

		return string(output), err
	}

	return string(output), err
}

func printStream(streamName string, reader io.Reader) {
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		fmt.Printf("[%s] %s\n", streamName, scanner.Text())
	}
}

func New() Command {
	return &cmd{}
}
