package socketiox

import (
	"net/http"
	"strings"

	"github.com/zeromicro/go-zero/core/logx"
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
		ip := clientIPFromRequest(r)
		if ip != "" {
			r.Header.Set(clientIPHeaderInternal, ip)
		} else {
			// 始终覆盖客户端可能伪造的内部头。
			r.Header.Del(clientIPHeaderInternal)
		}
		// websocket 直连时 SDK 不保存握手请求，OnConnection 里拿不到客户端 IP；
		// 这里在 HTTP 层对新会话的首次握手做一次审计，至少能按 IP 定位连接来源。
		if isNewSocketConnection(r) {
			logx.WithContext(r.Context()).Infof(
				"[socketio] new connection request: clientIp=%s transport=%s upgrade=%t path=%s ua=%q",
				ip, r.URL.Query().Get("transport"), isWebsocketUpgrade(r), r.URL.Path, r.UserAgent())
		}
		config.Server.HttpHandler().ServeHTTP(w, r)
	}
}

// isNewSocketConnection 判断请求是否在为新建的 engine.io 会话做首次握手：
// polling 的首次 GET 或 websocket 直连。带 sid 的请求复用已有会话，不重复审计。
func isNewSocketConnection(r *http.Request) bool {
	if r == nil || r.URL == nil || r.URL.Query().Get("sid") != "" {
		return false
	}
	switch r.URL.Query().Get("transport") {
	case "polling":
		return r.Method == http.MethodGet
	case "websocket":
		return isWebsocketUpgrade(r)
	default:
		return false
	}
}

// isWebsocketUpgrade 判断请求是否为 websocket 升级请求。
func isWebsocketUpgrade(r *http.Request) bool {
	if r == nil {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(r.Header.Get("Upgrade")), "websocket") &&
		strings.Contains(strings.ToLower(r.Header.Get("Connection")), "upgrade")
}

func SocketioHandler(server *Server) http.HandlerFunc {
	return NewSocketioHandler(WithServer(server))
}
