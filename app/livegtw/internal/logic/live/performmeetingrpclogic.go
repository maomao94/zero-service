package live

import (
	"context"

	"zero-service/app/live/live"
	"zero-service/app/livegtw/internal/svc"
	"zero-service/app/livegtw/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type PerformMeetingRpcLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewPerformMeetingRpcLogic(ctx context.Context, svcCtx *svc.ServiceContext) *PerformMeetingRpcLogic {
	return &PerformMeetingRpcLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *PerformMeetingRpcLogic) PerformMeetingRpc(req *types.PerformMeetingRpcRequest) (resp *types.PerformMeetingRpcReply, err error) {
	r, err := l.svcCtx.LiveRpcCli.PerformMeetingRpc(l.ctx, &live.PerformMeetingRpcReq{
		MeetingNo:         req.MeetingNo,
		Identity:          req.Identity,
		Method:            req.Method,
		Payload:           req.Payload,
		ResponseTimeoutMs: req.ResponseTimeoutMs,
	})
	if err != nil {
		return nil, err
	}
	return &types.PerformMeetingRpcReply{
		Response: r.GetResponse(),
	}, nil
}
