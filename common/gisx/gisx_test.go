package gisx

import (
	"testing"

	"github.com/paulmach/orb"
)

func TestValidateCoordinate(t *testing.T) {
	tests := []struct {
		name    string
		lat     float64
		lon     float64
		wantErr bool
	}{
		{"正常坐标-北京", 39.9042, 116.4074, false},
		{"正常坐标-南极", -90, 0, false},
		{"正常坐标-北极", 90, 0, false},
		{"正常坐标-日期变更线", 0, 180, false},
		{"纬度超上限", 90.1, 116, true},
		{"纬度超下限", -90.1, 116, true},
		{"经度超上限", 39, 180.1, true},
		{"经度超下限", 39, -180.1, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateCoordinate(tt.lat, tt.lon, 0)
			if tt.wantErr && err == nil {
				t.Error("期望返回错误，但得到 nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("期望无错误，但得到: %v", err)
			}
		})
	}
}

func TestIsOrbPointsEqual(t *testing.T) {
	if !IsOrbPointsEqual(orb.Point{116.4, 39.9}, orb.Point{116.4, 39.9}) {
		t.Error("相同点应该相等")
	}
	if !IsOrbPointsEqual(orb.Point{116.4, 39.9}, orb.Point{116.4 + 1e-10, 39.9}) {
		t.Error("精度范围内应该相等")
	}
	if IsOrbPointsEqual(orb.Point{116.4, 39.9}, orb.Point{116.5, 39.9}) {
		t.Error("不同点不应该相等")
	}
}

func TestOrbRingToH3LatLng_AutoClose(t *testing.T) {
	ring := orb.Ring{
		{116.0, 39.0},
		{117.0, 39.0},
		{117.0, 40.0},
		{116.0, 40.0},
	}
	result := OrbRingToH3LatLng(ring)
	if len(result) != 5 {
		t.Fatalf("未闭合 ring 应补一个点，期望 5 得到 %d", len(result))
	}
	if result[0] != result[4] {
		t.Error("首尾应相等")
	}
	if result[0].Lat != 39.0 || result[0].Lng != 116.0 {
		t.Errorf("坐标转换错误: got lat=%v lng=%v", result[0].Lat, result[0].Lng)
	}
}

func TestOrbRingToH3LatLng_AlreadyClosed(t *testing.T) {
	ring := orb.Ring{
		{116.0, 39.0},
		{117.0, 39.0},
		{117.0, 40.0},
		{116.0, 39.0},
	}
	result := OrbRingToH3LatLng(ring)
	if len(result) != 4 {
		t.Fatalf("已闭合 ring 不应重复加点，期望 4 得到 %d", len(result))
	}
}

func TestOrbPolygonToH3GeoPolygon(t *testing.T) {
	polygon := orb.Polygon{
		orb.Ring{
			{116.0, 39.0},
			{117.0, 39.0},
			{117.0, 40.0},
			{116.0, 40.0},
		},
	}

	gp, err := OrbPolygonToH3GeoPolygon(polygon)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(gp.GeoLoop) != 5 {
		t.Errorf("GeoLoop 期望 5 点，得到 %d", len(gp.GeoLoop))
	}
	if len(gp.Holes) != 0 {
		t.Error("不应有洞")
	}
}

func TestOrbPolygonToH3GeoPolygon_WithHole(t *testing.T) {
	polygon := orb.Polygon{
		orb.Ring{{116.0, 39.0}, {117.0, 39.0}, {117.0, 40.0}, {116.0, 40.0}},
		orb.Ring{{116.3, 39.3}, {116.7, 39.3}, {116.7, 39.7}, {116.3, 39.7}},
	}
	gp, err := OrbPolygonToH3GeoPolygon(polygon)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(gp.Holes) != 1 {
		t.Errorf("期望 1 个洞，得到 %d", len(gp.Holes))
	}
}

func TestOrbPolygonToH3GeoPolygon_Errors(t *testing.T) {
	if _, err := OrbPolygonToH3GeoPolygon(orb.Polygon{}); err == nil {
		t.Error("空多边形应返回错误")
	}
	if _, err := OrbPolygonToH3GeoPolygon(orb.Polygon{orb.Ring{{1, 1}, {2, 2}}}); err == nil {
		t.Error("不足3点应返回错误")
	}
}

func TestSegmentIntersect(t *testing.T) {
	tests := []struct {
		name string
		a1   orb.Point
		a2   orb.Point
		b1   orb.Point
		b2   orb.Point
		want bool
	}{
		{"交叉", orb.Point{0, 0}, orb.Point{2, 2}, orb.Point{0, 2}, orb.Point{2, 0}, true},
		{"平行不交", orb.Point{0, 0}, orb.Point{2, 0}, orb.Point{0, 1}, orb.Point{2, 1}, false},
		{"远离不交", orb.Point{0, 0}, orb.Point{1, 0}, orb.Point{3, 3}, orb.Point{4, 4}, false},
		{"端点接触", orb.Point{0, 0}, orb.Point{1, 1}, orb.Point{1, 1}, orb.Point{2, 0}, true},
		{"T形交叉", orb.Point{0, 0}, orb.Point{2, 0}, orb.Point{1, -1}, orb.Point{1, 1}, true},
		{"共线重叠", orb.Point{0, 0}, orb.Point{2, 0}, orb.Point{1, 0}, orb.Point{3, 0}, true},
		{"共线不重叠", orb.Point{0, 0}, orb.Point{1, 0}, orb.Point{2, 0}, orb.Point{3, 0}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SegmentIntersect(tt.a1, tt.a2, tt.b1, tt.b2)
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRingIntersect(t *testing.T) {
	r1 := orb.Ring{{0, 0}, {2, 0}, {2, 2}, {0, 2}, {0, 0}}
	r2 := orb.Ring{{1, 1}, {3, 1}, {3, 3}, {1, 3}, {1, 1}}
	r3 := orb.Ring{{5, 5}, {6, 5}, {6, 6}, {5, 6}, {5, 5}}

	if !RingIntersect(r1, r2) {
		t.Error("重叠 ring 应该相交")
	}
	if RingIntersect(r1, r3) {
		t.Error("远离 ring 不应该相交")
	}
}

func TestPolygonIntersect(t *testing.T) {
	p1 := orb.Polygon{orb.Ring{{0, 0}, {4, 0}, {4, 4}, {0, 4}, {0, 0}}}
	p2 := orb.Polygon{orb.Ring{{2, 2}, {6, 2}, {6, 6}, {2, 6}, {2, 2}}}
	p3 := orb.Polygon{orb.Ring{{10, 10}, {12, 10}, {12, 12}, {10, 12}, {10, 10}}}
	p4 := orb.Polygon{orb.Ring{{1, 1}, {3, 1}, {3, 3}, {1, 3}, {1, 1}}}

	if !PolygonIntersect(p1, p2) {
		t.Error("重叠多边形应相交")
	}
	if PolygonIntersect(p1, p3) {
		t.Error("远离多边形不应相交")
	}
	if !PolygonIntersect(p1, p4) {
		t.Error("包含关系应判定为相交")
	}
}
