// Package module provides the Runner which executes module steps in order,
// checking preconditions before running and supporting dry-run mode.
//
// Contract: any step that writes environment variables persistently MUST also
// call os.Setenv() so that subsequent steps within the same process see the
// updated value.
package module

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

// ModuleResult captures the outcome of running a single module.
type ModuleResult struct {
	// ModuleID is the ID of the module that was run.
	ModuleID string

	// Completed is the number of steps that ran successfully.
	Completed int

	// Skipped is the number of steps whose Check returned true.
	Skipped int

	// Total is the total number of steps in the module.
	Total int

	// FailedStep is the name of the step that failed, if any.
	FailedStep string

	// Err is the error returned by the failed step, or nil on success.
	Err error
}

// StepCallback is invoked after each step is processed (whether skipped, run,
// or failed). It allows the caller to display progress, update a UI, etc.
type StepCallback func(module *Module, step *Step, index int, total int, skipped bool, err error)

// PreStepCallback is invoked before each step begins processing.
type PreStepCallback func(module *Module, step *Step, index int, total int)

// Runner executes module steps with check-before-run semantics.
type Runner struct {
	logger      *slog.Logger
	dryRun      bool
	callback    StepCallback
	preCallback PreStepCallback
}

// NewRunner creates a Runner. When dryRun is true, steps are not executed;
// instead their DryRun description is logged.
func NewRunner(logger *slog.Logger, dryRun bool) *Runner {
	return &Runner{
		logger: logger,
		dryRun: dryRun,
	}
}

// SetCallback registers a callback that is invoked after each step is
// processed. Pass nil to clear.
func (r *Runner) SetCallback(cb StepCallback) {
	r.callback = cb
}

// SetPreStepCallback registers a callback that is invoked before each step
// begins processing. Pass nil to clear.
func (r *Runner) SetPreStepCallback(cb PreStepCallback) {
	r.preCallback = cb
}

// RunModule executes every step in the given module sequentially. For each
// step:
//   - If Check returns true the step is skipped.
//   - If the runner is in dry-run mode, DryRun is called and logged but Run is
//     not invoked.
//   - Otherwise Run is called; on error execution stops immediately.
func (r *Runner) RunModule(ctx context.Context, mod *Module) ModuleResult {
	result := ModuleResult{
		ModuleID: mod.ID,
		Total:    len(mod.Steps),
	}

	for i := range mod.Steps {
		step := &mod.Steps[i]

		if r.preCallback != nil {
			r.preCallback(mod, step, i, result.Total)
		}

		// Check precondition -- skip if already satisfied.
		if step.Check != nil && step.Check(ctx) {
			result.Skipped++
			r.logger.Info("step already satisfied, skipping",
				slog.String("module", mod.ID),
				slog.String("step", step.Name),
			)
			if r.callback != nil {
				r.callback(mod, step, i, result.Total, true, nil)
			}
			continue
		}

		// Dry-run mode -- describe but do not execute.
		if r.dryRun {
			desc := ""
			if step.DryRun != nil {
				desc = step.DryRun(ctx)
			}
			r.logger.Info("dry-run",
				slog.String("module", mod.ID),
				slog.String("step", step.Name),
				slog.String("would_do", desc),
			)
			if r.callback != nil {
				r.callback(mod, step, i, result.Total, true, nil)
			}
			continue
		}

		// Execute the step.
		start := time.Now()
		err := step.Run(ctx)
		elapsed := time.Since(start)

		if err != nil {
			result.FailedStep = step.Name
			result.Err = fmt.Errorf("step %q in module %q failed: %w", step.Name, mod.ID, err)
			r.logger.Error("step failed",
				slog.String("module", mod.ID),
				slog.String("step", step.Name),
				slog.Duration("elapsed", elapsed),
				slog.String("error", err.Error()),
			)
			if r.callback != nil {
				r.callback(mod, step, i, result.Total, false, err)
			}
			return result
		}

		result.Completed++
		r.logger.Info("step completed",
			slog.String("module", mod.ID),
			slog.String("step", step.Name),
			slog.Duration("elapsed", elapsed),
		)
		if r.callback != nil {
			r.callback(mod, step, i, result.Total, false, nil)
		}
	}

	return result
}

// RunModules resolves dependencies for the given module IDs using the registry,
// then runs each module in topological order. It stops on the first module
// failure.
func (r *Runner) RunModules(ctx context.Context, reg *Registry, moduleIDs []string) ([]ModuleResult, error) {
	sorted, err := reg.ResolveDeps(moduleIDs)
	if err != nil {
		return nil, fmt.Errorf("resolving dependencies: %w", err)
	}

	results := make([]ModuleResult, 0, len(sorted))
	for _, id := range sorted {
		mod := reg.Get(id)
		if mod == nil {
			return results, fmt.Errorf("module %q not found in registry", id)
		}

		result := r.RunModule(ctx, mod)
		results = append(results, result)

		if result.Err != nil {
			return results, result.Err
		}
	}

	return results, nil
}
