package module

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/druarnfield/shhh/internal/logging"
)

func nopLogger() *slog.Logger {
	return slog.New(logging.NopHandler{})
}

func TestRunner_ExecutesSteps(t *testing.T) {
	var executed []string
	mod := &Module{
		ID:   "test",
		Name: "Test",
		Steps: []Step{
			{
				Name:  "step1",
				Check: func(ctx context.Context) bool { return false },
				Run: func(ctx context.Context) error {
					executed = append(executed, "step1")
					return nil
				},
			},
			{
				Name:  "step2",
				Check: func(ctx context.Context) bool { return false },
				Run: func(ctx context.Context) error {
					executed = append(executed, "step2")
					return nil
				},
			},
		},
	}

	runner := NewRunner(nopLogger(), false)
	result := runner.RunModule(context.Background(), mod)

	if result.Err != nil {
		t.Fatalf("RunModule error: %v", result.Err)
	}
	if len(executed) != 2 {
		t.Fatalf("executed %d steps, want 2", len(executed))
	}
	if result.Skipped != 0 {
		t.Errorf("skipped = %d, want 0", result.Skipped)
	}
	if result.Completed != 2 {
		t.Errorf("completed = %d, want 2", result.Completed)
	}
}

func TestRunner_SkipsPassedChecks(t *testing.T) {
	ran := false
	mod := &Module{
		ID:   "test",
		Name: "Test",
		Steps: []Step{
			{
				Name:  "already done",
				Check: func(ctx context.Context) bool { return true },
				Run: func(ctx context.Context) error {
					ran = true
					return nil
				},
			},
		},
	}

	runner := NewRunner(nopLogger(), false)
	result := runner.RunModule(context.Background(), mod)

	if ran {
		t.Error("step should have been skipped")
	}
	if result.Skipped != 1 {
		t.Errorf("skipped = %d, want 1", result.Skipped)
	}
}

func TestRunner_StopsOnError(t *testing.T) {
	step2ran := false
	mod := &Module{
		ID:   "test",
		Name: "Test",
		Steps: []Step{
			{
				Name:  "fails",
				Check: func(ctx context.Context) bool { return false },
				Run: func(ctx context.Context) error {
					return errors.New("boom")
				},
			},
			{
				Name:  "should not run",
				Check: func(ctx context.Context) bool { return false },
				Run: func(ctx context.Context) error {
					step2ran = true
					return nil
				},
			},
		},
	}

	runner := NewRunner(nopLogger(), false)
	result := runner.RunModule(context.Background(), mod)

	if result.Err == nil {
		t.Error("expected error")
	}
	if step2ran {
		t.Error("step2 should not have run")
	}
	if result.FailedStep != "fails" {
		t.Errorf("FailedStep = %q, want %q", result.FailedStep, "fails")
	}
}

func TestRunner_DryRun(t *testing.T) {
	ran := false
	mod := &Module{
		ID:   "test",
		Name: "Test",
		Steps: []Step{
			{
				Name:  "step1",
				Check: func(ctx context.Context) bool { return false },
				Run: func(ctx context.Context) error {
					ran = true
					return nil
				},
				DryRun: func(ctx context.Context) string {
					return "would do the thing"
				},
			},
		},
	}

	runner := NewRunner(nopLogger(), true)
	result := runner.RunModule(context.Background(), mod)

	if ran {
		t.Error("Run should not be called in dry-run mode")
	}
	if result.Err != nil {
		t.Fatalf("unexpected error: %v", result.Err)
	}
}

func TestRunner_RunModules(t *testing.T) {
	var order []string

	reg := NewRegistry()
	reg.Register(&Module{
		ID: "base",
		Steps: []Step{{
			Name:  "base-step",
			Check: func(ctx context.Context) bool { return false },
			Run: func(ctx context.Context) error {
				order = append(order, "base")
				return nil
			},
		}},
	})
	reg.Register(&Module{
		ID:           "python",
		Dependencies: []string{"base"},
		Steps: []Step{{
			Name:  "python-step",
			Check: func(ctx context.Context) bool { return false },
			Run: func(ctx context.Context) error {
				order = append(order, "python")
				return nil
			},
		}},
	})

	runner := NewRunner(nopLogger(), false)
	results, err := runner.RunModules(context.Background(), reg, []string{"python"})
	if err != nil {
		t.Fatalf("RunModules: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("got %d results, want 2", len(results))
	}
	if order[0] != "base" || order[1] != "python" {
		t.Errorf("execution order = %v, want [base, python]", order)
	}
}
