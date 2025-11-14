package logic

import (
	"context"

	"zero-service/app/geo/geo"
	"zero-service/app/geo/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type NearbyFencesLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewNearbyFencesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *NearbyFencesLogic {
	return &NearbyFencesLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 获取某点附近多少 km 的围栏（粗过滤）
func (l *NearbyFencesLogic) NearbyFences(in *geo.NearbyFencesReq) (*geo.NearbyFencesRes, error) {
	// todo: add your logic here and delete this line

	return &geo.NearbyFencesRes{}, nil
}
