package handler

import (
	"context"

	"zero-service/common/isp"

	"github.com/zeromicro/go-zero/core/logx"
)

// logFields 入站/出站公用的日志字段（session 由 gnetx/OTel 自动注入）。
func logFields(msg *isp.Message) []logx.LogField {
	return []logx.LogField{
		logx.Field("name", msg.MessageName()),
		logx.Field("type", msg.Type),
		logx.Field("command", msg.Command),
		logx.Field("sendSeq", msg.SendSeq),
		logx.Field("recvSeq", msg.RecvSeq),
		logx.Field("sendCode", msg.SendCode),
	}
}

// LogInbound 入站消息日志（server→client）。
func LogInbound(ctx context.Context, msg *isp.Message) {
	logx.WithContext(ctx).Infow("recv", logFields(msg)...)
}

// LogOutbound 出站消息日志（client→server）。
func LogOutbound(ctx context.Context, msg *isp.Message) {
	logx.WithContext(ctx).Infow("send", logFields(msg)...)
}

// LogFallback 未匹配消息日志。
func LogFallback(ctx context.Context, msg *isp.Message) {
	logx.WithContext(ctx).Infow("fallback", logFields(msg)...)
}
