package logger

import (
	"context"
	"fmt"
	"github.com/fatih/color"
	c "github.com/life-stream-dev/life-stream-go-mqtt-broker/internal/config"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const (
	LevelFatal slog.Level = 12
)

type AsyncHandler struct {
	ch          chan []byte
	writer      io.Writer
	attrs       []slog.Attr
	currentDay  int      // 当前日志日期（day of year）
	currentFile *os.File // 当前日志文件
	basePath    string   // 日志文件基础路径
	group       string
	logLevel    slog.Level
	wg          sync.WaitGroup
}

func NewAsyncHandler(basePath string, logLevel slog.Level) *AsyncHandler {
	h := &AsyncHandler{
		ch:       make(chan []byte, 1024),
		logLevel: logLevel,
		basePath: basePath,
	}
	_ = h.rotateIfNeeded()
	h.wg.Add(1)
	go h.startWorker()
	return h
}

// 在rotateIfNeeded中添加
func (h *AsyncHandler) cleanOldLogs() {
	files, _ := filepath.Glob(h.basePath + "/*.log")
	now := time.Now()

	for _, f := range files {
		fi, _ := os.Stat(f)
		if now.Sub(fi.ModTime()) > 30*24*time.Hour {
			_ = os.Remove(f) // 删除30天前的日志
		}
	}
}

// 初始化或轮转日志文件
func (h *AsyncHandler) rotateIfNeeded() error {
	now := time.Now()
	currentDay := now.YearDay()

	// 检查是否需要轮转
	if currentDay == h.currentDay && h.currentFile != nil {
		return nil
	}

	// 关闭旧文件
	if h.currentFile != nil {
		if err := h.currentFile.Close(); err != nil {
			return fmt.Errorf("关闭日志文件失败: %w", err)
		}
	}

	// 创建新文件
	logPath := h.getLogPath()
	if err := os.MkdirAll(filepath.Dir(logPath), 0755); err != nil {
		return fmt.Errorf("创建日志目录失败: %w", err)
	}

	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("创建日志文件失败: %w", err)
	}

	// 更新状态
	h.currentFile = f
	h.currentDay = currentDay
	h.writer = io.MultiWriter(os.Stdout, h.currentFile)
	return nil
}

// 获取当前日志文件路径
func (h *AsyncHandler) getLogPath() string {
	now := time.Now()
	return fmt.Sprintf("%s/%s.log", h.basePath, now.Format("2006-01-02"))
}

func (h *AsyncHandler) startWorker() {
	defer h.wg.Done()
	for data := range h.ch {
		_, _ = h.writer.Write(data)
	}
}

func (h *AsyncHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.logLevel
}

func (h *AsyncHandler) Handle(_ context.Context, r slog.Record) error {
	level := r.Level.String()

	switch r.Level {
	case slog.LevelDebug:
		level = color.MagentaString(level)
	case slog.LevelInfo:
		level = color.BlueString(level)
	case slog.LevelWarn:
		level = color.YellowString(level)
	case slog.LevelError:
		level = color.RedString(level)
	case LevelFatal:
		level = color.HiRedString("FATAL")
	}

	// 基础格式：时间 | 级别 | 消息
	line := fmt.Sprintf(
		"%s | %-5s | %s",
		color.GreenString(r.Time.Format("2006-01-02T15:04:05")),
		level,
		color.CyanString(r.Message),
	)

	// 处理固定字段
	for _, attr := range h.attrs {
		line += color.CyanString(fmt.Sprintf(" %s=%v", attr.Key, attr.Value))
	}

	// 处理动态字段
	r.Attrs(func(attr slog.Attr) bool {
		line += color.CyanString(fmt.Sprintf(" %s=%v", attr.Key, attr.Value))
		return true
	})

	line += "\n"

	h.Write([]byte(line))
	return nil
}

func (h *AsyncHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	// 合并新旧字段
	newAttrs := make([]slog.Attr, 0, len(h.attrs)+len(attrs))
	newAttrs = append(newAttrs, h.attrs...)
	newAttrs = append(newAttrs, attrs...)

	return &AsyncHandler{
		writer:   h.writer,
		attrs:    newAttrs,
		group:    h.group,
		logLevel: h.logLevel,
	}
}

func (h *AsyncHandler) WithGroup(name string) slog.Handler {
	// 记录当前分组名称
	return &AsyncHandler{
		writer:   h.writer,
		attrs:    h.attrs,
		group:    name,
		logLevel: h.logLevel,
	}
}

func (h *AsyncHandler) Write(p []byte) {
	// 拷贝数据避免竞态
	pb := make([]byte, len(p))
	copy(pb, p)
	h.ch <- pb
}

func (h *AsyncHandler) Close() error {
	close(h.ch)
	h.wg.Wait()
	if f, ok := h.writer.(*os.File); ok {
		_ = f.Sync()
	}
	return nil
}

type ShutdownCallback struct {
	handler *AsyncHandler
}

func (lc *ShutdownCallback) Invoke(ctx context.Context) error {
	return lc.handler.Close()
}

func Init() *ShutdownCallback {
	var handler *AsyncHandler
	config, _ := c.GetConfig()
	if config.DebugMode {
		handler = NewAsyncHandler("logs", slog.LevelDebug)
	} else {
		handler = NewAsyncHandler("logs", slog.LevelInfo)
	}
	logger := slog.New(handler)
	slog.SetDefault(logger)
	slog.Debug("Logger initialized")
	return &ShutdownCallback{handler: handler}
}

func Debug(msg string, v ...interface{}) {
	slog.Debug(msg, v...)
}

func DebugF(msg string, v ...interface{}) {
	slog.Debug(fmt.Sprintf(msg, v...))
}

func Info(msg string, v ...interface{}) {
	slog.Info(msg, v...)
}

func InfoF(msg string, v ...interface{}) {
	slog.Info(fmt.Sprintf(msg, v...))
}

func Warn(msg string, v ...interface{}) {
	slog.Warn(msg, v...)
}

func WarnF(msg string, v ...interface{}) {
	slog.Warn(fmt.Sprintf(msg, v...))
}

func Error(msg string, v ...interface{}) {
	slog.Error(msg, v...)
}

func ErrorF(msg string, v ...interface{}) {
	slog.Error(fmt.Sprintf(msg, v...))
}

func Fatal(msg string, v ...interface{}) {
	slog.Log(context.Background(), LevelFatal, msg, v...)
}

func FatalF(msg string, v ...interface{}) {
	slog.Log(context.Background(), LevelFatal, fmt.Sprintf(msg, v...))
}
