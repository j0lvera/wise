package cmd

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/j0lvera/wise/agent"

	"github.com/spf13/cobra"
)

var runCmd = &cobra.Command{
	Use:   "run [task]",
	Short: "Run the agent with a task",
	Long: `Run the agent with a specific task.

Examples:
  # Run a task
  wise run "Create a file called hello.txt"

  # Verbose output (debug logging)
  wise run "List files" -v

  # Pipe task from stdin
  echo "Create hello.txt" | wise run -

  # JSON output for scripting
  wise run "List files" --json

  # Quiet mode (errors only)
  wise run "Build the project" -q`,
	Args: cobra.MaximumNArgs(1),
	RunE: runAgent,
}

func init() {
	rootCmd.AddCommand(runCmd)
}

type RunResult struct {
	Success  bool   `json:"success"`
	Task     string `json:"task"`
	Response string `json:"response,omitempty"`
	Error    string `json:"error,omitempty"`
}

func runAgent(cmd *cobra.Command, args []string) error {
	task := getTask(args)
	if task == "" {
		return userError("no task provided. Usage: wise run \"your task\"")
	}

	// Load config
	cfg, err := agent.LoadConfig(".")
	if err != nil {
		return handleError(err, "loading config")
	}

	// Set log level based on flags
	cfg.LogLevel = "warn"
	if quiet {
		cfg.LogLevel = "error"
	} else if verbose {
		cfg.LogLevel = "debug"
	}

	// Template the task into user prompt
	cfg.UserPrompt = strings.ReplaceAll(cfg.UserPrompt, "{{.Task}}", task)

	// Stream output to stdout unless quiet
	if !quiet {
		cfg.Output = os.Stdout
	}

	// Create and run agent
	a, err := agent.NewWithConfig(cfg)
	if err != nil {
		return handleError(err, "creating agent")
	}

	response, err := a.Run(context.Background())

	// Output result
	if jsonOut {
		return outputJSON(task, response, err)
	}

	if err != nil {
		return handleError(err, "running agent")
	}

	// Print the response or a simple confirmation
	if !quiet {
		if response != "" {
			fmt.Println(response)
		} else {
			fmt.Println("Done.")
		}
	}

	return nil
}

func getTask(args []string) string {
	if len(args) == 0 || args[0] == "-" {
		return readStdin()
	}
	return args[0]
}

func readStdin() string {
	stat, _ := os.Stdin.Stat()
	if (stat.Mode() & os.ModeCharDevice) != 0 {
		return ""
	}

	var lines []string
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "Error reading stdin: %v\n", err)
		return ""
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

func outputJSON(task, response string, err error) error {
	result := RunResult{
		Success:  err == nil,
		Task:     task,
		Response: response,
	}
	if err != nil {
		result.Error = err.Error()
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(result)
}

func userError(msg string) error {
	fmt.Fprintf(os.Stderr, "Error: %s\n", msg)
	os.Exit(ExitMisuse)
	return nil
}

func handleError(err error, context string) error {
	if jsonOut {
		return err
	}

	msg := err.Error()

	switch {
	case strings.Contains(msg, "OPENROUTER_API_KEY"):
		fmt.Fprintln(os.Stderr, "Error: API key not configured.")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "  export OPENROUTER_API_KEY=your-api-key")
		os.Exit(ExitMisuse)

	case strings.Contains(msg, "OPENROUTER_BASE_URL"):
		fmt.Fprintln(os.Stderr, "Error: API base URL not configured.")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "  export OPENROUTER_BASE_URL=https://openrouter.ai/api/v1")
		os.Exit(ExitMisuse)

	case strings.Contains(msg, "permission denied"):
		fmt.Fprintf(os.Stderr, "Error: Permission denied during %s.\n", context)
		os.Exit(ExitPermissionDenied)

	default:
		fmt.Fprintf(os.Stderr, "Error: %s\n", msg)
		os.Exit(ExitError)
	}

	return nil
}
