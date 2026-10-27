package middleware

import (
	"context"

	"zero-service/common/gtwx"
)

// RequestIDMiddleware 为每个请求准备 X-Request-Id。
// 优先复用调用方传入的 X-Request-Id，缺失或非法时生成一个，并写入响应头与上下文。
var RequestIDMiddleware = gtwx.RequestIDMiddleware

// RequestIDFromContext 读取当前请求的 X-Request-Id。
func RequestIDFromContext(ctx context.Context) string {
	return gtwx.RequestIDFromContext(ctx)
}
