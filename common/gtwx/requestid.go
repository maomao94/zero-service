package gtwx

import (
	"context"
	"net/http"
	"strings"

	"zero-service/common/tool"
)

type requestIDKey struct{}

// RequestIDMiddleware 为每个请求准备 X-Request-Id。
// 优先复用调用方传入的 X-Request-Id，缺失或非法时生成一个，并写入响应头与上下文。
func RequestIDMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		requestID := strings.TrimSpace(r.Header.Get("X-Request-Id"))
		if requestID == "" || len(requestID) > 128 || hasControlChars(requestID) {
			if id, err := tool.SimpleUUID(); err == nil {
				requestID = id
			} else {
				requestID = "req-" + strings.ReplaceAll(strings.TrimSpace(r.RemoteAddr), ":", "-")
			}
		}
		w.Header().Set("X-Request-Id", requestID)
		next(w, r.WithContext(context.WithValue(r.Context(), requestIDKey{}, requestID)))
	}
}

// RequestIDFromContext 读取当前请求的 X-Request-Id。
func RequestIDFromContext(ctx context.Context) string {
	requestID, _ := ctx.Value(requestIDKey{}).(string)
	return requestID
}

func hasControlChars(value string) bool {
	for i := 0; i < len(value); i++ {
		if value[i] < 0x20 || value[i] > 0x7E {
			return true
		}
	}
	return false
}
