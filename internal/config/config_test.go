package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadFromFile(t *testing.T) {
	content := `
[org]
name = "Test Org"

[proxy]
http = "http://proxy:8080"
https = "http://proxy:8080"
no_proxy = "localhost,127.0.0.1"

[certs]
source = "system"
extra = []

[git]
default_branch = "main"
ssh_hosts = ["gitlab.example.com"]

[gitlab]
host = "gitlab.example.com"
ssh_port = 22

[registries]
pypi_mirror = ""
npm_registry = ""
go_proxy = ""

[scoop]
buckets = ["extras"]

[tools]
core = ["git", "jq"]
data = ["sqlcmd"]
optional = ["bat"]

[python]
version = "3.12"

[golang]
version = "1.23"

[node]
version = "22"
`
	dir := t.TempDir()
	path := filepath.Join(dir, "shhh.toml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadFromFile(path)
	if err != nil {
		t.Fatalf("LoadFromFile: %v", err)
	}

	if cfg.Org.Name != "Test Org" {
		t.Errorf("org.name = %q, want %q", cfg.Org.Name, "Test Org")
	}
	if cfg.Proxy.HTTP != "http://proxy:8080" {
		t.Errorf("proxy.http = %q, want %q", cfg.Proxy.HTTP, "http://proxy:8080")
	}
	if cfg.Git.DefaultBranch != "main" {
		t.Errorf("git.default_branch = %q, want %q", cfg.Git.DefaultBranch, "main")
	}
	if len(cfg.Tools.Core) != 2 {
		t.Errorf("tools.core len = %d, want 2", len(cfg.Tools.Core))
	}
	if cfg.Python.Version != "3.12" {
		t.Errorf("python.version = %q, want %q", cfg.Python.Version, "3.12")
	}
}

func TestLoadFromFile_NotFound(t *testing.T) {
	_, err := LoadFromFile("/nonexistent/shhh.toml")
	if err == nil {
		t.Error("expected error for missing file")
	}
}

func TestDefaults(t *testing.T) {
	cfg := Defaults()
	if cfg.Git.DefaultBranch != "main" {
		t.Errorf("default git.default_branch = %q, want %q", cfg.Git.DefaultBranch, "main")
	}
	if cfg.Certs.Source != "system" {
		t.Errorf("default certs.source = %q, want %q", cfg.Certs.Source, "system")
	}
}
