package setup

import (
	"context"
	"os"
	"testing"

	"github.com/druarnfield/shhh/internal/config"
	"github.com/druarnfield/shhh/internal/exec"
	"github.com/druarnfield/shhh/internal/platform/mock"
	"github.com/druarnfield/shhh/internal/state"
)

func TestMain(m *testing.M) {
	code := m.Run()
	os.Unsetenv("HTTP_PROXY")
	os.Unsetenv("HTTPS_PROXY")
	os.Unsetenv("NO_PROXY")
	os.Unsetenv("SSL_CERT_FILE")
	os.Exit(code)
}

func testDeps() *Dependencies {
	return &Dependencies{
		Config:  testConfig(),
		Env:     mock.NewUserEnv(),
		Profile: mock.NewProfileManager("/tmp/test_profile.ps1"),
		Exec:    &exec.MockRunner{Results: map[string]exec.Result{}},
		State:   &state.State{},
	}
}

func testConfig() *config.Config {
	cfg := config.Defaults()
	cfg.Proxy.HTTP = "http://proxy:8080"
	cfg.Proxy.HTTPS = "http://proxy:8080"
	cfg.Proxy.NoProxy = "localhost,127.0.0.1,.internal"
	cfg.Certs.Source = "system"
	cfg.Git.DefaultBranch = "main"
	return cfg
}

func TestBaseModule_HasRequiredSteps(t *testing.T) {
	deps := testDeps()
	mod := NewBaseModule(deps)

	if mod.ID != "base" {
		t.Errorf("ID = %q, want %q", mod.ID, "base")
	}

	if len(mod.Steps) == 0 {
		t.Fatal("base module has no steps")
	}

	stepNames := make(map[string]bool)
	for _, s := range mod.Steps {
		stepNames[s.Name] = true
	}

	required := []string{"Set HTTP_PROXY", "Set HTTPS_PROXY", "Set NO_PROXY"}
	for _, name := range required {
		if !stepNames[name] {
			t.Errorf("missing required step: %q", name)
		}
	}
}

func TestProxySteps_SetEnvVars(t *testing.T) {
	deps := testDeps()
	mod := NewBaseModule(deps)
	ctx := context.Background()

	// Run proxy steps (first 3)
	for i := 0; i < 3 && i < len(mod.Steps); i++ {
		step := mod.Steps[i]
		if step.Check(ctx) {
			t.Errorf("step %q should not be already done", step.Name)
		}
		if err := step.Run(ctx); err != nil {
			t.Fatalf("step %q: %v", step.Name, err)
		}
	}

	// Verify env vars were set in mock
	val, _, err := deps.Env.Get("HTTP_PROXY")
	if err != nil {
		t.Fatalf("Get HTTP_PROXY: %v", err)
	}
	if val != "http://proxy:8080" {
		t.Errorf("HTTP_PROXY = %q, want %q", val, "http://proxy:8080")
	}

	// Verify in-process env was set
	if got := os.Getenv("HTTP_PROXY"); got != "http://proxy:8080" {
		t.Errorf("os.Getenv(HTTP_PROXY) = %q, want %q", got, "http://proxy:8080")
	}
}

func TestProxySteps_CheckSkipsIfDone(t *testing.T) {
	deps := testDeps()
	deps.Env.Set("HTTP_PROXY", "http://proxy:8080")
	os.Setenv("HTTP_PROXY", "http://proxy:8080")
	defer os.Unsetenv("HTTP_PROXY")

	mod := NewBaseModule(deps)
	ctx := context.Background()

	step := mod.Steps[0] // HTTP_PROXY step
	if !step.Check(ctx) {
		t.Error("step should be marked as done")
	}
}

func TestProxySteps_DryRun(t *testing.T) {
	deps := testDeps()
	mod := NewBaseModule(deps)
	ctx := context.Background()

	for i := 0; i < 3 && i < len(mod.Steps); i++ {
		step := mod.Steps[i]
		if step.DryRun == nil {
			t.Errorf("step %q has no DryRun", step.Name)
			continue
		}
		msg := step.DryRun(ctx)
		if msg == "" {
			t.Errorf("step %q DryRun returned empty", step.Name)
		}
	}
}
