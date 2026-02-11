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
	cmd.AddCommand(newSetupCmd())

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
