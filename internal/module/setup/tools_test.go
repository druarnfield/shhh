package setup

import (
	"context"
	"testing"

	"github.com/druarnfield/shhh/internal/exec"
	"github.com/druarnfield/shhh/internal/state"
)

func TestToolsModule_HasRequiredSteps(t *testing.T) {
	deps := testDeps()
	mod := NewToolsModule(deps)

	if mod.ID != "tools" {
		t.Errorf("ID = %q, want %q", mod.ID, "tools")
	}
	if len(mod.Dependencies) == 0 || mod.Dependencies[0] != "base" {
		t.Error("expected dependency on base")
	}

	stepNames := make(map[string]bool)
	for _, s := range mod.Steps {
		stepNames[s.Name] = true
	}

	required := []string{"Install core tools", "Install data tools", "Install optional tools"}
	for _, name := range required {
		if !stepNames[name] {
			t.Errorf("missing required step: %q", name)
		}
	}
}

func TestToolsModule_EmptyListOmitsStep(t *testing.T) {
	deps := testDeps()
	deps.Config.Tools.Core = nil
	deps.Config.Tools.Data = nil
	deps.Config.Tools.Optional = nil
	mod := NewToolsModule(deps)

	if len(mod.Steps) != 0 {
		t.Errorf("expected 0 steps with empty tool lists, got %d", len(mod.Steps))
	}
}

func TestScoopInstallStep_Check(t *testing.T) {
	deps := testDeps()
	mockExec := deps.Exec.(*exec.MockRunner)
	ctx := context.Background()

	tools := []string{"git", "jq", "ripgrep"}
	step := scoopInstallStep(deps, "Install core tools", "desc", "explain", tools)

	// Check returns false when scoop list fails.
	if step.Check(ctx) {
		t.Error("Check should return false when scoop list fails")
	}

	// Check returns false when not all tools present.
	mockExec.Results["scoop list"] = exec.Result{Stdout: "git\njq\n", ExitCode: 0}
	if step.Check(ctx) {
		t.Error("Check should return false when not all tools are present")
	}

	// Check returns true when all tools present.
	mockExec.Results["scoop list"] = exec.Result{Stdout: "git\njq\nripgrep\n", ExitCode: 0}
	if !step.Check(ctx) {
		t.Error("Check should return true when all tools are present")
	}
}

func TestScoopInstallStep_Run_PartialInstall(t *testing.T) {
	deps := testDeps()
	deps.State = &state.State{}
	mockExec := deps.Exec.(*exec.MockRunner)
	// git already installed, jq and ripgrep need install.
	mockExec.Results["scoop list"] = exec.Result{Stdout: "git\n", ExitCode: 0}
	mockExec.Results["scoop install jq"] = exec.Result{ExitCode: 0}
	mockExec.Results["scoop install ripgrep"] = exec.Result{ExitCode: 0}
	ctx := context.Background()

	tools := []string{"git", "jq", "ripgrep"}
	step := scoopInstallStep(deps, "Install core tools", "desc", "explain", tools)
	if err := step.Run(ctx); err != nil {
		t.Fatalf("Run: %v", err)
	}

	// Should not have called scoop install git.
	for _, call := range mockExec.Calls {
		if call == "scoop install git" {
			t.Error("should not install already-present tool 'git'")
		}
	}

	// jq and ripgrep should be in state.
	if len(deps.State.ScoopPackages) != 2 {
		t.Errorf("expected 2 scoop packages, got %d", len(deps.State.ScoopPackages))
	}
}

func TestScoopInstallStep_Run_AllMissing(t *testing.T) {
	deps := testDeps()
	deps.State = &state.State{}
	mockExec := deps.Exec.(*exec.MockRunner)
	mockExec.Results["scoop list"] = exec.Result{Stdout: "", ExitCode: 0}
	mockExec.Results["scoop install bat"] = exec.Result{ExitCode: 0}
	mockExec.Results["scoop install lazygit"] = exec.Result{ExitCode: 0}
	ctx := context.Background()

	tools := []string{"bat", "lazygit"}
	step := scoopInstallStep(deps, "Install optional tools", "desc", "explain", tools)
	if err := step.Run(ctx); err != nil {
		t.Fatalf("Run: %v", err)
	}

	if len(deps.State.ScoopPackages) != 2 {
		t.Errorf("expected 2 scoop packages, got %d", len(deps.State.ScoopPackages))
	}
}

func TestScoopInstallStep_DryRun(t *testing.T) {
	deps := testDeps()
	ctx := context.Background()
	tools := []string{"git", "jq"}
	step := scoopInstallStep(deps, "Install core tools", "desc", "explain", tools)
	msg := step.DryRun(ctx)
	if msg == "" {
		t.Error("DryRun returned empty string")
	}
}
