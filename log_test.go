package log

import (
	"log/slog"
	"testing"
)

func TestNewLogger(t *testing.T) {
	logger := NewLogger("test")
	if logger == nil {
		t.Error("NewLogger should not return nil")
	}
}

func TestSetLevel(t *testing.T) {
	SetLevel("INFO")

	level := levelVar.Level()
	if level != slog.LevelInfo {
		t.Errorf("Expected level to be slog.Info, got %v", level)
	}
}
