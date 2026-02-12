package setup

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/druarnfield/shhh/internal/config"
	shexec "github.com/druarnfield/shhh/internal/exec"
	"github.com/druarnfield/shhh/internal/module"
	"github.com/druarnfield/shhh/internal/platform"
	"github.com/druarnfield/shhh/internal/state"
)

// Dependencies holds all external dependencies needed by the base module.
type Dependencies struct {
	Config    *config.Config
	Env       platform.UserEnv
	Profile   platform.ProfileManager
	CertStore platform.CertStore
	Exec      shexec.Runner
	State     *state.State
}

// NewBaseModule creates the base setup module which configures proxy
// environment variables, git defaults, and certificate paths.
func NewBaseModule(deps *Dependencies) *module.Module {
	var steps []module.Step

	if deps.Config.Proxy.HTTP != "" {
		steps = append(steps, proxyStep(deps, "HTTP_PROXY", deps.Config.Proxy.HTTP))
	}
	if deps.Config.Proxy.HTTPS != "" {
		steps = append(steps, proxyStep(deps, "HTTPS_PROXY", deps.Config.Proxy.HTTPS))
	}
	if deps.Config.Proxy.NoProxy != "" {
		steps = append(steps, proxyStep(deps, "NO_PROXY", deps.Config.Proxy.NoProxy))
	}

	steps = append(steps, caBundleStep(deps))
	steps = append(steps, installScoopStep(deps))
	steps = append(steps, installGitStep(deps))
	if len(deps.Config.Scoop.Buckets) > 0 {
		steps = append(steps, scoopBucketsStep(deps))
	}
	steps = append(steps, gitSSLCAInfoStep(deps))
	steps = append(steps, gitDefaultBranchStep(deps))

	return &module.Module{
		ID:          "base",
		Name:        "Base",
		Description: "Configure proxy, certificates, and git defaults",
		Category:    module.CategoryBase,
		Steps:       steps,
	}
}

// proxyStep creates a step that sets a proxy-related environment variable
// in both the platform's persistent user environment and the current process.
func proxyStep(deps *Dependencies, key, value string) module.Step {
	return module.Step{
		Name:        fmt.Sprintf("Set %s", key),
		Description: fmt.Sprintf("Configure %s environment variable", key),
		Explain: fmt.Sprintf(
			"%s tells tools like git, curl, and pip how to reach the internet through your corporate proxy. "+
				"We set it in both your PowerShell $PROFILE (for interactive shells) and the Windows user "+
				"registry (for GUI apps and other shells).", key),
		Check: func(_ context.Context) bool {
			val, _, err := deps.Env.Get(key)
			if err == nil && val == value {
				return os.Getenv(key) == value
			}
			return false
		},
		Run: func(_ context.Context) error {
			if err := deps.Env.Set(key, value); err != nil {
				return fmt.Errorf("setting %s: %w", key, err)
			}
			os.Setenv(key, value)
			deps.State.AddEnvVar(key)
			return nil
		},
		DryRun: func(_ context.Context) string {
			return fmt.Sprintf("Would set %s=%q in user environment and current process", key, value)
		},
	}
}

// caBundleStep creates a step that extracts trusted root CAs from the OS
// certificate store, appends any configured extra PEM files, and writes the
// result as a single PEM bundle that tools like git, pip, and curl can use.
func caBundleStep(deps *Dependencies) module.Step {
	caPath := config.CABundlePath()

	return module.Step{
		Name:        "Build CA bundle",
		Description: "Extract OS root certificates and write PEM bundle",
		Explain: "Corporate networks often use TLS-intercepting proxies with custom root certificates. " +
			"Most dev tools (git, pip, npm, curl) need a PEM file with these certificates to verify " +
			"HTTPS connections. We extract them from your OS certificate store and bundle them into " +
			"a single file that all your tools can use.",
		Check: func(_ context.Context) bool {
			if deps.State.CABundleHash == "" {
				return false
			}
			if _, err := os.Stat(caPath); err != nil {
				return false
			}
			hash, err := computeBundleHash(deps)
			if err != nil {
				return false
			}
			return hash == deps.State.CABundleHash
		},
		Run: func(_ context.Context) error {
			certs, err := deps.CertStore.SystemRoots()
			if err != nil {
				return fmt.Errorf("reading system certificates: %w", err)
			}
			if len(certs) == 0 {
				return fmt.Errorf("no root certificates found in system store")
			}

			var buf []byte
			for _, cert := range certs {
				block := &pem.Block{
					Type:  "CERTIFICATE",
					Bytes: cert.Raw,
				}
				buf = append(buf, pem.EncodeToMemory(block)...)
			}

			// Append extra PEM files.
			for _, path := range deps.Config.Certs.Extra {
				data, err := os.ReadFile(path)
				if err != nil {
					return fmt.Errorf("reading extra cert file %q: %w", path, err)
				}
				// Validate it contains at least one PEM certificate.
				if block, _ := pem.Decode(data); block == nil {
					return fmt.Errorf("extra cert file %q contains no valid PEM data", path)
				}
				buf = append(buf, data...)
			}

			// Atomic write: temp file + rename.
			dir := filepath.Dir(caPath)
			if err := os.MkdirAll(dir, 0755); err != nil {
				return fmt.Errorf("creating config directory: %w", err)
			}
			tmp, err := os.CreateTemp(dir, "ca-bundle-*.pem.tmp")
			if err != nil {
				return fmt.Errorf("creating temp file: %w", err)
			}
			tmpPath := tmp.Name()

			if _, err := tmp.Write(buf); err != nil {
				tmp.Close()
				os.Remove(tmpPath)
				return fmt.Errorf("writing CA bundle: %w", err)
			}
			if err := tmp.Close(); err != nil {
				os.Remove(tmpPath)
				return fmt.Errorf("closing temp file: %w", err)
			}
			if err := os.Rename(tmpPath, caPath); err != nil {
				os.Remove(tmpPath)
				return fmt.Errorf("renaming CA bundle: %w", err)
			}

			// Compute and store hash.
			hash, err := computeBundleHash(deps)
			if err != nil {
				return fmt.Errorf("computing bundle hash: %w", err)
			}
			deps.State.CABundleHash = hash

			// Set SSL_CERT_FILE so tools like pip and curl use this bundle.
			os.Setenv("SSL_CERT_FILE", caPath)
			deps.State.AddEnvVar("SSL_CERT_FILE")
			if err := deps.Env.Set("SSL_CERT_FILE", caPath); err != nil {
				return fmt.Errorf("setting SSL_CERT_FILE: %w", err)
			}

			return nil
		},
		DryRun: func(_ context.Context) string {
			count := 0
			if certs, err := deps.CertStore.SystemRoots(); err == nil {
				count = len(certs)
			}
			return fmt.Sprintf("Would extract %d certs from system store and write to %s", count, caPath)
		},
	}
}

// computeBundleHash computes a deterministic SHA-256 hash over the system root
// certificates (sorted by raw DER bytes) and any configured extra PEM files.
func computeBundleHash(deps *Dependencies) (string, error) {
	certs, err := deps.CertStore.SystemRoots()
	if err != nil {
		return "", err
	}

	// Sort certs by raw DER bytes for deterministic hashing.
	sort.Slice(certs, func(i, j int) bool {
		a, b := certs[i].Raw, certs[j].Raw
		if len(a) != len(b) {
			return len(a) < len(b)
		}
		for k := range a {
			if a[k] != b[k] {
				return a[k] < b[k]
			}
		}
		return false
	})

	h := sha256.New()
	for _, cert := range certs {
		h.Write(cert.Raw)
	}

	for _, path := range deps.Config.Certs.Extra {
		data, err := os.ReadFile(path)
		if err != nil {
			return "", fmt.Errorf("reading extra cert file %q: %w", path, err)
		}
		h.Write(data)
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}

// installScoopStep creates a step that installs the Scoop package manager on Windows.
func installScoopStep(deps *Dependencies) module.Step {
	return module.Step{
		Name:        "Install Scoop",
		Description: "Install Scoop package manager",
		Explain:     "Scoop installs programs to your user directory without admin privileges.",
		Check: func(ctx context.Context) bool {
			_, err := deps.Exec.Run(ctx, "scoop", "--version")
			return err == nil
		},
		Run: func(ctx context.Context) error {
			_, err := deps.Exec.Run(ctx, "powershell", "-NoProfile", "-Command",
				"Set-ExecutionPolicy RemoteSigned -Scope CurrentUser -Force; irm get.scoop.sh | iex")
			if err != nil {
				return fmt.Errorf("installing scoop: %w", err)
			}
			home, _ := os.UserHomeDir()
			shimsDir := filepath.Join(home, "scoop", "shims")
			os.Setenv("PATH", shimsDir+string(os.PathListSeparator)+os.Getenv("PATH"))
			deps.State.AddPathEntry(shimsDir)
			return nil
		},
		DryRun: func(_ context.Context) string {
			return "Would install Scoop package manager via get.scoop.sh"
		},
	}
}

// installGitStep creates a step that installs git via Scoop. Git is required
// early in the base module because Scoop uses it to clone bucket repositories
// and subsequent steps configure git global settings.
func installGitStep(deps *Dependencies) module.Step {
	return module.Step{
		Name:        "Install git",
		Description: "Install git via Scoop",
		Explain: "Git is required by Scoop to manage bucket repositories, and by almost " +
			"every development workflow. We install it early so that all later steps " +
			"(adding buckets, configuring git settings) can succeed.",
		Check: func(ctx context.Context) bool {
			_, err := deps.Exec.Run(ctx, "git", "--version")
			return err == nil
		},
		Run: func(ctx context.Context) error {
			if _, err := deps.Exec.Run(ctx, "scoop", "install", "git"); err != nil {
				return fmt.Errorf("installing git: %w", err)
			}
			deps.State.AddScoopPackage("git")
			return nil
		},
		DryRun: func(_ context.Context) string {
			return "Would install git via scoop"
		},
	}
}

// scoopBucketsStep creates a step that adds configured Scoop buckets.
func scoopBucketsStep(deps *Dependencies) module.Step {
	buckets := deps.Config.Scoop.Buckets

	return module.Step{
		Name:        "Add Scoop buckets",
		Description: "Add Scoop buckets to expand available packages",
		Explain:     "Scoop buckets expand the pool of installable software.",
		Check: func(ctx context.Context) bool {
			result, err := deps.Exec.Run(ctx, "scoop", "bucket", "list")
			if err != nil {
				return false
			}
			for _, b := range buckets {
				if !strings.Contains(result.Stdout, b) {
					return false
				}
			}
			return true
		},
		Run: func(ctx context.Context) error {
			// Get current buckets to find which are missing.
			result, err := deps.Exec.Run(ctx, "scoop", "bucket", "list")
			existing := ""
			if err == nil {
				existing = result.Stdout
			}
			for _, b := range buckets {
				if strings.Contains(existing, b) {
					continue
				}
				if _, err := deps.Exec.Run(ctx, "scoop", "bucket", "add", b); err != nil {
					return fmt.Errorf("adding scoop bucket %q: %w", b, err)
				}
			}
			return nil
		},
		DryRun: func(_ context.Context) string {
			return fmt.Sprintf("Would add Scoop buckets: %s", strings.Join(buckets, ", "))
		},
	}
}

// gitDefaultBranchStep creates a step that configures the default git branch name.
func gitDefaultBranchStep(deps *Dependencies) module.Step {
	branch := deps.Config.Git.DefaultBranch
	if branch == "" {
		branch = "main"
	}

	return module.Step{
		Name:        "Set git default branch",
		Description: fmt.Sprintf("Set git init.defaultBranch to %s", branch),
		Explain: "When you run 'git init', git creates an initial branch. This sets the default name for that branch.",
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
		DryRun: func(_ context.Context) string {
			return fmt.Sprintf("Would run: git config --global init.defaultBranch %s", branch)
		},
	}
}

// gitSSLCAInfoStep creates a step that points git at the shhh-managed CA bundle.
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
			return err
		},
		DryRun: func(_ context.Context) string {
			return fmt.Sprintf("Would run: git config --global http.sslCAInfo %s", caPath)
		},
	}
}
