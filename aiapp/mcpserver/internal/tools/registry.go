package tools

import (
	"zero-service/aiapp/mcpserver/internal/svc"

	"github.com/zeromicro/go-zero/mcp"
)

// RegisterAll 注册所有 MCP 工具
func RegisterAll(server mcp.McpServer, svcCtx *svc.ServiceContext) {
	RegisterEcho(server)
	RegisterModbus(server, svcCtx)
}
