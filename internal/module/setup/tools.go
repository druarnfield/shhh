package setup

import (
	"context"
	"fmt"
	"strings"

	"github.com/druarnfield/shhh/internal/module"
)

// NewToolsModule creates the developer tools setup module.
func NewToolsModule(deps *Dependencies) *module.Module {
	var steps []module.Step

	if len(deps.Config.Tools.Core) > 0 {
		steps = append(steps, scoopInstallStep(deps,
			"Install core tools",
			"Install foundational developer tools via Scoop",
			"Foundational tools: git, jq, ripgrep, fd, fzf, delta, etc.",
			deps.Config.Tools.Core,
		))
	}
	if len(deps.Config.Tools.Data) > 0 {
		steps = append(steps, scoopInstallStep(deps,
			"Install data tools",
			"Install data engineering tools via Scoop",
			"Data engineering tools: sqlcmd, bcp, etc.",
			deps.Config.Tools.Data,
		))
	}
	if len(deps.Config.Tools.Optional) > 0 {
		steps = append(steps, scoopInstallStep(deps,
			"Install optional tools",
			"Install quality-of-life tools via Scoop",
			"Quality-of-life: bat, eza, lazygit, starship, etc.",
			deps.Config.Tools.Optional,
		))
	}

	return &module.Module{
		ID:           "tools",
		Name:         "Tools",
		Description:  "Install developer tools via Scoop",
		Category:     module.CategoryTool,
		Dependencies: []string{"base"},
		Steps:        steps,
	}
}

// scoopInstallStep creates a step that installs a set of tools via scoop.
func scoopInstallStep(deps *Dependencies, name, description, explain string, tools []string) module.Step {
	return module.Step{
		Name:        name,
		Description: description,
		Explain:     explain,
		Check: func(ctx context.Context) bool {
			result, err := deps.Exec.Run(ctx, "scoop", "list")
			if err != nil {
				return false
			}
			for _, tool := range tools {
				if !strings.Contains(result.Stdout, tool) {
					return false
				}
			}
			return true
		},
		Run: func(ctx context.Context) error {
			// Get installed list to only install missing tools.
			result, err := deps.Exec.Run(ctx, "scoop", "list")
			installed := ""
			if err == nil {
				installed = result.Stdout
			}
			for _, tool := range tools {
				if strings.Contains(installed, tool) {
					continue
				}
				if _, err := deps.Exec.Run(ctx, "scoop", "install", tool); err != nil {
					return fmt.Errorf("installing %s: %w", tool, err)
				}
				deps.State.AddScoopPackage(tool)
			}
			return nil
		},
		DryRun: func(_ context.Context) string {
			return fmt.Sprintf("Would install: %s", strings.Join(tools, ", "))
		},
	}
}
