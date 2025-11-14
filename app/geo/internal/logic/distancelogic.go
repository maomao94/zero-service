package logic

import (
	"context"

	"zero-service/app/geo/geo"
	"zero-service/app/geo/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type DistanceLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewDistanceLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DistanceLogic {
	return &DistanceLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 计算两个点之间的距离（米）
func (l *DistanceLogic) Distance(in *geo.DistanceReq) (*geo.DistanceRes, error) {
	// todo: add your logic here and delete this line

	return &geo.DistanceRes{}, nil
}
