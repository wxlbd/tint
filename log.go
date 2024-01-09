package log

import (
	"fmt"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/lmittmann/tint"
	"gopkg.in/natefinch/lumberjack.v2"
	"gorm.io/gorm/logger"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var levelVar = new(slog.LevelVar)

var Level = map[string]slog.Level{
	"INFO":  slog.LevelInfo,
	"WARN":  slog.LevelWarn,
	"DEBUG": slog.LevelDebug,
	"ERROR": slog.LevelError,
}

type Options struct {
	FilenamePrefix string // 日志文件前缀，文件名为 {FilenamePrefix}_{time}.log
	Level          string
	Filepath       string // 日志文件存放路径
	OutputType     string // 日志消息输出类型，“控制台”或“文件”
	MaxSize        int    // log file max size, MB
	MaxBackups     int    // log file max backups
	MaxAge         int    // log file max age, days
	Compress       bool   // log file compress
}

type Option func(*Options)

// WithLevel 设置日志级别 "INFO", "WARN", "DEBUG", "ERROR"
func WithLevel(level string) Option {
	return func(options *Options) {
		options.Level = level
	}
}

// WithOutputType 日志消息输出类型，“console”或“file”
func WithOutputType(outputType string) Option {
	return func(options *Options) {
		options.OutputType = outputType
	}
}

// WithMaxSize 日志文件最大值,单位为 MB
func WithMaxSize(maxSize int) Option {
	return func(options *Options) {
		options.MaxSize = maxSize
	}
}

// WithMaxBackups 日志文件最大备份数
func WithMaxBackups(maxBackups int) Option {
	return func(options *Options) {
		options.MaxBackups = maxBackups
	}
}

// WithMaxAge 日志文件最大保存天数
func WithMaxAge(maxAge int) Option {
	return func(options *Options) {
		options.MaxAge = maxAge
	}
}

// WithCompress 日志备份是否压缩
func WithCompress(compress bool) Option {
	return func(options *Options) {
		options.Compress = compress
	}
}

var defaultOptions = &Options{
	Level:      "INFO",
	OutputType: "console",
	MaxSize:    1,
	MaxBackups: 1,
	MaxAge:     1,
	Compress:   true,
}

type Logger struct {
	*slog.Logger
	gormLoggerConfig logger.Config
}

func (l *Logger) Log(level log.Level, args ...any) error {
	switch level {
	case log.LevelDebug:
		l.Logger.Debug("", args...)
	case log.LevelWarn:
		l.Logger.Warn("", args...)
	case log.LevelError:
		l.Logger.Error("", args...)
	case log.LevelInfo:
		l.Logger.Info("", args...)
	default:
		l.Logger.Error("", args...)
	}
	return nil
}

// NewLogger 实例化日志
func NewLogger(opts ...Option) *Logger {
	options := defaultOptions
	for _, opt := range opts {
		opt(options)
	}
	// 初始化日志级别
	var noColor bool
	// 初始化日志级别
	levelVar.Set(Level[strings.ToUpper(options.Level)])
	var writer io.Writer
	if options.OutputType == "file" {
		writer = &lumberjack.Logger{
			Filename:   filepath.Join(filepath.Clean(options.Filepath), fmt.Sprintf("%s_%s.log", options.FilenamePrefix, time.Now().Format(time.DateOnly))),
			MaxSize:    options.MaxSize,    // 文件大小限制,单位MB
			MaxBackups: options.MaxBackups, // 最大保留日志文件数量
			MaxAge:     options.MaxAge,     // 日志文件保留天数
			Compress:   options.Compress,   // 是否压缩处理
		}
		noColor = true
	} else {
		writer = os.Stdout
	}

	l := slog.New(tint.NewHandler(writer, &tint.Options{
		Level:      levelVar,
		TimeFormat: time.DateTime,
		NoColor:    noColor,
		AddSource:  true,
	}))

	return &Logger{Logger: l}
}

// SetLevel 设置日志级别
func SetLevel(level string) {
	levelVar.Set(Level[level])
}
