package logic

import (
	"context"

	"zero-service/app/gis/gis"
	"zero-service/app/gis/internal/svc"
	"zero-service/common/tool"
	"zero-service/third_party/extproto"

	"github.com/zeromicro/go-zero/core/logx"
)

type UpdateFenceLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewUpdateFenceLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateFenceLogic {
	return &UpdateFenceLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// UpdateFence 更新电子围栏多边形及索引。
// 流程：校验参数 → 重建多边形 → 重算 H3 + geohash cells → 覆盖写入 store。
func (l *UpdateFenceLogic) UpdateFence(in *gis.UpdateFenceReq) (*gis.UpdateFenceRes, error) {
	if in.FenceId == "" {
		return nil, tool.NewErrorByPbCode(extproto.Code__1_01_PARAM_MISSING, "fenceId")
	}

	polygon, err := pbPolygonToOrbPolygon(in.Polygon)
	if err != nil {
		return nil, err
	}

	resolution, err := resolveH3Resolution(in.H3Resolution)
	if err != nil {
		return nil, err
	}
	geohashPrecision, err := resolveGeohashPrecision(in.GeohashPrecision)
	if err != nil {
		return nil, err
	}

	cellStrings, geohashes, err := computeFenceCells(polygon, resolution, geohashPrecision)
	if err != nil {
		return nil, err
	}

	if err := l.svcCtx.FenceStore.UpdateFence(l.ctx, in.FenceId, in.Name, polygon, resolution, geohashPrecision, cellStrings, geohashes); err != nil {
		l.Logger.Errorf("更新围栏失败, fenceId=%s, err=%v", in.FenceId, err)
		return nil, err
	}

	return &gis.UpdateFenceRes{
		H3Cells:   cellStrings,
		Geohashes: geohashes,
	}, nil
}
