// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package middleware

import "net/http"

// NoStoreMiddleware 禁止管理面响应被浏览器或中间代理缓存。
//
// 管理面响应可能包含 secret/signKey 等凭据，缓存到磁盘或代理会扩大泄露面。
// 这里挂在整个 /open-admin 组上，而不是只给某个 handler，避免以后新增接口时遗漏。
type NoStoreMiddleware struct {
}

func NewNoStoreMiddleware() *NoStoreMiddleware {
	return &NoStoreMiddleware{}
}

func (m *NoStoreMiddleware) Handle(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Pragma", "no-cache")
		next(w, r)
	}
}
