package logic

import (
	"context"

	"zero-service/app/geo/geo"
	"zero-service/app/geo/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type PointInFencesLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewPointInFencesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *PointInFencesLogic {
	return &PointInFencesLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 点是否命中电子围栏（多个围栏）
func (l *PointInFencesLogic) PointInFences(in *geo.PointInFencesReq) (*geo.PointInFencesRes, error) {
	// todo: add your logic here and delete this line

	return &geo.PointInFencesRes{}, nil
}
