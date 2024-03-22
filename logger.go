package tint

import (
	"context"
	"fmt"
	"github.com/go-kratos/kratos/v2/log"
	"gorm.io/gorm/logger"
	"io"
	"log/slog"
	"runtime"
	"time"
)

var (
	_ logger.Interface = (*Logger)(nil)
	_ log.Logger       = (*Logger)(nil)
)

type Logger struct {
	*slog.Logger
	*Handler
}

func NewLogger(writer io.Writer, level slog.Level) *Logger {
	h := NewHandler(writer, &Options{
		TimeFormat: defaultTimeFormat,
		Level:      level,
	})
	return &Logger{
		Logger:  slog.New(h),
		Handler: h,
	}
}

func (h *Logger) Log(level log.Level, keyAndValues ...any) error {
	var pcs [1]uintptr
	runtime.Callers(4, pcs[:])
	pc := pcs[0]
	var r slog.Record
	switch level {
	case log.LevelDebug:
		r = slog.NewRecord(time.Now(), slog.LevelDebug, "", pc)
		r.Add(keyAndValues...)
	case log.LevelInfo:
		r = slog.NewRecord(time.Now(), slog.LevelInfo, "", pc)
		r.Add(keyAndValues...)
	case log.LevelWarn:
		r = slog.NewRecord(time.Now(), slog.LevelWarn, "", pc)
		r.Add(keyAndValues...)
	case log.LevelError:
		r = slog.NewRecord(time.Now(), slog.LevelError, "", pc)
		r.Add(keyAndValues...)
	case log.LevelFatal:
		r = slog.NewRecord(time.Now(), slog.LevelError, "", pc)
		r.Add(keyAndValues...)
	}
	return h.Handle(context.TODO(), r)
}
func (h *Logger) LogMode(_ logger.LogLevel) logger.Interface {
	return h
}

func (h *Logger) Info(ctx context.Context, s string, i ...any) {
	if h.Handler.Enabled(ctx, slog.LevelInfo) {
		var pcs [1]uintptr
		runtime.Callers(4, pcs[:])
		pc := pcs[0]
		r := slog.NewRecord(time.Now(), slog.LevelInfo, "", pc)
		r.AddAttrs(slog.String("msg", s))
		r.Add(i...)
		_ = h.Handle(ctx, r)
	}
}

func (h *Logger) Warn(ctx context.Context, s string, i ...interface{}) {
	if h.Handler.Enabled(ctx, slog.LevelWarn) {
		var pcs [1]uintptr
		runtime.Callers(4, pcs[:])
		pc := pcs[0]
		r := slog.NewRecord(time.Now(), slog.LevelInfo, "", pc)
		r.AddAttrs(slog.String("msg", s))
		r.Add(i...)
		_ = h.Handle(ctx, r)
	}
}

func (h *Logger) Error(ctx context.Context, s string, i ...interface{}) {
	if h.Handler.Enabled(ctx, slog.LevelError) {
		var pcs [1]uintptr
		runtime.Callers(4, pcs[:])
		pc := pcs[0]
		r := slog.NewRecord(time.Now(), slog.LevelInfo, "", pc)
		r.AddAttrs(slog.String("msg", s))
		r.Add(i...)
		_ = h.Handle(ctx, r)
	}
}

func (h *Logger) Trace(ctx context.Context, begin time.Time, fc func() (sql string, rowsAffected int64), err error) {
	if h.Handler.Enabled(ctx, slog.LevelInfo) {
		var pcs [1]uintptr
		runtime.Callers(4, pcs[:])
		pc := pcs[0]
		r := slog.NewRecord(time.Now(), slog.LevelInfo, "", pc)
		sql, rows := fc()
		elapsed := time.Since(begin)
		if err != nil {
			r.AddAttrs(Err(err))
		}
		if rows == -1 {
			r.AddAttrs(
				slog.String("time", fmt.Sprintf("%.3fms", float64(elapsed.Nanoseconds())/1e6)),
				slog.String("sql", "-"),
			)
		} else {
			r.AddAttrs(
				slog.String("time", fmt.Sprintf("%.3fms", float64(elapsed.Nanoseconds())/1e6)),
				slog.String("sql", sql),
			)
		}
		_ = h.Handle(ctx, r)
	}
}
