package logic

import (
	"context"
	"errors"
	"zero-service/app/gis/gis"

	"zero-service/app/gis/internal/svc"

	"github.com/paulmach/orb"
	"github.com/paulmach/orb/planar"
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
	var err error
	var polygon orb.Polygon
	if len(in.Fence.Points) > 0 {
		polygon, err = pbPointToOrbPolygon(in.Fence.Points)
		if err != nil {
			l.Logger.Error("构建多边形失败: ", err)
			return nil, err
		}
	} else if in.Fence.Id != "" {
		// TODO: 从数据库/缓存加载多边形（示例逻辑）
		// polygon, err = l.loadPolygonByFenceId(in.FenceId)
		// if err != nil {
		// 	return nil, err
		// }
		return nil, errors.New("FenceId加载逻辑未实现")
	} else {
		return nil, errors.New("必须提供Points或有效的FenceId")
	}
	point := orb.Point{in.Point.Lon, in.Point.Lat}
	hit := planar.PolygonContains(polygon, point)
	return &gis.PointInFenceRes{
		Hit: hit,
	}, nil
}
