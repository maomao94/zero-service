package logic

import (
	"context"

	"zero-service/app/gis/gis"
	"zero-service/app/gis/internal/svc"
	"zero-service/common/gisx"
	"zero-service/common/tool"
	"zero-service/third_party/extproto"

	"github.com/paulmach/orb"
	"github.com/uber/h3-go/v4"
	"github.com/zeromicro/go-zero/core/logx"
)

type CreateFenceLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewCreateFenceLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateFenceLogic {
	return &CreateFenceLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// CreateFence 新增电子围栏。
// 流程：校验参数 → 构建多边形 → 计算 H3 cells → 计算 geohash cells → 生成 ID → 持久化。
func (l *CreateFenceLogic) CreateFence(in *gis.CreateFenceReq) (*gis.CreateFenceRes, error) {
	if err := ValidatePoints(in.Points...); err != nil {
		return nil, err
	}
	if len(in.Points) < 3 {
		return nil, tool.NewErrorByPbCode(extproto.Code__1_01_PARAM, "多边形至少需要3个点")
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

	fenceId, err := tool.SimpleUUID()
	if err != nil {
		return nil, err
	}

	orbPoints := make([]orb.Point, len(in.Points))
	for i, p := range in.Points {
		orbPoints[i] = orb.Point{p.Lon, p.Lat}
	}

	if err := l.svcCtx.FenceStore.CreateFence(l.ctx, fenceId, in.Name, orbPoints, resolution, geohashPrecision, cellStrings, geohashes); err != nil {
		l.Logger.Errorf("创建围栏失败, err=%v", err)
		return nil, err
	}

	return &gis.CreateFenceRes{
		FenceId:   fenceId,
		H3Cells:   cellStrings,
		Geohashes: geohashes,
	}, nil
}
