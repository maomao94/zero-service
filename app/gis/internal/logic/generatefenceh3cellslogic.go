package logic

import (
	"context"
	"errors"
	"zero-service/app/gis/gis"
	"zero-service/app/gis/internal/svc"
	"zero-service/common/gisx"

	"github.com/paulmach/orb"
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

// 一次性生成围栏 H3 cells（小围栏）
func (l *GenerateFenceH3CellsLogic) GenerateFenceH3Cells(in *gis.GenFenceH3CellsReq) (*gis.GenFenceH3CellsRes, error) {
	var polygon orb.Polygon
	var err error

	if len(in.Points) > 0 {
		polygon, err = pbPointToOrbPolygon(in.Points)
		if err != nil {
			l.Logger.Error("构建多边形失败: ", err)
			return nil, err
		}
	} else if in.FenceId != "" {
		// TODO: 从数据库/缓存加载 polygon
		return nil, errors.New("FenceId加载逻辑未实现")
	} else {
		return nil, errors.New("必须提供Points或有效的FenceId")
	}

	resolution := in.Resolution
	if resolution <= 0 {
		resolution = 9 // 使用默认分辨率9
	}

	// 验证分辨率范围
	if resolution < 0 || resolution > 15 {
		return nil, errors.New("H3分辨率必须在0-15之间")
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
	// 返回结果
	return &gis.GenFenceH3CellsRes{
		H3Indexes: cellStrings,
	}, nil
}
