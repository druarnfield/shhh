package setup

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/druarnfield/shhh/internal/config"
	"github.com/druarnfield/shhh/internal/module"
)

// NewPythonModule creates the Python language setup module.
func NewPythonModule(deps *Dependencies) *module.Module {
	var steps []module.Step

	steps = append(steps, installUVStep(deps))
	steps = append(steps, installPythonStep(deps))
	steps = append(steps, configurePythonCertsStep(deps))
	steps = append(steps, setUVPythonPreferenceStep(deps))
	if deps.Config.Registries.PyPIMirror != "" {
		steps = append(steps, configurePyPIMirrorStep(deps))
	}

	return &module.Module{
		ID:           "python",
		Name:         "Python",
		Description:  "Install Python via uv and configure PyPI settings",
		Category:     module.CategoryLanguage,
		Dependencies: []string{"base"},
		Steps:        steps,
	}
}

func installUVStep(deps *Dependencies) module.Step {
	return module.Step{
		Name:        "Install uv",
		Description: "Install uv Python package manager via Scoop",
		Explain:     "uv is a fast Python package manager that also manages Python installations.",
		Check: func(ctx context.Context) bool {
			_, err := deps.Exec.Run(ctx, "uv", "--version")
			return err == nil
		},
		Run: func(ctx context.Context) error {
			if _, err := deps.Exec.Run(ctx, "scoop", "install", "uv"); err != nil {
				return fmt.Errorf("installing uv: %w", err)
			}
			deps.State.AddScoopPackage("uv")
			return nil
		},
		DryRun: func(_ context.Context) string {
			return "Would install uv via scoop"
		},
	}
}

func installPythonStep(deps *Dependencies) module.Step {
	version := deps.Config.Python.Version

	return module.Step{
		Name:        "Install Python",
		Description: fmt.Sprintf("Install Python %s via uv", version),
		Explain:     "Python is used for scripting, data engineering, and many internal tools.",
		Check: func(ctx context.Context) bool {
			result, err := deps.Exec.Run(ctx, "uv", "python", "list", "--only-installed")
			if err != nil {
				return false
			}
			return strings.Contains(result.Stdout, version)
		},
		Run: func(ctx context.Context) error {
			if _, err := deps.Exec.Run(ctx, "uv", "python", "install", version); err != nil {
				return fmt.Errorf("installing python %s: %w", version, err)
			}
			return nil
		},
		DryRun: func(_ context.Context) string {
			return fmt.Sprintf("Would install Python %s via uv", version)
		},
	}
}

func configurePythonCertsStep(deps *Dependencies) module.Step {
	caPath := config.CABundlePath()
	keys := []string{"REQUESTS_CA_BUNDLE", "PIP_CERT"}

	return module.Step{
		Name:        "Configure Python CA certificates",
		Description: "Point pip and requests at the shhh CA bundle",
		Explain: "Python's requests library and pip each have their own way of finding CA certificates. " +
			"REQUESTS_CA_BUNDLE tells the requests library (used by most Python HTTP clients) where to " +
			"find trusted CAs, and PIP_CERT tells pip directly. Without these, pip install and API calls " +
			"fail with SSL certificate verification errors behind corporate proxies.",
		Check: func(_ context.Context) bool {
			for _, key := range keys {
				val, _, err := deps.Env.Get(key)
				if err != nil || val != caPath {
					return false
				}
				if os.Getenv(key) != caPath {
					return false
				}
			}
			return true
		},
		Run: func(_ context.Context) error {
			for _, key := range keys {
				if err := deps.Env.Set(key, caPath); err != nil {
					return fmt.Errorf("setting %s: %w", key, err)
				}
				os.Setenv(key, caPath)
				deps.State.AddEnvVar(key)
			}
			return nil
		},
		DryRun: func(_ context.Context) string {
			return fmt.Sprintf("Would set REQUESTS_CA_BUNDLE=%s and PIP_CERT=%s", caPath, caPath)
		},
	}
}

func setUVPythonPreferenceStep(deps *Dependencies) module.Step {
	value := "only-managed"

	return module.Step{
		Name:        "Set UV_PYTHON_PREFERENCE",
		Description: "Set UV_PYTHON_PREFERENCE to only-managed",
		Explain:     "This tells uv to only use Python versions it manages, avoiding conflicts with system Python.",
		Check: func(_ context.Context) bool {
			val, _, err := deps.Env.Get("UV_PYTHON_PREFERENCE")
			if err != nil || val != value {
				return false
			}
			return os.Getenv("UV_PYTHON_PREFERENCE") == value
		},
		Run: func(_ context.Context) error {
			if err := deps.Env.Set("UV_PYTHON_PREFERENCE", value); err != nil {
				return fmt.Errorf("setting UV_PYTHON_PREFERENCE: %w", err)
			}
			os.Setenv("UV_PYTHON_PREFERENCE", value)
			deps.State.AddEnvVar("UV_PYTHON_PREFERENCE")
			return nil
		},
		DryRun: func(_ context.Context) string {
			return fmt.Sprintf("Would set UV_PYTHON_PREFERENCE=%s in user environment and current process", value)
		},
	}
}

func configurePyPIMirrorStep(deps *Dependencies) module.Step {
	mirror := deps.Config.Registries.PyPIMirror

	return module.Step{
		Name:        "Configure PyPI mirror",
		Description: fmt.Sprintf("Set UV_INDEX_URL and PIP_INDEX_URL to %s", mirror),
		Explain:     "Corporate environments often host an internal PyPI mirror for approved packages.",
		Check: func(_ context.Context) bool {
			uvVal, _, err := deps.Env.Get("UV_INDEX_URL")
			if err != nil || uvVal != mirror {
				return false
			}
			pipVal, _, err := deps.Env.Get("PIP_INDEX_URL")
			if err != nil || pipVal != mirror {
				return false
			}
			return os.Getenv("UV_INDEX_URL") == mirror && os.Getenv("PIP_INDEX_URL") == mirror
		},
		Run: func(_ context.Context) error {
			for _, key := range []string{"UV_INDEX_URL", "PIP_INDEX_URL"} {
				if err := deps.Env.Set(key, mirror); err != nil {
					return fmt.Errorf("setting %s: %w", key, err)
				}
				os.Setenv(key, mirror)
				deps.State.AddEnvVar(key)
			}
			return nil
		},
		DryRun: func(_ context.Context) string {
			return fmt.Sprintf("Would set UV_INDEX_URL=%s and PIP_INDEX_URL=%s", mirror, mirror)
		},
	}
}
