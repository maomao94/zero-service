package logic

import (
	"zero-service/common/tool"
	"zero-service/third_party/extproto"
)

// MCP 相关错误
var (
	ErrMcpClientNotConfigured          = tool.NewErrorByPbCode(extproto.Code__1_05_BIZ, "mcp client 未配置")
	ErrMcpToolNotFound                 = tool.NewErrorByPbCode(extproto.Code__1_05_BIZ, "mcp 工具未找到")
	ErrAsyncResultHandlerNotConfigured = tool.NewErrorByPbCode(extproto.Code__1_05_BIZ, "异步结果处理器未配置")
)
