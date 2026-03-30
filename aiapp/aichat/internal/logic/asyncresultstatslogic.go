package logic

import (
	"context"

	"zero-service/aiapp/aichat/aichat"
	"zero-service/aiapp/aichat/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type AsyncResultStatsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewAsyncResultStatsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AsyncResultStatsLogic {
	return &AsyncResultStatsLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// AsyncResultStats 获取异步结果统计信息
func (l *AsyncResultStatsLogic) AsyncResultStats(in *aichat.EmptyReq) (*aichat.AsyncResultStat, error) {
	store := l.svcCtx.AsyncResultStore
	if store == nil {
		return nil, ErrAsyncResultHandlerNotConfigured
	}

	stats, err := store.Stats(l.ctx)
	if err != nil {
		logx.WithContext(l.ctx).Errorf("[AsyncResultStats] stats error: %v", err)
		return nil, err
	}

	return &aichat.AsyncResultStat{
		Total:       stats.Total,
		Pending:     stats.Pending,
		Completed:   stats.Completed,
		Failed:      stats.Failed,
		SuccessRate: stats.SuccessRate,
	}, nil
}
