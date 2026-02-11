package exec

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// Result holds the output and exit code of a command execution.
type Result struct {
	Stdout   string
	Stderr   string
	ExitCode int
}

// Runner is an interface for executing external commands.
// Use DefaultRunner for real commands and MockRunner for tests.
type Runner interface {
	Run(ctx context.Context, name string, args ...string) (Result, error)
}

// DefaultRunner executes commands on the real system.
type DefaultRunner struct{}

// Run executes the named command with the given arguments using the real system.
func (d *DefaultRunner) Run(ctx context.Context, name string, args ...string) (Result, error) {
	return Run(ctx, name, args...)
}

// Run executes the named command with the given arguments and returns
// the captured stdout, stderr, and exit code.
func Run(ctx context.Context, name string, args ...string) (Result, error) {
	cmd := exec.CommandContext(ctx, name, args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	result := Result{
		Stdout: stdout.String(),
		Stderr: stderr.String(),
	}

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
		}
		return result, fmt.Errorf("command %q failed: %w\nstderr: %s", name, err, stderr.String())
	}

	return result, nil
}

// CommandExists checks whether a command is available on the system PATH.
func CommandExists(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

// MockRunner is a test double that returns pre-configured results for commands.
type MockRunner struct {
	Results map[string]Result
	Calls   []string
}

// Run looks up the command key in the Results map and returns the matching result.
// The key is formed as "name arg1 arg2 ...".
func (m *MockRunner) Run(ctx context.Context, name string, args ...string) (Result, error) {
	key := name
	if len(args) > 0 {
		key = name + " " + strings.Join(args, " ")
	}
	m.Calls = append(m.Calls, key)

	if result, ok := m.Results[key]; ok {
		if result.ExitCode != 0 {
			return result, fmt.Errorf("command %q exited with code %d", key, result.ExitCode)
		}
		return result, nil
	}

	return Result{}, fmt.Errorf("unexpected command: %q", key)
}
