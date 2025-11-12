package logger

import (
	"context"
	"log/slog"
	"os"
	"strings"
)

// LogLevel represents the logging level
type LogLevel string

const (
	LevelDebug LogLevel = "debug"
	LevelInfo  LogLevel = "info"
	LevelWarn  LogLevel = "warn"
	LevelError LogLevel = "error"
)

// Config holds logger configuration
type Config struct {
	Level  LogLevel `json:"level"`
	Format string   `json:"format"` // "json" or "text"
}

// Logger wraps slog.Logger with additional functionality
type Logger struct {
	*slog.Logger
}

// Global logger instance
var defaultLogger *Logger

// Init initializes the global logger
func Init(config Config) {
	var level slog.Level
	switch strings.ToLower(string(config.Level)) {
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

	var handler slog.Handler
	if config.Format == "json" {
		handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: level,
		})
	} else {
		handler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: level,
		})
	}

	defaultLogger = &Logger{
		Logger: slog.New(handler),
	}
}

// GetLogger returns the global logger instance
func GetLogger() *Logger {
	if defaultLogger == nil {
		// Initialize with default config if not initialized
		Init(Config{
			Level:  LevelInfo,
			Format: "text",
		})
	}
	return defaultLogger
}

// Debug logs at debug level
func Debug(msg string, args ...any) {
	GetLogger().Debug(msg, args...)
}

// Info logs at info level
func Info(msg string, args ...any) {
	GetLogger().Info(msg, args...)
}

// Warn logs at warn level
func Warn(msg string, args ...any) {
	GetLogger().Warn(msg, args...)
}

// Error logs at error level
func Error(msg string, args ...any) {
	GetLogger().Error(msg, args...)
}

// WithContext returns a logger with context
func WithContext(ctx context.Context) *Logger {
	return &Logger{
		Logger: GetLogger().Logger.With(),
	}
}

// WithFields returns a logger with additional fields
func WithFields(fields map[string]any) *Logger {
	var args []any
	for k, v := range fields {
		args = append(args, k, v)
	}
	return &Logger{
		Logger: GetLogger().Logger.With(args...),
	}
}

// WithComponent returns a logger with component field
func WithComponent(component string) *Logger {
	return &Logger{
		Logger: GetLogger().Logger.With("component", component),
	}
}
