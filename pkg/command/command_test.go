package command

import (
	"strings"
	"testing"
)

func TestRunSuccess(t *testing.T) {
	cmd := New()
	out, err := cmd.Run("echo", "hello")
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(out) != "hello" {
		t.Fatalf("unexpected output: %q", out)
	}
}

func TestRunFailure(t *testing.T) {
	cmd := New()
	if _, err := cmd.Run("false"); err == nil {
		t.Fatal("expected error")
	}
}

func TestRunAsyncSuccess(t *testing.T) {
	cmd := New()
	if _, err := cmd.RunAsync("sh", "-c", "printf ok"); err != nil {
		t.Fatal(err)
	}
}

func TestRunAsyncFailure(t *testing.T) {
	cmd := New()
	if _, err := cmd.RunAsync("sh", "-c", "exit 1"); err == nil {
		t.Fatal("expected error")
	}
}

func TestPrintStream(t *testing.T) {
	printStream("stdout", strings.NewReader("line1\nline2\n"))
}
