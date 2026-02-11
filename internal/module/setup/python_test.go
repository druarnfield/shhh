package setup

import (
	"context"
	"os"
	"testing"

	"github.com/druarnfield/shhh/internal/config"
	"github.com/druarnfield/shhh/internal/exec"
	"github.com/druarnfield/shhh/internal/state"
)

func TestPythonModule_HasRequiredSteps(t *testing.T) {
	deps := testDeps()
	mod := NewPythonModule(deps)

	if mod.ID != "python" {
		t.Errorf("ID = %q, want %q", mod.ID, "python")
	}
	if len(mod.Dependencies) == 0 || mod.Dependencies[0] != "base" {
		t.Error("expected dependency on base")
	}

	stepNames := make(map[string]bool)
	for _, s := range mod.Steps {
		stepNames[s.Name] = true
	}

	required := []string{"Install uv", "Install Python", "Configure Python CA certificates", "Set UV_PYTHON_PREFERENCE", "Configure PyPI mirror"}
	for _, name := range required {
		if !stepNames[name] {
			t.Errorf("missing required step: %q", name)
		}
	}
}

func TestInstallUVStep_Check(t *testing.T) {
	deps := testDeps()
	mockExec := deps.Exec.(*exec.MockRunner)
	ctx := context.Background()

	step := installUVStep(deps)

	if step.Check(ctx) {
		t.Error("Check should return false when uv is not installed")
	}

	mockExec.Results["uv --version"] = exec.Result{Stdout: "uv 0.4.0\n", ExitCode: 0}
	if !step.Check(ctx) {
		t.Error("Check should return true when uv is installed")
	}
}

func TestInstallUVStep_Run(t *testing.T) {
	deps := testDeps()
	deps.State = &state.State{}
	mockExec := deps.Exec.(*exec.MockRunner)
	mockExec.Results["scoop install uv"] = exec.Result{ExitCode: 0}
	ctx := context.Background()

	step := installUVStep(deps)
	if err := step.Run(ctx); err != nil {
		t.Fatalf("Run: %v", err)
	}

	if len(deps.State.ScoopPackages) == 0 || deps.State.ScoopPackages[0] != "uv" {
		t.Error("expected 'uv' in scoop packages")
	}
}

func TestInstallUVStep_DryRun(t *testing.T) {
	deps := testDeps()
	ctx := context.Background()
	step := installUVStep(deps)
	msg := step.DryRun(ctx)
	if msg == "" {
		t.Error("DryRun returned empty string")
	}
}

func TestInstallPythonStep_Check(t *testing.T) {
	deps := testDeps()
	mockExec := deps.Exec.(*exec.MockRunner)
	ctx := context.Background()

	step := installPythonStep(deps)

	if step.Check(ctx) {
		t.Error("Check should return false when python not installed via uv")
	}

	mockExec.Results["uv python list --only-installed"] = exec.Result{Stdout: "cpython-3.11.0\n", ExitCode: 0}
	if step.Check(ctx) {
		t.Error("Check should return false for wrong version")
	}

	mockExec.Results["uv python list --only-installed"] = exec.Result{Stdout: "cpython-3.12.0\n", ExitCode: 0}
	if !step.Check(ctx) {
		t.Error("Check should return true when version matches")
	}
}

func TestInstallPythonStep_Run(t *testing.T) {
	deps := testDeps()
	mockExec := deps.Exec.(*exec.MockRunner)
	mockExec.Results["uv python install 3.12"] = exec.Result{ExitCode: 0}
	ctx := context.Background()

	step := installPythonStep(deps)
	if err := step.Run(ctx); err != nil {
		t.Fatalf("Run: %v", err)
	}
}

func TestInstallPythonStep_DryRun(t *testing.T) {
	deps := testDeps()
	ctx := context.Background()
	step := installPythonStep(deps)
	msg := step.DryRun(ctx)
	if msg == "" {
		t.Error("DryRun returned empty string")
	}
}

func TestConfigurePythonCertsStep_Check(t *testing.T) {
	deps := testDeps()
	ctx := context.Background()
	caPath := config.CABundlePath()

	step := configurePythonCertsStep(deps)

	if step.Check(ctx) {
		t.Error("Check should return false initially")
	}

	// Only one key set.
	deps.Env.Set("REQUESTS_CA_BUNDLE", caPath)
	if step.Check(ctx) {
		t.Error("Check should return false when only REQUESTS_CA_BUNDLE is set")
	}

	// Both in env but not process.
	deps.Env.Set("PIP_CERT", caPath)
	if step.Check(ctx) {
		t.Error("Check should return false when not in process env")
	}

	// All set.
	os.Setenv("REQUESTS_CA_BUNDLE", caPath)
	os.Setenv("PIP_CERT", caPath)
	t.Cleanup(func() {
		os.Unsetenv("REQUESTS_CA_BUNDLE")
		os.Unsetenv("PIP_CERT")
	})
	if !step.Check(ctx) {
		t.Error("Check should return true when all set")
	}
}

func TestConfigurePythonCertsStep_Run(t *testing.T) {
	deps := testDeps()
	deps.State = &state.State{}
	ctx := context.Background()
	caPath := config.CABundlePath()

	step := configurePythonCertsStep(deps)
	if err := step.Run(ctx); err != nil {
		t.Fatalf("Run: %v", err)
	}

	for _, key := range []string{"REQUESTS_CA_BUNDLE", "PIP_CERT"} {
		val, _, err := deps.Env.Get(key)
		if err != nil {
			t.Fatalf("Get %s: %v", key, err)
		}
		if val != caPath {
			t.Errorf("%s = %q, want %q", key, val, caPath)
		}
		if got := os.Getenv(key); got != caPath {
			t.Errorf("os.Getenv(%s) = %q, want %q", key, got, caPath)
		}
	}
	t.Cleanup(func() {
		os.Unsetenv("REQUESTS_CA_BUNDLE")
		os.Unsetenv("PIP_CERT")
	})

	if len(deps.State.ManagedEnvVars) < 2 {
		t.Errorf("expected at least 2 managed env vars, got %d", len(deps.State.ManagedEnvVars))
	}
}

func TestConfigurePythonCertsStep_DryRun(t *testing.T) {
	deps := testDeps()
	ctx := context.Background()
	step := configurePythonCertsStep(deps)
	msg := step.DryRun(ctx)
	if msg == "" {
		t.Error("DryRun returned empty string")
	}
}

func TestSetUVPythonPreferenceStep_Check(t *testing.T) {
	deps := testDeps()
	ctx := context.Background()

	step := setUVPythonPreferenceStep(deps)

	if step.Check(ctx) {
		t.Error("Check should return false initially")
	}

	deps.Env.Set("UV_PYTHON_PREFERENCE", "only-managed")
	if step.Check(ctx) {
		t.Error("Check should return false when only in mock env")
	}

	os.Setenv("UV_PYTHON_PREFERENCE", "only-managed")
	t.Cleanup(func() { os.Unsetenv("UV_PYTHON_PREFERENCE") })
	if !step.Check(ctx) {
		t.Error("Check should return true when set in both")
	}
}

func TestSetUVPythonPreferenceStep_Run(t *testing.T) {
	deps := testDeps()
	deps.State = &state.State{}
	ctx := context.Background()

	step := setUVPythonPreferenceStep(deps)
	if err := step.Run(ctx); err != nil {
		t.Fatalf("Run: %v", err)
	}

	val, _, err := deps.Env.Get("UV_PYTHON_PREFERENCE")
	if err != nil {
		t.Fatalf("Get UV_PYTHON_PREFERENCE: %v", err)
	}
	if val != "only-managed" {
		t.Errorf("UV_PYTHON_PREFERENCE = %q, want %q", val, "only-managed")
	}
	if got := os.Getenv("UV_PYTHON_PREFERENCE"); got != "only-managed" {
		t.Errorf("os.Getenv(UV_PYTHON_PREFERENCE) = %q, want %q", got, "only-managed")
	}
	t.Cleanup(func() { os.Unsetenv("UV_PYTHON_PREFERENCE") })
}

func TestConfigurePyPIMirrorStep_Check(t *testing.T) {
	deps := testDeps()
	ctx := context.Background()
	mirror := deps.Config.Registries.PyPIMirror

	step := configurePyPIMirrorStep(deps)

	if step.Check(ctx) {
		t.Error("Check should return false initially")
	}

	// Only UV_INDEX_URL set.
	deps.Env.Set("UV_INDEX_URL", mirror)
	if step.Check(ctx) {
		t.Error("Check should return false when only UV_INDEX_URL is set")
	}

	// Both set in env but not process.
	deps.Env.Set("PIP_INDEX_URL", mirror)
	if step.Check(ctx) {
		t.Error("Check should return false when not in process env")
	}

	// All set.
	os.Setenv("UV_INDEX_URL", mirror)
	os.Setenv("PIP_INDEX_URL", mirror)
	t.Cleanup(func() {
		os.Unsetenv("UV_INDEX_URL")
		os.Unsetenv("PIP_INDEX_URL")
	})
	if !step.Check(ctx) {
		t.Error("Check should return true when all set")
	}
}

func TestConfigurePyPIMirrorStep_Run(t *testing.T) {
	deps := testDeps()
	deps.State = &state.State{}
	ctx := context.Background()
	mirror := deps.Config.Registries.PyPIMirror

	step := configurePyPIMirrorStep(deps)
	if err := step.Run(ctx); err != nil {
		t.Fatalf("Run: %v", err)
	}

	for _, key := range []string{"UV_INDEX_URL", "PIP_INDEX_URL"} {
		val, _, err := deps.Env.Get(key)
		if err != nil {
			t.Fatalf("Get %s: %v", key, err)
		}
		if val != mirror {
			t.Errorf("%s = %q, want %q", key, val, mirror)
		}
		if got := os.Getenv(key); got != mirror {
			t.Errorf("os.Getenv(%s) = %q, want %q", key, got, mirror)
		}
	}
	t.Cleanup(func() {
		os.Unsetenv("UV_INDEX_URL")
		os.Unsetenv("PIP_INDEX_URL")
	})
}

func TestConfigurePyPIMirrorStep_DryRun(t *testing.T) {
	deps := testDeps()
	ctx := context.Background()
	step := configurePyPIMirrorStep(deps)
	msg := step.DryRun(ctx)
	if msg == "" {
		t.Error("DryRun returned empty string")
	}
}

func TestPythonModule_PyPIOmitted_WhenEmpty(t *testing.T) {
	deps := testDeps()
	deps.Config.Registries.PyPIMirror = ""
	mod := NewPythonModule(deps)

	for _, s := range mod.Steps {
		if s.Name == "Configure PyPI mirror" {
			t.Error("Configure PyPI mirror step should be omitted when PyPIMirror is empty")
		}
	}
}
