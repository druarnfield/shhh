# shhh — Developer Environment Bootstrapper

## Overview

`shhh` is a single Go binary that bootstraps and manages developer environments on locked-down Windows workstations without admin privileges. It combines a guided TUI setup wizard with daily-use CLI utilities, teaching users what it's doing at each step.

**Design principles:**
- Single binary, zero dependencies at runtime
- No admin required — everything targets user-level configuration
- Idempotent — safe to re-run, skips what's already done
- Educational — explains *why*, not just *what*
- Practical — becomes a daily-use tool, not a one-time script

---

## Project Structure

```
shhh/
├── cmd/
│   └── shhh/
│       └── main.go                 # Entrypoint, root cobra command
│
├── internal/
│   ├── cli/                        # CLI command definitions
│   │   ├── root.go                 # Root command, global flags (--explain, --quiet, --dry-run)
│   │   ├── setup.go                # `shhh setup` — launches TUI wizard
│   │   ├── doctor.go               # `shhh doctor` — health checks
│   │   ├── path.go                 # `shhh path add|remove|list|clean|check`
│   │   ├── env.go                  # `shhh env set|get|list|source`
│   │   ├── profile.go              # `shhh profile edit|reload|append|show|diff`
│   │   ├── proxy.go                # `shhh proxy on|off|test`
│   │   ├── cert.go                 # `shhh cert check|export`
│   │   ├── ssh.go                  # `shhh ssh keygen|test|config`
│   │   ├── which.go                # `shhh which <cmd>`
│   │   ├── port.go                 # `shhh port <host:port>`
│   │   ├── update.go               # `shhh update` — update managed tools
│   │   └── config.go               # `shhh config show|edit|init`
│   │
│   ├── tui/                        # Bubble Tea TUI components
│   │   ├── wizard/
│   │   │   ├── wizard.go           # Main setup wizard model (step flow)
│   │   │   ├── picker.go           # Module multi-select picker
│   │   │   ├── progress.go         # Step execution with progress/output
│   │   │   └── explain.go          # Explanation panels (the "teaching" UI)
│   │   ├── doctor/
│   │   │   └── doctor.go           # Doctor results dashboard
│   │   ├── status/
│   │   │   └── status.go           # `shhh status` dashboard
│   │   └── components/
│   │       ├── spinner.go          # Shared spinner component
│   │       ├── banner.go           # shhh ASCII banner
│   │       └── styles.go           # Lipgloss shared styles
│   │
│   ├── module/                     # Module system
│   │   ├── module.go               # Module interface + registry
│   │   ├── runner.go               # Module execution engine
│   │   ├── setup/                  # Setup modules (the guided installers)
│   │   │   ├── base.go             # Proxy, certs, scoop, git
│   │   │   ├── python.go           # uv, profile config, pyproject template
│   │   │   ├── golang.go           # Go toolchain, env vars, GOPROXY
│   │   │   ├── node.go             # fnm, npm config, registry
│   │   │   ├── rust.go             # rustup, cargo config
│   │   │   ├── ssh.go              # SSH key generation, gitlab config
│   │   │   └── tools.go            # bcp, sqlcmd, quality-of-life tools
│   │   └── doctor/                 # Doctor check modules
│   │       ├── proxy.go            # Proxy connectivity checks
│   │       ├── certs.go            # CA bundle validation
│   │       ├── git.go              # Git config health
│   │       ├── path.go             # PATH sanity checks
│   │       └── tools.go            # Tool version checks
│   │
│   ├── platform/                   # Windows-specific operations
│   │   ├── platform.go             # Platform interface definitions
│   │   ├── registry_windows.go     # User-level registry (env vars, PATH)
│   │   ├── registry_other.go       # Stub/mock for non-Windows builds
│   │   ├── profile_windows.go      # PowerShell $PROFILE management
│   │   ├── profile_other.go        # Stub/mock for non-Windows builds
│   │   ├── cert_windows.go         # Certificate store access (CryptoAPI)
│   │   ├── cert_other.go           # Stub/mock for non-Windows builds
│   │   ├── env_windows.go          # Environment variable read/write
│   │   ├── env_other.go            # Stub/mock for non-Windows builds
│   │   ├── path_windows.go         # PATH manipulation (user-level)
│   │   └── path_other.go           # Stub/mock for non-Windows builds
│   │
│   ├── config/                     # Configuration
│   │   ├── config.go               # shhh.toml parsing + defaults
│   │   └── locations.go            # XDG-style paths for Windows
│   │
│   ├── exec/                       # Command execution
│   │   ├── run.go                  # Run external commands with output capture
│   │   ├── powershell.go           # PowerShell command helpers
│   │   └── scoop.go                # Scoop install/update helpers
│   │
│   ├── logging/                    # Structured logging
│   │   └── logging.go              # slog setup, file + TUI output
│   │
│   └── explain/                    # Explanation engine
│       ├── explain.go              # Explanation registry + rendering
│       └── topics/                 # Explanation content (embedded)
│           ├── proxy.go            # Why proxies, what env vars do
│           ├── certs.go            # What CA bundles are, why tools break
│           ├── ssh.go              # SSH keys, ed25519 vs RSA
│           ├── path.go             # How PATH works on Windows
│           └── profile.go          # What $PROFILE is, load order
│
├── shhh.example.toml               # Example org config
├── go.mod
├── go.sum
├── Makefile                         # Build targets
└── README.md
```

---

## Core Interfaces

### Module Interface

Every setup module implements the same interface. This keeps adding new modules dead simple.

```go
// internal/module/module.go
package module

import "context"

// Step is a single action within a module.
// Each step is displayed individually in the TUI with its own
// explanation, progress, and status.
type Step struct {
    Name        string                          // Short display name: "Set HTTP_PROXY"
    Description string                          // One-liner: "Configure proxy environment variable"
    Explain     string                          // Teaching text: "HTTP_PROXY tells tools like git..."
    Check       func(ctx context.Context) bool  // Already done? Skip if true.
    Run         func(ctx context.Context) error // Do the thing.
    DryRun      func(ctx context.Context) string // Describe what would happen.
}

// Module is a logical group of setup steps.
type Module struct {
    ID           string   // "python", "golang", "base"
    Name         string   // "Python (uv)"
    Description  string   // "Install uv and configure Python development environment"
    Category     Category // CategoryBase, CategoryLanguage, CategoryTool
    Dependencies []string // Module IDs that must run first: ["base"]
    Steps        []Step
}

type Category int

const (
    CategoryBase Category = iota
    CategoryLanguage
    CategoryTool
)

// Registry holds all available modules.
type Registry struct {
    modules map[string]*Module
}

func NewRegistry() *Registry { ... }
func (r *Registry) Register(m *Module) { ... }
func (r *Registry) Get(id string) *Module { ... }
func (r *Registry) ByCategory(c Category) []*Module { ... }
func (r *Registry) ResolveDeps(ids []string) ([]string, error) { ... } // topological sort
```

### Error Handling Strategy

There is no rollback mechanism by design. The module system leans fully into idempotency:

- If a step fails, execution stops and the error is displayed with context (what failed, why, and how to retry).
- Every `Step.Check` function means re-running `shhh setup` safely skips completed steps and picks up where it left off.
- The runner logs the failure to `shhh.log` so the user (or whoever is helping them) can see exactly what happened.

This keeps the module interface simple and matches the educational philosophy — users learn to troubleshoot rather than relying on magic undo.

### Doctor Check Interface

```go
// internal/module/doctor.go
package module

type CheckResult struct {
    Name    string
    Status  CheckStatus // OK, Warn, Fail, Skip
    Message string      // "HTTP_PROXY = http://proxy:8080"
    Fix     string      // "Run: shhh setup base"
}

type CheckStatus int

const (
    CheckOK CheckStatus = iota
    CheckWarn
    CheckFail
    CheckSkip
)

type DoctorCheck interface {
    Name() string
    Category() string
    Run(ctx context.Context) []CheckResult
}
```

### Platform Interface

```go
// internal/platform/platform.go
package platform

// UserEnv manages user-level environment variables via the Windows
// registry (HKCU\Environment). Changes persist across sessions
// without requiring admin. Broadcasts WM_SETTINGCHANGE after writes
// so new terminal windows pick up changes immediately.
type UserEnv interface {
    Get(key string) (value string, source EnvSource, err error)
    Set(key, value string) error          // persistent, user-level
    Delete(key string) error
    AppendPath(dir string) error          // add to user PATH
    RemovePath(dir string) error          // remove from user PATH
    ListPath() ([]PathEntry, error)       // all PATH entries with metadata
}

type EnvSource int

const (
    SourceProcess EnvSource = iota
    SourceUser
    SourceSystem
)

type PathEntry struct {
    Dir    string
    Source EnvSource
    Exists bool // does the directory actually exist?
}
```

### Profile Manager

```go
// internal/platform/profile.go
package platform

// ProfileManager handles PowerShell $PROFILE modifications.
// All shhh-managed content lives between sentinel markers:
//
//   # >>> shhh managed - do not edit >>>
//   $env:HTTP_PROXY = "http://proxy:8080"
//   # <<< shhh managed <<<
//
// Content outside the markers is never touched.
type ProfileManager interface {
    Path() string                                    // resolve $PROFILE path
    Read() (string, error)                           // full content
    ManagedBlock() (string, error)                   // just shhh's section
    SetManagedBlock(content string) error             // replace shhh's section
    AppendToManagedBlock(line string) error           // add a line
    Diff() (string, error)                           // show pending changes
    Exists() bool
    EnsureExists() error                             // create if missing
}
```

---

## Config File

Lives at `~/.config/shhh/shhh.toml` (or next to the binary for portability).

```toml
# shhh.toml — Organisation configuration
# Share this file with your team. New hires get the binary + this file.

[org]
name = "Health Data Services"

[proxy]
http  = "http://proxy.health.gov:8080"
https = "http://proxy.health.gov:8080"
no_proxy = "localhost,127.0.0.1,.health.gov,.internal"

[certs]
# "system" extracts from Windows cert store
# can also be a URL or file path
source = "system"
# additional CAs to bundle (internal intermediates etc)
extra = []

[git]
default_branch = "main"
# auto-configure these remotes to use SSH
ssh_hosts = ["gitlab.health.gov"]

[gitlab]
host = "gitlab.health.gov"
ssh_port = 22

[registries]
pypi_mirror   = ""  # leave empty if not applicable
npm_registry  = ""
go_proxy      = ""

[scoop]
# extra scoop buckets to add
buckets = ["extras", "versions"]

[tools]
# tools to install via scoop during setup
core = [
    "git", "7zip", "curl", "jq", "yq",
    "ripgrep", "fd", "fzf", "delta",
    "neovim",
]

# data engineering tools
data = [
    "sqlcmd",
    # bcp installed via sqlcmd tools
]

# optional extras the user can pick
optional = [
    "dbeaver", "wezterm", "starship",
    "lazygit", "bat", "eza", "dust", "tokei",
]

[python]
# default python version for uv
version = "3.12"

[golang]
version = "1.23"

[node]
version = "22"
```

---

## Command Flows

### `shhh setup` — Guided TUI Wizard

```
┌─────────────────────────────────────────────────────┐
│  shhh — Developer Environment Setup                 │
│                                                     │
│  Welcome! This wizard will set up your dev          │
│  environment. Each step explains what it's doing    │
│  so you can do it yourself next time.               │
│                                                     │
│  Config: ~/.config/shhh/shhh.toml                   │
│  Mode: [x] Explain   [ ] Quick                      │
│                                                     │
│  Select modules to install:                         │
│                                                     │
│  Base (required)                                    │
│  [x] Proxy & certificates                           │
│  [x] Scoop package manager                          │
│  [x] Git configuration                              │
│                                                     │
│  Languages                                          │
│  [x] Python (uv)                                    │
│  [ ] Go                                             │
│  [ ] Node.js (fnm)                                  │
│  [ ] Rust                                           │
│                                                     │
│  Tools                                              │
│  [x] Data tools (bcp, sqlcmd)                       │
│  [x] CLI essentials (ripgrep, fzf, jq, delta...)   │
│  [ ] Optional extras (dbeaver, wezterm...)          │
│                                                     │
│  SSH                                                │
│  [x] Generate SSH key for GitLab                    │
│                                                     │
│  ↑↓ navigate  space select  enter continue          │
└─────────────────────────────────────────────────────┘
```

After selection, each step runs sequentially:

```
┌─────────────────────────────────────────────────────┐
│  shhh — Setting up proxy                            │
│  Step 2/14  ████████░░░░░░░░░░░░░░░░░░░░░░  14%    │
│                                                     │
│  ┌ What's happening ─────────────────────────────┐  │
│  │                                               │  │
│  │  Your network uses a proxy server to reach    │  │
│  │  the internet. Tools like git, pip, and curl  │  │
│  │  need to know about it via environment vars.  │  │
│  │                                               │  │
│  │  We're setting these in your PowerShell       │  │
│  │  $PROFILE (~\Documents\PowerShell\...):       │  │
│  │                                               │  │
│  │    $env:HTTP_PROXY  = "http://proxy:8080"     │  │
│  │    $env:HTTPS_PROXY = "http://proxy:8080"     │  │
│  │    $env:NO_PROXY    = "localhost,.internal"    │  │
│  │                                               │  │
│  │  And persisting them to your user registry    │  │
│  │  so non-PowerShell tools pick them up too.    │  │
│  │                                               │  │
│  │  To do this manually next time:               │  │
│  │  > [System.Environment]::SetEnvironmentVar... │  │
│  │                                               │  │
│  └───────────────────────────────────────────────┘  │
│                                                     │
│  ✓ Set HTTP_PROXY in $PROFILE                       │
│  ✓ Set HTTPS_PROXY in $PROFILE                      │
│  ✓ Set NO_PROXY in $PROFILE                         │
│  ● Persisting to user environment...                │
│  ○ Verify proxy connectivity                        │
│                                                     │
│  enter continue  q quit  ? toggle explanation        │
└─────────────────────────────────────────────────────┘
```

### `shhh doctor` — Health Dashboard

```
┌─────────────────────────────────────────────────────┐
│  shhh doctor                                        │
│                                                     │
│  Network                                            │
│  ✓ HTTP_PROXY configured                            │
│  ✓ HTTPS_PROXY configured                           │
│  ✓ Proxy reachable (180ms)                          │
│  ✓ External connectivity via proxy                  │
│  ⚠ NO_PROXY missing .gitlab.health.gov              │
│    fix: shhh env set NO_PROXY="..,.gitlab.health.." │
│                                                     │
│  Certificates                                       │
│  ✓ CA bundle exists (~/.config/shhh/ca-bundle.pem)  │
│  ✓ CA bundle valid (expires 2026-03-15)             │
│  ✓ SSL_CERT_FILE set                                │
│  ✓ git ssl.cainfo set                               │
│                                                     │
│  Git                                                │
│  ✓ git installed (2.43.0)                           │
│  ✓ user.name configured                             │
│  ✗ user.email not set                               │
│    fix: git config --global user.email "you@org"    │
│  ✓ credential.helper configured                     │
│                                                     │
│  Tools                                              │
│  ✓ scoop (0.4.1)                                    │
│  ✓ uv (0.5.2)                                       │
│  ✓ go (1.23.4)                                      │
│  ⚠ bcp not found                                    │
│    fix: shhh setup tools                            │
│                                                     │
│  5 passed  2 warnings  1 failed                     │
└─────────────────────────────────────────────────────┘
```

### `shhh ssh keygen` — SSH Key Flow

```
┌─────────────────────────────────────────────────────┐
│  shhh — SSH Key Setup for GitLab                    │
│                                                     │
│  ┌ Why ed25519? ─────────────────────────────────┐  │
│  │ Ed25519 keys are shorter, faster, and more    │  │
│  │ secure than RSA. They're the modern default.  │  │
│  └───────────────────────────────────────────────┘  │
│                                                     │
│  ✓ Generated ~/.ssh/id_ed25519_gitlab               │
│  ✓ Created ~/.ssh/config entry:                     │
│    Host gitlab.health.gov                           │
│      IdentityFile ~/.ssh/id_ed25519_gitlab          │
│      ProxyCommand connect-proxy -H proxy:8080 %h %p│
│  ✓ Public key copied to clipboard                   │
│                                                     │
│  ┌ Next step ────────────────────────────────────┐  │
│  │ Opening GitLab SSH keys page in your browser. │  │
│  │ Paste the key (it's on your clipboard) and    │  │
│  │ click "Add key".                              │  │
│  │                                               │  │
│  │ Press enter when done, we'll test it.         │  │
│  └───────────────────────────────────────────────┘  │
│                                                     │
│  Waiting for you...  enter when ready               │
└─────────────────────────────────────────────────────┘
```

---

## Key Design Decisions

### 1. Module Dependencies via Topological Sort

Modules declare dependencies by ID. The runner resolves execution order automatically.

```
base (no deps) → python (depends: base) → tools (depends: base)
                → golang (depends: base)
                → ssh (depends: base, git)
```

If someone selects `python` but not `base`, `base` is auto-included. The TUI shows this: "Python requires Base — adding automatically."

### 2. Managed $PROFILE Blocks

All `$PROFILE` modifications live between sentinel comments:

```powershell
# normal user stuff above is untouched

# >>> shhh managed - do not edit >>>
$env:HTTP_PROXY = "http://proxy.health.gov:8080"
$env:HTTPS_PROXY = "http://proxy.health.gov:8080"
$env:NO_PROXY = "localhost,127.0.0.1,.health.gov"
$env:SSL_CERT_FILE = "$HOME\.config\shhh\ca-bundle.pem"
$env:GOPATH = "$HOME\go"
$env:GOBIN = "$HOME\go\bin"
$env:UV_PYTHON_PREFERENCE = "only-managed"
# <<< shhh managed <<<

# normal user stuff below is untouched
```

This makes `shhh profile diff` and `shhh profile clean` trivial to implement, and the user always knows what shhh owns.

### 3. Dual Persistence Strategy

Environment variables are set in **two places**:
1. **PowerShell `$PROFILE`** — for interactive shells
2. **User registry** (`HKCU\Environment`) — for GUI apps, other shells, and child processes

The explanation engine teaches users why both are needed.

### 4. In-Process Environment Propagation

Steps that set environment variables (proxy, certs, PATH) must also call `os.Setenv()` to propagate values to the **current shhh process**. This is critical because later steps (e.g., installing Scoop) spawn child processes that inherit the current process environment — writing to `$PROFILE` or the registry alone won't help until a new shell is opened.

The execution order in the base module is: proxy → certs → scoop → git. The proxy and cert steps must set in-process env vars *before* the scoop step tries to download anything over the network.

The runner in `runner.go` should document this contract: any step that writes an environment variable persistently must also set it in-process.

### 5. Explanation System

Explanations are embedded Go strings (not external files) organized by topic. Each step references an explanation key. Three verbosity modes:

- `--explain` (default in TUI): shows the teaching panel
- `--quiet`: suppresses explanations, just shows progress
- `--dry-run`: shows what *would* happen without doing it

### 6. Idempotency via Check Functions (No Rollback)

Every `Step` has a `Check` function. Before running, the runner calls `Check()`:
- Returns `true` → skip, show "already configured"
- Returns `false` → run the step

This makes re-running safe and fast. `shhh setup` on an already-configured machine finishes in seconds.

There is no rollback mechanism by design. If a step fails:
- Execution stops with a clear error message (what failed, why, how to retry)
- The failure is logged to `shhh.log`
- The user re-runs `shhh setup` — `Check` functions skip completed steps and pick up where it left off

This keeps the module interface simple and matches the educational philosophy.

### 7. State Tracking

shhh tracks what it's installed/configured in `~/.config/shhh/state.json`:

```json
{
  "installed_modules": ["base", "python", "tools"],
  "last_run": "2025-02-10T14:30:00Z",
  "managed_env_vars": ["HTTP_PROXY", "HTTPS_PROXY", "NO_PROXY", "SSL_CERT_FILE"],
  "managed_path_entries": ["C:\\Users\\dru\\scoop\\shims", "C:\\Users\\dru\\go\\bin"],
  "scoop_packages": ["git", "jq", "ripgrep", "neovim"],
  "ca_bundle_hash": "sha256:abc123...",
  "shhh_version": "0.1.0"
}
```

This powers `shhh doctor`, `shhh update`, and `shhh profile clean`.

### 8. Structured Logging

All operations are logged to `~/.config/shhh/shhh.log` using Go's `log/slog` with structured JSON output. The runner automatically logs:
- Each step's `Check` result (skipped or needs to run)
- `Run` output and duration
- Any errors with full context

The log file is capped at a reasonable size (e.g., 5MB, rotated on startup if exceeded). This enables remote troubleshooting — "send me your shhh log" beats trying to reproduce the issue.

Log output is separate from TUI output. The `--verbose` flag can optionally surface log-level detail in the terminal.

### 9. Cross-Platform Development Strategy

The target platform is Windows. However, the codebase supports development and testing on macOS/Linux:

**Build tags** split platform-specific code:
- `*_windows.go` — real implementations (registry, CryptoAPI, PowerShell)
- `*_other.go` — stubs that return sensible defaults or `ErrNotSupported`

**Interface-based testing** — All platform operations go through interfaces (`UserEnv`, `ProfileManager`). Unit tests use in-memory fakes to test module logic, dependency resolution, config parsing, and TUI flows on any OS.

**Demo mode** — On non-Windows, the TUI wizard runs with mock platform backends so the UI can be developed and visually verified without a Windows machine. This is a development convenience, not a supported feature.

No effort is spent making shhh actually *work* on macOS/Linux. The build tags and mocks exist solely to enable development and testing.

---

## Build & Distribution

```makefile
# Makefile
VERSION := $(shell git describe --tags --always --dirty)

build:
	GOOS=windows GOARCH=amd64 go build \
		-ldflags "-s -w -X main.version=$(VERSION)" \
		-o shhh.exe ./cmd/shhh

# single binary, copy to a shared network drive or attach to a wiki page
# new hire downloads shhh.exe + shhh.toml and they're off
```

### Go Dependencies

```
github.com/spf13/cobra             # CLI framework
github.com/charmbracelet/bubbletea # TUI framework
github.com/charmbracelet/bubbles   # TUI components (spinner, list, progress)
github.com/charmbracelet/lipgloss  # TUI styling
github.com/pelletier/go-toml/v2    # Config parsing
golang.org/x/sys/windows/registry  # Windows registry access (user env vars)
golang.org/x/crypto/ssh            # SSH key generation
```

No CGo. Pure Go. Cross-compile from your Mac or Linux box.

---

## Implementation Order

Suggested build order, each phase is shippable:

### Phase 1 — Core + Base Module
- Project skeleton, cobra commands, config loading
- Platform layer: interfaces, build tags, registry, env vars, PATH, $PROFILE management
- Logging setup (`slog` to file)
- Base setup module: proxy, certs, scoop, git
- In-process env propagation for proxy/cert steps
- Simple CLI output (no TUI yet — just structured terminal output)
- `shhh setup base` works end-to-end
- Unit tests with mock platform backends

### Phase 2 — TUI + Explain
- Bubble Tea wizard flow
- Explanation engine with teaching panels
- Module picker with multi-select
- Progress display during step execution
- `shhh setup` with full TUI experience

### Phase 3 — Language Modules
- Python (uv) module
- Go module
- Node module (if needed)
- Tools module (bcp, sqlcmd, quality-of-life)

### Phase 4 — Daily-Use CLI
- `shhh path` commands
- `shhh env` commands
- `shhh profile` commands
- `shhh which`, `shhh port`
- `shhh proxy on/off/test`

### Phase 5 — Doctor + SSH
- Doctor check framework
- All health checks
- SSH key generation flow
- GitLab integration + test

### Phase 6 — Polish
- `shhh update` — update all managed tools
- `shhh export` — dump config from working machine
- `shhh status` — at-a-glance dashboard
- Edge case handling
