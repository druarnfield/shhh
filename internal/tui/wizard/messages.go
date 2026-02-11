package wizard

import "github.com/druarnfield/shhh/internal/module"

// StepStartMsg is sent when a step begins executing.
type StepStartMsg struct {
	ModuleID string
	StepName string
	Explain  string
	Index    int
	Total    int
}

// StepDoneMsg is sent when a step completes (success or skip).
type StepDoneMsg struct {
	ModuleID string
	StepName string
	Index    int
	Total    int
	Skipped  bool
}

// StepErrorMsg is sent when a step fails.
type StepErrorMsg struct {
	ModuleID string
	StepName string
	Index    int
	Total    int
	Err      error
}

// ModuleStartMsg is sent when a module begins.
type ModuleStartMsg struct {
	ModuleID string
	Name     string
	Steps    []module.Step
}

// AllDoneMsg is sent when all modules have finished.
type AllDoneMsg struct {
	Results []module.ModuleResult
}

// RunErrorMsg is sent if the runner itself fails (e.g. dep resolution).
type RunErrorMsg struct {
	Err error
}
