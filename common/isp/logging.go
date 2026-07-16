package isp

import (
	"context"
	"errors"

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

// LogErrorResponse 记录由 handler error 转换出的 ISP 通用应答。
func LogErrorResponse(ctx context.Context, req *Message, err error, code string) {
	if code == StatusSuccess || code == StatusRetry {
		return
	}
	name := code
	switch code {
	case StatusReject:
		name = "拒绝(400)"
	case StatusError:
		name = "错误(500)"
	}
	var ie *IspError
	if errors.As(err, &ie) {
		logx.WithContext(ctx).Errorf("[isp] 回复%s type=%d command=%d reqCode=%s msg=%s", name, req.Type, req.Command, req.Code, ie.Msg)
	} else if err != nil {
		logx.WithContext(ctx).Errorf("[isp] 回复%s type=%d command=%d reqCode=%s err=%v", name, req.Type, req.Command, req.Code, err)
	}
}
