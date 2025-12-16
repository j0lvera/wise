# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build & Run Commands

```bash
# Build
task build                   # Output: bin/wise
go build -o bin/wise         # Direct build

# Run
./bin/wise run "your task"   # Run a task
./bin/wise run -             # Read task from stdin
./bin/wise run "task" -v     # Verbose (debug logging)
./bin/wise run "task" -q     # Quiet (errors only)
./bin/wise run "task" --json # JSON output

# Test
go test ./...                 # Run all tests
go test ./agent               # Test agent package
```

## Configuration

Two-layer config system:
- **Environment variables**: API keys and runtime settings (required: `OPENROUTER_API_KEY`, `OPENROUTER_BASE_URL`)
- **config.toml**: Prompts and templates (use `config.example.toml` as reference)

The `{{.Task}}` placeholder in `user_prompt` gets replaced with the CLI argument.

## Architecture

Bash-only agent loop inspired by mini-swe-agent. Each command runs in a fresh bash process (stateless execution).

### Core Components (agent/ package)

- **Agent interface** (`agent.go`): Main loop with `Run()` (full execution) and `Step()` (single iteration). Uses composable components via `NewWithComponents()` for testing.
- **Querier** (`querier.go`): LLM API calls via OpenRouter/OpenAI-compatible endpoints using langchaingo.
- **Parser** (`parser.go`): Extracts bash commands from LLM responses (expects THOUGHT + code block format).
- **Executor** (`executor.go`): Runs bash commands with timeout. Includes `BlocklistValidator` with safety patterns.

### Agent Loop Flow

1. Query LLM with message history
2. Parse action (bash command) from response
3. Execute command with timeout
4. Format output as observation
5. Check for `TASK_COMPLETE` marker or continue

### Termination

- LLM signals completion by running `echo TASK_COMPLETE`
- `TerminatingErr` with `ReasonComplete` or `ReasonStepLimit`
- `ProcessErr` for recoverable errors (added as feedback, loop continues)

### Safety

Commands are validated against `DefaultBlockedPatterns` in executor.go before execution. Patterns block destructive operations like `rm -rf /`, `curl | sh`, system commands.
