package logic

import (
	"context"

	"zero-service/app/gis/gis"

	"zero-service/app/gis/internal/svc"

	"github.com/paulmach/orb"
	"github.com/paulmach/orb/geo"
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

// Distance 计算两点间的大圆距离（Haversine 公式，单位：米）。
func (l *DistanceLogic) Distance(in *gis.DistanceReq) (*gis.DistanceRes, error) {
	if err := ValidatePoints(in.A, in.B); err != nil {
		return nil, err
	}
	a := orb.Point{in.A.Lon, in.A.Lat}
	b := orb.Point{in.B.Lon, in.B.Lat}
	distance := geo.Distance(a, b)
	return &gis.DistanceRes{
		Meters: distance,
	}, nil
}
