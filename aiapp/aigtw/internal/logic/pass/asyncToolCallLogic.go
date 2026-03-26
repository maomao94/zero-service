package pass

import (
	"context"
	"encoding/json"

	aichat "zero-service/aiapp/aichat/aichat"
	"zero-service/aiapp/aigtw/internal/svc"
	"zero-service/aiapp/aigtw/internal/types"

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

func (l *AsyncToolCallLogic) AsyncToolCall(req *types.AsyncToolCallRequest) (*types.AsyncToolCallResponse, error) {
	// 将参数转为 JSON 字符串
	argsJSON, err := json.Marshal(req.Args)
	if err != nil {
		logx.WithContext(l.ctx).Errorf("[AsyncToolCall] marshal args error: %v", err)
		return nil, err
	}

	resp, err := l.svcCtx.AiChatCli.AsyncToolCall(l.ctx, &aichat.AsyncToolCallReq{
		Server: req.Server,
		Tool:   req.Tool,
		Args:   string(argsJSON),
	})
	if err != nil {
		logx.WithContext(l.ctx).Errorf("[AsyncToolCall] rpc error: %v", err)
		return nil, err
	}

	return &types.AsyncToolCallResponse{
		TaskID: resp.TaskId,
		Status: resp.Status,
	}, nil
}
