package observability

import (
	"context"
	"io"
	"os"
	"strings"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/vihan/taskorchestrator/internal/infrastructure/config"
)

const TraceIDKey = "trace_id"

type Logger struct {
	*zerolog.Logger
}

// NewLogger creates a new structured logger based on configuration
func NewLogger(cfg *config.ObservabilityConfig) *Logger {
	var output io.Writer = os.Stdout

	// Configure zerolog
	logLevel := parseLogLevel(cfg.LogLevel)
	zerolog.SetGlobalLevel(logLevel)

	// Format output
	if cfg.LogFormat == "text" {
		output = zerolog.ConsoleWriter{Out: os.Stdout}
	}

	logger := zerolog.New(output).
		Level(logLevel).
		With().
		Timestamp().
		Logger()

	return &Logger{Logger: &logger}
}

// WithTraceID returns a new logger with trace ID attached
func (l *Logger) WithTraceID(ctx context.Context, traceID string) *Logger {
	logger := l.With().Str(TraceIDKey, traceID).Logger()
	return &Logger{Logger: &logger}
}

// WithOrchestrationID returns a new logger with orchestration ID
func (l *Logger) WithOrchestrationID(orchestrationID string) *Logger {
	logger := l.With().Str("orchestration_id", orchestrationID).Logger()
	return &Logger{Logger: &logger}
}

// WithActivityName returns a new logger with activity name
func (l *Logger) WithActivityName(activityName string) *Logger {
	logger := l.With().Str("activity", activityName).Logger()
	return &Logger{Logger: &logger}
}

// WithError returns a new logger with error attached
func (l *Logger) WithError(err error) *Logger {
	logger := l.With().Err(err).Logger()
	return &Logger{Logger: &logger}
}

// Info logs an info level message
func (l *Logger) Info(msg string, fields ...interface{}) {
	l.Logger.Info().Msgf(msg)
}

// Error logs an error level message
func (l *Logger) Error(msg string, err error, fields ...interface{}) {
	l.Logger.Error().Err(err).Msg(msg)
}

// Debug logs a debug level message
func (l *Logger) Debug(msg string, fields ...interface{}) {
	l.Logger.Debug().Msgf(msg)
}

// parseLogLevel converts string to zerolog level
func parseLogLevel(levelStr string) zerolog.Level {
	switch strings.ToLower(levelStr) {
	case "debug":
		return zerolog.DebugLevel
	case "info":
		return zerolog.InfoLevel
	case "warn":
		return zerolog.WarnLevel
	case "error":
		return zerolog.ErrorLevel
	case "fatal":
		return zerolog.FatalLevel
	default:
		return zerolog.InfoLevel
	}
}

// GetGlobalLogger returns the global logger
func GetGlobalLogger() *Logger {
	logger := log.Logger
	return &Logger{Logger: &logger}
}
