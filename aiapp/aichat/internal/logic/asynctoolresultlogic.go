package logic

import (
	"context"

	"zero-service/aiapp/aichat/aichat"
	"zero-service/aiapp/aichat/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type AsyncToolResultLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewAsyncToolResultLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AsyncToolResultLogic {
	return &AsyncToolResultLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// AsyncToolResult 查询异步工具调用结果
func (l *AsyncToolResultLogic) AsyncToolResult(in *aichat.AsyncToolResultReq) (*aichat.AsyncToolResultRes, error) {
	store := l.svcCtx.AsyncResultStore
	if store == nil {
		return nil, ErrAsyncResultHandlerNotConfigured
	}

	result, err := store.Get(l.ctx, in.TaskId)
	if err != nil {
		logx.WithContext(l.ctx).Errorf("[AsyncToolResult] get result error: %v", err)
		return nil, err
	}

	// 转换 messages
	messages := make([]*aichat.ProgressMessagePb, len(result.Messages))
	for i, msg := range result.Messages {
		messages[i] = &aichat.ProgressMessagePb{
			Progress: msg.Progress,
			Total:    msg.Total,
			Message:  msg.Message,
			Time:     msg.Time,
		}
	}

	return &aichat.AsyncToolResultRes{
		TaskId:   result.TaskID,
		Status:   result.Status,
		Progress: result.Progress,
		Result:   result.Result,
		Error:    result.Error,
		Messages: messages,
	}, nil
}
