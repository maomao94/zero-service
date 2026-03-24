package ctxdata

import "context"

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

// PropField 定义单个上下文传递字段在三种传输层中的 key。
type PropField struct {
	CtxKey     string // context.WithValue 的 key
	GrpcHeader string // gRPC metadata key（全小写）
	HttpHeader string // HTTP header key（canonical form）
	Sensitive  bool   // 日志中是否脱敏
}

// PropFields 全量传递字段列表（唯一数据源）。
// 新增传递字段只需在此追加，所有 gRPC/HTTP 转换自动生效。
var PropFields = []PropField{
	{CtxAuthorizationKey, HeaderAuthorization, "Authorization", true},
	{CtxUserIdKey, HeaderUserId, "X-User-Id", false},
	{CtxUserNameKey, HeaderUserName, "X-User-Name", false},
	{CtxDeptCodeKey, HeaderDeptCode, "X-Dept-Code", false},
	{CtxTraceIdKey, HeaderTraceId, "X-Trace-Id", false},
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
