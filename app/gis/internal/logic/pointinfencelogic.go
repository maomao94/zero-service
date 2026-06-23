package logic

import (
	"context"
	"zero-service/app/gis/gis"

	"zero-service/app/gis/internal/svc"
	"zero-service/common/tool"
	"zero-service/third_party/extproto"

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

// PointInFence 判断点是否在单个电子围栏内（射线法 point-in-polygon）。
// 围栏来源：优先使用请求中的顶点列表，其次按 fence_id 从 store 加载。
func (l *PointInFenceLogic) PointInFence(in *gis.PointInFenceReq) (*gis.PointInFenceRes, error) {
	var err error
	err = ValidatePoints(in.Point)
	if err != nil {
		return nil, err
	}
	if in.Fence == nil {
		return nil, tool.NewErrorByPbCode(extproto.Code__1_01_PARAM_MISSING, "fence")
	}
	var polygon orb.Polygon
	if len(in.Fence.Points) > 0 {
		polygon, err = pbPointToOrbPolygon(in.Fence.Points)
		if err != nil {
			l.Logger.Error("构建多边形失败: ", err)
			return nil, err
		}
	} else if in.Fence.FenceId != "" {
		polygon, err = l.svcCtx.FenceStore.LoadFencePolygon(l.ctx, in.Fence.FenceId)
		if err != nil {
			l.Logger.Errorf("加载围栏多边形失败, fenceId=%s, err=%v", in.Fence.FenceId, err)
			return nil, err
		}
	} else {
		return nil, tool.NewErrorByPbCode(extproto.Code__1_01_PARAM_MISSING, "Points或FenceId")
	}
	point := orb.Point{in.Point.Lon, in.Point.Lat}
	hit := planar.PolygonContains(polygon, point)
	return &gis.PointInFenceRes{
		Hit: hit,
	}, nil
}
