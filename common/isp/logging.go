package isp

import (
	"context"

	"github.com/zeromicro/go-zero/core/logx"
)

// LogFields 返回 ISP 消息的日志字段（session 由 gnetx/OTel 自动注入）。
func LogFields(msg *Message) []logx.LogField {
	return []logx.LogField{
		logx.Field("name", msg.MessageName()),
		logx.Field("type", msg.Type),
		logx.Field("command", msg.Command),
		logx.Field("sendSeq", msg.SendSeq),
		logx.Field("recvSeq", msg.RecvSeq),
		logx.Field("sendCode", msg.SendCode),
	}
}

// LogInbound 入站消息日志（client→server）。
func LogInbound(ctx context.Context, msg *Message) {
	logx.WithContext(ctx).Infow("recv", LogFields(msg)...)
}

// LogOutbound 出站消息日志（server→client）。
func LogOutbound(ctx context.Context, msg *Message) {
	logx.WithContext(ctx).Infow("send", LogFields(msg)...)
}

// LogFallback 未匹配消息日志。
func LogFallback(ctx context.Context, msg *Message) {
	logx.WithContext(ctx).Infow("fallback", LogFields(msg)...)
}
