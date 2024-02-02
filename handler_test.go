package tint

import (
	"errors"
	"github.com/go-kratos/kratos/v2/log"
	"log/slog"
	"os"
	"testing"
	"time"
)

func Example() {
	handler := NewHandler(os.Stderr, &Options{
		Level:      slog.LevelDebug,
		TimeFormat: time.DateTime,
	})
	slog.SetDefault(slog.New(handler))

	slog.Info("Starting server", "addr", ":8080", "env", "production")
	slog.Debug("Connected to DB", "db", "myapp", "host", "localhost:5432")
	slog.Warn("Slow request", "method", "GET", "path", "/users", "duration", 497*time.Millisecond)
	slog.Error("DB connection lost", Err(errors.New("connection reset")), "db", "myapp")
	// Output:
}

func TestHandler_Log(t *testing.T) {

	handler := NewHandler(os.Stderr, &Options{
		Level:      slog.LevelDebug,
		TimeFormat: time.DateTime,
	})
	handler.Log(log.LevelDebug, "msg", "Starting server", "addr", ":8080", "env", "production")
}
