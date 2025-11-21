package logic

import (
	"context"
	"zero-service/app/gis/gis"

	"zero-service/app/gis/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type PointInFenceLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewPointInFenceLogic(ctx context.Context, svcCtx *svc.ServiceContext) *PointInFenceLogic {
	return &PointInFenceLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 点是否命中电子围栏（单个）
func (l *PointInFenceLogic) PointInFence(in *gis.PointInFenceReq) (*gis.PointInFenceRes, error) {
	// todo: add your logic here and delete this line

	return &gis.PointInFenceRes{}, nil
}
