package internal

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/druarnfield/shhh/internal/config"
	"github.com/druarnfield/shhh/internal/exec"
	"github.com/druarnfield/shhh/internal/logging"
	"github.com/druarnfield/shhh/internal/module"
	"github.com/druarnfield/shhh/internal/module/setup"
	"github.com/druarnfield/shhh/internal/platform/mock"
	"github.com/druarnfield/shhh/internal/state"
)

func TestFullSetupFlow(t *testing.T) {
	t.Cleanup(func() {
		os.Unsetenv("HTTP_PROXY")
		os.Unsetenv("HTTPS_PROXY")
		os.Unsetenv("NO_PROXY")
		os.Unsetenv("SSL_CERT_FILE")
	})

	// Config
	cfg := config.Defaults()
	cfg.Org.Name = "Test Org"
	cfg.Proxy.HTTP = "http://proxy:8080"
	cfg.Proxy.HTTPS = "http://proxy:8080"
	cfg.Proxy.NoProxy = "localhost,127.0.0.1"
	cfg.Git.DefaultBranch = "main"

	// Mocks
	env := mock.NewUserEnv()
	prof := mock.NewProfileManager("/tmp/test_profile.ps1")
	mockExec := &exec.MockRunner{
		Results: map[string]exec.Result{
			"git config --global init.defaultBranch":      {Stdout: "", ExitCode: 1},
			"git config --global init.defaultBranch main": {Stdout: "", ExitCode: 0},
			"git config --global http.sslCAInfo":          {Stdout: "", ExitCode: 1},
			"git config --global http.sslCAInfo " + config.CABundlePath(): {Stdout: "", ExitCode: 0},
		},
	}
	st := &state.State{}

	deps := &setup.Dependencies{
		Config:  cfg,
		Env:     env,
		Profile: prof,
		Exec:    mockExec,
		State:   st,
	}

	// Registry
	reg := module.NewRegistry()
	reg.Register(setup.NewBaseModule(deps))

	// Run
	logger := slog.New(logging.NopHandler{})
	runner := module.NewRunner(logger, false)

	results, err := runner.RunModules(context.Background(), reg, []string{"base"})
	if err != nil {
		t.Fatalf("RunModules: %v", err)
	}

	// Verify results
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	r := results[0]
	if r.Err != nil {
		t.Errorf("module error: %v", r.Err)
	}
	if r.Completed == 0 {
		t.Error("no steps completed")
	}

	// Verify env vars set in mock
	val, _, err := env.Get("HTTP_PROXY")
	if err != nil {
		t.Errorf("HTTP_PROXY not set in mock env: %v", err)
	}
	if val != "http://proxy:8080" {
		t.Errorf("HTTP_PROXY = %q", val)
	}

	// Verify in-process env
	if got := os.Getenv("HTTP_PROXY"); got != "http://proxy:8080" {
		t.Errorf("os HTTP_PROXY = %q", got)
	}

	// Verify state updated
	if len(st.ManagedEnvVars) == 0 {
		t.Error("state has no managed env vars")
	}

	// Run again — should skip all steps (idempotency)
	// Need fresh mock exec that returns the "already configured" values
	mockExec2 := &exec.MockRunner{
		Results: map[string]exec.Result{
			"git config --global init.defaultBranch": {Stdout: "main\n", ExitCode: 0},
			"git config --global http.sslCAInfo":     {Stdout: config.CABundlePath() + "\n", ExitCode: 0},
		},
	}
	deps2 := &setup.Dependencies{
		Config:  cfg,
		Env:     env,
		Profile: prof,
		Exec:    mockExec2,
		State:   st,
	}
	reg2 := module.NewRegistry()
	reg2.Register(setup.NewBaseModule(deps2))

	runner2 := module.NewRunner(logger, false)
	results2, err := runner2.RunModules(context.Background(), reg2, []string{"base"})
	if err != nil {
		t.Fatalf("second RunModules: %v", err)
	}
	if results2[0].Skipped == 0 {
		t.Error("second run should have skipped steps")
	}
	if results2[0].Completed != 0 {
		t.Errorf("second run completed %d steps, want 0", results2[0].Completed)
	}

	// Verify state round-trip
	dir := t.TempDir()
	statePath := filepath.Join(dir, "state.json")
	if err := state.Save(statePath, st); err != nil {
		t.Fatalf("Save state: %v", err)
	}
	loaded, err := state.Load(statePath)
	if err != nil {
		t.Fatalf("Load state: %v", err)
	}
	if len(loaded.ManagedEnvVars) != len(st.ManagedEnvVars) {
		t.Error("state round-trip mismatch")
	}
}
