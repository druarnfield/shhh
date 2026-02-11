package setup

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/druarnfield/shhh/internal/config"
	"github.com/druarnfield/shhh/internal/module"
)

// NewNodeModule creates the Node.js language setup module.
func NewNodeModule(deps *Dependencies) *module.Module {
	var steps []module.Step

	steps = append(steps, installFnmStep(deps))
	steps = append(steps, configureFnmShellStep(deps))
	steps = append(steps, installNodeStep(deps))
	steps = append(steps, configureNodeCertsStep(deps))
	if deps.Config.Registries.NPMRegistry != "" {
		steps = append(steps, configureNPMRegistryStep(deps))
	}

	return &module.Module{
		ID:           "node",
		Name:         "Node.js",
		Description:  "Install Node.js via fnm and configure npm registry",
		Category:     module.CategoryLanguage,
		Dependencies: []string{"base"},
		Steps:        steps,
	}
}

func installFnmStep(deps *Dependencies) module.Step {
	return module.Step{
		Name:        "Install fnm",
		Description: "Install fnm (Fast Node Manager) via Scoop",
		Explain:     "fnm manages multiple Node.js versions, letting you switch between projects easily.",
		Check: func(ctx context.Context) bool {
			_, err := deps.Exec.Run(ctx, "fnm", "--version")
			return err == nil
		},
		Run: func(ctx context.Context) error {
			if _, err := deps.Exec.Run(ctx, "scoop", "install", "fnm"); err != nil {
				return fmt.Errorf("installing fnm: %w", err)
			}
			deps.State.AddScoopPackage("fnm")
			return nil
		},
		DryRun: func(_ context.Context) string {
			return "Would install fnm via scoop"
		},
	}
}

func configureFnmShellStep(deps *Dependencies) module.Step {
	fnmInitLine := `fnm env --use-on-cd --shell power-shell | Out-String | Invoke-Expression`

	return module.Step{
		Name:        "Configure fnm shell",
		Description: "Add fnm shell initialization to PowerShell profile",
		Explain:     "This adds fnm's shell integration so that Node.js versions are activated automatically.",
		Check: func(_ context.Context) bool {
			block, err := deps.Profile.ManagedBlock()
			if err != nil {
				return false
			}
			return strings.Contains(block, "fnm env")
		},
		Run: func(_ context.Context) error {
			if err := deps.Profile.AppendToManagedBlock(fnmInitLine); err != nil {
				return fmt.Errorf("adding fnm init to profile: %w", err)
			}
			return nil
		},
		DryRun: func(_ context.Context) string {
			return "Would add fnm shell initialization to PowerShell profile"
		},
	}
}

func installNodeStep(deps *Dependencies) module.Step {
	version := deps.Config.Node.Version

	return module.Step{
		Name:        "Install Node.js",
		Description: fmt.Sprintf("Install Node.js %s via fnm", version),
		Explain:     "Node.js is the JavaScript runtime used for frontend tooling and many internal services.",
		Check: func(ctx context.Context) bool {
			result, err := deps.Exec.Run(ctx, "fnm", "list")
			if err != nil {
				return false
			}
			return strings.Contains(result.Stdout, version)
		},
		Run: func(ctx context.Context) error {
			if _, err := deps.Exec.Run(ctx, "fnm", "install", version); err != nil {
				return fmt.Errorf("installing node %s: %w", version, err)
			}
			if _, err := deps.Exec.Run(ctx, "fnm", "default", version); err != nil {
				return fmt.Errorf("setting default node version: %w", err)
			}
			return nil
		},
		DryRun: func(_ context.Context) string {
			return fmt.Sprintf("Would install Node.js %s via fnm and set as default", version)
		},
	}
}

func configureNodeCertsStep(deps *Dependencies) module.Step {
	caPath := config.CABundlePath()
	version := deps.Config.Node.Version

	return module.Step{
		Name:        "Configure Node.js CA certificates",
		Description: "Point Node.js and npm at the shhh CA bundle",
		Explain: "Node.js has its own built-in CA certificate list that doesn't include corporate proxy CAs. " +
			"NODE_EXTRA_CA_CERTS tells Node to load additional certificates, and npm's cafile setting " +
			"tells npm specifically where to find trusted CAs. Without these, npm install and any Node.js " +
			"HTTPS request will fail with UNABLE_TO_VERIFY_LEAF_SIGNATURE behind corporate proxies.",
		Check: func(ctx context.Context) bool {
			val, _, err := deps.Env.Get("NODE_EXTRA_CA_CERTS")
			if err != nil || val != caPath {
				return false
			}
			if os.Getenv("NODE_EXTRA_CA_CERTS") != caPath {
				return false
			}
			result, err := deps.Exec.Run(ctx, "fnm", "exec", "--using", version, "--", "npm", "config", "get", "cafile")
			if err != nil {
				return false
			}
			return strings.TrimSpace(result.Stdout) == caPath
		},
		Run: func(ctx context.Context) error {
			if err := deps.Env.Set("NODE_EXTRA_CA_CERTS", caPath); err != nil {
				return fmt.Errorf("setting NODE_EXTRA_CA_CERTS: %w", err)
			}
			os.Setenv("NODE_EXTRA_CA_CERTS", caPath)
			deps.State.AddEnvVar("NODE_EXTRA_CA_CERTS")

			if _, err := deps.Exec.Run(ctx, "fnm", "exec", "--using", version, "--", "npm", "config", "set", "cafile", caPath); err != nil {
				return fmt.Errorf("setting npm cafile: %w", err)
			}
			return nil
		},
		DryRun: func(_ context.Context) string {
			return fmt.Sprintf("Would set NODE_EXTRA_CA_CERTS=%s and npm config set cafile %s", caPath, caPath)
		},
	}
}

func configureNPMRegistryStep(deps *Dependencies) module.Step {
	registry := deps.Config.Registries.NPMRegistry
	version := deps.Config.Node.Version

	return module.Step{
		Name:        "Configure npm registry",
		Description: fmt.Sprintf("Set npm registry to %s", registry),
		Explain:     "Corporate environments often host an internal npm registry for approved packages.",
		Check: func(ctx context.Context) bool {
			result, err := deps.Exec.Run(ctx, "fnm", "exec", "--using", version, "--", "npm", "config", "get", "registry")
			if err != nil {
				return false
			}
			got := strings.TrimRight(strings.TrimSpace(result.Stdout), "/")
			want := strings.TrimRight(registry, "/")
			return got == want
		},
		Run: func(ctx context.Context) error {
			if _, err := deps.Exec.Run(ctx, "fnm", "exec", "--using", version, "--", "npm", "config", "set", "registry", registry); err != nil {
				return fmt.Errorf("setting npm registry: %w", err)
			}
			return nil
		},
		DryRun: func(_ context.Context) string {
			return fmt.Sprintf("Would run: fnm exec --using %s -- npm config set registry %s", version, registry)
		},
	}
}
