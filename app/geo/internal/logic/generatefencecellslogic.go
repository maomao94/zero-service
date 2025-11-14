package logic

import (
	"context"
	"errors"
	"math"

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
	// 默认精度 7
	precision := int(in.Precision)
	if precision <= 0 {
		precision = 7
	}
	// polygon 转 orb.Polygon
	var polygon orb.Polygon
	if len(in.Points) > 0 {
		var ring orb.Ring
		for _, p := range in.Points {
			ring = append(ring, orb.Point{p.Lon, p.Lat}) // orb 使用 (lon, lat)
		}
		// 检查闭合：首尾点是否相等
		if len(ring) > 0 && (ring[0][0] != ring[len(ring)-1][0] || ring[0][1] != ring[len(ring)-1][1]) {
			ring = append(ring, ring[0])
		}
		polygon = orb.Polygon{ring}
	} else if in.FenceId != "" {
		// TODO: 从数据库或内存加载 polygon
		return nil, errors.New("参数错误")
	} else {
		return nil, errors.New("参数错误")
	}
	// 计算 polygon bbox
	bbox := polygon.Bound()
	latMin := bbox.Min.Y()
	latMax := bbox.Max.Y()
	lonMin := bbox.Min.X()
	lonMax := bbox.Max.X()
	// 计算步长（粗略，10 等分 bbox）
	geohashSet := make(map[string]struct{})
	latStep, lonStep := geohashCellSize(precision, (latMin+latMax)/2)
	// 遍历 bbox 生成 candidate geohash
	for lat := latMin; lat <= latMax; lat += latStep {
		for lon := lonMin; lon <= lonMax; lon += lonStep {
			hash := geohash.EncodeWithPrecision(lat, lon, uint(precision))
			// 生成格子 polygon（四角 + 闭合）
			box := geohash.BoundingBox(hash)
			cell := orb.Polygon{{
				{box.MinLng, box.MinLat}, // 左下
				{box.MinLng, box.MaxLat}, // 左上
				{box.MaxLng, box.MaxLat}, // 右上
				{box.MaxLng, box.MinLat}, // 右下
				{box.MinLng, box.MinLat}, // 闭合
			}}
			cellBound := cell.Bound()
			// 精过滤：格子与 polygon 相交
			cLat, cLon := geohash.DecodeCenter(hash)
			if planar.PolygonContains(polygon, orb.Point{cLon, cLat}) {
				geohashSet[hash] = struct{}{}
				// includeNeighbors
				if in.IncludeNeighbors {
					for _, n := range geohash.Neighbors(hash) {
						geohashSet[n] = struct{}{}
					}
				}
			} else {
				if bBoxIntersect(cellBound, bbox) {
					// 包含
					geohashSet[hash] = struct{}{}
					// includeNeighbors
					if in.IncludeNeighbors {
						for _, n := range geohash.Neighbors(hash) {
							geohashSet[n] = struct{}{}
						}
					}
				}
			}
		}
	}

	result := make([]string, 0, len(geohashSet))
	for h := range geohashSet {
		result = append(result, h)
	}

	return &geo.GenFenceCellsRes{
		Geohashes: result,
	}, nil
}

func bBoxIntersect(b1, b2 orb.Bound) bool {
	if b1.Max[1] < b2.Min[1] || b1.Min[1] > b2.Max[1] { // 纬度
		return false
	}
	if b1.Max[0] < b2.Min[0] || b1.Min[0] > b2.Max[0] { // 经度
		return false
	}
	return true
}

// geohashCellSize 返回给定精度 geohash 格子的大约宽度和高度（单位：米）
func geohashCellSize(precision int, lat float64) (widthM, heightM float64) {
	// 每个 geohash 精度的格子大小，纬度方向大致固定，精度7大约150m
	// 这里只列出常用精度的参考值（米）
	// 精度 1 ~ 12
	latHeight := []float64{
		5000e3, 1250e3, 156e3, 39.1e3, 4.89e3, 1.22e3,
		153, 38.2, 4.77, 1.19, 0.149, 0.0372,
	}
	lonWidth := []float64{
		5000e3, 625e3, 156e3, 39.1e3, 4.89e3, 1.22e3,
		153, 38.2, 4.77, 1.19, 0.149, 0.0372,
	}

	if precision < 1 {
		precision = 1
	} else if precision > 12 {
		precision = 12
	}

	// 经度米数换成度数，纬度米数换成度数
	widthDeg := lonWidth[precision-1] / (111320 * math.Cos(lat*math.Pi/180))
	heightDeg := latHeight[precision-1] / 110540 // 纬度 1度大约 110.54 km

	return widthDeg, heightDeg
}
