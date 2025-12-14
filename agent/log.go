package agent

import (
	"os"
	"time"

	"github.com/rs/zerolog"
)

// isDevelopmentEnv returns true if running in development or test mode.
// Returns true for ENV values: "dev", "test", or empty string.
// Production environments should set ENV explicitly (e.g., "prod").
func isDevelopmentEnv() bool {
	env := os.Getenv("ENV")
	return env == "dev" || env == "test" || env == ""
}

// defaultLogLevel returns the default log level based on environment.
// Returns debug in development/test, info in production.
func defaultLogLevel() zerolog.Level {
	if isDevelopmentEnv() {
		return zerolog.DebugLevel
	}
	return zerolog.InfoLevel
}

// NewLogger creates a new zerolog logger with the specified level.
//
// Log Level Configuration:
//   - Accepts: trace, debug, info, warn, error, fatal, panic
//   - Invalid values trigger a warning and fall back to environment defaults
//   - Empty string: defaults based on ENV (debug for dev/test, info for prod)
//
// Output Format:
//   - Development/test mode (ENV=dev, ENV=test, or unset): Console writer with colors
//   - Production mode: JSON output to stderr
func NewLogger(logLevel string) zerolog.Logger {
	// Determine log level using zerolog.ParseLevel
	var level zerolog.Level
	if logLevel != "" {
		parsedLevel, err := zerolog.ParseLevel(logLevel)
		if err != nil {
			level = defaultLogLevel()
			tempLogger := zerolog.New(os.Stderr).With().Timestamp().Logger()
			tempLogger.Warn().
				Err(err).
				Str("provided", logLevel).
				Str("fallback", level.String()).
				Msg("unrecognized log level; valid: trace, debug, info, warn, error, fatal, panic")
		} else {
			level = parsedLevel
		}
	} else {
		level = defaultLogLevel()
	}

	// Development/test mode: pretty console output with caller info
	if isDevelopmentEnv() {
		writer := zerolog.ConsoleWriter{
			Out:        os.Stderr,
			TimeFormat: time.RFC3339,
		}
		return zerolog.New(writer).Level(level).With().Timestamp().Caller().Logger()
	}

	// Production mode: JSON output
	return zerolog.New(os.Stderr).Level(level).With().Timestamp().Logger()
}
