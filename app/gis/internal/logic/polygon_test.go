package logic

import (
	"testing"

	"zero-service/app/gis/gis"
)

func TestPbRingToOrbRing_Normal(t *testing.T) {
	points := []*gis.Point{{Lon: 116.3, Lat: 39.9}, {Lon: 116.4, Lat: 39.9}, {Lon: 116.4, Lat: 40.0}}
	ring, err := pbRingToOrbRing(points, "测试")
	if err != nil {
		t.Fatalf("预期通过，得到: %v", err)
	}
	if len(ring) != 3 {
		t.Fatalf("len=%d, want 3", len(ring))
	}
}

func TestPbRingToOrbRing_LessThan3(t *testing.T) {
	_, err := pbRingToOrbRing([]*gis.Point{{Lon: 116.3, Lat: 39.9}, {Lon: 116.4, Lat: 39.9}}, "测试")
	if err == nil {
		t.Fatal("预期错误，得到 nil")
	}
}

func TestPbRingToOrbRing_NilPoint(t *testing.T) {
	points := []*gis.Point{{Lon: 116.3, Lat: 39.9}, nil, {Lon: 116.4, Lat: 40.0}}
	_, err := pbRingToOrbRing(points, "测试")
	if err == nil {
		t.Fatal("预期 nil point 错误，得到 nil")
	}
}

func pbPoint(lon, lat float64) *gis.Point {
	return &gis.Point{Lon: lon, Lat: lat}
}

func pbRing(pts ...*gis.Point) *gis.Ring {
	return &gis.Ring{Points: pts}
}

func pbPolygon(outer *gis.Ring, holes ...*gis.Ring) *gis.Polygon {
	return &gis.Polygon{Outer: outer, Holes: holes}
}

func TestPbPolygonToOrbPolygon_Nil(t *testing.T) {
	_, err := pbPolygonToOrbPolygon(nil)
	if err == nil {
		t.Fatal("预期 nil polygon 错误")
	}
	_, err = pbPolygonToOrbPolygon(&gis.Polygon{})
	if err == nil {
		t.Fatal("预期 nil outer 错误")
	}
}

func TestPbPolygonToOrbPolygon_NoHole(t *testing.T) {
	p := pbPolygon(pbRing(pbPoint(116.3, 39.9), pbPoint(116.4, 39.9), pbPoint(116.4, 40.0)))
	_, err := pbPolygonToOrbPolygon(p)
	if err != nil {
		t.Fatalf("无洞单环应通过: %v", err)
	}
}

func TestPbPolygonToOrbPolygon_ValidHole(t *testing.T) {
	outer := pbRing(pbPoint(0, 0), pbPoint(10, 0), pbPoint(10, 10), pbPoint(0, 10))
	hole := pbRing(pbPoint(2, 2), pbPoint(4, 2), pbPoint(4, 4), pbPoint(2, 4))
	p := pbPolygon(outer, hole)

	_, err := pbPolygonToOrbPolygon(p)
	if err != nil {
		t.Fatalf("有效洞应通过: %v", err)
	}
}

func TestPbPolygonToOrbPolygon_HoleOutsideOuter(t *testing.T) {
	outer := pbRing(pbPoint(0, 0), pbPoint(10, 0), pbPoint(10, 10), pbPoint(0, 10))
	hole := pbRing(pbPoint(5, 5), pbPoint(15, 5), pbPoint(15, 15), pbPoint(5, 5))
	p := pbPolygon(outer, hole)

	_, err := pbPolygonToOrbPolygon(p)
	if err == nil {
		t.Fatal("洞超出外环应被拒绝")
	}
}

func TestPbPolygonToOrbPolygon_OverlappingHoles(t *testing.T) {
	outer := pbRing(pbPoint(0, 0), pbPoint(10, 0), pbPoint(10, 10), pbPoint(0, 10))
	holeA := pbRing(pbPoint(2, 2), pbPoint(6, 2), pbPoint(6, 6), pbPoint(2, 6))
	holeB := pbRing(pbPoint(4, 4), pbPoint(8, 4), pbPoint(8, 8), pbPoint(4, 8))
	p := pbPolygon(outer, holeA, holeB)

	_, err := pbPolygonToOrbPolygon(p)
	if err == nil {
		t.Fatal("重叠洞应被拒绝")
	}
}

func TestPbPolygonToOrbPolygon_MultipleValidHoles(t *testing.T) {
	outer := pbRing(pbPoint(0, 0), pbPoint(10, 0), pbPoint(10, 10), pbPoint(0, 10))
	holeA := pbRing(pbPoint(2, 2), pbPoint(3, 2), pbPoint(3, 3), pbPoint(2, 3))
	holeB := pbRing(pbPoint(5, 5), pbPoint(6, 5), pbPoint(6, 6), pbPoint(5, 6))
	holeC := pbRing(pbPoint(8, 2), pbPoint(9, 2), pbPoint(9, 3), pbPoint(8, 3))
	p := pbPolygon(outer, holeA, holeB, holeC)

	_, err := pbPolygonToOrbPolygon(p)
	if err != nil {
		t.Fatalf("多个不重叠的洞应通过: %v", err)
	}
}
