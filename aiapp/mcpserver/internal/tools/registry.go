package tools

import (
	"zero-service/aiapp/mcpserver/internal/svc"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

// RegisterAll 注册所有 MCP 工具
func RegisterAll(server *sdkmcp.Server, svcCtx *svc.ServiceContext) {
	RegisterEcho(server)
	RegisterModbus(server, svcCtx)
}
