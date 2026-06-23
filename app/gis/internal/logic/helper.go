package logic

import (
	"fmt"
	"math"

	"zero-service/app/gis/gis"
	"zero-service/common/gisx"
	"zero-service/common/gisx/geos/orbconv"
	"zero-service/common/tool"
	"zero-service/third_party/extproto"

	"github.com/mmcloughlin/geohash"
	"github.com/paulmach/orb"
	"github.com/paulmach/orb/planar"
	"github.com/uber/h3-go/v4"
)

// ValidateH3Resolution 校验 H3 分辨率，返回 int 类型或 nil。
func ValidateH3Resolution(resolution uint32) (int, error) {
	r := int(resolution)
	if r > 15 {
		return 0, tool.NewErrorByPbCode(extproto.Code__1_01_PARAM, "H3分辨率必须为0-15")
	}
	return r, nil
}

// ValidateGeoHashPrecision 校验 geohash 精度，返回 int 类型或 nil。
func ValidateGeoHashPrecision(precision uint32) (int, error) {
	p := int(precision)
	if p < 1 || p > 12 {
		return 0, tool.NewErrorByPbCode(extproto.Code__1_01_PARAM, "geohash精度必须为1-12")
	}
	return p, nil
}

// resolveH3Resolution 校验 H3 分辨率，0 时默认 9。
// 注意：proto3 中 uint32 默认值为 0，无法区分"用户未设置"和"用户传 0"。
// H3 分辨率 0 是合法值（全球约 122 cells），若需支持传 0 请改用 proto optional 或 wrapper 类型。
func resolveH3Resolution(r uint32) (int, error) {
	resolution := int(r)
	if resolution == 0 {
		resolution = 9
	} else if resolution > 15 {
		return 0, tool.NewErrorByPbCode(extproto.Code__1_01_PARAM, "H3分辨率必须在0-15之间")
	}
	return resolution, nil
}

// resolveGeohashPrecision 校验 geohash 精度，<=0 时默认 7。
func resolveGeohashPrecision(p uint32) (int, error) {
	precision := int(p)
	if precision <= 0 {
		precision = 7
	} else if precision > 12 {
		return 0, tool.NewErrorByPbCode(extproto.Code__1_01_PARAM, "geohash精度最大为12")
	}
	return precision, nil
}

// computeFenceCells 计算多边形覆盖的 H3 cells + geohash cells。
func computeFenceCells(polygon orb.Polygon, h3Resolution, geohashPrecision int) (h3CellStrings []string, geohashes []string, err error) {
	geoPolygon, err := gisx.OrbPolygonToH3GeoPolygon(polygon)
	if err != nil {
		return nil, nil, err
	}
	cells, err := h3.PolygonToCellsExperimental(geoPolygon, h3Resolution, h3.ContainmentOverlapping, 1000)
	if err != nil {
		return nil, nil, err
	}
	h3CellStrings = make([]string, len(cells))
	for i, c := range cells {
		h3CellStrings[i] = c.String()
	}
	geohashes = computeGeohashCells(polygon, geohashPrecision)
	return h3CellStrings, geohashes, nil
}

// scanGeohashCells 扫描多边形 bbox，收集所有被覆盖的 geohash cells（核心算法）。
// includeNeighbors：是否扩展 8 邻域。返回去重的 geohash 集合。
func scanGeohashCells(polygon orb.Polygon, precision int, includeNeighbors bool) (map[string]struct{}, error) {
	bbox := polygon.Bound()
	latMin, latMax := bbox.Min.Y(), bbox.Max.Y()
	lonMin, lonMax := bbox.Min.X(), bbox.Max.X()

	geohashSet := make(map[string]struct{}, 1024)
	centerLat := (latMin + latMax) / 2
	lonStep, latStep := geohashCellSize(precision, centerLat)
	latStep /= 2
	lonStep /= 2
	epsilon := 1e-8

	for lat := latMin; lat <= latMax+epsilon; lat += latStep {
		for lon := lonMin; lon <= lonMax+epsilon; lon += lonStep {
			hash := geohash.EncodeWithPrecision(lat, lon, uint(precision))
			if len(hash) != precision {
				continue
			}

			box := geohash.BoundingBox(hash)
			cellOrb := orb.Polygon{
				orb.Ring{
					{box.MinLng, box.MinLat},
					{box.MinLng, box.MaxLat},
					{box.MaxLng, box.MaxLat},
					{box.MaxLng, box.MinLat},
					{box.MinLng, box.MinLat},
				},
			}

			cLat, cLon := geohash.DecodeCenter(hash)
			isInside := planar.PolygonContains(polygon, orb.Point{cLon, cLat})
			isIntersect, err := orbconv.IntersectsOrb(polygon, cellOrb)
			if err != nil {
				return nil, fmt.Errorf("GEOS intersect 失败: %w", err)
			}

			if isInside || isIntersect {
				geohashSet[hash] = struct{}{}
			}
		}
	}

	if includeNeighbors {
		for hash := range geohashSet {
			for _, neighbor := range geohash.Neighbors(hash) {
				if len(neighbor) == precision {
					geohashSet[neighbor] = struct{}{}
				}
			}
		}
	}

	return geohashSet, nil
}

// computeGeohashCells 计算覆盖多边形的 geohash cells（不含邻居扩展）。
func computeGeohashCells(polygon orb.Polygon, precision int) []string {
	result, err := scanGeohashCells(polygon, precision, false)
	if err != nil {
		return nil
	}
	out := make([]string, 0, len(result))
	for h := range result {
		out = append(out, h)
	}
	return out
}

// geohashCellSize 根据 geohash 位划分精确计算单个格子的经纬度跨度（单位：度）。
// 返回值：widthDeg 为经度方向跨度，heightDeg 为纬度方向跨度。
func geohashCellSize(precision int, _ float64) (widthDeg, heightDeg float64) {
	totalBits := precision * 5
	lonBits := (totalBits + 1) / 2
	latBits := totalBits / 2
	return 360 / math.Pow(2, float64(lonBits)), 180 / math.Pow(2, float64(latBits))
}

// validateCoordType 校验坐标系类型，合法值为 1(WGS84)、2(GCJ02)、3(BD09)。
func validateCoordType(t gis.CoordType) error {
	val := uint32(t)
	if val < 1 || val > 3 {
		return tool.NewErrorByPbCode(extproto.Code__1_01_PARAM_INVALID,
			fmt.Sprintf("invalid coord type: %v (only support 1=WGS84, 2=GCJ02, 3=BD09)", val))
	}
	return nil
}

// EncodeH3Cell 将经纬度编码为 H3 cell。
func EncodeH3Cell(point *gis.Point, resolution int) (h3.Cell, error) {
	latLng := h3.NewLatLng(point.Lat, point.Lon)
	return h3.LatLngToCell(latLng, resolution)
}

// ValidatePoints 批量校验 pb Point 列表：非空、非 nil、经纬度范围合法。
func ValidatePoints(points ...*gis.Point) error {
	if len(points) == 0 {
		return tool.NewErrorByPbCode(extproto.Code__1_01_PARAM_MISSING, "points")
	}
	for i, p := range points {
		if p == nil {
			return tool.NewErrorByPbCode(extproto.Code__1_01_PARAM_MISSING, fmt.Sprintf("第 %d 个 point", i))
		}
		if err := gisx.ValidateCoordinate(p.Lon, p.Lat, i); err != nil {
			return tool.NewErrorByPbCode(extproto.Code__1_01_PARAM_INVALID, err.Error())
		}
	}
	return nil
}

// pbPointToOrbPolygon 将 pb Point 切片转换为 orb.Polygon（单外环，无洞）。
// 步骤：校验点数 → 坐标范围检查 → 构建 ring → 自动闭合（gisx.EnsurePolygonClosed）。
func pbPointToOrbPolygon(points []*gis.Point) (orb.Polygon, error) {
	if len(points) < 3 {
		return nil, tool.NewErrorByPbCode(extproto.Code__1_01_PARAM, "多边形至少需要3个点")
	}

	var ring orb.Ring
	for i, p := range points {
		if err := gisx.ValidateCoordinate(p.Lon, p.Lat, i); err != nil {
			return nil, tool.NewErrorByPbCode(extproto.Code__1_01_PARAM_INVALID, err.Error())
		}
		ring = append(ring, orb.Point{p.Lon, p.Lat})
	}

	return gisx.EnsurePolygonClosed(orb.Polygon{ring})
}
