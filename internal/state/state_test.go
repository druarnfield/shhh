package state

import (
	"path/filepath"
	"testing"
	"time"
)

func TestState_SaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")

	s := &State{
		InstalledModules:   []string{"base", "python"},
		LastRun:            time.Date(2025, 2, 10, 14, 30, 0, 0, time.UTC),
		ManagedEnvVars:     []string{"HTTP_PROXY", "HTTPS_PROXY"},
		ManagedPathEntries: []string{`C:\Users\dru\scoop\shims`},
		ScoopPackages:      []string{"git", "jq"},
		CABundleHash:       "sha256:abc123",
		ShhhVersion:        "0.1.0",
	}

	if err := Save(path, s); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if len(loaded.InstalledModules) != 2 {
		t.Errorf("InstalledModules = %v", loaded.InstalledModules)
	}
	if loaded.ShhhVersion != "0.1.0" {
		t.Errorf("ShhhVersion = %q", loaded.ShhhVersion)
	}
	if loaded.CABundleHash != "sha256:abc123" {
		t.Errorf("CABundleHash = %q", loaded.CABundleHash)
	}
}

func TestLoad_Missing(t *testing.T) {
	s, err := Load("/nonexistent/state.json")
	if err != nil {
		t.Fatalf("Load should not error on missing file: %v", err)
	}
	if s == nil {
		t.Fatal("should return empty state")
	}
	if len(s.InstalledModules) != 0 {
		t.Error("should be empty state")
	}
}

func TestState_AddModule(t *testing.T) {
	s := &State{}
	s.AddModule("base")
	s.AddModule("python")
	s.AddModule("base") // duplicate

	if len(s.InstalledModules) != 2 {
		t.Errorf("InstalledModules = %v, want [base, python]", s.InstalledModules)
	}
}

func TestState_AddEnvVar(t *testing.T) {
	s := &State{}
	s.AddEnvVar("HTTP_PROXY")
	s.AddEnvVar("HTTP_PROXY") // duplicate

	if len(s.ManagedEnvVars) != 1 {
		t.Errorf("ManagedEnvVars = %v", s.ManagedEnvVars)
	}
}
