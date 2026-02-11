package setup

import (
	"context"
	"os"
	"testing"

	"github.com/druarnfield/shhh/internal/config"
	"github.com/druarnfield/shhh/internal/exec"
	"github.com/druarnfield/shhh/internal/state"
)

func TestNodeModule_HasRequiredSteps(t *testing.T) {
	deps := testDeps()
	mod := NewNodeModule(deps)

	if mod.ID != "node" {
		t.Errorf("ID = %q, want %q", mod.ID, "node")
	}
	if len(mod.Dependencies) == 0 || mod.Dependencies[0] != "base" {
		t.Error("expected dependency on base")
	}

	stepNames := make(map[string]bool)
	for _, s := range mod.Steps {
		stepNames[s.Name] = true
	}

	required := []string{"Install fnm", "Configure fnm shell", "Install Node.js", "Configure Node.js CA certificates", "Configure npm registry"}
	for _, name := range required {
		if !stepNames[name] {
			t.Errorf("missing required step: %q", name)
		}
	}
}

func TestInstallFnmStep_Check(t *testing.T) {
	deps := testDeps()
	mockExec := deps.Exec.(*exec.MockRunner)
	ctx := context.Background()

	step := installFnmStep(deps)

	if step.Check(ctx) {
		t.Error("Check should return false when fnm is not installed")
	}

	mockExec.Results["fnm --version"] = exec.Result{Stdout: "fnm 1.37.0\n", ExitCode: 0}
	if !step.Check(ctx) {
		t.Error("Check should return true when fnm is installed")
	}
}

func TestInstallFnmStep_Run(t *testing.T) {
	deps := testDeps()
	deps.State = &state.State{}
	mockExec := deps.Exec.(*exec.MockRunner)
	mockExec.Results["scoop install fnm"] = exec.Result{ExitCode: 0}
	ctx := context.Background()

	step := installFnmStep(deps)
	if err := step.Run(ctx); err != nil {
		t.Fatalf("Run: %v", err)
	}

	if len(deps.State.ScoopPackages) == 0 || deps.State.ScoopPackages[0] != "fnm" {
		t.Error("expected 'fnm' in scoop packages")
	}
}

func TestInstallFnmStep_DryRun(t *testing.T) {
	deps := testDeps()
	ctx := context.Background()
	step := installFnmStep(deps)
	msg := step.DryRun(ctx)
	if msg == "" {
		t.Error("DryRun returned empty string")
	}
}

func TestConfigureFnmShellStep_Check(t *testing.T) {
	deps := testDeps()
	ctx := context.Background()

	step := configureFnmShellStep(deps)

	if step.Check(ctx) {
		t.Error("Check should return false when profile has no fnm init")
	}

	deps.Profile.AppendToManagedBlock(`fnm env --use-on-cd --shell power-shell | Out-String | Invoke-Expression`)
	if !step.Check(ctx) {
		t.Error("Check should return true after fnm init added to profile")
	}
}

func TestConfigureFnmShellStep_Run(t *testing.T) {
	deps := testDeps()
	ctx := context.Background()

	step := configureFnmShellStep(deps)
	if err := step.Run(ctx); err != nil {
		t.Fatalf("Run: %v", err)
	}

	block, _ := deps.Profile.ManagedBlock()
	if block == "" {
		t.Error("managed block should not be empty after Run")
	}
}

func TestConfigureFnmShellStep_DryRun(t *testing.T) {
	deps := testDeps()
	ctx := context.Background()
	step := configureFnmShellStep(deps)
	msg := step.DryRun(ctx)
	if msg == "" {
		t.Error("DryRun returned empty string")
	}
}

func TestInstallNodeStep_Check(t *testing.T) {
	deps := testDeps()
	mockExec := deps.Exec.(*exec.MockRunner)
	ctx := context.Background()

	step := installNodeStep(deps)

	if step.Check(ctx) {
		t.Error("Check should return false when fnm list fails")
	}

	mockExec.Results["fnm list"] = exec.Result{Stdout: "* v20.11.0\n", ExitCode: 0}
	if step.Check(ctx) {
		t.Error("Check should return false for wrong version")
	}

	mockExec.Results["fnm list"] = exec.Result{Stdout: "* v22.0.0\n  v20.11.0\n", ExitCode: 0}
	if !step.Check(ctx) {
		t.Error("Check should return true when version matches")
	}
}

func TestInstallNodeStep_Run(t *testing.T) {
	deps := testDeps()
	mockExec := deps.Exec.(*exec.MockRunner)
	mockExec.Results["fnm install 22"] = exec.Result{ExitCode: 0}
	mockExec.Results["fnm default 22"] = exec.Result{ExitCode: 0}
	ctx := context.Background()

	step := installNodeStep(deps)
	if err := step.Run(ctx); err != nil {
		t.Fatalf("Run: %v", err)
	}
}

func TestInstallNodeStep_DryRun(t *testing.T) {
	deps := testDeps()
	ctx := context.Background()
	step := installNodeStep(deps)
	msg := step.DryRun(ctx)
	if msg == "" {
		t.Error("DryRun returned empty string")
	}
}

func TestConfigureNodeCertsStep_Check(t *testing.T) {
	deps := testDeps()
	mockExec := deps.Exec.(*exec.MockRunner)
	ctx := context.Background()
	caPath := config.CABundlePath()

	step := configureNodeCertsStep(deps)

	// Check returns false initially.
	if step.Check(ctx) {
		t.Error("Check should return false initially")
	}

	// Env var set but not process env.
	deps.Env.Set("NODE_EXTRA_CA_CERTS", caPath)
	if step.Check(ctx) {
		t.Error("Check should return false when only in mock env")
	}

	// Process env set too but npm cafile not set.
	os.Setenv("NODE_EXTRA_CA_CERTS", caPath)
	t.Cleanup(func() { os.Unsetenv("NODE_EXTRA_CA_CERTS") })
	if step.Check(ctx) {
		t.Error("Check should return false when npm cafile not set")
	}

	// All set.
	mockExec.Results["fnm exec --using 22 -- npm config get cafile"] = exec.Result{Stdout: caPath + "\n", ExitCode: 0}
	if !step.Check(ctx) {
		t.Error("Check should return true when all set")
	}
}

func TestConfigureNodeCertsStep_Run(t *testing.T) {
	deps := testDeps()
	deps.State = &state.State{}
	mockExec := deps.Exec.(*exec.MockRunner)
	caPath := config.CABundlePath()
	mockExec.Results["fnm exec --using 22 -- npm config set cafile "+caPath] = exec.Result{ExitCode: 0}
	ctx := context.Background()

	step := configureNodeCertsStep(deps)
	if err := step.Run(ctx); err != nil {
		t.Fatalf("Run: %v", err)
	}

	val, _, err := deps.Env.Get("NODE_EXTRA_CA_CERTS")
	if err != nil {
		t.Fatalf("Get NODE_EXTRA_CA_CERTS: %v", err)
	}
	if val != caPath {
		t.Errorf("NODE_EXTRA_CA_CERTS = %q, want %q", val, caPath)
	}
	if got := os.Getenv("NODE_EXTRA_CA_CERTS"); got != caPath {
		t.Errorf("os.Getenv(NODE_EXTRA_CA_CERTS) = %q, want %q", got, caPath)
	}
	t.Cleanup(func() { os.Unsetenv("NODE_EXTRA_CA_CERTS") })

	if len(deps.State.ManagedEnvVars) == 0 {
		t.Error("expected NODE_EXTRA_CA_CERTS in managed env vars")
	}
}

func TestConfigureNodeCertsStep_DryRun(t *testing.T) {
	deps := testDeps()
	ctx := context.Background()
	step := configureNodeCertsStep(deps)
	msg := step.DryRun(ctx)
	if msg == "" {
		t.Error("DryRun returned empty string")
	}
}

func TestConfigureNPMRegistryStep_Check(t *testing.T) {
	deps := testDeps()
	mockExec := deps.Exec.(*exec.MockRunner)
	ctx := context.Background()

	step := configureNPMRegistryStep(deps)

	if step.Check(ctx) {
		t.Error("Check should return false when npm config fails")
	}

	// Wrong registry.
	mockExec.Results["fnm exec --using 22 -- npm config get registry"] = exec.Result{Stdout: "https://registry.npmjs.org/\n", ExitCode: 0}
	if step.Check(ctx) {
		t.Error("Check should return false for wrong registry")
	}

	// Correct registry (with trailing slash normalization).
	mockExec.Results["fnm exec --using 22 -- npm config get registry"] = exec.Result{Stdout: "https://npm.example.com/\n", ExitCode: 0}
	if !step.Check(ctx) {
		t.Error("Check should return true when registry matches (with trailing slash)")
	}
}

func TestConfigureNPMRegistryStep_Run(t *testing.T) {
	deps := testDeps()
	mockExec := deps.Exec.(*exec.MockRunner)
	mockExec.Results["fnm exec --using 22 -- npm config set registry https://npm.example.com/"] = exec.Result{ExitCode: 0}
	ctx := context.Background()

	step := configureNPMRegistryStep(deps)
	if err := step.Run(ctx); err != nil {
		t.Fatalf("Run: %v", err)
	}
}

func TestConfigureNPMRegistryStep_DryRun(t *testing.T) {
	deps := testDeps()
	ctx := context.Background()
	step := configureNPMRegistryStep(deps)
	msg := step.DryRun(ctx)
	if msg == "" {
		t.Error("DryRun returned empty string")
	}
}

func TestNodeModule_NPMRegistryOmitted_WhenEmpty(t *testing.T) {
	deps := testDeps()
	deps.Config.Registries.NPMRegistry = ""
	mod := NewNodeModule(deps)

	for _, s := range mod.Steps {
		if s.Name == "Configure npm registry" {
			t.Error("Configure npm registry step should be omitted when NPMRegistry is empty")
		}
	}
}
