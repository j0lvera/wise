# agent

Small agent heavily inspired by [mini-swe-agent](https://github.com/SWE-agent/mini-swe-agent), the 100-line Python agent that scores 74% on SWE-bench. This is a Go implementation of the same philosophy.

## Design

### Bash only

Instead of implementing custom tools for every task (file reading, searching, git operations), the agent focuses on letting the LLM use the shell with limited access and resources, e.g., want it to open a PR? Don't implement a GitHub tool. Tell it to use `gh pr create`.

### Stateless Execution

Each command runs in a fresh bash process:

```go
exec.CommandContext(ctx, "bash", "-c", command)
```

No persistent shell session. This gives you:

- **Stability** — no shell state corruption between commands
- **Sandboxing** — easy to swap `exec.Command` with `docker exec`
- **Debugging** — each command is independent and reproducible

### Linear History

Every step appends to the message list. No branching, no complex state management. The trajectory *is* the conversation — great for debugging and understanding what the LLM sees.

## Installation

```bash
go install github.com/j0lvera/agent@latest
```

Or build from source:

```bash
git clone https://github.com/j0lvera/agent
cd agent
go build -o agent .
```

## Configuration

Only works with OpenRouter-compatible models. If you need support for other providers, please submit an pull request, but ideally, you'd fork it and make it your own.

### Environment Variables

```bash
export OPENROUTER_API_KEY="your-api-key"
export OPENROUTER_BASE_URL="https://openrouter.ai/api/v1"

# Optional settings with defaults
export MODEL="anthropic/claude-3.5-sonnet"   # Default model
export MAX_STEPS=25                          # Max iterations
export COMMAND_TIMEOUT=30s                   # Command timeout
export ENV=dev                               # dev, test, or prod
```

### Config File (config.toml)

Copy the example and customize:

```bash
cp config.example.toml config.toml
```

See [config.example.toml](config.example.toml) for all options.

## Usage

### Basic

```bash
agent run "List all Go files in this directory"
```

### From Stdin

```bash
echo "Create a file called hello.txt with 'Hello World'" | agent run -
```

### Flags

```bash
agent run "your task" -v    # Verbose (debug logging)
agent run "your task" -q    # Quiet (errors only)
agent run "your task" --json # JSON output
```

### Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | Error |
| 2 | Misuse (missing args, bad config) |
| 126 | Permission denied |
| 127 | Command not found |

## Safety

The agent blocks dangerous commands by default:

- `rm -rf /` and similar destructive patterns
- `mkfs`, `dd` to devices
- `curl | sh` patterns
- System commands (`shutdown`, `reboot`, `halt`)
- Fork bombs

See `agent/executor.go` for the full blocklist.

## License

MIT
