package gisx

import (
	"errors"
	"math"

	"github.com/paulmach/orb"
	"github.com/uber/h3-go/v4"
)

// OrbPolygonToH3GeoPolygon 将 orb.Polygon 转换为 H3 PolygonToCells 所需的 GeoPolygon 格式。
// 多边形结构约定：polygon[0] 为外环，polygon[1:] 为洞（hole）。
// 外环至少需要 3 个顶点；无效洞（< 3 点）会被静默跳过。
func OrbPolygonToH3GeoPolygon(polygon orb.Polygon) (h3.GeoPolygon, error) {
	var geoPolygon h3.GeoPolygon

	if len(polygon) == 0 {
		return geoPolygon, errors.New("多边形至少包含一个外环")
	}

	// --- 处理外环 ---
	outerRing := polygon[0]
	if len(outerRing) < 3 {
		return geoPolygon, errors.New("外环至少需要3个点")
	}

	geoPolygon.GeoLoop = OrbRingToH3LatLng(outerRing)

	// --- 处理洞 ---
	for i := 1; i < len(polygon); i++ {
		holeRing := polygon[i]
		if len(holeRing) < 3 {
			continue // 忽略无效洞
		}
		hole := OrbRingToH3LatLng(holeRing)
		geoPolygon.Holes = append(geoPolygon.Holes, hole)
	}

	return geoPolygon, nil
}

// IsOrbPointsEqual 判断两个坐标点是否相等（浮点精度容差 1e-9，约 0.1mm）。
// 主要用于检测 ring 首尾是否闭合。
func IsOrbPointsEqual(p1, p2 orb.Point) bool {
	const epsilon = 1e-9
	return math.Abs(p1[0]-p2[0]) < epsilon && math.Abs(p1[1]-p2[1]) < epsilon
}

// OrbRingToH3LatLng 将 orb.Ring（经度在前 [lon, lat]）转换为 H3 的 []LatLng（纬度在前）。
// 若 ring 未闭合（首尾不相等），自动追加首点使其闭合。
func OrbRingToH3LatLng(ring orb.Ring) []h3.LatLng {
	if !IsOrbPointsEqual(ring[0], ring[len(ring)-1]) {
		ring = append(ring, ring[0])
	}
	res := make([]h3.LatLng, len(ring))
	for i, pt := range ring {
		res[i] = h3.LatLng{Lat: pt[1], Lng: pt[0]}
	}
	return res
}
