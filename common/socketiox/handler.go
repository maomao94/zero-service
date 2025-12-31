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
		config.Server.HttpHandler().ServeHTTP(w, r)
	}
}

func SocketioHandler(server *Server) http.HandlerFunc {
	return NewSocketioHandler(WithServer(server))
}
