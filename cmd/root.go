package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var Version = "0.1.0"

var (
	verbose bool
	quiet   bool
	jsonOut bool
)

// Exit codes
const (
	ExitSuccess          = 0
	ExitError            = 1
	ExitMisuse           = 2
	ExitPermissionDenied = 126
	ExitNotFound         = 127
)

var rootCmd = &cobra.Command{
	Use:     "agent",
	Short:   "An LLM-powered command execution agent",
	Version: Version,
	Long: `Agent is a minimal software engineering agent that executes
shell commands based on LLM responses.

Exit codes:
  0    Success
  1    General error
  2    Command misuse
  126  Permission denied
  127  Command not found

Examples:
  agent run "Create a hello.txt file"
  agent run "List files" -v
  agent run "Create file" -q
  echo "Create hello.txt" | agent run -`,
	PersistentPreRunE: validateFlags,
}

func validateFlags(cmd *cobra.Command, args []string) error {
	if verbose && quiet {
		return fmt.Errorf("--verbose and --quiet cannot be used together")
	}
	return nil
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(ExitError)
	}
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output (debug logs)")
	rootCmd.PersistentFlags().BoolVarP(&quiet, "quiet", "q", false, "quiet mode (errors only)")
	rootCmd.PersistentFlags().BoolVar(&jsonOut, "json", false, "output as JSON")
}
