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
	"github.com/twpayne/go-geom"
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
	// 2. 构建多边形（校验有效性）
	var polygon orb.Polygon
	var err error
	if len(in.Points) > 0 {
		polygon, err = buildPolygonFromPoints(in.Points)
		if err != nil {
			l.Logger.Error("构建多边形失败: ", err)
			return nil, err
		}
	} else if in.FenceId != "" {
		// TODO: 从数据库/缓存加载多边形（示例逻辑）
		// polygon, err = l.loadPolygonByFenceId(in.FenceId)
		// if err != nil {
		// 	return nil, err
		// }
		return nil, errors.New("FenceId加载逻辑未实现")
	} else {
		return nil, errors.New("必须提供Points或有效的FenceId")
	}

	// 计算多边形边界框（用于遍历范围）
	bbox := polygon.Bound()
	latMin, latMax := bbox.Min.Y(), bbox.Max.Y()
	lonMin, lonMax := bbox.Min.X(), bbox.Max.X()
	l.Logger.Debugf("围栏边界框: 纬度[%v,%v], 经度[%v,%v]", latMin, latMax, lonMin, lonMax)

	// 初始化变量与步长计算（步长减半避免遗漏）
	geohashSet := make(map[string]struct{}, 1024) // 预设容量减少扩容
	centerLat := (latMin + latMax) / 2            // 用区域中心纬度计算步长更准确
	latStep, lonStep := geohashCellSize(precision, centerLat)
	latStep /= 2 // 步长减半，确保覆盖所有可能格子
	lonStep /= 2
	epsilon := 1e-8 // 浮点数精度补偿（约0.01米误差）

	// 遍历边界框生成候选geohash并过滤
	for lat := latMin; lat <= latMax+epsilon; lat += latStep {
		for lon := lonMin; lon <= lonMax+epsilon; lon += lonStep {
			// 生成当前点的geohash
			hash := geohash.EncodeWithPrecision(lat, lon, uint(precision))
			if len(hash) != precision {
				l.Logger.Errorf("无效geohash生成: %s（精度不匹配）", hash)
				continue
			}

			// 生成geohash格子的多边形（用于相交判断）
			box := geohash.BoundingBox(hash)
			cellOrb := orb.Polygon{
				orb.Ring{ // 直接构建ring，减少内存分配
					{box.MinLng, box.MinLat}, // 左下
					{box.MinLng, box.MaxLat}, // 左上
					{box.MaxLng, box.MaxLat}, // 右上
					{box.MaxLng, box.MinLat}, // 右下
					{box.MinLng, box.MinLat}, // 闭合
				},
			}

			// 精过滤：格子中心在多边形内 或 格子与多边形相交
			cLat, cLon := geohash.DecodeCenter(hash)
			isInside := planar.PolygonContains(polygon, orb.Point{cLon, cLat})
			isIntersect := PolygonIntersect(polygon, cellOrb)

			if isInside || isIntersect {
				geohashSet[hash] = struct{}{}
				l.Logger.Debugf("命中有效格子: %s（中心在内部: %v, 相交: %v）", hash, isInside, isIntersect)

				// 8. 处理邻居格子（过滤无效邻居）
				if in.IncludeNeighbors {
					for _, neighbor := range geohash.Neighbors(hash) {
						if len(neighbor) == precision { // 确保邻居精度匹配
							geohashSet[neighbor] = struct{}{}
						}
					}
				}
			}
		}
	}

	// 9. 转换结果并返回
	result := make([]string, 0, len(geohashSet))
	for h := range geohashSet {
		result = append(result, h)
	}
	l.Logger.Infof("生成围栏geohash完成，共%d个格子", len(result))

	return &geo.GenFenceCellsRes{
		Geohashes: result,
	}, nil
}

// 构建多边形并校验有效性
func buildPolygonFromPoints(points []*geo.Point) (orb.Polygon, error) {
	if len(points) < 3 {
		return nil, errors.New("多边形至少需要3个点")
	}

	var ring orb.Ring
	for _, p := range points {
		if p.Lon < -180 || p.Lon > 180 || p.Lat < -90 || p.Lat > 90 {
			return nil, errors.New("经纬度超出有效范围（经度-180~180，纬度-90~90）")
		}
		ring = append(ring, orb.Point{p.Lon, p.Lat})
	}

	// 确保多边形闭合（处理浮点数精度）
	first, last := ring[0], ring[len(ring)-1]
	if !(math.Abs(first[0]-last[0]) < 1e-8 && math.Abs(first[1]-last[1]) < 1e-8) {
		ring = append(ring, first)
	}

	return orb.Polygon{ring}, nil
}

// 将orb.Polygon转换为go-geom.Polygon（适配库函数）
func orbToGeomPolygon(orbPoly orb.Polygon) *geom.Polygon {
	if len(orbPoly) == 0 {
		return nil
	}

	// go-geom的多边形格式：外层是多边形，内层是环（每个环是[]Coord）
	geomRings := make([][]geom.Coord, 0, len(orbPoly))
	for _, ring := range orbPoly {
		geomRing := make([]geom.Coord, 0, len(ring))
		for _, point := range ring {
			// orb的点格式是 [lon, lat]，与go-geom的[x, y]一致
			geomRing = append(geomRing, geom.Coord{point[0], point[1]})
		}
		geomRings = append(geomRings, geomRing)
	}

	// 创建go-geom多边形（使用XY坐标类型，WGS84坐标系）
	return geom.NewPolygon(geom.XY).MustSetCoords(geomRings)
}

// 计算geohash格子尺寸（度）
func geohashCellSize(precision int, lat float64) (widthDeg, heightDeg float64) {
	latHeights := []float64{5000e3, 1250e3, 156e3, 39.1e3, 4.89e3, 1.22e3, 153, 38.2, 4.77, 1.19, 0.149, 0.0372}
	lonWidths := []float64{5000e3, 625e3, 156e3, 39.1e3, 4.89e3, 1.22e3, 153, 38.2, 4.77, 1.19, 0.149, 0.0372}

	latIdx := precision - 1
	heightDeg = latHeights[latIdx] / 110540                             // 纬度1度≈110.54km
	widthDeg = lonWidths[latIdx] / (111320 * math.Cos(lat*math.Pi/180)) // 经度1度随纬度变化

	return widthDeg, heightDeg
}

// ---------------------------------------------
// 线段相交判断
// ---------------------------------------------

// SegmentIntersect 判断两条线段是否相交（含端点接触与共线重叠）
func SegmentIntersect(a1, a2, b1, b2 orb.Point) bool {
	// 快速排除 --- bbox 不重叠
	if math.Max(a1.X(), a2.X()) < math.Min(b1.X(), b2.X()) ||
		math.Max(b1.X(), b2.X()) < math.Min(a1.X(), a2.X()) ||
		math.Max(a1.Y(), a2.Y()) < math.Min(b1.Y(), b2.Y()) ||
		math.Max(b1.Y(), b2.Y()) < math.Min(a1.Y(), a2.Y()) {
		return false
	}

	// 方向判断（叉积）
	d1 := cross(b1, b2, a1)
	d2 := cross(b1, b2, a2)
	d3 := cross(a1, a2, b1)
	d4 := cross(a1, a2, b2)

	// 一般相交（跨立相交）
	if (d1 > 0 && d2 < 0 || d1 < 0 && d2 > 0) &&
		(d3 > 0 && d4 < 0 || d3 < 0 && d4 > 0) {
		return true
	}

	// 特殊情况：共线
	if d1 == 0 && onSegment(b1, b2, a1) {
		return true
	}
	if d2 == 0 && onSegment(b1, b2, a2) {
		return true
	}
	if d3 == 0 && onSegment(a1, a2, b1) {
		return true
	}
	if d4 == 0 && onSegment(a1, a2, b2) {
		return true
	}

	return false
}

// 叉积 (b - a) × (c - a)
func cross(a, b, c orb.Point) float64 {
	return (b.X()-a.X())*(c.Y()-a.Y()) - (b.Y()-a.Y())*(c.X()-a.X())
}

// onSegment 判断点 c 是否在线段 ab 上（仅在保证共线后使用）
func onSegment(a, b, c orb.Point) bool {
	return c.X() >= math.Min(a.X(), b.X()) &&
		c.X() <= math.Max(a.X(), b.X()) &&
		c.Y() >= math.Min(a.Y(), b.Y()) &&
		c.Y() <= math.Max(a.Y(), b.Y())
}

// ---------------------------------------------
// Ring 相交判断
// ---------------------------------------------

// RingIntersect 判断两个 ring 的边界是否相交
func RingIntersect(r1, r2 orb.Ring) bool {
	n1 := len(r1)
	n2 := len(r2)

	for i := 0; i < n1; i++ {
		a1 := r1[i]
		a2 := r1[(i+1)%n1]

		for j := 0; j < n2; j++ {
			b1 := r2[j]
			b2 := r2[(j+1)%n2]

			if SegmentIntersect(a1, a2, b1, b2) {
				return true
			}
		}
	}
	return false
}

// ---------------------------------------------
// Polygon 相交判断（包含 + 边界）
// ---------------------------------------------

// PolygonIntersect 判断两个多边形是否相交
// p1、p2 都为 orb.Polygon（外圈 r[0]）
func PolygonIntersect(p1, p2 orb.Polygon) bool {
	r1 := p1[0]
	r2 := p2[0]

	// 1. 顶点包含（内部包含情况）
	for _, pt := range r1 {
		if planar.PolygonContains(p2, pt) {
			return true
		}
	}
	for _, pt := range r2 {
		if planar.PolygonContains(p1, pt) {
			return true
		}
	}

	// 2. 边界相交（交叉情况）
	if RingIntersect(r1, r2) {
		return true
	}

	return false
}
