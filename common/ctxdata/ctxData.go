package ctxdata

import "context"

const (
	CtxUserIdKey        = "user-id"
	CtxUserNameKey      = "user-name"
	CtxDeptCodeKey      = "dept-code"
	CtxAuthorizationKey = "authorization"
	CtxAuthTypeKey      = "auth-type"
	CtxMetaKey          = "_meta" // MCP _meta 透传 key，存储原始 _meta 数据供业务层使用
)

// gRPC metadata header key（必须小写）
const (
	HeaderUserId        = "x-user-id"
	HeaderUserName      = "x-user-name"
	HeaderDeptCode      = "x-dept-code"
	HeaderAuthorization = "authorization"
	HeaderAuthType      = "x-auth-type"
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
	{CtxAuthTypeKey, HeaderAuthType, "X-Auth-Type", false},
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

func GetDeptCode(ctx context.Context) string {
	if v, ok := ctx.Value(CtxDeptCodeKey).(string); ok {
		return v
	}
	return ""
}

// GetMeta 获取 MCP _meta 透传数据。
// 返回原始 _meta map[string]any，业务层自行解析。
func GetMeta(ctx context.Context) map[string]any {
	if v, ok := ctx.Value(CtxMetaKey).(map[string]any); ok {
		return v
	}
	return nil
}
