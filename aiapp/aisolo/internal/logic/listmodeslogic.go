package logic

import (
	"context"

	"github.com/zeromicro/go-zero/core/logx"

	"zero-service/aiapp/aisolo/aisolo"
	"zero-service/aiapp/aisolo/internal/svc"
)

type ListModesLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewListModesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListModesLogic {
	return &ListModesLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *ListModesLogic) ListModes(_ *aisolo.ListModesReq) (*aisolo.ListModesResp, error) {
	if l.svcCtx.Registry == nil {
		return &aisolo.ListModesResp{}, nil
	}
	return &aisolo.ListModesResp{Modes: l.svcCtx.Registry.List()}, nil
}
