package live

import (
	"context"

	"zero-service/app/live/live"
	"zero-service/app/livegtw/internal/svc"
	"zero-service/app/livegtw/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type UpdateSipProviderLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUpdateSipProviderLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateSipProviderLogic {
	return &UpdateSipProviderLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UpdateSipProviderLogic) UpdateSipProvider(req *types.UpdateSipProviderRequest) (resp *types.UpdateSipProviderReply, err error) {
	r, err := l.svcCtx.LiveRpcCli.UpdateSipProvider(l.ctx, &live.UpdateSipProviderReq{
		Id:           req.Id,
		Name:         req.Name,
		Address:      req.Address,
		Numbers:      req.Numbers,
		AuthUsername: req.AuthUsername,
		AuthPassword: req.AuthPassword,
		Status:       req.Status,
	})
	if err != nil {
		return nil, err
	}
	return &types.UpdateSipProviderReply{
		Provider: toSipProviderInfo(r.GetProvider()),
	}, nil
}
