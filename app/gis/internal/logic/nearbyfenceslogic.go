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

// NearbyFences 先用 H3 查询附近 km 范围内的候选围栏，再用围栏多边形精确过滤命中点。
// 依赖 FenceStore 的空间索引实现；若 store 不可用则返回空结果。
func (l *NearbyFencesLogic) NearbyFences(in *gis.NearbyFencesReq) (*gis.NearbyFencesRes, error) {
	if err := ValidatePoints(in.Point); err != nil {
		return nil, err
	}
	if in.Km <= 0 {
		return nil, tool.NewErrorByPbCode(extproto.Code__1_01_PARAM_INVALID, "km必须大于0")
	}

	fenceIds, err := l.svcCtx.FenceStore.FindNearbyFenceIds(l.ctx, in.Point.Lon, in.Point.Lat, in.Km)
	if err != nil {
		l.Logger.Errorf("FenceStore.FindNearbyFenceIds 失败: %v", err)
		return nil, err
	}

	point := orb.Point{in.Point.Lon, in.Point.Lat}
	hitFenceIds := make([]string, 0, len(fenceIds))
	for _, fenceId := range fenceIds {
		polygon, err := l.svcCtx.FenceStore.LoadFencePolygon(l.ctx, fenceId)
		if err != nil {
			l.Logger.Errorf("加载围栏多边形失败, fenceId=%s, err=%v", fenceId, err)
			continue
		}
		if planar.PolygonContains(polygon, point) {
			hitFenceIds = append(hitFenceIds, fenceId)
		}
	}

	return &gis.NearbyFencesRes{FenceIds: hitFenceIds}, nil
}
