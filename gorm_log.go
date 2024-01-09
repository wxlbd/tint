package log

import (
	"context"
	"errors"
	"fmt"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/utils"
	"log/slog"
	"time"
)

func (l *Logger) LogMode(level logger.LogLevel) logger.Interface {
	l.gormLoggerConfig.LogLevel = level
	return l
}

func (l *Logger) Info(ctx context.Context, s string, i ...any) {
	l.Logger.InfoContext(ctx, s, i...)
}

func (l *Logger) Warn(ctx context.Context, s string, i ...any) {
	l.Logger.WarnContext(ctx, s, i...)
}

func (l *Logger) Error(ctx context.Context, s string, i ...any) {
	l.Logger.ErrorContext(ctx, s, i...)
}

func (l *Logger) Trace(ctx context.Context, begin time.Time, fc func() (sql string, rowsAffected int64), err error) {
	if l.gormLoggerConfig.LogLevel <= logger.Silent {
		return
	}
	elapsed := time.Since(begin)
	switch {
	case err != nil && l.gormLoggerConfig.LogLevel >= logger.Error && (!errors.Is(err, gorm.ErrRecordNotFound) || !l.gormLoggerConfig.IgnoreRecordNotFoundError):
		sql, rows := fc()
		if rows == -1 {
			l.Logger.ErrorContext(ctx, err.Error(), slog.String("line", utils.FileWithLineNum()), slog.String("elapsed", fmt.Sprintf("%.3fms", float64(elapsed.Nanoseconds())/1e6)), slog.String("sql", "-"))
		} else {
			l.Logger.ErrorContext(ctx, err.Error(), slog.String("line", utils.FileWithLineNum()), slog.String("elapsed", fmt.Sprintf("%.3fms", float64(elapsed.Nanoseconds())/1e6)), slog.String("sql", sql))
		}
	case elapsed > l.gormLoggerConfig.SlowThreshold && l.gormLoggerConfig.SlowThreshold != 0 && l.gormLoggerConfig.LogLevel >= logger.Warn:
		sql, rows := fc()
		slowLog := fmt.Sprintf("SLOW SQL >= %v", l.gormLoggerConfig.SlowThreshold)
		if rows == -1 {
			l.Logger.WarnContext(ctx, slowLog, slog.String("line", utils.FileWithLineNum()), slog.String("elapsed", fmt.Sprintf("%.3fms", float64(elapsed.Nanoseconds())/1e6)), slog.String("sql", "-"))
		} else {
			l.Logger.WarnContext(ctx, slowLog, slog.String("line", utils.FileWithLineNum()), slog.String("elapsed", fmt.Sprintf("%.3fms", float64(elapsed.Nanoseconds())/1e6)), slog.String("sql", sql))
		}
	case l.gormLoggerConfig.LogLevel == logger.Info:
		sql, rows := fc()
		if rows == -1 {
			l.Logger.InfoContext(ctx, "", slog.String("line", utils.FileWithLineNum()), slog.String("elapsed", fmt.Sprintf("%.3fms", float64(elapsed.Nanoseconds())/1e6)), slog.String("sql", "-"))
		} else {
			l.Logger.InfoContext(ctx, "", slog.String("line", utils.FileWithLineNum()), slog.String("elapsed", fmt.Sprintf("%.3fms", float64(elapsed.Nanoseconds())/1e6)), slog.String("sql", sql))
		}
	}
}
