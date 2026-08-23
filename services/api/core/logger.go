package core

import (
	"log/slog"
	"os"
	"time"
)

// Logger wraps slog with structured fields for VaultForge.
type Logger struct {
	*slog.Logger
}

// NewLogger creates a structured JSON logger configured for the given environment.
func NewLogger(env, level string) *Logger {
	var lvl slog.Level
	switch level {
	case "debug":
		lvl = slog.LevelDebug
	case "warn":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		lvl = slog.LevelInfo
	}

	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level:     lvl,
		AddSource: env == "development",
	})

	return &Logger{
		Logger: slog.New(handler).With(
			"service", "vaultforge-api",
			"environment", env,
		),
	}
}

// Request returns a logger enriched with request-scoped fields.
func (l *Logger) Request(requestID, method, path, clientIP string) *slog.Logger {
	return l.Logger.With(
		"request_id", requestID,
		"method", method,
		"path", path,
		"client_ip", clientIP,
	)
}

// Intent returns a logger enriched with intent-scoped fields.
func (l *Logger) Intent(requestID, intentID, tenantID, actor string) *slog.Logger {
	return l.Logger.With(
		"request_id", requestID,
		"intent_id", intentID,
		"tenant_id", tenantID,
		"actor", actor,
	)
}

// LogRequestStart logs the beginning of an HTTP request.
func (l *Logger) LogRequestStart(requestID, method, path string) {
	l.Logger.Info("request started",
		"request_id", requestID,
		"method", method,
		"path", path,
		"timestamp", time.Now().UTC().Format(time.RFC3339Nano),
	)
}

// LogRequestComplete logs the completion of an HTTP request.
func (l *Logger) LogRequestComplete(requestID, method, path string, status int, duration time.Duration) {
	l.Logger.Info("request completed",
		"request_id", requestID,
		"method", method,
		"path", path,
		"status", status,
		"duration_ms", duration.Milliseconds(),
		"timestamp", time.Now().UTC().Format(time.RFC3339Nano),
	)
}
