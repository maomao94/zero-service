package pass

import (
	"context"

	aichat "zero-service/aiapp/aichat/aichat"
	"zero-service/aiapp/aigtw/internal/svc"
	"zero-service/aiapp/aigtw/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type AsyncResultStatsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 获取异步结果统计信息
func NewAsyncResultStatsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AsyncResultStatsLogic {
	return &AsyncResultStatsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *AsyncResultStatsLogic) AsyncResultStats() (resp *types.AsyncResultStatsResponse, err error) {
	rpcResp, err := l.svcCtx.AiChatCli.AsyncResultStats(l.ctx, &aichat.EmptyReq{})
	if err != nil {
		return nil, err
	}

	return &types.AsyncResultStatsResponse{
		Total:       rpcResp.Total,
		Pending:     rpcResp.Pending,
		Completed:   rpcResp.Completed,
		Failed:      rpcResp.Failed,
		SuccessRate: rpcResp.SuccessRate,
	}, nil
}
