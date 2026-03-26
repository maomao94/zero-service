// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package pass

import (
	"context"

	aichat "zero-service/aiapp/aichat/aichat"
	"zero-service/aiapp/aigtw/internal/svc"
	"zero-service/aiapp/aigtw/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type AsyncToolResultLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 查询异步工具调用的执行状态和结果，建议轮询间隔 1~2 秒
func NewAsyncToolResultLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AsyncToolResultLogic {
	return &AsyncToolResultLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *AsyncToolResultLogic) AsyncToolResult(req *types.AsyncToolResultRequest) (resp *types.AsyncToolResultResponse, err error) {
	l.Infof("[AsyncToolResult] task_id: %s", req.TaskID)

	rpcResp, err := l.svcCtx.AiChatCli.AsyncToolResult(l.ctx, &aichat.AsyncToolResultReq{
		TaskId: req.TaskID,
	})
	if err != nil {
		l.Errorf("[AsyncToolResult] rpc error: %v", err)
		return nil, err
	}

	return &types.AsyncToolResultResponse{
		TaskID:   rpcResp.TaskId,
		Status:   rpcResp.Status,
		Progress: rpcResp.Progress,
		Result:   rpcResp.Result,
		Error:    rpcResp.Error,
	}, nil
}
