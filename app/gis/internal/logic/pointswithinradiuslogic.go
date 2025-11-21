package logic

import (
	"context"

	"zero-service/app/gis/gis"
	"zero-service/app/gis/internal/svc"

	"github.com/paulmach/orb"
	"github.com/paulmach/orb/geo"
	"github.com/zeromicro/go-zero/core/logx"
)

type PointsWithinRadiusLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewPointsWithinRadiusLogic(ctx context.Context, svcCtx *svc.ServiceContext) *PointsWithinRadiusLogic {
	return &PointsWithinRadiusLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 获取半径内的点
func (l *PointsWithinRadiusLogic) PointsWithinRadius(in *gis.PointsWithinRadiusReq) (*gis.PointsWithinRadiusRes, error) {
	// 校验中心点和点列表
	if err := ValidatePoints(append([]*gis.Point{in.Center}, in.Points...)...); err != nil {
		return nil, err
	}
	center := orb.Point{in.Center.Lon, in.Center.Lat}
	hitIndexs := make([]int32, 0)
	for i, p := range in.Points {
		orbP := orb.Point{p.Lon, p.Lat}
		distance := geo.Distance(center, orbP)
		if distance <= in.RadiusMeters {
			hitIndexs = append(hitIndexs, int32(i))
		}
	}
	return &gis.PointsWithinRadiusRes{
		HitIndexes: hitIndexs,
	}, nil
}
