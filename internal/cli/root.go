package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/user/gitx/internal/ui"
)

var (
	verbose bool
	jsonOut bool
	noColor bool
)

func Execute() error {
	return rootCmd.Execute()
}

var rootCmd = &cobra.Command{
	Use:   "gitx",
	Short: "AI-powered Git assistant",
	Long: `GitX extends Git with intelligent workflows.

Generate commit messages, pull request descriptions,
and changelogs using AI.

Git remains the source of truth. GitX provides intelligence
and automation around existing Git workflows.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Silences errors on commands so we control formatting
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "enable verbose output")
	rootCmd.PersistentFlags().BoolVar(&jsonOut, "json", false, "output in JSON format")
	rootCmd.PersistentFlags().BoolVar(&noColor, "no-color", false, "disable colored output")

	rootCmd.AddCommand(newCommitCmd())
	rootCmd.AddCommand(newDescribeCmd())
	rootCmd.AddCommand(newChangelogCmd())
	rootCmd.AddCommand(newConfigCmd())
	rootCmd.AddCommand(newSetupCmd())
	rootCmd.AddCommand(newDoctorCmd())

	// Disable the built-in completion command
	rootCmd.CompletionOptions.DisableDefaultCmd = true

	// Silence usage on errors so we print the error message directly
	rootCmd.SilenceUsage = true
	rootCmd.SilenceErrors = true
}

// outputLevel returns the UI output level based on global flags.
func outputLevel() ui.OutputLevel {
	if jsonOut {
		return ui.OutputJSON
	}
	if verbose {
		return ui.OutputVerbose
	}
	return ui.OutputNormal
}

func verboseLn(msg string) {
	if verbose {
		fmt.Println(msg)
	}
}
