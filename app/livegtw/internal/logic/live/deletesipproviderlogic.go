package live

import (
	"context"

	"zero-service/app/live/live"
	"zero-service/app/livegtw/internal/svc"
	"zero-service/app/livegtw/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type DeleteSipProviderLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewDeleteSipProviderLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteSipProviderLogic {
	return &DeleteSipProviderLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *DeleteSipProviderLogic) DeleteSipProvider(req *types.DeleteSipProviderRequest) (resp *types.DeleteSipProviderReply, err error) {
	_, err = l.svcCtx.LiveRpcCli.DeleteSipProvider(l.ctx, &live.DeleteSipProviderReq{
		Id: req.Id,
	})
	if err != nil {
		return nil, err
	}
	return &types.DeleteSipProviderReply{}, nil
}
