package gisx

import (
	"errors"
	"math"

	"github.com/paulmach/orb"
	"github.com/uber/h3-go/v4"
)

// polygonToH3GeoPolygon 将orb.Polygon转换为H3库需要的GeoPolygon格式
// 严格按照用户要求处理多边形结构：
// - ring[0]: 电子围栏外环
// - ring[1...]: 电子围栏的洞
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

// isPointsEqual 检查两个orb.Point是否相等（用于验证多边形闭合性）
func IsOrbPointsEqual(p1, p2 orb.Point) bool {
	// 考虑浮点精度问题的坐标比较
	const epsilon = 1e-9
	return math.Abs(p1[0]-p2[0]) < epsilon && math.Abs(p1[1]-p2[1]) < epsilon
}

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
