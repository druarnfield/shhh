package setup

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/druarnfield/shhh/internal/module"
)

// NewGolangModule creates the Go language setup module.
func NewGolangModule(deps *Dependencies) *module.Module {
	var steps []module.Step

	steps = append(steps, installGoStep(deps))
	steps = append(steps, setGOPATHStep(deps))
	steps = append(steps, addGOBINStep(deps))
	if deps.Config.Registries.GoProxy != "" {
		steps = append(steps, configureGOPROXYStep(deps))
	}

	return &module.Module{
		ID:           "golang",
		Name:         "Go",
		Description:  "Install Go and configure GOPATH, GOBIN, and GOPROXY",
		Category:     module.CategoryLanguage,
		Dependencies: []string{"base"},
		Steps:        steps,
	}
}

func installGoStep(deps *Dependencies) module.Step {
	version := deps.Config.Golang.Version

	return module.Step{
		Name:        "Install Go",
		Description: fmt.Sprintf("Install Go %s via Scoop", version),
		Explain:     "Go is the programming language used for many internal tools and services.",
		Check: func(ctx context.Context) bool {
			result, err := deps.Exec.Run(ctx, "go", "version")
			if err != nil {
				return false
			}
			return strings.Contains(result.Stdout, version)
		},
		Run: func(ctx context.Context) error {
			if _, err := deps.Exec.Run(ctx, "scoop", "install", "go"); err != nil {
				return fmt.Errorf("installing go: %w", err)
			}
			deps.State.AddScoopPackage("go")
			return nil
		},
		DryRun: func(_ context.Context) string {
			return fmt.Sprintf("Would install Go %s via scoop", version)
		},
	}
}

func setGOPATHStep(deps *Dependencies) module.Step {
	home, _ := os.UserHomeDir()
	gopath := filepath.Join(home, "go")

	return module.Step{
		Name:        "Set GOPATH",
		Description: "Set GOPATH to ~/go",
		Explain:     "GOPATH tells Go where to store downloaded modules and build artifacts.",
		Check: func(_ context.Context) bool {
			val, _, err := deps.Env.Get("GOPATH")
			if err != nil || val != gopath {
				return false
			}
			return os.Getenv("GOPATH") == gopath
		},
		Run: func(_ context.Context) error {
			if err := deps.Env.Set("GOPATH", gopath); err != nil {
				return fmt.Errorf("setting GOPATH: %w", err)
			}
			os.Setenv("GOPATH", gopath)
			deps.State.AddEnvVar("GOPATH")
			return nil
		},
		DryRun: func(_ context.Context) string {
			return fmt.Sprintf("Would set GOPATH=%s in user environment and current process", gopath)
		},
	}
}

func addGOBINStep(deps *Dependencies) module.Step {
	home, _ := os.UserHomeDir()
	gobin := filepath.Join(home, "go", "bin")

	return module.Step{
		Name:        "Add GOBIN to PATH",
		Description: "Add ~/go/bin to PATH",
		Explain:     "Adding GOBIN to your PATH lets you run Go-installed tools directly from the command line.",
		Check: func(_ context.Context) bool {
			entries, err := deps.Env.ListPath()
			if err != nil {
				return false
			}
			for _, e := range entries {
				if e.Dir == gobin {
					return true
				}
			}
			return false
		},
		Run: func(_ context.Context) error {
			if err := deps.Env.AppendPath(gobin); err != nil {
				return fmt.Errorf("appending GOBIN to PATH: %w", err)
			}
			deps.State.AddPathEntry(gobin)
			return nil
		},
		DryRun: func(_ context.Context) string {
			return fmt.Sprintf("Would add %s to PATH", gobin)
		},
	}
}

func configureGOPROXYStep(deps *Dependencies) module.Step {
	goProxy := deps.Config.Registries.GoProxy

	return module.Step{
		Name:        "Configure GOPROXY",
		Description: fmt.Sprintf("Set GOPROXY to %s", goProxy),
		Explain:     "GOPROXY tells Go where to download modules from. Corporate environments often use an internal proxy.",
		Check: func(ctx context.Context) bool {
			result, err := deps.Exec.Run(ctx, "go", "env", "GOPROXY")
			if err != nil {
				return false
			}
			return strings.TrimSpace(result.Stdout) == goProxy
		},
		Run: func(ctx context.Context) error {
			if _, err := deps.Exec.Run(ctx, "go", "env", "-w", "GOPROXY="+goProxy); err != nil {
				return fmt.Errorf("setting GOPROXY: %w", err)
			}
			os.Setenv("GOPROXY", goProxy)
			deps.State.AddEnvVar("GOPROXY")
			return nil
		},
		DryRun: func(_ context.Context) string {
			return fmt.Sprintf("Would run: go env -w GOPROXY=%s", goProxy)
		},
	}
}
