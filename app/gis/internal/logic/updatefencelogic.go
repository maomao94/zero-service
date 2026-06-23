package logic

import (
	"context"

	"zero-service/app/gis/gis"
	"zero-service/app/gis/internal/svc"
	"zero-service/common/gisx"
	"zero-service/common/tool"
	"zero-service/third_party/extproto"

	"github.com/uber/h3-go/v4"
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
// 流程：校验参数 → 重建多边形 → 重算 H3 cells + geohash cells → 覆盖写入 store。
func (l *UpdateFenceLogic) UpdateFence(in *gis.UpdateFenceReq) (*gis.UpdateFenceRes, error) {
	if in.FenceId == "" {
		return nil, tool.NewErrorByPbCode(extproto.Code__1_01_PARAM_MISSING, "fenceId")
	}
	if len(in.Points) < 3 {
		return nil, tool.NewErrorByPbCode(extproto.Code__1_01_PARAM, "多边形至少需要3个点")
	}
	if err := ValidatePoints(in.Points...); err != nil {
		return nil, err
	}

	polygon, err := pbPointToOrbPolygon(in.Points)
	if err != nil {
		return nil, err
	}

	resolution := int(in.H3Resolution)
	if resolution <= 0 {
		resolution = 9
	} else if resolution > 15 {
		return nil, tool.NewErrorByPbCode(extproto.Code__1_01_PARAM, "H3分辨率必须在0-15之间")
	}

	geoPolygon, err := gisx.OrbPolygonToH3GeoPolygon(polygon)
	if err != nil {
		return nil, err
	}

	cells, err := h3.PolygonToCellsExperimental(geoPolygon, resolution, h3.ContainmentOverlapping, 1000)
	if err != nil {
		return nil, err
	}

	cellStrings := make([]string, len(cells))
	for i, c := range cells {
		cellStrings[i] = c.String()
	}

	geohashPrecision := int(in.GeohashPrecision)
	if geohashPrecision <= 0 {
		geohashPrecision = 7
	} else if geohashPrecision > 12 {
		return nil, tool.NewErrorByPbCode(extproto.Code__1_01_PARAM, "geohash精度最大为12")
	}
	geohashes := computeGeohashCells(polygon, geohashPrecision)

	if err := l.svcCtx.FenceStore.UpdateFence(l.ctx, in.FenceId, in.Name, polygon, resolution, geohashPrecision, cellStrings, geohashes); err != nil {
		l.Logger.Errorf("更新围栏失败, fenceId=%s, err=%v", in.FenceId, err)
		return nil, err
	}

	return &gis.UpdateFenceRes{
		H3Cells:   cellStrings,
		Geohashes: geohashes,
	}, nil
}
