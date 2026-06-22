package logic

import (
	"context"

	"zero-service/app/gis/gis"
	"zero-service/app/gis/internal/svc"
	"zero-service/common/tool"
	"zero-service/third_party/extproto"

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

// PointsWithinRadius 筛选出距中心点距离 ≤ radius_meters 的所有点，返回命中点的索引和距离。
func (l *PointsWithinRadiusLogic) PointsWithinRadius(in *gis.PointsWithinRadiusReq) (*gis.PointsWithinRadiusRes, error) {
	if err := ValidatePoints(append([]*gis.Point{in.Center}, in.Points...)...); err != nil {
		return nil, err
	}
	if in.RadiusMeters <= 0 {
		return nil, tool.NewErrorByPbCode(extproto.Code__1_01_PARAM_INVALID, "radius_meters必须大于0")
	}
	center := orb.Point{in.Center.Lon, in.Center.Lat}
	hits := make([]*gis.RadiusHit, 0)
	for i, p := range in.Points {
		orbP := orb.Point{p.Lon, p.Lat}
		distance := geo.Distance(center, orbP)
		if distance <= in.RadiusMeters {
			hits = append(hits, &gis.RadiusHit{
				Index:           int32(i),
				DistanceMeters:  distance,
			})
		}
	}
	return &gis.PointsWithinRadiusRes{
		Hits: hits,
	}, nil
}
