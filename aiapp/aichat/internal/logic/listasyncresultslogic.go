package logic

import (
	"context"

	"zero-service/aiapp/aichat/aichat"
	"zero-service/aiapp/aichat/internal/svc"
	"zero-service/common/mcpx"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListAsyncResultsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewListAsyncResultsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListAsyncResultsLogic {
	return &ListAsyncResultsLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// ListAsyncResults 分页查询异步结果列表。
func (l *ListAsyncResultsLogic) ListAsyncResults(in *aichat.ListAsyncResultsReq) (*aichat.ListAsyncResultsResp, error) {
	store := l.svcCtx.AsyncResultStore
	if store == nil {
		return nil, ErrAsyncResultHandlerNotConfigured
	}

	// 调用存储层
	storeReq := &mcpx.ListAsyncResultsReq{
		Status:    in.Status,
		StartTime: in.StartTime,
		EndTime:   in.EndTime,
		Page:      int(in.Page),
		PageSize:  int(in.PageSize),
		SortField: in.SortField,
		SortOrder: in.SortOrder,
	}

	storeResp, err := store.List(l.ctx, storeReq)
	if err != nil {
		logx.WithContext(l.ctx).Errorf("[ListAsyncResults] store list error: %v", err)
		return nil, err
	}

	// 转换响应
	items := make([]*aichat.AsyncToolResultRes, len(storeResp.Items))
	for i, item := range storeResp.Items {
		messages := make([]*aichat.ProgressMessagePb, len(item.Messages))
		for j, msg := range item.Messages {
			messages[j] = &aichat.ProgressMessagePb{
				Progress: msg.Progress,
				Total:    msg.Total,
				Message:  msg.Message,
				Time:     msg.Time,
			}
		}
		items[i] = &aichat.AsyncToolResultRes{
			TaskId:   item.TaskID,
			Status:   item.Status,
			Progress: item.Progress,
			Result:   item.Result,
			Error:    item.Error,
			Messages: messages,
		}
	}

	return &aichat.ListAsyncResultsResp{
		Items:      items,
		Total:      storeResp.Total,
		Page:       int32(storeResp.Page),
		PageSize:   int32(storeResp.PageSize),
		TotalPages: int32(storeResp.TotalPages),
	}, nil
}
