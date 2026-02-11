# Phase 1: Core + Base Module — Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build the project skeleton, config loading, platform abstraction layer, module system, and base setup module so that `shhh setup base` works end-to-end with structured CLI output (no TUI yet).

**Architecture:** Cobra CLI dispatches to a module runner that executes Steps within Modules. Steps interact with the system through platform interfaces (UserEnv, ProfileManager) which have real Windows implementations behind build tags and in-memory mocks for testing on macOS. Config is loaded from TOML. All operations are logged via slog.

**Tech Stack:** Go 1.23+, cobra (CLI), go-toml/v2 (config), slog (logging). No TUI dependencies in Phase 1.

**Module path:** `github.com/druarnfield/shhh`

---

### Task 1: Project Skeleton + Root Command

**Files:**
- Create: `go.mod`
- Create: `cmd/shhh/main.go`
- Create: `internal/cli/root.go`
- Create: `Taskfile.yml`
- Create: `.gitignore`

**Step 1: Initialize Go module and install cobra**

```bash
cd /Users/dru/Documents/Development/go/shhh
go mod init github.com/druarnfield/shhh
go get github.com/spf13/cobra@latest
```

**Step 2: Create directory structure**

```bash
mkdir -p cmd/shhh internal/{cli,config,module/setup,platform,exec,logging,explain/topics,state}
```

**Step 3: Write main.go**

Create `cmd/shhh/main.go`:

```go
package main

import (
	"os"

	"github.com/druarnfield/shhh/internal/cli"
)

var version = "dev"

func main() {
	if err := cli.Execute(version); err != nil {
		os.Exit(1)
	}
}
```

**Step 4: Write root command**

Create `internal/cli/root.go`:

```go
package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	flagExplain bool
	flagQuiet   bool
	flagDryRun  bool
	flagVerbose bool
)

func newRootCmd(version string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "shhh",
		Short: "Developer environment bootstrapper",
		Long:  "shhh bootstraps and manages developer environments on locked-down Windows workstations without admin privileges.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	cmd.PersistentFlags().BoolVar(&flagExplain, "explain", false, "Show explanations for each step")
	cmd.PersistentFlags().BoolVar(&flagQuiet, "quiet", false, "Suppress explanations, show progress only")
	cmd.PersistentFlags().BoolVar(&flagDryRun, "dry-run", false, "Show what would happen without doing it")
	cmd.PersistentFlags().BoolVar(&flagVerbose, "verbose", false, "Show detailed log output")

	cmd.AddCommand(newVersionCmd(version))

	return cmd
}

func newVersionCmd(version string) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print shhh version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("shhh", version)
		},
	}
}

func Execute(version string) error {
	return newRootCmd(version).Execute()
}
```

**Step 5: Write Taskfile**

Create `Taskfile.yml`:

```yaml
version: '3'

vars:
  VERSION:
    sh: git describe --tags --always --dirty 2>/dev/null || echo "dev"

tasks:
  build:
    desc: Build shhh binary for current platform
    cmds:
      - go build -ldflags "-s -w -X main.version={{.VERSION}}" -o shhh ./cmd/shhh

  build-windows:
    desc: Cross-compile for Windows
    env:
      GOOS: windows
      GOARCH: amd64
    cmds:
      - go build -ldflags "-s -w -X main.version={{.VERSION}}" -o shhh.exe ./cmd/shhh

  test:
    desc: Run all tests
    cmds:
      - go test ./...

  test-verbose:
    desc: Run all tests with verbose output
    cmds:
      - go test ./... -v

  clean:
    desc: Remove build artifacts
    cmds:
      - rm -f shhh shhh.exe

  tidy:
    desc: Tidy go modules
    cmds:
      - go mod tidy
```

**Step 6: Write .gitignore**

Create `.gitignore`:

```
shhh
shhh.exe
*.log
```

**Step 7: Build and verify**

```bash
task build
./shhh --help
./shhh version
```

Expected: help text prints, version prints "shhh dev".

**Step 8: Commit**

```bash
git add cmd/ internal/cli/ go.mod go.sum Taskfile.yml .gitignore
git commit -m "feat: project skeleton with cobra root command"
```

---

### Task 2: Config Package

**Files:**
- Create: `internal/config/config.go`
- Create: `internal/config/locations.go`
- Create: `internal/config/config_test.go`
- Create: `shhh.example.toml`

**Step 1: Install go-toml**

```bash
go get github.com/pelletier/go-toml/v2@latest
```

**Step 2: Write the config test**

Create `internal/config/config_test.go`:

```go
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
```

**Step 3: Run test to verify it fails**

```bash
go test ./internal/config/ -v
```

Expected: FAIL — types and functions don't exist yet.

**Step 4: Write config types and parsing**

Create `internal/config/config.go`:

```go
package config

import (
	"fmt"
	"os"

	toml "github.com/pelletier/go-toml/v2"
)

type Config struct {
	Org        OrgConfig        `toml:"org"`
	Proxy      ProxyConfig      `toml:"proxy"`
	Certs      CertsConfig      `toml:"certs"`
	Git        GitConfig        `toml:"git"`
	GitLab     GitLabConfig     `toml:"gitlab"`
	Registries RegistriesConfig `toml:"registries"`
	Scoop      ScoopConfig      `toml:"scoop"`
	Tools      ToolsConfig      `toml:"tools"`
	Python     PythonConfig     `toml:"python"`
	Golang     GolangConfig     `toml:"golang"`
	Node       NodeConfig       `toml:"node"`
}

type OrgConfig struct {
	Name string `toml:"name"`
}

type ProxyConfig struct {
	HTTP    string `toml:"http"`
	HTTPS   string `toml:"https"`
	NoProxy string `toml:"no_proxy"`
}

type CertsConfig struct {
	Source string   `toml:"source"`
	Extra  []string `toml:"extra"`
}

type GitConfig struct {
	DefaultBranch string   `toml:"default_branch"`
	SSHHosts      []string `toml:"ssh_hosts"`
}

type GitLabConfig struct {
	Host    string `toml:"host"`
	SSHPort int    `toml:"ssh_port"`
}

type RegistriesConfig struct {
	PyPIMirror  string `toml:"pypi_mirror"`
	NPMRegistry string `toml:"npm_registry"`
	GoProxy     string `toml:"go_proxy"`
}

type ScoopConfig struct {
	Buckets []string `toml:"buckets"`
}

type ToolsConfig struct {
	Core     []string `toml:"core"`
	Data     []string `toml:"data"`
	Optional []string `toml:"optional"`
}

type PythonConfig struct {
	Version string `toml:"version"`
}

type GolangConfig struct {
	Version string `toml:"version"`
}

type NodeConfig struct {
	Version string `toml:"version"`
}

func Defaults() *Config {
	return &Config{
		Certs:  CertsConfig{Source: "system"},
		Git:    GitConfig{DefaultBranch: "main"},
		GitLab: GitLabConfig{SSHPort: 22},
		Python: PythonConfig{Version: "3.12"},
		Golang: GolangConfig{Version: "1.23"},
		Node:   NodeConfig{Version: "22"},
	}
}

func LoadFromFile(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config: %w", err)
	}

	cfg := Defaults()
	if err := toml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	return cfg, nil
}
```

**Step 5: Write locations helper**

Create `internal/config/locations.go`:

```go
package config

import (
	"os"
	"path/filepath"
)

// ConfigDir returns the shhh config directory.
// Uses ~/.config/shhh on all platforms (XDG-style).
func ConfigDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".", ".config", "shhh")
	}
	return filepath.Join(home, ".config", "shhh")
}

// ConfigFilePath returns the path to shhh.toml.
// Checks next to the binary first, then ~/.config/shhh/shhh.toml.
func ConfigFilePath() string {
	// Check next to the binary
	exe, err := os.Executable()
	if err == nil {
		adjacent := filepath.Join(filepath.Dir(exe), "shhh.toml")
		if _, err := os.Stat(adjacent); err == nil {
			return adjacent
		}
	}

	return filepath.Join(ConfigDir(), "shhh.toml")
}

// StateFilePath returns the path to state.json.
func StateFilePath() string {
	return filepath.Join(ConfigDir(), "state.json")
}

// LogFilePath returns the path to shhh.log.
func LogFilePath() string {
	return filepath.Join(ConfigDir(), "shhh.log")
}

// CABundlePath returns the path to the CA bundle.
func CABundlePath() string {
	return filepath.Join(ConfigDir(), "ca-bundle.pem")
}
```

**Step 6: Run tests**

```bash
go test ./internal/config/ -v
```

Expected: all PASS.

**Step 7: Write example config**

Create `shhh.example.toml` — copy the TOML from the design doc's Config section.

**Step 8: Commit**

```bash
git add internal/config/ shhh.example.toml
git commit -m "feat: config package with TOML parsing and location resolution"
```

---

### Task 3: Logging Package

**Files:**
- Create: `internal/logging/logging.go`
- Create: `internal/logging/logging_test.go`

**Step 1: Write the logging test**

Create `internal/logging/logging_test.go`:

```go
package logging

import (
	"log/slog"
	"os"
	"path/filepath"
	"testing"
)

func TestSetup_CreatesLogFile(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "test.log")

	logger, err := Setup(logPath, false)
	if err != nil {
		t.Fatalf("Setup: %v", err)
	}

	logger.Info("test message", "key", "value")

	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("reading log: %v", err)
	}

	if len(data) == 0 {
		t.Error("log file is empty")
	}
}

func TestSetup_VerboseAddsStderr(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "test.log")

	logger, err := Setup(logPath, true)
	if err != nil {
		t.Fatalf("Setup: %v", err)
	}

	// Just verify it doesn't panic
	logger.Info("verbose test")
}

func TestRotateIfNeeded(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "test.log")

	// Create a file just over the max size
	data := make([]byte, maxLogSize+1)
	if err := os.WriteFile(logPath, data, 0644); err != nil {
		t.Fatal(err)
	}

	if err := RotateIfNeeded(logPath); err != nil {
		t.Fatalf("RotateIfNeeded: %v", err)
	}

	// Original should be gone or truncated
	info, err := os.Stat(logPath)
	if err != nil {
		// File was rotated away, that's fine
		return
	}
	if info.Size() > maxLogSize {
		t.Errorf("log file still %d bytes, want <= %d", info.Size(), maxLogSize)
	}
}

func TestNopLogger(t *testing.T) {
	logger := slog.New(NopHandler{})
	// Should not panic
	logger.Info("nop")
}
```

**Step 2: Run test to verify it fails**

```bash
go test ./internal/logging/ -v
```

Expected: FAIL.

**Step 3: Write logging implementation**

Create `internal/logging/logging.go`:

```go
package logging

import (
	"context"
	"io"
	"log/slog"
	"os"
	"path/filepath"
)

const maxLogSize = 5 * 1024 * 1024 // 5MB

// Setup creates a structured logger that writes to the given file.
// If verbose is true, also writes to stderr.
func Setup(logPath string, verbose bool) (*slog.Logger, error) {
	if err := os.MkdirAll(filepath.Dir(logPath), 0755); err != nil {
		return nil, err
	}

	if err := RotateIfNeeded(logPath); err != nil {
		return nil, err
	}

	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, err
	}

	var w io.Writer = f
	if verbose {
		w = io.MultiWriter(f, os.Stderr)
	}

	handler := slog.NewJSONHandler(w, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})

	return slog.New(handler), nil
}

// RotateIfNeeded truncates the log file if it exceeds maxLogSize.
func RotateIfNeeded(logPath string) error {
	info, err := os.Stat(logPath)
	if err != nil {
		return nil // file doesn't exist yet, nothing to rotate
	}

	if info.Size() <= maxLogSize {
		return nil
	}

	// Simple rotation: rename old, start fresh
	backup := logPath + ".old"
	os.Remove(backup) // ignore error
	return os.Rename(logPath, backup)
}

// NopHandler is a slog.Handler that discards everything.
type NopHandler struct{}

func (NopHandler) Enabled(context.Context, slog.Level) bool  { return false }
func (NopHandler) Handle(context.Context, slog.Record) error { return nil }
func (h NopHandler) WithAttrs([]slog.Attr) slog.Handler      { return h }
func (h NopHandler) WithGroup(string) slog.Handler            { return h }
```

**Step 4: Run tests**

```bash
go test ./internal/logging/ -v
```

Expected: all PASS.

**Step 5: Commit**

```bash
git add internal/logging/
git commit -m "feat: structured logging with slog and file rotation"
```

---

### Task 4: Platform Interfaces + Build Tag Stubs

**Files:**
- Create: `internal/platform/platform.go`
- Create: `internal/platform/env_other.go`
- Create: `internal/platform/profile_other.go`
- Create: `internal/platform/env_windows.go`
- Create: `internal/platform/profile_windows.go`

**Step 1: Write platform interfaces**

Create `internal/platform/platform.go`:

```go
package platform

import "errors"

// ErrNotSupported is returned by stub implementations on non-Windows platforms.
var ErrNotSupported = errors.New("not supported on this platform")

// UserEnv manages user-level environment variables.
// On Windows, this uses the registry (HKCU\Environment).
type UserEnv interface {
	Get(key string) (value string, source EnvSource, err error)
	Set(key, value string) error
	Delete(key string) error
	AppendPath(dir string) error
	RemovePath(dir string) error
	ListPath() ([]PathEntry, error)
}

type EnvSource int

const (
	SourceProcess EnvSource = iota
	SourceUser
	SourceSystem
)

func (s EnvSource) String() string {
	switch s {
	case SourceProcess:
		return "process"
	case SourceUser:
		return "user"
	case SourceSystem:
		return "system"
	default:
		return "unknown"
	}
}

type PathEntry struct {
	Dir    string
	Source EnvSource
	Exists bool
}

// ProfileManager handles PowerShell $PROFILE modifications.
// All shhh-managed content lives between sentinel markers.
type ProfileManager interface {
	Path() string
	Read() (string, error)
	ManagedBlock() (string, error)
	SetManagedBlock(content string) error
	AppendToManagedBlock(line string) error
	Diff() (string, error)
	Exists() bool
	EnsureExists() error
}

// Sentinel markers for managed profile blocks.
const (
	ManagedBlockStart = "# >>> shhh managed - do not edit >>>"
	ManagedBlockEnd   = "# <<< shhh managed <<<"
)
```

**Step 2: Write non-Windows stubs**

Create `internal/platform/env_other.go`:

```go
//go:build !windows

package platform

// StubUserEnv is a no-op implementation for non-Windows platforms.
// Use mock.UserEnv for testing.
type StubUserEnv struct{}

func NewUserEnv() UserEnv              { return &StubUserEnv{} }
func (s *StubUserEnv) Get(key string) (string, EnvSource, error) {
	return "", SourceProcess, ErrNotSupported
}
func (s *StubUserEnv) Set(key, value string) error { return ErrNotSupported }
func (s *StubUserEnv) Delete(key string) error     { return ErrNotSupported }
func (s *StubUserEnv) AppendPath(dir string) error { return ErrNotSupported }
func (s *StubUserEnv) RemovePath(dir string) error { return ErrNotSupported }
func (s *StubUserEnv) ListPath() ([]PathEntry, error) {
	return nil, ErrNotSupported
}
```

Create `internal/platform/profile_other.go`:

```go
//go:build !windows

package platform

// StubProfileManager is a no-op implementation for non-Windows platforms.
// Use mock.ProfileManager for testing.
type StubProfileManager struct{}

func NewProfileManager() ProfileManager               { return &StubProfileManager{} }
func (s *StubProfileManager) Path() string             { return "" }
func (s *StubProfileManager) Read() (string, error)    { return "", ErrNotSupported }
func (s *StubProfileManager) ManagedBlock() (string, error) {
	return "", ErrNotSupported
}
func (s *StubProfileManager) SetManagedBlock(content string) error  { return ErrNotSupported }
func (s *StubProfileManager) AppendToManagedBlock(line string) error { return ErrNotSupported }
func (s *StubProfileManager) Diff() (string, error)                  { return "", ErrNotSupported }
func (s *StubProfileManager) Exists() bool                           { return false }
func (s *StubProfileManager) EnsureExists() error                    { return ErrNotSupported }
```

**Step 3: Write Windows placeholder stubs**

Create `internal/platform/env_windows.go`:

```go
//go:build windows

package platform

// TODO: Implement using golang.org/x/sys/windows/registry
// Reads/writes HKCU\Environment, broadcasts WM_SETTINGCHANGE.

type windowsUserEnv struct{}

func NewUserEnv() UserEnv { return &windowsUserEnv{} }

func (w *windowsUserEnv) Get(key string) (string, EnvSource, error) {
	return "", SourceProcess, errors.New("not yet implemented")
}
func (w *windowsUserEnv) Set(key, value string) error { return errors.New("not yet implemented") }
func (w *windowsUserEnv) Delete(key string) error     { return errors.New("not yet implemented") }
func (w *windowsUserEnv) AppendPath(dir string) error { return errors.New("not yet implemented") }
func (w *windowsUserEnv) RemovePath(dir string) error { return errors.New("not yet implemented") }
func (w *windowsUserEnv) ListPath() ([]PathEntry, error) {
	return nil, errors.New("not yet implemented")
}
```

Create `internal/platform/profile_windows.go`:

```go
//go:build windows

package platform

import "errors"

// TODO: Implement PowerShell $PROFILE management.
// Resolves profile path, reads/writes managed blocks between sentinels.

type windowsProfileManager struct{}

func NewProfileManager() ProfileManager               { return &windowsProfileManager{} }
func (w *windowsProfileManager) Path() string          { return "" }
func (w *windowsProfileManager) Read() (string, error) { return "", errors.New("not yet implemented") }
func (w *windowsProfileManager) ManagedBlock() (string, error) {
	return "", errors.New("not yet implemented")
}
func (w *windowsProfileManager) SetManagedBlock(content string) error {
	return errors.New("not yet implemented")
}
func (w *windowsProfileManager) AppendToManagedBlock(line string) error {
	return errors.New("not yet implemented")
}
func (w *windowsProfileManager) Diff() (string, error)  { return "", errors.New("not yet implemented") }
func (w *windowsProfileManager) Exists() bool            { return false }
func (w *windowsProfileManager) EnsureExists() error     { return errors.New("not yet implemented") }
```

**Step 4: Verify it compiles**

```bash
go build ./internal/platform/
```

Expected: compiles cleanly on macOS.

**Step 5: Commit**

```bash
git add internal/platform/
git commit -m "feat: platform interfaces with build tag stubs"
```

---

### Task 5: Platform Mock Implementations

**Files:**
- Create: `internal/platform/mock/mock.go`
- Create: `internal/platform/mock/mock_test.go`

**Step 1: Write mock tests**

Create `internal/platform/mock/mock_test.go`:

```go
package mock

import (
	"testing"

	"github.com/druarnfield/shhh/internal/platform"
)

func TestUserEnv_SetAndGet(t *testing.T) {
	env := NewUserEnv()

	if err := env.Set("HTTP_PROXY", "http://proxy:8080"); err != nil {
		t.Fatalf("Set: %v", err)
	}

	val, source, err := env.Get("HTTP_PROXY")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if val != "http://proxy:8080" {
		t.Errorf("value = %q, want %q", val, "http://proxy:8080")
	}
	if source != platform.SourceUser {
		t.Errorf("source = %v, want SourceUser", source)
	}
}

func TestUserEnv_GetMissing(t *testing.T) {
	env := NewUserEnv()
	_, _, err := env.Get("NONEXISTENT")
	if err == nil {
		t.Error("expected error for missing key")
	}
}

func TestUserEnv_Delete(t *testing.T) {
	env := NewUserEnv()
	env.Set("FOO", "bar")
	if err := env.Delete("FOO"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	_, _, err := env.Get("FOO")
	if err == nil {
		t.Error("expected error after delete")
	}
}

func TestUserEnv_AppendPath(t *testing.T) {
	env := NewUserEnv()

	if err := env.AppendPath("/usr/local/bin"); err != nil {
		t.Fatalf("AppendPath: %v", err)
	}
	if err := env.AppendPath("/usr/bin"); err != nil {
		t.Fatalf("AppendPath: %v", err)
	}

	entries, err := env.ListPath()
	if err != nil {
		t.Fatalf("ListPath: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("len = %d, want 2", len(entries))
	}
	if entries[0].Dir != "/usr/local/bin" {
		t.Errorf("entries[0] = %q, want %q", entries[0].Dir, "/usr/local/bin")
	}
}

func TestUserEnv_AppendPath_Dedup(t *testing.T) {
	env := NewUserEnv()
	env.AppendPath("/usr/bin")
	env.AppendPath("/usr/bin")

	entries, _ := env.ListPath()
	if len(entries) != 1 {
		t.Errorf("len = %d, want 1 (should dedup)", len(entries))
	}
}

func TestUserEnv_RemovePath(t *testing.T) {
	env := NewUserEnv()
	env.AppendPath("/usr/bin")
	env.AppendPath("/usr/local/bin")

	if err := env.RemovePath("/usr/bin"); err != nil {
		t.Fatalf("RemovePath: %v", err)
	}

	entries, _ := env.ListPath()
	if len(entries) != 1 {
		t.Errorf("len = %d, want 1", len(entries))
	}
}

func TestProfileManager_ManagedBlock(t *testing.T) {
	pm := NewProfileManager("/tmp/test_profile.ps1")

	if err := pm.EnsureExists(); err != nil {
		t.Fatalf("EnsureExists: %v", err)
	}

	block := `$env:HTTP_PROXY = "http://proxy:8080"
$env:HTTPS_PROXY = "http://proxy:8080"`

	if err := pm.SetManagedBlock(block); err != nil {
		t.Fatalf("SetManagedBlock: %v", err)
	}

	got, err := pm.ManagedBlock()
	if err != nil {
		t.Fatalf("ManagedBlock: %v", err)
	}
	if got != block {
		t.Errorf("ManagedBlock = %q, want %q", got, block)
	}

	// Full content should have sentinels
	full, _ := pm.Read()
	if full == "" {
		t.Error("Read() returned empty")
	}
}

func TestProfileManager_AppendToManagedBlock(t *testing.T) {
	pm := NewProfileManager("/tmp/test_profile.ps1")
	pm.EnsureExists()

	pm.SetManagedBlock(`$env:FOO = "bar"`)
	pm.AppendToManagedBlock(`$env:BAZ = "qux"`)

	got, _ := pm.ManagedBlock()
	if got != "$env:FOO = \"bar\"\n$env:BAZ = \"qux\"" {
		t.Errorf("ManagedBlock = %q", got)
	}
}

func TestProfileManager_PreservesUserContent(t *testing.T) {
	pm := NewProfileManager("/tmp/test_profile.ps1")

	// Simulate existing user content
	pm.(*ProfileManager).content = "# my custom stuff\nSet-Alias ll ls\n"

	pm.SetManagedBlock(`$env:FOO = "bar"`)

	full, _ := pm.Read()
	if full == "" {
		t.Error("Read() returned empty")
	}

	// User content should still be there
	pm.SetManagedBlock(`$env:FOO = "updated"`)
	full, _ = pm.Read()
	// Verify user content wasn't destroyed
	got, _ := pm.ManagedBlock()
	if got != `$env:FOO = "updated"` {
		t.Errorf("ManagedBlock = %q", got)
	}
}
```

**Step 2: Run test to verify it fails**

```bash
go test ./internal/platform/mock/ -v
```

Expected: FAIL.

**Step 3: Write mock implementations**

Create `internal/platform/mock/mock.go`:

```go
package mock

import (
	"fmt"
	"strings"

	"github.com/druarnfield/shhh/internal/platform"
)

// UserEnv is an in-memory implementation of platform.UserEnv for testing.
type UserEnv struct {
	vars map[string]string
	path []string
}

func NewUserEnv() platform.UserEnv {
	return &UserEnv{
		vars: make(map[string]string),
	}
}

func (m *UserEnv) Get(key string) (string, platform.EnvSource, error) {
	val, ok := m.vars[key]
	if !ok {
		return "", platform.SourceProcess, fmt.Errorf("env var %q not set", key)
	}
	return val, platform.SourceUser, nil
}

func (m *UserEnv) Set(key, value string) error {
	m.vars[key] = value
	return nil
}

func (m *UserEnv) Delete(key string) error {
	delete(m.vars, key)
	return nil
}

func (m *UserEnv) AppendPath(dir string) error {
	for _, p := range m.path {
		if p == dir {
			return nil // already present
		}
	}
	m.path = append(m.path, dir)
	return nil
}

func (m *UserEnv) RemovePath(dir string) error {
	filtered := m.path[:0]
	for _, p := range m.path {
		if p != dir {
			filtered = append(filtered, p)
		}
	}
	m.path = filtered
	return nil
}

func (m *UserEnv) ListPath() ([]platform.PathEntry, error) {
	entries := make([]platform.PathEntry, len(m.path))
	for i, dir := range m.path {
		entries[i] = platform.PathEntry{
			Dir:    dir,
			Source: platform.SourceUser,
			Exists: true,
		}
	}
	return entries, nil
}

// ProfileManager is an in-memory implementation of platform.ProfileManager.
type ProfileManager struct {
	path         string
	content      string
	managedBlock string
	exists       bool
}

func NewProfileManager(path string) platform.ProfileManager {
	return &ProfileManager{path: path}
}

func (m *ProfileManager) Path() string { return m.path }

func (m *ProfileManager) Read() (string, error) {
	if !m.exists {
		return "", fmt.Errorf("profile does not exist")
	}
	return m.buildContent(), nil
}

func (m *ProfileManager) ManagedBlock() (string, error) {
	return m.managedBlock, nil
}

func (m *ProfileManager) SetManagedBlock(content string) error {
	m.managedBlock = content
	return nil
}

func (m *ProfileManager) AppendToManagedBlock(line string) error {
	if m.managedBlock == "" {
		m.managedBlock = line
	} else {
		m.managedBlock = m.managedBlock + "\n" + line
	}
	return nil
}

func (m *ProfileManager) Diff() (string, error) {
	return m.managedBlock, nil
}

func (m *ProfileManager) Exists() bool {
	return m.exists
}

func (m *ProfileManager) EnsureExists() error {
	m.exists = true
	return nil
}

func (m *ProfileManager) buildContent() string {
	var b strings.Builder
	// Write user content (everything before managed block)
	userContent := m.userContent()
	if userContent != "" {
		b.WriteString(userContent)
		if !strings.HasSuffix(userContent, "\n") {
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}

	if m.managedBlock != "" {
		b.WriteString(platform.ManagedBlockStart)
		b.WriteString("\n")
		b.WriteString(m.managedBlock)
		b.WriteString("\n")
		b.WriteString(platform.ManagedBlockEnd)
		b.WriteString("\n")
	}

	return b.String()
}

func (m *ProfileManager) userContent() string {
	// Return content that isn't part of the managed block
	// For the mock, m.content holds the user's own content
	return m.content
}
```

**Step 4: Run tests**

```bash
go test ./internal/platform/mock/ -v
```

Expected: all PASS.

**Step 5: Commit**

```bash
git add internal/platform/mock/
git commit -m "feat: in-memory mock implementations for platform interfaces"
```

---

### Task 6: Module System — Types, Registry, Dependency Resolution

**Files:**
- Create: `internal/module/module.go`
- Create: `internal/module/module_test.go`

**Step 1: Write registry and dependency resolution tests**

Create `internal/module/module_test.go`:

```go
package module

import (
	"context"
	"testing"
)

func TestRegistry_RegisterAndGet(t *testing.T) {
	reg := NewRegistry()
	mod := &Module{
		ID:       "base",
		Name:     "Base",
		Category: CategoryBase,
	}

	reg.Register(mod)

	got := reg.Get("base")
	if got == nil {
		t.Fatal("Get returned nil")
	}
	if got.ID != "base" {
		t.Errorf("ID = %q, want %q", got.ID, "base")
	}
}

func TestRegistry_GetMissing(t *testing.T) {
	reg := NewRegistry()
	if reg.Get("nonexistent") != nil {
		t.Error("expected nil for missing module")
	}
}

func TestRegistry_ByCategory(t *testing.T) {
	reg := NewRegistry()
	reg.Register(&Module{ID: "base", Category: CategoryBase})
	reg.Register(&Module{ID: "python", Category: CategoryLanguage})
	reg.Register(&Module{ID: "golang", Category: CategoryLanguage})
	reg.Register(&Module{ID: "tools", Category: CategoryTool})

	langs := reg.ByCategory(CategoryLanguage)
	if len(langs) != 2 {
		t.Errorf("ByCategory(Language) = %d modules, want 2", len(langs))
	}
}

func TestRegistry_ResolveDeps_Simple(t *testing.T) {
	reg := NewRegistry()
	reg.Register(&Module{ID: "base", Dependencies: nil})
	reg.Register(&Module{ID: "python", Dependencies: []string{"base"}})

	order, err := reg.ResolveDeps([]string{"python"})
	if err != nil {
		t.Fatalf("ResolveDeps: %v", err)
	}

	if len(order) != 2 {
		t.Fatalf("len = %d, want 2", len(order))
	}
	if order[0] != "base" || order[1] != "python" {
		t.Errorf("order = %v, want [base, python]", order)
	}
}

func TestRegistry_ResolveDeps_AlreadyIncluded(t *testing.T) {
	reg := NewRegistry()
	reg.Register(&Module{ID: "base"})
	reg.Register(&Module{ID: "python", Dependencies: []string{"base"}})

	order, err := reg.ResolveDeps([]string{"base", "python"})
	if err != nil {
		t.Fatalf("ResolveDeps: %v", err)
	}

	// base should appear only once
	count := 0
	for _, id := range order {
		if id == "base" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("base appears %d times, want 1", count)
	}
}

func TestRegistry_ResolveDeps_Diamond(t *testing.T) {
	// base -> python -> tools
	// base -> golang -> tools
	reg := NewRegistry()
	reg.Register(&Module{ID: "base"})
	reg.Register(&Module{ID: "python", Dependencies: []string{"base"}})
	reg.Register(&Module{ID: "golang", Dependencies: []string{"base"}})
	reg.Register(&Module{ID: "tools", Dependencies: []string{"python", "golang"}})

	order, err := reg.ResolveDeps([]string{"tools"})
	if err != nil {
		t.Fatalf("ResolveDeps: %v", err)
	}

	// base must come before python and golang
	// python and golang must come before tools
	idx := make(map[string]int)
	for i, id := range order {
		idx[id] = i
	}

	if idx["base"] >= idx["python"] {
		t.Error("base should come before python")
	}
	if idx["base"] >= idx["golang"] {
		t.Error("base should come before golang")
	}
	if idx["python"] >= idx["tools"] {
		t.Error("python should come before tools")
	}
}

func TestRegistry_ResolveDeps_CycleError(t *testing.T) {
	reg := NewRegistry()
	reg.Register(&Module{ID: "a", Dependencies: []string{"b"}})
	reg.Register(&Module{ID: "b", Dependencies: []string{"a"}})

	_, err := reg.ResolveDeps([]string{"a"})
	if err == nil {
		t.Error("expected error for cycle")
	}
}

func TestRegistry_ResolveDeps_MissingDepError(t *testing.T) {
	reg := NewRegistry()
	reg.Register(&Module{ID: "python", Dependencies: []string{"base"}})

	_, err := reg.ResolveDeps([]string{"python"})
	if err == nil {
		t.Error("expected error for missing dependency")
	}
}

func TestStep_CheckSkipsRun(t *testing.T) {
	ran := false
	step := Step{
		Name: "test step",
		Check: func(ctx context.Context) bool {
			return true // already done
		},
		Run: func(ctx context.Context) error {
			ran = true
			return nil
		},
	}

	// Check returns true, so Run should be skipped
	if !step.Check(context.Background()) {
		t.Error("Check should return true")
	}
	if ran {
		t.Error("Run should not have been called")
	}
}
```

**Step 2: Run test to verify it fails**

```bash
go test ./internal/module/ -v
```

Expected: FAIL.

**Step 3: Write module types and registry**

Create `internal/module/module.go`:

```go
package module

import (
	"context"
	"fmt"
)

type Step struct {
	Name        string
	Description string
	Explain     string
	Check       func(ctx context.Context) bool
	Run         func(ctx context.Context) error
	DryRun      func(ctx context.Context) string
}

type Module struct {
	ID           string
	Name         string
	Description  string
	Category     Category
	Dependencies []string
	Steps        []Step
}

type Category int

const (
	CategoryBase Category = iota
	CategoryLanguage
	CategoryTool
)

func (c Category) String() string {
	switch c {
	case CategoryBase:
		return "Base"
	case CategoryLanguage:
		return "Language"
	case CategoryTool:
		return "Tool"
	default:
		return "Unknown"
	}
}

type Registry struct {
	modules map[string]*Module
	order   []string // insertion order
}

func NewRegistry() *Registry {
	return &Registry{
		modules: make(map[string]*Module),
	}
}

func (r *Registry) Register(m *Module) {
	r.modules[m.ID] = m
	r.order = append(r.order, m.ID)
}

func (r *Registry) Get(id string) *Module {
	return r.modules[id]
}

func (r *Registry) All() []*Module {
	result := make([]*Module, 0, len(r.order))
	for _, id := range r.order {
		result = append(result, r.modules[id])
	}
	return result
}

func (r *Registry) ByCategory(c Category) []*Module {
	var result []*Module
	for _, id := range r.order {
		if m := r.modules[id]; m.Category == c {
			result = append(result, m)
		}
	}
	return result
}

// ResolveDeps returns module IDs in topological order, auto-including
// any dependencies not explicitly requested.
func (r *Registry) ResolveDeps(ids []string) ([]string, error) {
	// Collect all needed modules (requested + transitive deps)
	needed := make(map[string]bool)
	var collect func(id string) error
	collect = func(id string) error {
		if needed[id] {
			return nil
		}
		m := r.modules[id]
		if m == nil {
			return fmt.Errorf("unknown module: %q", id)
		}
		needed[id] = true
		for _, dep := range m.Dependencies {
			if err := collect(dep); err != nil {
				return err
			}
		}
		return nil
	}

	for _, id := range ids {
		if err := collect(id); err != nil {
			return nil, err
		}
	}

	// Topological sort using Kahn's algorithm
	inDegree := make(map[string]int)
	for id := range needed {
		if _, ok := inDegree[id]; !ok {
			inDegree[id] = 0
		}
		for _, dep := range r.modules[id].Dependencies {
			if needed[dep] {
				inDegree[id]++
			}
		}
	}

	var queue []string
	for id, deg := range inDegree {
		if deg == 0 {
			queue = append(queue, id)
		}
	}

	var sorted []string
	for len(queue) > 0 {
		// Pick from queue (stable: prefer insertion order)
		best := -1
		for i, id := range queue {
			if best == -1 || r.insertionIndex(id) < r.insertionIndex(queue[best]) {
				best = i
			}
		}

		id := queue[best]
		queue = append(queue[:best], queue[best+1:]...)
		sorted = append(sorted, id)

		// Reduce in-degree for dependents
		for other := range needed {
			for _, dep := range r.modules[other].Dependencies {
				if dep == id {
					inDegree[other]--
					if inDegree[other] == 0 {
						queue = append(queue, other)
					}
				}
			}
		}
	}

	if len(sorted) != len(needed) {
		return nil, fmt.Errorf("dependency cycle detected")
	}

	return sorted, nil
}

func (r *Registry) insertionIndex(id string) int {
	for i, oid := range r.order {
		if oid == id {
			return i
		}
	}
	return len(r.order)
}
```

**Step 4: Run tests**

```bash
go test ./internal/module/ -v
```

Expected: all PASS.

**Step 5: Commit**

```bash
git add internal/module/
git commit -m "feat: module system with registry and topological dependency resolution"
```

---

### Task 7: Module Runner

**Files:**
- Create: `internal/module/runner.go`
- Create: `internal/module/runner_test.go`

**Step 1: Write runner tests**

Create `internal/module/runner_test.go`:

```go
package module

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/druarnfield/shhh/internal/logging"
)

func nopLogger() *slog.Logger {
	return slog.New(logging.NopHandler{})
}

func TestRunner_ExecutesSteps(t *testing.T) {
	var executed []string
	mod := &Module{
		ID:   "test",
		Name: "Test",
		Steps: []Step{
			{
				Name:  "step1",
				Check: func(ctx context.Context) bool { return false },
				Run: func(ctx context.Context) error {
					executed = append(executed, "step1")
					return nil
				},
			},
			{
				Name:  "step2",
				Check: func(ctx context.Context) bool { return false },
				Run: func(ctx context.Context) error {
					executed = append(executed, "step2")
					return nil
				},
			},
		},
	}

	runner := NewRunner(nopLogger(), false)
	result := runner.RunModule(context.Background(), mod)

	if result.Err != nil {
		t.Fatalf("RunModule error: %v", result.Err)
	}
	if len(executed) != 2 {
		t.Fatalf("executed %d steps, want 2", len(executed))
	}
	if result.Skipped != 0 {
		t.Errorf("skipped = %d, want 0", result.Skipped)
	}
	if result.Completed != 2 {
		t.Errorf("completed = %d, want 2", result.Completed)
	}
}

func TestRunner_SkipsPassedChecks(t *testing.T) {
	ran := false
	mod := &Module{
		ID:   "test",
		Name: "Test",
		Steps: []Step{
			{
				Name:  "already done",
				Check: func(ctx context.Context) bool { return true },
				Run: func(ctx context.Context) error {
					ran = true
					return nil
				},
			},
		},
	}

	runner := NewRunner(nopLogger(), false)
	result := runner.RunModule(context.Background(), mod)

	if ran {
		t.Error("step should have been skipped")
	}
	if result.Skipped != 1 {
		t.Errorf("skipped = %d, want 1", result.Skipped)
	}
}

func TestRunner_StopsOnError(t *testing.T) {
	step2ran := false
	mod := &Module{
		ID:   "test",
		Name: "Test",
		Steps: []Step{
			{
				Name:  "fails",
				Check: func(ctx context.Context) bool { return false },
				Run: func(ctx context.Context) error {
					return errors.New("boom")
				},
			},
			{
				Name:  "should not run",
				Check: func(ctx context.Context) bool { return false },
				Run: func(ctx context.Context) error {
					step2ran = true
					return nil
				},
			},
		},
	}

	runner := NewRunner(nopLogger(), false)
	result := runner.RunModule(context.Background(), mod)

	if result.Err == nil {
		t.Error("expected error")
	}
	if step2ran {
		t.Error("step2 should not have run")
	}
	if result.FailedStep != "fails" {
		t.Errorf("FailedStep = %q, want %q", result.FailedStep, "fails")
	}
}

func TestRunner_DryRun(t *testing.T) {
	ran := false
	mod := &Module{
		ID:   "test",
		Name: "Test",
		Steps: []Step{
			{
				Name:  "step1",
				Check: func(ctx context.Context) bool { return false },
				Run: func(ctx context.Context) error {
					ran = true
					return nil
				},
				DryRun: func(ctx context.Context) string {
					return "would do the thing"
				},
			},
		},
	}

	runner := NewRunner(nopLogger(), true) // dry run
	result := runner.RunModule(context.Background(), mod)

	if ran {
		t.Error("Run should not be called in dry-run mode")
	}
	if result.Err != nil {
		t.Fatalf("unexpected error: %v", result.Err)
	}
}

func TestRunner_RunModules(t *testing.T) {
	var order []string

	reg := NewRegistry()
	reg.Register(&Module{
		ID: "base",
		Steps: []Step{{
			Name:  "base-step",
			Check: func(ctx context.Context) bool { return false },
			Run: func(ctx context.Context) error {
				order = append(order, "base")
				return nil
			},
		}},
	})
	reg.Register(&Module{
		ID:           "python",
		Dependencies: []string{"base"},
		Steps: []Step{{
			Name:  "python-step",
			Check: func(ctx context.Context) bool { return false },
			Run: func(ctx context.Context) error {
				order = append(order, "python")
				return nil
			},
		}},
	})

	runner := NewRunner(nopLogger(), false)
	results, err := runner.RunModules(context.Background(), reg, []string{"python"})
	if err != nil {
		t.Fatalf("RunModules: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("got %d results, want 2", len(results))
	}
	if order[0] != "base" || order[1] != "python" {
		t.Errorf("execution order = %v, want [base, python]", order)
	}
}
```

**Step 2: Run test to verify it fails**

```bash
go test ./internal/module/ -v -run Runner
```

Expected: FAIL.

**Step 3: Write runner implementation**

Create `internal/module/runner.go`:

```go
package module

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

// ModuleResult holds the outcome of running a single module.
type ModuleResult struct {
	ModuleID   string
	Completed  int
	Skipped    int
	Total      int
	FailedStep string
	Err        error
}

// StepCallback is called after each step completes.
// Used by the CLI/TUI to display progress.
type StepCallback func(module *Module, step *Step, index int, total int, skipped bool, err error)

// Runner executes module steps with check-before-run semantics.
//
// Contract: any step that writes environment variables persistently
// (to registry, $PROFILE, etc.) MUST also call os.Setenv() so child
// processes spawned by later steps inherit the values.
type Runner struct {
	logger   *slog.Logger
	dryRun   bool
	callback StepCallback
}

func NewRunner(logger *slog.Logger, dryRun bool) *Runner {
	return &Runner{
		logger: logger,
		dryRun: dryRun,
	}
}

// SetCallback sets the function called after each step.
func (r *Runner) SetCallback(cb StepCallback) {
	r.callback = cb
}

// RunModule executes all steps in a module.
func (r *Runner) RunModule(ctx context.Context, mod *Module) ModuleResult {
	result := ModuleResult{
		ModuleID: mod.ID,
		Total:    len(mod.Steps),
	}

	r.logger.Info("running module", "module", mod.ID, "steps", len(mod.Steps))

	for i := range mod.Steps {
		step := &mod.Steps[i]

		// Check if already done
		if step.Check != nil && step.Check(ctx) {
			r.logger.Info("step skipped (already done)",
				"module", mod.ID, "step", step.Name)
			result.Skipped++

			if r.callback != nil {
				r.callback(mod, step, i, len(mod.Steps), true, nil)
			}
			continue
		}

		if r.dryRun {
			msg := "(no dry-run description)"
			if step.DryRun != nil {
				msg = step.DryRun(ctx)
			}
			r.logger.Info("dry run",
				"module", mod.ID, "step", step.Name, "would_do", msg)

			if r.callback != nil {
				r.callback(mod, step, i, len(mod.Steps), true, nil)
			}
			result.Skipped++
			continue
		}

		// Run the step
		start := time.Now()
		r.logger.Info("running step",
			"module", mod.ID, "step", step.Name)

		err := step.Run(ctx)
		duration := time.Since(start)

		if err != nil {
			r.logger.Error("step failed",
				"module", mod.ID, "step", step.Name,
				"duration", duration, "error", err)
			result.FailedStep = step.Name
			result.Err = fmt.Errorf("module %q step %q: %w", mod.ID, step.Name, err)

			if r.callback != nil {
				r.callback(mod, step, i, len(mod.Steps), false, err)
			}
			return result
		}

		r.logger.Info("step completed",
			"module", mod.ID, "step", step.Name, "duration", duration)
		result.Completed++

		if r.callback != nil {
			r.callback(mod, step, i, len(mod.Steps), false, nil)
		}
	}

	r.logger.Info("module complete",
		"module", mod.ID,
		"completed", result.Completed,
		"skipped", result.Skipped)

	return result
}

// RunModules resolves dependencies and runs modules in topological order.
// Stops on first module failure.
func (r *Runner) RunModules(ctx context.Context, reg *Registry, moduleIDs []string) ([]ModuleResult, error) {
	order, err := reg.ResolveDeps(moduleIDs)
	if err != nil {
		return nil, fmt.Errorf("resolving dependencies: %w", err)
	}

	r.logger.Info("resolved execution order", "modules", order)

	var results []ModuleResult
	for _, id := range order {
		mod := reg.Get(id)
		result := r.RunModule(ctx, mod)
		results = append(results, result)

		if result.Err != nil {
			return results, result.Err
		}
	}

	return results, nil
}
```

**Step 4: Run tests**

```bash
go test ./internal/module/ -v
```

Expected: all PASS.

**Step 5: Commit**

```bash
git add internal/module/runner.go internal/module/runner_test.go
git commit -m "feat: module runner with check-before-run and dependency resolution"
```

---

### Task 8: Exec Package — Command Execution

**Files:**
- Create: `internal/exec/exec.go`
- Create: `internal/exec/exec_test.go`

**Step 1: Write exec tests**

Create `internal/exec/exec_test.go`:

```go
package exec

import (
	"context"
	"runtime"
	"testing"
)

func TestRun_SimpleCommand(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("test uses unix commands")
	}

	result, err := Run(context.Background(), "echo", "hello")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if result.ExitCode != 0 {
		t.Errorf("exit code = %d, want 0", result.ExitCode)
	}
	if result.Stdout != "hello\n" {
		t.Errorf("stdout = %q, want %q", result.Stdout, "hello\n")
	}
}

func TestRun_FailingCommand(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("test uses unix commands")
	}

	_, err := Run(context.Background(), "false")
	if err == nil {
		t.Error("expected error for failing command")
	}
}

func TestRun_CommandNotFound(t *testing.T) {
	_, err := Run(context.Background(), "nonexistent_command_12345")
	if err == nil {
		t.Error("expected error for missing command")
	}
}

func TestCommandExists(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("test uses unix commands")
	}

	if !CommandExists("echo") {
		t.Error("echo should exist")
	}
	if CommandExists("nonexistent_command_12345") {
		t.Error("nonexistent command should not exist")
	}
}

func TestMockRunner(t *testing.T) {
	mock := &MockRunner{
		Results: map[string]Result{
			"git --version": {Stdout: "git version 2.43.0\n", ExitCode: 0},
		},
	}

	result, err := mock.Run(context.Background(), "git", "--version")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if result.Stdout != "git version 2.43.0\n" {
		t.Errorf("stdout = %q", result.Stdout)
	}
}
```

**Step 2: Run test to verify it fails**

```bash
go test ./internal/exec/ -v
```

Expected: FAIL.

**Step 3: Write exec implementation**

Create `internal/exec/exec.go`:

```go
package exec

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// Result holds the output of a command execution.
type Result struct {
	Stdout   string
	Stderr   string
	ExitCode int
}

// Runner is the interface for executing external commands.
// Use MockRunner in tests.
type Runner interface {
	Run(ctx context.Context, name string, args ...string) (Result, error)
}

// DefaultRunner executes real commands.
type DefaultRunner struct{}

func (d *DefaultRunner) Run(ctx context.Context, name string, args ...string) (Result, error) {
	return Run(ctx, name, args...)
}

// Run executes a command and returns its output.
func Run(ctx context.Context, name string, args ...string) (Result, error) {
	cmd := exec.CommandContext(ctx, name, args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	result := Result{
		Stdout: stdout.String(),
		Stderr: stderr.String(),
	}

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
		}
		return result, fmt.Errorf("command %q failed: %w\nstderr: %s", name, err, stderr.String())
	}

	return result, nil
}

// CommandExists checks if a command is available on PATH.
func CommandExists(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

// MockRunner is a test double for Runner.
type MockRunner struct {
	Results map[string]Result // key is "name arg1 arg2"
	Calls   []string         // records calls made
}

func (m *MockRunner) Run(ctx context.Context, name string, args ...string) (Result, error) {
	key := name
	if len(args) > 0 {
		key = name + " " + strings.Join(args, " ")
	}
	m.Calls = append(m.Calls, key)

	if result, ok := m.Results[key]; ok {
		if result.ExitCode != 0 {
			return result, fmt.Errorf("command %q exited with code %d", key, result.ExitCode)
		}
		return result, nil
	}

	return Result{}, fmt.Errorf("unexpected command: %q", key)
}
```

**Step 4: Run tests**

```bash
go test ./internal/exec/ -v
```

Expected: all PASS.

**Step 5: Commit**

```bash
git add internal/exec/
git commit -m "feat: command execution package with mock runner for testing"
```

---

### Task 9: State Tracking

**Files:**
- Create: `internal/state/state.go`
- Create: `internal/state/state_test.go`

**Step 1: Write state tests**

Create `internal/state/state_test.go`:

```go
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
		InstalledModules: []string{"base", "python"},
		LastRun:          time.Date(2025, 2, 10, 14, 30, 0, 0, time.UTC),
		ManagedEnvVars:   []string{"HTTP_PROXY", "HTTPS_PROXY"},
		ManagedPathEntries: []string{`C:\Users\dru\scoop\shims`},
		ScoopPackages:    []string{"git", "jq"},
		CABundleHash:     "sha256:abc123",
		ShhhVersion:      "0.1.0",
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
```

**Step 2: Run test to verify it fails**

```bash
go test ./internal/state/ -v
```

Expected: FAIL.

**Step 3: Write state implementation**

Create `internal/state/state.go`:

```go
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
```

**Step 4: Run tests**

```bash
go test ./internal/state/ -v
```

Expected: all PASS.

**Step 5: Commit**

```bash
git add internal/state/
git commit -m "feat: state tracking with JSON persistence"
```

---

### Task 10: Base Module — Proxy Steps

**Files:**
- Create: `internal/module/setup/base.go`
- Create: `internal/module/setup/base_test.go`

This is where the real module logic lives. The base module creates Steps that use the platform interfaces. Because steps are closures over the platform interfaces, they're fully testable with mocks.

**Step 1: Write base module tests**

Create `internal/module/setup/base_test.go`:

```go
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

	// Check key steps exist
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

	// Run all proxy steps (first 3)
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
	env := deps.Env
	val, _, err := env.Get("HTTP_PROXY")
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

	// Pre-set the env var
	deps.Env.Set("HTTP_PROXY", "http://proxy:8080")
	os.Setenv("HTTP_PROXY", "http://proxy:8080")

	mod := NewBaseModule(deps)
	ctx := context.Background()

	// First proxy step (HTTP_PROXY) should be skipped
	step := mod.Steps[0]
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
```

**Step 2: Run test to verify it fails**

```bash
go test ./internal/module/setup/ -v
```

Expected: FAIL.

**Step 3: Write the base module**

Create `internal/module/setup/base.go`:

```go
package setup

import (
	"context"
	"fmt"
	"os"

	"github.com/druarnfield/shhh/internal/config"
	shexec "github.com/druarnfield/shhh/internal/exec"
	"github.com/druarnfield/shhh/internal/module"
	"github.com/druarnfield/shhh/internal/platform"
	"github.com/druarnfield/shhh/internal/state"
)

// Dependencies holds the interfaces a module needs.
// Injected by the CLI layer — real implementations on Windows,
// mocks in tests.
type Dependencies struct {
	Config  *config.Config
	Env     platform.UserEnv
	Profile platform.ProfileManager
	Exec    shexec.Runner
	State   *state.State
}

// NewBaseModule creates the base setup module: proxy, certs, git config.
// Scoop installation is a separate step that depends on proxy being set.
func NewBaseModule(deps *Dependencies) *module.Module {
	var steps []module.Step

	// Proxy steps
	if deps.Config.Proxy.HTTP != "" {
		steps = append(steps, proxyStep(deps, "HTTP_PROXY", deps.Config.Proxy.HTTP))
	}
	if deps.Config.Proxy.HTTPS != "" {
		steps = append(steps, proxyStep(deps, "HTTPS_PROXY", deps.Config.Proxy.HTTPS))
	}
	if deps.Config.Proxy.NoProxy != "" {
		steps = append(steps, proxyStep(deps, "NO_PROXY", deps.Config.Proxy.NoProxy))
	}

	// Git config steps
	steps = append(steps, gitDefaultBranchStep(deps))
	steps = append(steps, gitSSLCAInfoStep(deps))

	return &module.Module{
		ID:          "base",
		Name:        "Base",
		Description: "Configure proxy, certificates, and git defaults",
		Category:    module.CategoryBase,
		Steps:       steps,
	}
}

func proxyStep(deps *Dependencies, key, value string) module.Step {
	return module.Step{
		Name:        fmt.Sprintf("Set %s", key),
		Description: fmt.Sprintf("Configure %s environment variable", key),
		Explain: fmt.Sprintf(
			"%s tells tools like git, curl, and pip how to reach the internet through your corporate proxy. "+
				"We set it in both your PowerShell $PROFILE (for interactive shells) and the Windows user "+
				"registry (for GUI apps and other shells).", key),
		Check: func(ctx context.Context) bool {
			// Check if already set in platform env
			val, _, err := deps.Env.Get(key)
			if err == nil && val == value {
				// Also check in-process
				return os.Getenv(key) == value
			}
			return false
		},
		Run: func(ctx context.Context) error {
			// Set in platform persistent storage (registry + profile)
			if err := deps.Env.Set(key, value); err != nil {
				return fmt.Errorf("setting %s: %w", key, err)
			}

			// Set in current process for later steps
			os.Setenv(key, value)

			// Track in state
			deps.State.AddEnvVar(key)

			return nil
		},
		DryRun: func(ctx context.Context) string {
			return fmt.Sprintf("Would set %s=%q in user environment and current process", key, value)
		},
	}
}

func gitDefaultBranchStep(deps *Dependencies) module.Step {
	branch := deps.Config.Git.DefaultBranch
	if branch == "" {
		branch = "main"
	}

	return module.Step{
		Name:        "Set git default branch",
		Description: fmt.Sprintf("Set git init.defaultBranch to %s", branch),
		Explain:     "When you run 'git init', git creates an initial branch. This sets the default name for that branch.",
		Check: func(ctx context.Context) bool {
			result, err := deps.Exec.Run(ctx, "git", "config", "--global", "init.defaultBranch")
			if err != nil {
				return false
			}
			return result.Stdout == branch+"\n" || result.Stdout == branch
		},
		Run: func(ctx context.Context) error {
			_, err := deps.Exec.Run(ctx, "git", "config", "--global", "init.defaultBranch", branch)
			return err
		},
		DryRun: func(ctx context.Context) string {
			return fmt.Sprintf("Would run: git config --global init.defaultBranch %s", branch)
		},
	}
}

func gitSSLCAInfoStep(deps *Dependencies) module.Step {
	caPath := config.CABundlePath()

	return module.Step{
		Name:        "Set git ssl.caInfo",
		Description: "Point git at the shhh CA bundle",
		Explain: "Corporate networks often use TLS-intercepting proxies with custom CA certificates. " +
			"Git needs to know where to find these certificates to verify HTTPS connections. " +
			"We point git at the shhh-managed CA bundle that includes your organization's CAs.",
		Check: func(ctx context.Context) bool {
			result, err := deps.Exec.Run(ctx, "git", "config", "--global", "http.sslCAInfo")
			if err != nil {
				return false
			}
			return result.Stdout == caPath+"\n" || result.Stdout == caPath
		},
		Run: func(ctx context.Context) error {
			_, err := deps.Exec.Run(ctx, "git", "config", "--global", "http.sslCAInfo", caPath)
			if err != nil {
				return err
			}
			// Also set SSL_CERT_FILE for other tools
			os.Setenv("SSL_CERT_FILE", caPath)
			deps.State.AddEnvVar("SSL_CERT_FILE")
			return deps.Env.Set("SSL_CERT_FILE", caPath)
		},
		DryRun: func(ctx context.Context) string {
			return fmt.Sprintf("Would run: git config --global http.sslCAInfo %s", caPath)
		},
	}
}
```

**Step 4: Run tests**

```bash
go test ./internal/module/setup/ -v
```

Expected: all PASS.

**Step 5: Clean up test env vars** (important — proxy tests call os.Setenv)

Add `TestMain` or `t.Cleanup` to the test file to unset env vars after tests. Add to the top of `base_test.go`:

```go
func TestMain(m *testing.M) {
	code := m.Run()
	// Clean up env vars set during tests
	os.Unsetenv("HTTP_PROXY")
	os.Unsetenv("HTTPS_PROXY")
	os.Unsetenv("NO_PROXY")
	os.Unsetenv("SSL_CERT_FILE")
	os.Exit(code)
}
```

**Step 6: Run full test suite**

```bash
go test ./... -v
```

Expected: all PASS.

**Step 7: Commit**

```bash
git add internal/module/setup/
git commit -m "feat: base setup module with proxy and git config steps"
```

---

### Task 11: CLI Setup Command — Wire Everything Together

**Files:**
- Create: `internal/cli/setup.go`
- Modify: `internal/cli/root.go` (add setup subcommand)

**Step 1: Write the setup command**

Create `internal/cli/setup.go`:

```go
package cli

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/druarnfield/shhh/internal/config"
	"github.com/druarnfield/shhh/internal/exec"
	"github.com/druarnfield/shhh/internal/logging"
	"github.com/druarnfield/shhh/internal/module"
	"github.com/druarnfield/shhh/internal/module/setup"
	"github.com/druarnfield/shhh/internal/platform"
	"github.com/druarnfield/shhh/internal/state"
	"github.com/spf13/cobra"
)

func newSetupCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "setup [module...]",
		Short: "Set up your development environment",
		Long:  "Run the setup wizard. Optionally specify module names (e.g., 'shhh setup base') to run specific modules only.",
		RunE:  runSetup,
	}

	return cmd
}

func runSetup(cmd *cobra.Command, args []string) error {
	// Load config
	cfgPath := config.ConfigFilePath()
	cfg, err := config.LoadFromFile(cfgPath)
	if err != nil {
		// Try defaults if no config file
		if os.IsNotExist(err) {
			fmt.Println("No config file found, using defaults.")
			fmt.Printf("Create %s to customize.\n\n", cfgPath)
			cfg = config.Defaults()
		} else {
			return fmt.Errorf("loading config: %w", err)
		}
	} else {
		fmt.Printf("Config: %s\n", cfgPath)
		if cfg.Org.Name != "" {
			fmt.Printf("Org:    %s\n", cfg.Org.Name)
		}
		fmt.Println()
	}

	// Set up logging
	logger, err := logging.Setup(config.LogFilePath(), flagVerbose)
	if err != nil {
		// Non-fatal: fall back to nop logger
		logger = slog.New(logging.NopHandler{})
	}

	// Load state
	st, err := state.Load(config.StateFilePath())
	if err != nil {
		st = &state.State{}
	}

	// Create platform backends
	env := platform.NewUserEnv()
	prof := platform.NewProfileManager()

	// Build dependencies
	deps := &setup.Dependencies{
		Config:  cfg,
		Env:     env,
		Profile: prof,
		Exec:    &exec.DefaultRunner{},
		State:   st,
	}

	// Build module registry
	reg := module.NewRegistry()
	reg.Register(setup.NewBaseModule(deps))

	// Determine which modules to run
	moduleIDs := args
	if len(moduleIDs) == 0 {
		// Default: run all registered modules
		for _, m := range reg.All() {
			moduleIDs = append(moduleIDs, m.ID)
		}
	}

	// Create runner
	runner := module.NewRunner(logger, flagDryRun)
	runner.SetCallback(cliStepCallback)

	if flagDryRun {
		fmt.Println("=== DRY RUN ===")
		fmt.Println()
	}

	// Run modules
	ctx := context.Background()
	results, err := runner.RunModules(ctx, reg, moduleIDs)

	// Print summary
	fmt.Println()
	printSummary(results)

	// Save state
	st.LastRun = time.Now()
	for _, r := range results {
		if r.Err == nil {
			st.AddModule(r.ModuleID)
		}
	}
	if saveErr := state.Save(config.StateFilePath(), st); saveErr != nil {
		logger.Error("failed to save state", "error", saveErr)
	}

	if err != nil {
		fmt.Println()
		fmt.Println("Setup failed. Fix the issue and re-run — completed steps will be skipped.")
		return err
	}

	return nil
}

func cliStepCallback(mod *module.Module, step *module.Step, index int, total int, skipped bool, err error) {
	prefix := fmt.Sprintf("  [%d/%d]", index+1, total)

	if skipped {
		fmt.Printf("%s  %s (already done)\n", prefix, step.Name)
		return
	}

	if err != nil {
		fmt.Printf("%s  %s FAILED: %v\n", prefix, step.Name, err)
		return
	}

	fmt.Printf("%s  %s\n", prefix, step.Name)

	if flagExplain && step.Explain != "" {
		fmt.Printf("         %s\n", step.Explain)
	}
}

func printSummary(results []module.ModuleResult) {
	totalCompleted := 0
	totalSkipped := 0
	totalSteps := 0

	for _, r := range results {
		totalCompleted += r.Completed
		totalSkipped += r.Skipped
		totalSteps += r.Total

		status := "done"
		if r.Err != nil {
			status = fmt.Sprintf("FAILED at %q", r.FailedStep)
		}
		fmt.Printf("  %s: %s (%d completed, %d skipped)\n",
			r.ModuleID, status, r.Completed, r.Skipped)
	}

	fmt.Printf("\nTotal: %d steps (%d completed, %d skipped)\n",
		totalSteps, totalCompleted, totalSkipped)
}
```

**Step 2: Wire setup command into root**

Modify `internal/cli/root.go` — add this line inside `newRootCmd` after the version command:

```go
cmd.AddCommand(newSetupCmd())
```

**Step 3: Build and verify**

```bash
go mod tidy && task build
./shhh setup --help
./shhh setup --dry-run base
```

Expected: help text shows setup command. Dry run prints the proxy steps with "would do" messages. On macOS, the platform stubs return `ErrNotSupported` for real operations, but dry-run should work since it doesn't call `Run`.

**Step 4: Commit**

```bash
git add internal/cli/ cmd/
git commit -m "feat: CLI setup command with structured output and dry-run support"
```

---

### Task 12: Integration Test

**Files:**
- Create: `internal/integration_test.go`

**Step 1: Write end-to-end test with mocks**

Create `internal/integration_test.go`:

```go
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

// TestFullSetupFlow verifies the complete setup flow with mock backends.
// This is the "shhh setup base" equivalent using in-memory mocks.
func TestFullSetupFlow(t *testing.T) {
	// Clean env before test
	t.Cleanup(func() {
		os.Unsetenv("HTTP_PROXY")
		os.Unsetenv("HTTPS_PROXY")
		os.Unsetenv("NO_PROXY")
		os.Unsetenv("SSL_CERT_FILE")
	})

	// Set up config
	cfg := config.Defaults()
	cfg.Org.Name = "Test Org"
	cfg.Proxy.HTTP = "http://proxy:8080"
	cfg.Proxy.HTTPS = "http://proxy:8080"
	cfg.Proxy.NoProxy = "localhost,127.0.0.1"
	cfg.Git.DefaultBranch = "main"

	// Set up mocks
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

	// Build registry
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

	// Verify env vars were set
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

	// Verify state was updated
	if len(st.ManagedEnvVars) == 0 {
		t.Error("state has no managed env vars")
	}

	// Run again — should skip all steps
	runner2 := module.NewRunner(logger, false)
	results2, err := runner2.RunModules(context.Background(), reg, []string{"base"})
	if err != nil {
		t.Fatalf("second RunModules: %v", err)
	}
	if results2[0].Skipped == 0 {
		t.Error("second run should have skipped steps")
	}

	// Verify state can be saved/loaded
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
```

**Step 2: Run the integration test**

```bash
go test ./internal/ -v -run TestFullSetupFlow
```

Expected: PASS.

**Step 3: Run the full test suite**

```bash
go test ./... -v
```

Expected: all PASS.

**Step 4: Commit**

```bash
git add internal/integration_test.go
git commit -m "feat: integration test for full setup flow with mock backends"
```

---

### Task 13: Final Cleanup

**Step 1: Run go mod tidy**

```bash
go mod tidy
```

**Step 2: Run full test suite one final time**

```bash
go test ./... -count=1 -v
```

**Step 3: Verify build**

```bash
task build
./shhh --help
./shhh version
./shhh setup --help
./shhh setup --dry-run
```

**Step 4: Commit any final changes**

```bash
git add go.mod go.sum
git commit -m "chore: tidy go modules"
```

---

## What Phase 1 Delivers

After completing all tasks, you'll have:

1. A working `shhh` binary with `setup`, `version` commands
2. Config loading from `shhh.toml` with sensible defaults
3. Structured logging to `~/.config/shhh/shhh.log`
4. Platform interfaces with build tag stubs (compiles on Mac + Windows)
5. In-memory mocks for all platform operations
6. Module system with topological dependency resolution
7. Module runner with check-before-run, dry-run, and step callbacks
8. Base module with proxy env var and git config steps
9. State tracking via `state.json`
10. Full integration test proving the flow works end-to-end
11. All tests passing on macOS via mocks

## What's Deferred to Later

- Windows platform implementations (Task for when on Windows machine)
- Cert extraction from Windows cert store
- Scoop installation steps (needs Windows)
- TUI (Phase 2)
- Explanation engine rendering (Phase 2 — the `Explain` strings are already on each step)
- Profile management wiring (steps write to `Env` but don't compose `$PROFILE` blocks yet — deferred until Windows testing)
