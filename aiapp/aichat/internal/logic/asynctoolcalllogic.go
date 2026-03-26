package logic

import (
	"context"
	"encoding/json"

	"zero-service/aiapp/aichat/aichat"
	"zero-service/aiapp/aichat/internal/svc"
	"zero-service/common/mcpx"

	"github.com/zeromicro/go-zero/core/logx"
)

type AsyncToolCallLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewAsyncToolCallLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AsyncToolCallLogic {
	return &AsyncToolCallLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// AsyncToolCall 异步调用 MCP 工具
func (l *AsyncToolCallLogic) AsyncToolCall(in *aichat.AsyncToolCallReq) (*aichat.AsyncToolCallRes, error) {
	mcpClient := l.svcCtx.McpClient
	if mcpClient == nil {
		return nil, ErrMcpClientNotConfigured
	}

	// 获取 ResultHandler
	handler := l.svcCtx.AsyncResultHandler
	if handler == nil {
		return nil, ErrAsyncResultHandlerNotConfigured
	}

	// 解析 JSON 参数
	args := make(map[string]any)
	if in.Args != "" {
		if err := json.Unmarshal([]byte(in.Args), &args); err != nil {
			logx.WithContext(l.ctx).Errorf("[AsyncToolCall] parse args error: %v", err)
			return nil, err
		}
	}

	// 构建工具名称（带服务器前缀）
	toolName := in.Server + mcpx.ToolNameSeparator + in.Tool

	// 调用异步方法
	taskID, err := mcpClient.CallToolAsync(l.ctx, &mcpx.CallToolAsyncRequest{
		Name:          toolName,
		Args:          args,
		ResultHandler: handler,
	})
	if err != nil {
		logx.WithContext(l.ctx).Errorf("[AsyncToolCall] call tool async error: %v", err)
		return nil, err
	}

	return &aichat.AsyncToolCallRes{
		TaskId: taskID,
		Status: "pending",
	}, nil
}
