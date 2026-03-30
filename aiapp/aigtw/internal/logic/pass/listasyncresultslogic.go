package pass

import (
	"context"

	aichat "zero-service/aiapp/aichat/aichat"
	"zero-service/aiapp/aigtw/internal/svc"
	"zero-service/aiapp/aigtw/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListAsyncResultsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 分页查询异步结果列表，支持按状态、时间范围过滤，支持多字段排序
func NewListAsyncResultsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListAsyncResultsLogic {
	return &ListAsyncResultsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ListAsyncResultsLogic) ListAsyncResults(req *types.ListAsyncResultsRequest) (resp *types.ListAsyncResultsResponse, err error) {
	rpcResp, err := l.svcCtx.AiChatCli.ListAsyncResults(l.ctx, &aichat.ListAsyncResultsReq{
		Status:    req.Status,
		StartTime: req.StartTime,
		EndTime:   req.EndTime,
		Page:      int32(req.Page),
		PageSize:  int32(req.PageSize),
		SortField: req.SortField,
		SortOrder: req.SortOrder,
	})
	if err != nil {
		return nil, err
	}

	// 转换响应
	items := make([]types.AsyncToolResultResponse, len(rpcResp.Items))
	for i, item := range rpcResp.Items {
		messages := make([]types.ProgressMessage, len(item.Messages))
		for j, msg := range item.Messages {
			messages[j] = types.ProgressMessage{
				Progress: msg.Progress,
				Total:    msg.Total,
				Message:  msg.Message,
				Time:     msg.Time,
			}
		}
		items[i] = types.AsyncToolResultResponse{
			TaskID:   item.TaskId,
			Status:   item.Status,
			Progress: item.Progress,
			Result:   item.Result,
			Error:    item.Error,
			Messages: messages,
		}
	}

	return &types.ListAsyncResultsResponse{
		Items:      items,
		Total:      rpcResp.Total,
		Page:       int(rpcResp.Page),
		PageSize:   int(rpcResp.PageSize),
		TotalPages: int(rpcResp.TotalPages),
	}, nil
}
