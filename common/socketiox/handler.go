package socketiox

import (
	"net/http"
)

type HandlerOption func(h *HandlerConfig)

type HandlerConfig struct {
	Server *Server
}

func WithServer(server *Server) HandlerOption {
	return func(h *HandlerConfig) {
		h.Server = server
	}
}

func NewSocketioHandler(opts ...HandlerOption) http.HandlerFunc {
	// 默认配置
	config := &HandlerConfig{}

	// 应用配置选项
	for _, opt := range opts {
		opt(config)
	}

	// 验证配置
	if config.Server == nil {
		panic("socketio server is required")
	}

	return func(w http.ResponseWriter, r *http.Request) {
		// 握手请求是唯一能拿到完整 HTTP 请求的地方，这里先把客户端 IP 落到内部头，
		// Session 在 OnConnection 时再读取，供连接/断开日志使用。
		if ip := clientIPFromRequest(r); ip != "" {
			r.Header.Set(clientIPHeaderInternal, ip)
		} else {
			// 始终覆盖客户端可能伪造的内部头。
			r.Header.Del(clientIPHeaderInternal)
		}
		config.Server.HttpHandler().ServeHTTP(w, r)
	}
}

func SocketioHandler(server *Server) http.HandlerFunc {
	return NewSocketioHandler(WithServer(server))
}
