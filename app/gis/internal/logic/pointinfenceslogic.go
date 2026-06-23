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

// PointInFences 批量判断点是否命中多个围栏，返回所有命中的围栏 ID。
// 逐围栏执行 point-in-polygon 检测，单个围栏加载失败时跳过并继续。
func (l *PointInFencesLogic) PointInFences(in *gis.PointInFencesReq) (*gis.PointInFencesRes, error) {
	if err := ValidatePoints(in.Point); err != nil {
		return nil, err
	}

	hitFenceIds := make([]string, 0)
	point := orb.Point{in.Point.Lon, in.Point.Lat}

	for key, fence := range in.Fences {
		if len(fence.Points) == 0 && fence.FenceId == "" {
			continue
		}

		var polygon orb.Polygon
		var err error
		if len(fence.Points) > 0 {
			polygon, err = pbPointToOrbPolygon(fence.Points)
			if err != nil {
				l.Logger.Error("构建多边形失败, fenceId=", fence.FenceId, err)
				continue
			}
		} else if fence.FenceId != "" {
			polygon, err = l.svcCtx.FenceStore.LoadFencePolygon(l.ctx, fence.FenceId)
			if err != nil {
				l.Logger.Errorf("加载围栏多边形失败, fenceId=%s, err=%v", fence.FenceId, err)
				continue
			}
		}

		if planar.PolygonContains(polygon, point) {
			if len(fence.FenceId) == 0 {
				fence.FenceId = strconv.Itoa(key)
			}
			hitFenceIds = append(hitFenceIds, fence.FenceId)
		}
	}

	return &gis.PointInFencesRes{
		HitFenceIds: hitFenceIds,
	}, nil
}
