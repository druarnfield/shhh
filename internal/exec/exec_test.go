package exec

import (
	"context"
	"runtime"
	"testing"
)

func TestRun_SimpleCommand(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("test uses unix commands")
	}

	result, err := Run(context.Background(), "echo", "hello")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if result.ExitCode != 0 {
		t.Errorf("exit code = %d, want 0", result.ExitCode)
	}
	if result.Stdout != "hello\n" {
		t.Errorf("stdout = %q, want %q", result.Stdout, "hello\n")
	}
}

func TestRun_FailingCommand(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("test uses unix commands")
	}

	_, err := Run(context.Background(), "false")
	if err == nil {
		t.Error("expected error for failing command")
	}
}

func TestRun_CommandNotFound(t *testing.T) {
	_, err := Run(context.Background(), "nonexistent_command_12345")
	if err == nil {
		t.Error("expected error for missing command")
	}
}

func TestCommandExists(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("test uses unix commands")
	}

	if !CommandExists("echo") {
		t.Error("echo should exist")
	}
	if CommandExists("nonexistent_command_12345") {
		t.Error("nonexistent command should not exist")
	}
}

func TestMockRunner(t *testing.T) {
	mock := &MockRunner{
		Results: map[string]Result{
			"git --version": {Stdout: "git version 2.43.0\n", ExitCode: 0},
		},
	}

	result, err := mock.Run(context.Background(), "git", "--version")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if result.Stdout != "git version 2.43.0\n" {
		t.Errorf("stdout = %q", result.Stdout)
	}
}
