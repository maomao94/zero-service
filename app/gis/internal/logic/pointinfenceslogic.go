package logic

import (
	"context"
	"strconv"
	"zero-service/app/gis/gis"
	"zero-service/app/gis/internal/svc"

	"github.com/paulmach/orb"
	"github.com/paulmach/orb/planar"
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
func (l *PointInFencesLogic) PointInFences(in *gis.PointInFencesReq) (*gis.PointInFencesRes, error) {
	if err := ValidatePoints(in.Point); err != nil {
		return nil, err
	}

	hitFenceIds := make([]string, 0)
	point := orb.Point{in.Point.Lon, in.Point.Lat}

	for key, fence := range in.Fences {
		if len(fence.Points) == 0 && fence.Id == "" {
			continue
		}

		var polygon orb.Polygon
		var err error
		if len(fence.Points) > 0 {
			polygon, err = pbPointToOrbPolygon(fence.Points)
			if err != nil {
				l.Logger.Error("构建多边形失败, fenceId=", fence.Id, err)
				continue
			}
		} else if fence.Id != "" {
			// TODO: 从数据库/缓存加载多边形
			continue
		}

		if planar.PolygonContains(polygon, point) {
			if len(fence.Id) == 0 {
				fence.Id = strconv.Itoa(key)
			}
			hitFenceIds = append(hitFenceIds, fence.Id)
		}
	}

	return &gis.PointInFencesRes{
		HitFenceIds: hitFenceIds,
	}, nil
}
