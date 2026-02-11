package cli

import (
	"context"
	"errors"
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
	return &cobra.Command{
		Use:   "setup [module...]",
		Short: "Set up your development environment",
		Long:  "Run the setup wizard. Optionally specify module names (e.g., 'shhh setup base') to run specific modules only.",
		RunE:  runSetup,
	}
}

func runSetup(cmd *cobra.Command, args []string) error {
	// Load config
	cfgPath := config.ConfigFilePath()
	cfg, err := config.LoadFromFile(cfgPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
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
