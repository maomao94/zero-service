package live

import (
	"context"

	"zero-service/app/live/live"
	"zero-service/app/livegtw/internal/svc"
	"zero-service/app/livegtw/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type CreateSipProviderLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewCreateSipProviderLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateSipProviderLogic {
	return &CreateSipProviderLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *CreateSipProviderLogic) CreateSipProvider(req *types.CreateSipProviderRequest) (resp *types.CreateSipProviderReply, err error) {
	r, err := l.svcCtx.LiveRpcCli.CreateSipProvider(l.ctx, &live.CreateSipProviderReq{
		Code:         req.Code,
		Name:         req.Name,
		Address:      req.Address,
		Numbers:      req.Numbers,
		AuthUsername: req.AuthUsername,
		AuthPassword: req.AuthPassword,
	})
	if err != nil {
		return nil, err
	}
	return &types.CreateSipProviderReply{
		Provider: toSipProviderInfo(r.GetProvider()),
	}, nil
}
