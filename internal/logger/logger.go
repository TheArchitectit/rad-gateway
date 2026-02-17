// Package logger provides structured logging for RAD Gateway.
// All modules should use this package for consistent logging.
package logger

import (
	"log/slog"
	"os"
	"sync"
)

var (
	instance *slog.Logger
	once     sync.Once
)

// Config holds logger configuration
// Can be loaded from Infisical for production
type Config struct {
	Level     string `json:"level"`     // debug, info, warn, error
	Format    string `json:"format"`    // json, text
	Output    string `json:"output"`    // stdout, file path
	AddSource bool   `json:"addSource"` // include source file/line
}

// DefaultConfig returns sensible defaults
func DefaultConfig() Config {
	return Config{
		Level:     getEnv("LOG_LEVEL", "info"),
		Format:    getEnv("LOG_FORMAT", "json"),
		Output:    getEnv("LOG_OUTPUT", "stdout"),
		AddSource: getEnv("LOG_SOURCE", "false") == "true",
	}
}

// Init initializes the global logger
func Init(cfg Config) {
	once.Do(func() {
		var level slog.Level
		switch cfg.Level {
		case "debug":
			level = slog.LevelDebug
		case "info":
			level = slog.LevelInfo
		case "warn":
			level = slog.LevelWarn
		case "error":
			level = slog.LevelError
		default:
			level = slog.LevelInfo
		}

		opts := &slog.HandlerOptions{
			Level:     level,
			AddSource: cfg.AddSource,
		}

		var handler slog.Handler
		if cfg.Format == "json" {
			handler = slog.NewJSONHandler(os.Stdout, opts)
		} else {
			handler = slog.NewTextHandler(os.Stdout, opts)
		}

		instance = slog.New(handler)
		slog.SetDefault(instance)
	})
}

// Get returns the global logger instance
func Get() *slog.Logger {
	if instance == nil {
		Init(DefaultConfig())
	}
	return instance
}

// Debug logs a debug message
func Debug(msg string, args ...any) {
	Get().Debug(msg, args...)
}

// Info logs an info message
func Info(msg string, args ...any) {
	Get().Info(msg, args...)
}

// Warn logs a warning message
func Warn(msg string, args ...any) {
	Get().Warn(msg, args...)
}

// Error logs an error message
func Error(msg string, err error, args ...any) {
	if err != nil {
		args = append(args, "error", err.Error())
	}
	Get().Error(msg, args...)
}

// WithComponent returns a logger with component attribute
func WithComponent(component string) *slog.Logger {
	return Get().With("component", component)
}

// WithRequestID returns a logger with request ID
func WithRequestID(requestID string) *slog.Logger {
	return Get().With("request_id", requestID)
}

// Helper function
func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}
