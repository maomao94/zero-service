package mcpx

import (
	"net/http"

	"github.com/modelcontextprotocol/go-sdk/auth"
	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/mcp"
	"github.com/zeromicro/go-zero/rest"
)

// McpServerConf 扩展 go-zero McpConf，添加鉴权配置。
// 当 go-zero 原生支持 auth 后，可直接切回 mcp.McpConf。
type McpServerConf struct {
	mcp.McpConf
	Auth struct {
		JwtSecrets   []string `json:",optional"`
		ServiceToken string   `json:",optional"`
	}
}

// McpServer 带鉴权的 MCP 服务器封装。
// 与 go-zero mcp.McpServer 接口对齐（Start/Stop），
// 额外暴露 Server() 用于注册工具。
type McpServer struct {
	sdkServer  *sdkmcp.Server
	httpServer *rest.Server
	conf       McpServerConf
}

// NewMcpServer 创建带鉴权的 MCP 服务器。
// 内部逻辑与 go-zero mcp.NewMcpServer 一致，仅在 handler 外层包装 auth 中间件。
func NewMcpServer(c McpServerConf) *McpServer {
	// 1. 创建 REST HTTP server（与 go-zero 一致）
	var httpServer *rest.Server
	if len(c.Mcp.Cors) == 0 {
		httpServer = rest.MustNewServer(c.RestConf)
	} else {
		httpServer = rest.MustNewServer(c.RestConf, rest.WithCors(c.Mcp.Cors...))
	}

	// 2. 设置默认值（与 go-zero 一致）
	if len(c.Mcp.Name) == 0 {
		c.Mcp.Name = c.Name
	}
	if len(c.Mcp.Version) == 0 {
		c.Mcp.Version = "1.0.0"
	}

	// 3. 创建 MCP SDK server（与 go-zero 一致）
	sdkServer := sdkmcp.NewServer(&sdkmcp.Implementation{
		Name:    c.Mcp.Name,
		Version: c.Mcp.Version,
	}, nil)

	s := &McpServer{
		sdkServer:  sdkServer,
		httpServer: httpServer,
		conf:       c,
	}

	// 4. 设置 transport 并注册路由
	if c.Mcp.UseStreamable {
		s.setupStreamableTransport()
	} else {
		s.setupSSETransport()
	}

	return s
}

// Server 返回底层 SDK Server，用于注册工具。
// 使用方式：sdkmcp.AddTool(server.Server(), tool, handler)
func (s *McpServer) Server() *sdkmcp.Server {
	return s.sdkServer
}

// Start 启动 HTTP 服务器。
func (s *McpServer) Start() {
	logx.Infof("Starting MCP server %s v%s on %s:%d",
		s.conf.Mcp.Name, s.conf.Mcp.Version, s.conf.Host, s.conf.Port)
	s.httpServer.Start()
}

// Stop 停止 HTTP 服务器。
func (s *McpServer) Stop() {
	logx.Info("Stopping MCP server")
	s.httpServer.Stop()
}

// setupSSETransport 配置 SSE transport（2024-11-05 spec）。
func (s *McpServer) setupSSETransport() {
	handler := sdkmcp.NewSSEHandler(func(r *http.Request) *sdkmcp.Server {
		logx.Infof("New SSE connection from %s", r.RemoteAddr)
		return s.sdkServer
	}, nil)

	s.registerRoutes(s.wrapAuth(handler), s.conf.Mcp.SseEndpoint)
}

// setupStreamableTransport 配置 Streamable HTTP transport（2025-03-26 spec）。
func (s *McpServer) setupStreamableTransport() {
	handler := sdkmcp.NewStreamableHTTPHandler(func(r *http.Request) *sdkmcp.Server {
		logx.Infof("New streamable connection from %s", r.RemoteAddr)
		return s.sdkServer
	}, nil)

	s.registerRoutes(s.wrapAuth(handler), s.conf.Mcp.MessageEndpoint)
}

// wrapAuth 如果配置了鉴权，则包装 auth 中间件；否则原样返回。
func (s *McpServer) wrapAuth(handler http.Handler) http.Handler {
	if len(s.conf.Auth.JwtSecrets) == 0 && s.conf.Auth.ServiceToken == "" {
		return handler
	}
	verifier := NewDualTokenVerifier(s.conf.Auth.JwtSecrets, s.conf.Auth.ServiceToken)
	return auth.RequireBearerToken(verifier, nil)(handler)
}

// registerRoutes 注册 MCP 路由（与 go-zero 一致，增加 DELETE 支持 Streamable HTTP）。
func (s *McpServer) registerRoutes(handler http.Handler, endpoint string) {
	s.httpServer.AddRoute(rest.Route{
		Method:  http.MethodGet,
		Path:    endpoint,
		Handler: handler.ServeHTTP,
	}, rest.WithSSE(), rest.WithTimeout(s.conf.Mcp.SseTimeout))

	s.httpServer.AddRoute(rest.Route{
		Method:  http.MethodPost,
		Path:    endpoint,
		Handler: handler.ServeHTTP,
	}, rest.WithTimeout(s.conf.Mcp.MessageTimeout))

	s.httpServer.AddRoute(rest.Route{
		Method:  http.MethodDelete,
		Path:    endpoint,
		Handler: handler.ServeHTTP,
	})
}
