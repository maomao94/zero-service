package logic

import (
	"context"
	"errors"

	"zero-service/app/geo/geo"
	"zero-service/app/geo/internal/svc"

	"github.com/mmcloughlin/geohash"
	"github.com/paulmach/orb"
	"github.com/paulmach/orb/planar"
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

// 一次性生成围栏 cells（小围栏）
func (l *GenerateFenceCellsLogic) GenerateFenceCells(in *geo.GenFenceCellsReq) (*geo.GenFenceCellsRes, error) {
	// 1. polygon 转 orb.Polygon
	var polygon orb.Polygon
	if len(in.Points) > 0 {
		var ring orb.Ring
		for _, p := range in.Points {
			ring = append(ring, orb.Point{p.Lon, p.Lat}) // orb 使用 (lon, lat)
		}
		polygon = orb.Polygon{ring}
	} else if in.FenceId != "" {
		// TODO: 从数据库或内存加载 polygon
		return nil, errors.New("参数错误")
	} else {
		return nil, errors.New("参数错误")
	}
	// 2. 计算 polygon bbox
	bbox := polygon.Bound()
	latMin := bbox.Min.Y()
	latMax := bbox.Max.Y()
	lonMin := bbox.Min.X()
	lonMax := bbox.Max.X()
	// 3. 计算步长（粗略，10 等分 bbox）
	geohashSet := make(map[string]struct{})
	latStep := (latMax - latMin) / 10.0
	lonStep := (lonMax - lonMin) / 10.0

	// 4. 遍历 bbox 生成 candidate geohash
	for lat := latMin; lat <= latMax; lat += latStep {
		for lon := lonMin; lon <= lonMax; lon += lonStep {
			hash := geohash.EncodeWithPrecision(lat, lon, uint(in.Precision))
			// 5. 精过滤：格子中心点在 polygon 内
			cLat, cLon := geohash.DecodeCenter(hash)
			if planar.PolygonContains(polygon, orb.Point{cLon, cLat}) {
				geohashSet[hash] = struct{}{}
				// 6. includeNeighbors
				if in.IncludeNeighbors {
					for _, n := range geohash.Neighbors(hash) {
						geohashSet[n] = struct{}{}
					}
				}
			}
		}
	}

	// 7. 去重，转换为切片
	result := make([]string, 0, len(geohashSet))
	for h := range geohashSet {
		result = append(result, h)
	}

	return &geo.GenFenceCellsRes{
		Geohashes: result,
	}, nil
}
