package gisx

import (
	"errors"

	"github.com/paulmach/orb"
	"github.com/uber/h3-go/v4"
)

// OrbPolygonToH3GeoPolygon 将 orb.Polygon 转换为 H3 PolygonToCells 所需的 GeoPolygon 格式。
// 多边形结构约定：polygon[0] 为外环，polygon[1:] 为洞（hole）。
// 外环至少 3 个顶点，不足或空返回 error；无效洞（< 3 点）静默跳过。
// 不闭合的环会自动闭合后转换（不修改入参）。
func OrbPolygonToH3GeoPolygon(polygon orb.Polygon) (h3.GeoPolygon, error) {
	var geoPolygon h3.GeoPolygon

	if len(polygon) == 0 {
		return geoPolygon, errors.New("多边形至少包含一个外环")
	}

	outerRing := polygon[0]
	geoLoop, err := OrbRingToH3LatLng(outerRing)
	if err != nil {
		return geoPolygon, err
	}
	geoPolygon.GeoLoop = geoLoop

	for i := 1; i < len(polygon); i++ {
		hole, err := OrbRingToH3LatLng(polygon[i])
		if err != nil {
			continue
		}
		geoPolygon.Holes = append(geoPolygon.Holes, hole)
	}

	return geoPolygon, nil
}

// OrbRingToH3LatLng 将 orb.Ring（经度在前 [lon, lat]）转换为 H3 的 []LatLng（纬度在前）。
// 若 ring 未闭合，自动闭合后转换（不修改入参）。
// 空 ring 或不足 3 点返回 error。
func OrbRingToH3LatLng(ring orb.Ring) ([]h3.LatLng, error) {
	closed, err := EnsureRingClosed(ring)
	if err != nil {
		return nil, err
	}
	res := make([]h3.LatLng, len(closed))
	for i, pt := range closed {
		res[i] = h3.LatLng{Lat: pt.Lat(), Lng: pt.Lon()}
	}
	return res, nil
}

// IsRingClosed 判断 orb.Ring 首尾是否精确闭合（必须完全相同，GEOS 不支持容差）。
// 空 ring 返回 false。
func IsRingClosed(ring orb.Ring) bool {
	if len(ring) < 3 {
		return false
	}
	first, last := ring[0], ring[len(ring)-1]
	return first.Lon() == last.Lon() && first.Lat() == last.Lat()
}

// EnsureRingClosed 确保 orb.Ring 首尾精确闭合，不修改入参。
// 少于 3 个点返回 error；已闭合返回原值；未闭合追加首点返回新 ring。
func EnsureRingClosed(ring orb.Ring) (orb.Ring, error) {
	if len(ring) < 3 {
		return nil, errors.New("ring 至少需要 3 个点")
	}
	first, last := ring[0], ring[len(ring)-1]
	if first.Lon() == last.Lon() && first.Lat() == last.Lat() {
		return ring, nil
	}
	closed := make(orb.Ring, len(ring)+1)
	copy(closed, ring)
	closed[len(ring)] = ring[0]
	return closed, nil
}

// EnsurePolygonClosed 确保 orb.Polygon 所有环（外环 + 洞）首尾精确闭合，不修改入参。
// 任一环 < 3 点返回 error。
func EnsurePolygonClosed(poly orb.Polygon) (orb.Polygon, error) {
	result := make(orb.Polygon, len(poly))
	for i, ring := range poly {
		closed, err := EnsureRingClosed(ring)
		if err != nil {
			return nil, err
		}
		result[i] = closed
	}
	return result, nil
}

// H3LatLngsToOrbRing 将 H3 的 []LatLng（纬度在前）转换为 orb.Ring（经度在前 [lon, lat]）。
// 与 OrbRingToH3LatLng 互为反向操作。
// 首尾相同的闭合环原样保留（不会去重最后一点）。
func H3LatLngsToOrbRing(latlngs []h3.LatLng) orb.Ring {
	ring := make(orb.Ring, len(latlngs))
	for i, ll := range latlngs {
		ring[i] = orb.Point{ll.Lng, ll.Lat}
	}
	return ring
}

// H3LatLngsToOrbPolygon 将 H3 GeoPolygon 的外环和洞转换为 orb.Polygon。
// 洞为空时返回无洞多边形。
func H3LatLngsToOrbPolygon(gp h3.GeoPolygon) orb.Polygon {
	poly := orb.Polygon{H3LatLngsToOrbRing(gp.GeoLoop)}
	for _, hole := range gp.Holes {
		poly = append(poly, H3LatLngsToOrbRing(hole))
	}
	return poly
}
