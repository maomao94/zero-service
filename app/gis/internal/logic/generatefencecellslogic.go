package logic

import (
	"context"

	"zero-service/app/gis/gis"

	"zero-service/app/gis/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type GenerateFenceCellsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGenerateFenceCellsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GenerateFenceCellsLogic {
	return &GenerateFenceCellsLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// GenerateFenceCells 生成覆盖围栏多边形的 geohash cells。
// 委托 scanGeohashCells 执行核心扫描。
func (l *GenerateFenceCellsLogic) GenerateFenceCells(in *gis.GenFenceCellsReq) (*gis.GenFenceCellsRes, error) {
	precision, err := resolveGeohashPrecision(in.Precision)
	if err != nil {
		return nil, err
	}

	polygon, err := pbPolygonToOrbPolygon(in.Polygon)
	if err != nil {
		l.Logger.Error("构建多边形失败: ", err)
		return nil, err
	}

	geohashSet, err := scanGeohashCells(polygon, precision, in.IncludeNeighbors)
	if err != nil {
		l.Logger.Errorf("扫描 geohash cells 失败: %v", err)
		return nil, err
	}

	result := make([]string, 0, len(geohashSet))
	for h := range geohashSet {
		result = append(result, h)
	}

	l.Logger.Infof("生成围栏geohash完成，共%d个格子", len(result))
	return &gis.GenFenceCellsRes{
		Geohashes: result,
	}, nil
}
