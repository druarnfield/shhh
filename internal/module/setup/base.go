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

// Dependencies holds all external dependencies needed by the base module.
type Dependencies struct {
	Config  *config.Config
	Env     platform.UserEnv
	Profile platform.ProfileManager
	Exec    shexec.Runner
	State   *state.State
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

// proxyStep creates a step that sets a proxy-related environment variable
// in both the platform's persistent user environment and the current process.
func proxyStep(deps *Dependencies, key, value string) module.Step {
	return module.Step{
		Name:        fmt.Sprintf("Set %s", key),
		Description: fmt.Sprintf("Configure %s environment variable", key),
		Explain: func(_ context.Context) string {
			return fmt.Sprintf(
				"%s tells tools like git, curl, and pip how to reach the internet through your corporate proxy. "+
					"We set it in both your PowerShell $PROFILE (for interactive shells) and the Windows user "+
					"registry (for GUI apps and other shells).", key)
		},
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

// gitDefaultBranchStep creates a step that configures the default git branch name.
func gitDefaultBranchStep(deps *Dependencies) module.Step {
	branch := deps.Config.Git.DefaultBranch
	if branch == "" {
		branch = "main"
	}

	return module.Step{
		Name:        "Set git default branch",
		Description: fmt.Sprintf("Set git init.defaultBranch to %s", branch),
		Explain: func(_ context.Context) string {
			return "When you run 'git init', git creates an initial branch. This sets the default name for that branch."
		},
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
		Explain: func(_ context.Context) string {
			return "Corporate networks often use TLS-intercepting proxies with custom CA certificates. " +
				"Git needs to know where to find these certificates to verify HTTPS connections. " +
				"We point git at the shhh-managed CA bundle that includes your organization's CAs."
		},
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
			os.Setenv("SSL_CERT_FILE", caPath)
			deps.State.AddEnvVar("SSL_CERT_FILE")
			return deps.Env.Set("SSL_CERT_FILE", caPath)
		},
		DryRun: func(_ context.Context) string {
			return fmt.Sprintf("Would run: git config --global http.sslCAInfo %s", caPath)
		},
	}
}
