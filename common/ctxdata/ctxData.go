package ctxdata

import (
	"context"

	"go.opentelemetry.io/otel/propagation"
)

const (
	CtxUserIdKey        = "user-id"
	CtxUserNameKey      = "user-name"
	CtxDeptCodeKey      = "dept-code"
	CtxAuthorizationKey = "authorization"
	CtxTraceIdKey       = "trace-id"
)

// gRPC metadata header key（必须小写）
const (
	HeaderUserId        = "x-user-id"
	HeaderUserName      = "x-user-name"
	HeaderDeptCode      = "x-dept-code"
	HeaderAuthorization = "authorization"
	HeaderTraceId       = "x-trace-id"
)

type MsgBody struct {
	MsgId   string                     `json:"msgId,omitempty"`
	Carrier *propagation.HeaderCarrier `json:"carrier"`
	Msg     string                     `json:"msg,omitempty"`
	Url     string                     `json:"url" validate:"required"`
}

type ProtoMsgBody struct {
	MsgId          string                     `json:"msgId,omitempty"`
	Carrier        *propagation.HeaderCarrier `json:"carrier"`
	GrpcServer     string                     `json:"grpcServer" validate:"required"`
	Method         string                     `json:"method" validate:"required"`
	Payload        string                     `json:"payload" validate:"required"`
	RequestTimeout int64                      `json:"requestTimeout"`
}

func GetUserId(ctx context.Context) string {
	if v, ok := ctx.Value(CtxUserIdKey).(string); ok {
		return v
	}
	return ""
}

func GetUserName(ctx context.Context) string {
	if v, ok := ctx.Value(CtxUserNameKey).(string); ok {
		return v
	}
	return ""
}

func GetAuthorization(ctx context.Context) string {
	if v, ok := ctx.Value(CtxAuthorizationKey).(string); ok {
		return v
	}
	return ""
}

func GetTraceId(ctx context.Context) string {
	if v, ok := ctx.Value(CtxTraceIdKey).(string); ok {
		return v
	}
	return ""
}

func GetDeptCode(ctx context.Context) string {
	if v, ok := ctx.Value(CtxDeptCodeKey).(string); ok {
		return v
	}
	return ""
}
