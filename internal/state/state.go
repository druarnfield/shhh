package state

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

type State struct {
	InstalledModules   []string  `json:"installed_modules"`
	LastRun            time.Time `json:"last_run"`
	ManagedEnvVars     []string  `json:"managed_env_vars"`
	ManagedPathEntries []string  `json:"managed_path_entries"`
	ScoopPackages      []string  `json:"scoop_packages"`
	CABundleHash       string    `json:"ca_bundle_hash"`
	ShhhVersion        string    `json:"shhh_version"`
}

func Load(path string) (*State, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &State{}, nil
		}
		return nil, err
	}

	var s State
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, err
	}
	return &s, nil
}

func Save(path string, s *State) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func (s *State) AddModule(id string) {
	if !contains(s.InstalledModules, id) {
		s.InstalledModules = append(s.InstalledModules, id)
	}
}

func (s *State) AddEnvVar(key string) {
	if !contains(s.ManagedEnvVars, key) {
		s.ManagedEnvVars = append(s.ManagedEnvVars, key)
	}
}

func (s *State) AddPathEntry(dir string) {
	if !contains(s.ManagedPathEntries, dir) {
		s.ManagedPathEntries = append(s.ManagedPathEntries, dir)
	}
}

func (s *State) AddScoopPackage(pkg string) {
	if !contains(s.ScoopPackages, pkg) {
		s.ScoopPackages = append(s.ScoopPackages, pkg)
	}
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
