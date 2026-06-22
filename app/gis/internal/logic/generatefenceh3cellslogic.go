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

type GenerateFenceH3CellsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGenerateFenceH3CellsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GenerateFenceH3CellsLogic {
	return &GenerateFenceH3CellsLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// GenerateFenceH3Cells 生成覆盖围栏多边形的 H3 六边形索引。
// 算法：将多边形转换为 H3 GeoPolygon，调用 PolygonToCellsExperimental 获取所有重叠 cell。
func (l *GenerateFenceH3CellsLogic) GenerateFenceH3Cells(in *gis.GenFenceH3CellsReq) (*gis.GenFenceH3CellsRes, error) {
	// 本接口为纯计算，仅支持请求中直接传入顶点
	if len(in.Points) < 3 {
		return nil, tool.NewErrorByPbCode(extproto.Code__1_01_PARAM_MISSING, "围栏至少需要3个顶点")
	}
	polygon, err := pbPointToOrbPolygon(in.Points)
	if err != nil {
		l.Logger.Error("构建多边形失败: ", err)
		return nil, err
	}

	resolution := in.Resolution
	if resolution == 0 {
		resolution = 9
	}

	if resolution > 15 {
		return nil, tool.NewErrorByPbCode(extproto.Code__1_01_PARAM, "H3分辨率必须在0-15之间")
	}

	geoPolygon, err := gisx.OrbPolygonToH3GeoPolygon(polygon)
	if err != nil {
		l.Logger.Error("多边形格式转换失败: ", err)
		return nil, err
	}

	cell, err := h3.PolygonToCellsExperimental(geoPolygon, int(resolution), h3.ContainmentOverlapping, 1000)
	if err != nil {
		l.Logger.Error("生成H3 cells失败: ", err)
		return nil, err
	}

	cellStrings := make([]string, len(cell))
	for i, c := range cell {
		cellStrings[i] = c.String()
	}

	return &gis.GenFenceH3CellsRes{
		H3Indexes: cellStrings,
	}, nil
}
