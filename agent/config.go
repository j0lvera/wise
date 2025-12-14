package agent

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/kelseyhightower/envconfig"
)

// EnvConfig holds environment-specific configuration loaded from env vars.
// These are typically sensitive or deployment-specific values.
type EnvConfig struct {
	// API settings
	APIKey  string `envconfig:"OPENROUTER_API_KEY" required:"true"`
	BaseURL string `envconfig:"OPENROUTER_BASE_URL" required:"true"`

	// Runtime settings
	Env   string `envconfig:"ENV" default:"dev"`
	Model string `envconfig:"MODEL" default:"anthropic/claude-3.5-sonnet"`

	// Limits
	MaxSteps       int           `envconfig:"MAX_STEPS" default:"25"`
	CommandTimeout time.Duration `envconfig:"COMMAND_TIMEOUT" default:"30s"`
}

// LoadEnv loads configuration from environment variables.
func LoadEnv() (*EnvConfig, error) {
	var cfg EnvConfig
	if err := envconfig.Process("", &cfg); err != nil {
		return nil, fmt.Errorf("failed to load env config: %w", err)
	}
	return &cfg, nil
}

// UserConfig holds user-facing configuration loaded from TOML.
// These are non-sensitive values like prompts and templates.
type UserConfig struct {
	// Prompts
	SystemPrompt string `toml:"system_prompt"`
	UserPrompt   string `toml:"user_prompt"`

	// Optional overrides (env vars take precedence if set)
	Model          string `toml:"model"`
	MaxSteps       int    `toml:"max_steps"`
	CommandTimeout string `toml:"command_timeout"`

	// Working directory for command execution
	WorkingDir string `toml:"working_dir"`
}

// LoadUserConfig reads user configuration from a TOML file.
func LoadUserConfig(dir string) (*UserConfig, error) {
	var cfg UserConfig

	configPath := filepath.Join(dir, "config.toml")

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("config file not found: %s", configPath)
	}

	_, err := toml.DecodeFile(configPath, &cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to load user config: %w", err)
	}

	return &cfg, nil
}

// Config is the combined configuration used by the agent.
type Config struct {
	// From EnvConfig
	APIKey  string
	BaseURL string
	Env     string
	Model   string

	// Limits
	MaxSteps       int
	CommandTimeout time.Duration

	// From UserConfig
	SystemPrompt string
	UserPrompt   string
	WorkingDir   string

	// Output writer for streaming results (optional, defaults to io.Discard)
	Output io.Writer

	// LogLevel sets the logging verbosity (trace, debug, info, warn, error)
	// If empty, defaults based on Env (debug for dev/test, info for prod)
	LogLevel string
}

// Validate checks that required configuration fields are present.
func (c *Config) Validate() error {
	if c.APIKey == "" {
		return fmt.Errorf("OPENROUTER_API_KEY is required")
	}
	if c.BaseURL == "" {
		return fmt.Errorf("OPENROUTER_BASE_URL is required")
	}
	if strings.TrimSpace(c.SystemPrompt) == "" {
		return fmt.Errorf("system_prompt is required in config.toml")
	}
	if strings.TrimSpace(c.UserPrompt) == "" {
		return fmt.Errorf("user_prompt is required in config.toml")
	}
	if c.WorkingDir != "" {
		if _, err := os.Stat(c.WorkingDir); os.IsNotExist(err) {
			return fmt.Errorf("working_dir does not exist: %s", c.WorkingDir)
		}
	}
	return nil
}

// LoadConfig loads and merges configuration from env vars and TOML file.
// Environment variables take precedence over TOML values for shared fields.
func LoadConfig(dir string) (*Config, error) {
	// Load env config
	envCfg, err := LoadEnv()
	if err != nil {
		return nil, err
	}

	// Load user config from TOML
	userCfg, err := LoadUserConfig(dir)
	if err != nil {
		return nil, err
	}

	// Merge configs (env takes precedence)
	cfg := &Config{
		// From env (required)
		APIKey:  envCfg.APIKey,
		BaseURL: envCfg.BaseURL,
		Env:     envCfg.Env,

		// From env with TOML fallback
		Model:          envCfg.Model,
		MaxSteps:       envCfg.MaxSteps,
		CommandTimeout: envCfg.CommandTimeout,

		// From TOML only
		SystemPrompt: userCfg.SystemPrompt,
		UserPrompt:   userCfg.UserPrompt,
		WorkingDir:   userCfg.WorkingDir,
	}

	// Allow TOML to override defaults if env uses defaults
	if userCfg.Model != "" && envCfg.Model == "anthropic/claude-3.5-sonnet" {
		cfg.Model = userCfg.Model
	}
	if userCfg.MaxSteps > 0 && envCfg.MaxSteps == 25 {
		cfg.MaxSteps = userCfg.MaxSteps
	}
	if userCfg.CommandTimeout != "" && envCfg.CommandTimeout == 30*time.Second {
		d, err := time.ParseDuration(userCfg.CommandTimeout)
		if err != nil {
			return nil, fmt.Errorf("invalid command_timeout in config.toml: %w", err)
		}
		cfg.CommandTimeout = d
	}

	// Set default output writer
	cfg.Output = io.Discard

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return cfg, nil
}
