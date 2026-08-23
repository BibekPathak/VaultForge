package core

import (
	"log/slog"
	"os"
	"testing"
)

func TestNewLogger_CreatesLogger(t *testing.T) {
	logger := NewLogger("development", "debug")
	if logger == nil {
		t.Fatal("logger should not be nil")
	}
	if logger.Logger == nil {
		t.Fatal("underlying slog.Logger should not be nil")
	}
}

func TestNewLogger_WithOutput(t *testing.T) {
	// Capture log output
	r, w, _ := os.Pipe()
	old := os.Stdout
	os.Stdout = w

	logger := NewLogger("development", "info")
	logger.Info("test message", "key", "value")

	w.Close()
	os.Stdout = old

	buf := make([]byte, 1024)
	n, _ := r.Read(buf)
	output := string(buf[:n])

	if len(output) == 0 {
		t.Error("expected log output")
	}
}

func TestLogger_Request(t *testing.T) {
	logger := NewLogger("development", "debug")
	reqLogger := logger.Request("req-1", "GET", "/v1/intents", "127.0.0.1")
	if reqLogger == nil {
		t.Fatal("request logger should not be nil")
	}
}

func TestLogger_Intent(t *testing.T) {
	logger := NewLogger("development", "debug")
	intentLogger := logger.Intent("req-1", "intent-1", "tenant-1", "user-1")
	if intentLogger == nil {
		t.Fatal("intent logger should not be nil")
	}
}

func TestNewLogger_LogLevels(t *testing.T) {
	levels := []string{"debug", "info", "warn", "error"}
	for _, level := range levels {
		l := NewLogger("development", level)
		if l == nil {
			t.Errorf("logger should not be nil for level %s", level)
		}
	}
}

func TestNewLogger_DefaultLevel(t *testing.T) {
	l := NewLogger("development", "invalid")
	if l == nil {
		t.Fatal("logger should not be nil for invalid level (should default to info)")
	}
}

func TestLogger_HandlerLevel(t *testing.T) {
	// Verify debug logger is at debug level
	l := NewLogger("development", "debug")
	if l.Logger.Enabled(nil, slog.LevelDebug) != true {
		t.Error("debug logger should enable debug level")
	}

	// Verify info logger does NOT enable debug level
	l = NewLogger("development", "info")
	if l.Logger.Enabled(nil, slog.LevelDebug) != false {
		t.Error("info logger should not enable debug level")
	}
}
