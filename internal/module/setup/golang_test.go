package setup

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/druarnfield/shhh/internal/exec"
	"github.com/druarnfield/shhh/internal/state"
)

func TestGolangModule_HasRequiredSteps(t *testing.T) {
	deps := testDeps()
	mod := NewGolangModule(deps)

	if mod.ID != "golang" {
		t.Errorf("ID = %q, want %q", mod.ID, "golang")
	}
	if len(mod.Dependencies) == 0 || mod.Dependencies[0] != "base" {
		t.Error("expected dependency on base")
	}

	stepNames := make(map[string]bool)
	for _, s := range mod.Steps {
		stepNames[s.Name] = true
	}

	required := []string{"Install Go", "Set GOPATH", "Add GOBIN to PATH", "Configure GOPROXY"}
	for _, name := range required {
		if !stepNames[name] {
			t.Errorf("missing required step: %q", name)
		}
	}
}

func TestInstallGoStep_Check(t *testing.T) {
	deps := testDeps()
	mockExec := deps.Exec.(*exec.MockRunner)
	ctx := context.Background()

	step := installGoStep(deps)

	// Check returns false when go not found.
	if step.Check(ctx) {
		t.Error("Check should return false when go is not installed")
	}

	// Check returns false when wrong version.
	mockExec.Results["go version"] = exec.Result{Stdout: "go version go1.21.0 windows/amd64\n", ExitCode: 0}
	if step.Check(ctx) {
		t.Error("Check should return false for wrong version")
	}

	// Check returns true when version matches.
	mockExec.Results["go version"] = exec.Result{Stdout: "go version go1.23.0 windows/amd64\n", ExitCode: 0}
	if !step.Check(ctx) {
		t.Error("Check should return true when version matches")
	}
}

func TestInstallGoStep_Run(t *testing.T) {
	deps := testDeps()
	deps.State = &state.State{}
	mockExec := deps.Exec.(*exec.MockRunner)
	mockExec.Results["scoop install go"] = exec.Result{ExitCode: 0}
	ctx := context.Background()

	step := installGoStep(deps)
	if err := step.Run(ctx); err != nil {
		t.Fatalf("Run: %v", err)
	}

	if len(deps.State.ScoopPackages) == 0 || deps.State.ScoopPackages[0] != "go" {
		t.Error("expected 'go' in scoop packages")
	}
}

func TestInstallGoStep_DryRun(t *testing.T) {
	deps := testDeps()
	ctx := context.Background()
	step := installGoStep(deps)
	msg := step.DryRun(ctx)
	if msg == "" {
		t.Error("DryRun returned empty string")
	}
}

func TestSetGOPATHStep_Check(t *testing.T) {
	deps := testDeps()
	ctx := context.Background()
	home, _ := os.UserHomeDir()
	gopath := filepath.Join(home, "go")

	step := setGOPATHStep(deps)

	// Check returns false initially.
	if step.Check(ctx) {
		t.Error("Check should return false initially")
	}

	// Set in mock env but not in process.
	deps.Env.Set("GOPATH", gopath)
	if step.Check(ctx) {
		t.Error("Check should return false when only in mock env")
	}

	// Set in both.
	os.Setenv("GOPATH", gopath)
	t.Cleanup(func() { os.Unsetenv("GOPATH") })
	if !step.Check(ctx) {
		t.Error("Check should return true when set in both")
	}
}

func TestSetGOPATHStep_Run(t *testing.T) {
	deps := testDeps()
	deps.State = &state.State{}
	ctx := context.Background()
	home, _ := os.UserHomeDir()
	gopath := filepath.Join(home, "go")

	step := setGOPATHStep(deps)
	if err := step.Run(ctx); err != nil {
		t.Fatalf("Run: %v", err)
	}

	val, _, err := deps.Env.Get("GOPATH")
	if err != nil {
		t.Fatalf("Get GOPATH: %v", err)
	}
	if val != gopath {
		t.Errorf("GOPATH = %q, want %q", val, gopath)
	}
	if got := os.Getenv("GOPATH"); got != gopath {
		t.Errorf("os.Getenv(GOPATH) = %q, want %q", got, gopath)
	}
	t.Cleanup(func() { os.Unsetenv("GOPATH") })
}

func TestAddGOBINStep_Check(t *testing.T) {
	deps := testDeps()
	ctx := context.Background()
	home, _ := os.UserHomeDir()
	gobin := filepath.Join(home, "go", "bin")

	step := addGOBINStep(deps)

	// Check returns false initially.
	if step.Check(ctx) {
		t.Error("Check should return false initially")
	}

	// After appending to path, check returns true.
	deps.Env.AppendPath(gobin)
	if !step.Check(ctx) {
		t.Error("Check should return true after path is added")
	}
}

func TestAddGOBINStep_Run(t *testing.T) {
	deps := testDeps()
	deps.State = &state.State{}
	ctx := context.Background()
	home, _ := os.UserHomeDir()
	gobin := filepath.Join(home, "go", "bin")

	step := addGOBINStep(deps)
	if err := step.Run(ctx); err != nil {
		t.Fatalf("Run: %v", err)
	}

	entries, _ := deps.Env.ListPath()
	found := false
	for _, e := range entries {
		if e.Dir == gobin {
			found = true
		}
	}
	if !found {
		t.Error("GOBIN not found in PATH entries")
	}
	if len(deps.State.ManagedPathEntries) == 0 {
		t.Error("expected GOBIN in managed path entries")
	}
}

func TestConfigureGOPROXYStep_Check(t *testing.T) {
	deps := testDeps()
	mockExec := deps.Exec.(*exec.MockRunner)
	ctx := context.Background()

	step := configureGOPROXYStep(deps)

	// Check returns false when go env fails.
	if step.Check(ctx) {
		t.Error("Check should return false when go env GOPROXY fails")
	}

	// Check returns false for wrong proxy.
	mockExec.Results["go env GOPROXY"] = exec.Result{Stdout: "https://proxy.golang.org,direct\n", ExitCode: 0}
	if step.Check(ctx) {
		t.Error("Check should return false for wrong proxy")
	}

	// Check returns true when matching.
	mockExec.Results["go env GOPROXY"] = exec.Result{Stdout: "https://goproxy.example.com\n", ExitCode: 0}
	if !step.Check(ctx) {
		t.Error("Check should return true when proxy matches")
	}
}

func TestConfigureGOPROXYStep_Run(t *testing.T) {
	deps := testDeps()
	mockExec := deps.Exec.(*exec.MockRunner)
	mockExec.Results["go env -w GOPROXY=https://goproxy.example.com"] = exec.Result{ExitCode: 0}
	deps.State = &state.State{}
	ctx := context.Background()

	step := configureGOPROXYStep(deps)
	if err := step.Run(ctx); err != nil {
		t.Fatalf("Run: %v", err)
	}

	if got := os.Getenv("GOPROXY"); got != "https://goproxy.example.com" {
		t.Errorf("os.Getenv(GOPROXY) = %q, want %q", got, "https://goproxy.example.com")
	}
	t.Cleanup(func() { os.Unsetenv("GOPROXY") })
}

func TestGolangModule_GOPROXYOmitted_WhenEmpty(t *testing.T) {
	deps := testDeps()
	deps.Config.Registries.GoProxy = ""
	mod := NewGolangModule(deps)

	for _, s := range mod.Steps {
		if s.Name == "Configure GOPROXY" {
			t.Error("Configure GOPROXY step should be omitted when GoProxy is empty")
		}
	}
}
