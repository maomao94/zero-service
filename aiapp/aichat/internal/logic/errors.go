package logic

import (
	"zero-service/common/errx"
	"zero-service/third_party/extproto"
)

// MCP 相关错误
var (
	ErrMcpClientNotConfigured          = errx.NewErrorByPbCode(extproto.Code__1_00_BIZ, "mcp client 未配置")
	ErrMcpToolNotFound                 = errx.NewErrorByPbCode(extproto.Code__1_00_BIZ, "mcp 工具未找到")
	ErrAsyncResultHandlerNotConfigured = errx.NewErrorByPbCode(extproto.Code__1_00_BIZ, "异步结果处理器未配置")
)
