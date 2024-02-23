/*
Package tint 实现了一个 [slog.Handler]，可以写入着色（colorized）的日志。
输出格式受 [zerolog.ConsoleWriter] 和 [slog.TextHandler] 的启发。

可以通过 [Config] 自定义输出格式，它是[slog.HandlerOptions]的直接替代品。

# 定制属性

可以在写入之前使用 Config.ReplaceAttr 来修改或删除属性。
如果设置了该属性，将在每个非组属性上调用它。
详情请参阅 [slog.HandlerOptions]。

	w := os.Stderr
	logger := slog.New(
		tint.NewHandler(w, &tint.Config{
			ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
				if a.Key == slog.TimeKey && len(groups) == 0 {
					return slog.Attr{}
				}
				return a
			},
		}),
	)

# 自动启用颜色

默认情况下启用了颜色，可以使用 Config.NoColor 属性禁用颜色。
要根据终端功能自动启用颜色，请使用例如 [go-isatty] 包。

	w := os.Stderr
	logger := slog.New(
		tint.NewHandler(w, &tint.Config{
			NoColor: !isatty.IsTerminal(w.Fd()),
		}),
	)
*/
package tint

import (
	"context"
	"encoding"
	"fmt"
	"github.com/go-kratos/kratos/v2/log"
	"gorm.io/gorm/logger"
	"io"
	"log/slog"
	"path/filepath"
	"runtime"
	"strconv"
	"sync"
	"time"
	"unicode"
)

// ANSI modes
const (
	ansiReset          = "\033[0m"    // 重置文本属性为默认颜色
	ansiFaint          = "\033[2m"    // 设置文本为虚幻颜色
	ansiResetFaint     = "\033[22m"   // 重置文本属性为默认虚幻颜色
	ansiBrightRed      = "\033[91m"   // 设置文本为亮红色
	ansiBrightGreen    = "\033[92m"   // 设置文本为亮绿色
	ansiBrightYellow   = "\033[93m"   // 设置文本为亮黄色
	ansiBrightRedFaint = "\033[91;2m" // 设置虚幻的亮红色文本
	ansiBrightBlue     = "\033[34;1m" // 设置文本为亮蓝色
)

const errKey = "err"

var (
	defaultLevel      = slog.LevelInfo
	defaultTimeFormat = time.DateTime
)

// Options 写有染色日志的slog.Handler的选项。零值Options完全由默认值组成。
//
// 选项可以作为[slog.HandlerOptions]的drop-in替代品使用。
type Options struct {
	// 启用源代码位置（默认值：false）
	AddSource bool

	// 要记录的最低级别（默认值：slog.LevelInfo）
	Level slog.Leveler

	// 在记录之前，调用ReplaceAttr重写每个非组属性。
	// 详情请参考https://pkg.go.dev/log/slog#HandlerOptions。
	ReplaceAttr func(groups []string, attr slog.Attr) slog.Attr

	// 时间格式（默认值：time.DateTime）
	TimeFormat string

	// 禁用颜色（默认值：false）
	NoColor bool

	// 跳过栈帧数（默认值：4）
	Skip int
}

// NewHandler 使用默认选项将彩色日志写入Writer w的[slog.Handler]。如果opts为nil，则使用默认选项。
func NewHandler(w io.Writer, opts *Options) *Handler {
	h := &Handler{
		w:          w,
		level:      defaultLevel,
		timeFormat: defaultTimeFormat,
	}
	if opts == nil {
		return h
	}

	// 设置添加源
	h.addSource = opts.AddSource
	// 设置级别
	if opts.Level != nil {
		h.level = opts.Level
	}
	// 设置替换属性
	h.replaceAttr = opts.ReplaceAttr
	// 设置时间格式
	if opts.TimeFormat != "" {
		h.timeFormat = opts.TimeFormat
	}
	// 设置是否不使用颜色
	h.noColor = opts.NoColor

	// 设置跳过行数
	if opts.Skip > 0 {
		h.skip = opts.Skip
	} else {
		h.skip = 4
	}
	return h
}

// Handler 实现了 [slog.Handler] 接口.
type Handler struct {
	attrsPrefix string
	groupPrefix string
	groups      []string

	mu sync.Mutex
	w  io.Writer

	addSource   bool
	level       slog.Leveler
	replaceAttr func([]string, slog.Attr) slog.Attr
	timeFormat  string
	noColor     bool
	skip        int
}

func (h *Handler) Log(level log.Level, keyvals ...any) error {
	var pcs [1]uintptr
	runtime.Callers(4, pcs[:])
	pc := pcs[0]
	var r slog.Record
	switch level {
	case log.LevelDebug:
		r = slog.NewRecord(time.Now(), slog.LevelDebug, "", pc)
		r.Add(keyvals...)
	case log.LevelInfo:
		r = slog.NewRecord(time.Now(), slog.LevelInfo, "", pc)
		r.Add(keyvals...)
	case log.LevelWarn:
		r = slog.NewRecord(time.Now(), slog.LevelWarn, "", pc)
		r.Add(keyvals...)
	case log.LevelError:
		r = slog.NewRecord(time.Now(), slog.LevelError, "", pc)
		r.Add(keyvals...)
	case log.LevelFatal:
		r = slog.NewRecord(time.Now(), slog.LevelError, "", pc)
		r.Add(keyvals...)
	}
	return h.Handle(context.TODO(), r)
}

func (h *Handler) LogMode(_ logger.LogLevel) logger.Interface {
	return h
}

func (h *Handler) Info(ctx context.Context, s string, i ...any) {
	if h.Enabled(ctx, slog.LevelInfo) {
		var pcs [1]uintptr
		runtime.Callers(4, pcs[:])
		pc := pcs[0]
		r := slog.NewRecord(time.Now(), slog.LevelInfo, "", pc)
		r.AddAttrs(slog.String("msg", s))
		r.Add(i...)
		_ = h.Handle(ctx, r)
	}
}

func (h *Handler) Warn(ctx context.Context, s string, i ...interface{}) {
	if h.Enabled(ctx, slog.LevelWarn) {
		var pcs [1]uintptr
		runtime.Callers(4, pcs[:])
		pc := pcs[0]
		r := slog.NewRecord(time.Now(), slog.LevelInfo, "", pc)
		r.AddAttrs(slog.String("msg", s))
		r.Add(i...)
		_ = h.Handle(ctx, r)
	}
}

func (h *Handler) Error(ctx context.Context, s string, i ...interface{}) {
	if h.Enabled(ctx, slog.LevelError) {
		var pcs [1]uintptr
		runtime.Callers(4, pcs[:])
		pc := pcs[0]
		r := slog.NewRecord(time.Now(), slog.LevelInfo, "", pc)
		r.AddAttrs(slog.String("msg", s))
		r.Add(i...)
		_ = h.Handle(ctx, r)
	}
}

func (h *Handler) Trace(ctx context.Context, begin time.Time, fc func() (sql string, rowsAffected int64), err error) {
	if h.Enabled(ctx, slog.LevelInfo) {
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

func (h *Handler) clone() *Handler {
	return &Handler{
		attrsPrefix: h.attrsPrefix,
		groupPrefix: h.groupPrefix,
		groups:      h.groups,
		w:           h.w,
		addSource:   h.addSource,
		level:       h.level,
		replaceAttr: h.replaceAttr,
		timeFormat:  h.timeFormat,
		noColor:     h.noColor,
	}
}

// Enabled 函数用于检查日志级别是否在处理器中启用
func (h *Handler) Enabled(_ context.Context, level slog.Level) bool {
	// 检查日志级别是否大于或等于处理器的日志级别
	return level >= h.level.Level()
}

// Handle 处理记录并将其写入日志。
func (h *Handler) Handle(_ context.Context, r slog.Record) error {
	// 从同步池中获取一个缓冲区，处理完毕后返回给池。
	buf := newBuffer()
	defer buf.Free()

	// 获取替换属性
	rep := h.replaceAttr

	// 写入时间
	if !r.Time.IsZero() {
		// 去除单调时间，以匹配Attr的行为
		val := r.Time.Round(0)
		if rep == nil {
			h.appendTime(buf, r.Time)
			buf.WriteByte(' ')
		} else if a := rep(nil /* groups */, slog.Time(slog.TimeKey, val)); a.Key != "" {
			if a.Value.Kind() == slog.KindTime {
				h.appendTime(buf, a.Value.Time())
			} else {
				h.appendValue(buf, a.Value, false)
			}
			buf.WriteByte(' ')
		}
	}

	// 写入级别
	if rep == nil {
		h.appendLevel(buf, r.Level)
		buf.WriteByte(' ')
	} else if a := rep(nil /* groups */, slog.Any(slog.LevelKey, r.Level)); a.Key != "" {
		h.appendValue(buf, a.Value, false)
		buf.WriteByte(' ')
	}

	// 写入源代码文件位置
	if h.addSource {
		fs := runtime.CallersFrames([]uintptr{r.PC})
		f, _ := fs.Next()
		if f.File != "" {
			src := &slog.Source{
				Function: f.Function,
				File:     f.File,
				Line:     f.Line,
			}

			if rep == nil {
				h.appendSource(buf, src)
				buf.WriteByte(' ')
			} else if a := rep(nil /* groups */, slog.Any(slog.SourceKey, src)); a.Key != "" {
				h.appendValue(buf, a.Value, false)
				buf.WriteByte(' ')
			}
		}
	}

	// 写入消息
	if rep == nil {
		buf.WriteString(r.Message)
		buf.WriteByte(' ')
	} else if a := rep(nil /* groups */, slog.String(slog.MessageKey, r.Message)); a.Key != "" {
		h.appendValue(buf, a.Value, false)
		buf.WriteByte(' ')
	}

	// 写入处理程序的属性
	if len(h.attrsPrefix) > 0 {
		buf.WriteString(h.attrsPrefix)
	}

	// 写入属性
	r.Attrs(func(attr slog.Attr) bool {
		h.appendAttr(buf, attr, h.groupPrefix, h.groups)
		return true
	})

	if len(*buf) == 0 {
		return nil
	}
	(*buf)[len(*buf)-1] = '\n' // 将最后一个空格替换为换行符

	// 写入日志
	h.mu.Lock()
	defer h.mu.Unlock()

	_, err := h.w.Write(*buf)
	return err
}

// WithAttrs 函数为handler类型的一个方法，用于返回一个具有指定属性的新的handler实例。
func (h *Handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	// 如果属性列表为空，则直接返回原始的handler实例
	if len(attrs) == 0 {
		return h
	}

	// 克隆原始的handler实例并赋值给新的变量h2
	h2 := h.clone()

	// 创建一个新的缓冲区，并在函数结束时释放缓冲区
	buf := newBuffer()
	defer buf.Free()

	// 将属性写入缓冲区
	for _, attr := range attrs {
		h.appendAttr(buf, attr, h.groupPrefix, h.groups)
	}

	// 将属性前缀添加到属性字符串前，并将结果赋值给h2的attrsPrefix字段
	h2.attrsPrefix = h.attrsPrefix + string(*buf)

	// 返回处理后的handler实例h2
	return h2
}

// WithGroup 函数为slog.Handler类型的method，用于给handler添加一个group。
// 参数name为要添加的group的名称。
// 如果name为空字符串，则返回原始的handler。
// 否则，克隆原始的handler并返回处理后的handler。
// 处理后的handler的groupPrefix字段会在原始字段的基础上加上新的group名称，并将新的group名称添加到groups字段中。
func (h *Handler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}
	h2 := h.clone()
	h2.groupPrefix += name + "."
	h2.groups = append(h2.groups, name)
	return h2
}

func (h *Handler) appendTime(buf *buffer, t time.Time) {
	buf.WriteStringIf(!h.noColor, ansiFaint)  // 在buf中添加ansiFaint字符，如果h.noColor为false
	*buf = t.AppendFormat(*buf, h.timeFormat) // 将t格式化为字符串并追加到buf
	buf.WriteStringIf(!h.noColor, ansiReset)  // 在buf中添加ansiReset字符,如果h.noColor为false
}

// appendLevel 方法根据日志级别将相应的级别字符串和相对应的级别差值添加到buf中
func (h *Handler) appendLevel(buf *buffer, level slog.Level) {
	switch {
	case level < slog.LevelInfo:
		buf.WriteStringIf(!h.noColor, ansiBrightBlue) // 如果noColor为false，则添加亮青色前景色代码
		buf.WriteString("DEBUG")
		appendLevelDelta(buf, level-slog.LevelDebug)
		buf.WriteStringIf(!h.noColor, ansiReset)
	case level < slog.LevelWarn:
		buf.WriteStringIf(!h.noColor, ansiBrightGreen) // 如果noColor为false，则添加亮绿色前景色代码
		buf.WriteString("INFO")                        // 添加"INFO"字符串
		appendLevelDelta(buf, level-slog.LevelInfo)    // 添加级别差值
		buf.WriteStringIf(!h.noColor, ansiReset)       // 如果noColor为false，则添加重置代码
	case level < slog.LevelError:
		buf.WriteStringIf(!h.noColor, ansiBrightYellow) // 如果noColor为false，则添加亮黄色前景色代码
		buf.WriteString("WARN")                         // 添加"WARN"字符串
		appendLevelDelta(buf, level-slog.LevelWarn)     // 添加级别差值
		buf.WriteStringIf(!h.noColor, ansiReset)        // 如果noColor为false，则添加重置代码
	default:
		buf.WriteStringIf(!h.noColor, ansiBrightRed) // 如果noColor为false，则添加亮红色前景色代码
		buf.WriteString("ERROR")                     // 添加"ERROR"字符串
		appendLevelDelta(buf, level-slog.LevelError) // 添加级别差值
		buf.WriteStringIf(!h.noColor, ansiReset)     // 如果noColor为false，则添加重置代码
	}
}

func appendLevelDelta(buf *buffer, delta slog.Level) {
	if delta == 0 {
		return
	} else if delta > 0 {
		buf.WriteByte('+')
	}
	*buf = strconv.AppendInt(*buf, int64(delta), 10)
}

func (h *Handler) appendSource(buf *buffer, src *slog.Source) {
	dir, file := filepath.Split(src.File)

	buf.WriteStringIf(!h.noColor, ansiFaint)
	buf.WriteString(filepath.Join(filepath.Base(dir), file))
	buf.WriteByte(':')
	buf.WriteString(strconv.Itoa(src.Line))
	buf.WriteStringIf(!h.noColor, ansiReset)
}

func (h *Handler) appendAttr(buf *buffer, attr slog.Attr, groupsPrefix string, groups []string) {
	attr.Value = attr.Value.Resolve()
	if rep := h.replaceAttr; rep != nil && attr.Value.Kind() != slog.KindGroup {
		attr = rep(groups, attr)
		attr.Value = attr.Value.Resolve()
	}

	if attr.Equal(slog.Attr{}) {
		return
	}

	if attr.Value.Kind() == slog.KindGroup {
		if attr.Key != "" {
			groupsPrefix += attr.Key + "."
			groups = append(groups, attr.Key)
		}
		for _, groupAttr := range attr.Value.Group() {
			h.appendAttr(buf, groupAttr, groupsPrefix, groups)
		}
	} else {
		if err, ok := attr.Value.Any().(Error); ok {
			// append Error
			h.appendError(buf, err, groupsPrefix)
			buf.WriteByte(' ')
		} else {
			h.appendKey(buf, attr.Key, groupsPrefix)
			h.appendValue(buf, attr.Value, true)
			buf.WriteByte(' ')
		}
	}
}

func (h *Handler) appendKey(buf *buffer, key, groups string) {
	buf.WriteStringIf(!h.noColor, ansiFaint)
	appendString(buf, groups+key, true)
	buf.WriteByte('=')
	buf.WriteStringIf(!h.noColor, ansiReset)
}

// appendValue 根据传入的值类型将值附加到buf中
func (h *Handler) appendValue(buf *buffer, v slog.Value, quote bool) {
	switch v.Kind() {
	case slog.KindString:
		// 如果值的类型是字符串，调用appendString函数将字符串附加到buf中
		appendString(buf, v.String(), quote)
	case slog.KindInt64:
		// 如果值的类型是int64，使用strconv包中的AppendInt函数将int64类型的值附加到buf中
		*buf = strconv.AppendInt(*buf, v.Int64(), 10)
	case slog.KindUint64:
		// 如果值的类型是uint64，使用strconv包中的AppendUint函数将uint64类型的值附加到buf中
		*buf = strconv.AppendUint(*buf, v.Uint64(), 10)
	case slog.KindFloat64:
		// 如果值的类型是float64，使用strconv包中的AppendFloat函数将float64类型的值附加到buf中
		*buf = strconv.AppendFloat(*buf, v.Float64(), 'g', -1, 64)
	case slog.KindBool:
		// 如果值的类型是布尔值，使用strconv包中的AppendBool函数将布尔值附加到buf中
		*buf = strconv.AppendBool(*buf, v.Bool())
	case slog.KindDuration:
		// 如果值的类型是时间段，调用appendString函数将时间段附加到buf中
		appendString(buf, v.Duration().String(), quote)
	case slog.KindTime:
		// 如果值的类型是时间，调用appendString函数将时间附加到buf中
		appendString(buf, v.Time().String(), quote)
	case slog.KindAny:
		// 如果值的类型是任意类型
		switch cv := v.Any().(type) {
		case slog.Level:
			// 如果值的类型是slog.Level，调用appendLevel函数将Level值附加到buf中
			h.appendLevel(buf, cv)
		case encoding.TextMarshaler:
			// 如果值的类型实现了 [encoding.TextMarshaler] 接口
			data, err := cv.MarshalText()
			if err != nil {
				break
			}
			// 将MarshalText返回的字符串附加到buf中
			appendString(buf, string(data), quote)
		case *slog.Source:
			// 如果值的类型是slog.Source指针
			// 调用appendSource函数将Source附加到buf中
			h.appendSource(buf, cv)
		default:
			// 对于其他任意类型的值
			// 调用fmt.Sprint将任意类型的值转换为字符串，并将字符串附加到buf中
			appendString(buf, fmt.Sprint(v.Any()), quote)
		}
	default:
		return
	}
}

func (h *Handler) appendError(buf *buffer, err error, groupsPrefix string) {
	buf.WriteStringIf(!h.noColor, ansiBrightRedFaint)
	appendString(buf, groupsPrefix+errKey, true)
	buf.WriteByte('=')
	buf.WriteStringIf(!h.noColor, ansiResetFaint)
	appendString(buf, err.Error(), true)
	buf.WriteStringIf(!h.noColor, ansiReset)
}

func appendString(buf *buffer, s string, quote bool) {
	if quote && needsQuoting(s) {
		*buf = strconv.AppendQuote(*buf, s)
	} else {
		buf.WriteString(s)
	}
}

func needsQuoting(s string) bool {
	// 如果字符串为空，需要引号标识
	if len(s) == 0 {
		return true
	}
	// 遍历字符串中的每个字符
	for _, r := range s {
		// 如果字符是空格、双引号、等号或者不是可打印字符，则需要引号标识
		if unicode.IsSpace(r) || r == '"' || r == '=' || !unicode.IsPrint(r) {
			return true
		}
	}
	// 不需要引号标识
	return false
}

type Error struct{ error }

// Err 返回一个着色（颜色化）的 [slog.Attr]，通过 [Handler] 将该 [slog.Attr] 写为红色。
// 当与其他[slog.Handler]一起使用时，它表现如
//
//	slog.Any("err", err)
func Err(err error) slog.Attr {
	if err != nil {
		err = Error{err}
	}
	return slog.Any(errKey, err)
}
