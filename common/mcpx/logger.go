package mcpx

import (
	"context"
	"log/slog"

	"github.com/zeromicro/go-zero/core/logx"
)

// logxHandler 将 slog 日志桥接到 go-zero logx。
type logxHandler struct{}

func newLogxLogger() *slog.Logger {
	return slog.New(&logxHandler{})
}

func (h *logxHandler) Enabled(_ context.Context, _ slog.Level) bool {
	return true
}

func (h *logxHandler) Handle(ctx context.Context, r slog.Record) error {
	msg := r.Message
	// 收集 attrs 拼接到消息后面
	r.Attrs(func(a slog.Attr) bool {
		msg += " " + a.Key + "=" + a.Value.String()
		return true
	})

	switch {
	case r.Level >= slog.LevelError:
		logx.WithContext(ctx).Error(msg)
	case r.Level >= slog.LevelWarn:
		logx.WithContext(ctx).Error(msg) // logx 没有 Warn，用 Error
	case r.Level >= slog.LevelInfo:
		logx.WithContext(ctx).Info(msg)
	default:
		logx.WithContext(ctx).Info(msg)
	}
	return nil
}

func (h *logxHandler) WithAttrs(_ []slog.Attr) slog.Handler { return h }
func (h *logxHandler) WithGroup(_ string) slog.Handler      { return h }
