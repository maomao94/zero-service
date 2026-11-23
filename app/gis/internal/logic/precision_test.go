package logic

import (
	"fmt"
	"math"
	"testing"

	"zero-service/app/gis/gis"
)

// TestRoundCoord 校验响应出口坐标按 6 位小数四舍五入。
func TestRoundCoord(t *testing.T) {
	cases := []struct {
		in   float64
		want float64
	}{
		{121.46719514053864, 121.467195},
		{31.22434200031416, 31.224342},
		{121.4671955, 121.467196},
		{-0.0000004, 0},
		{0, 0},
	}
	for _, c := range cases {
		if got := roundCoord(c.in); got != c.want {
			t.Errorf("roundCoord(%v) = %v, want %v", c.in, got, c.want)
		}
	}

	// -0 归一化：JSON 不应出现 "-0"
	if got := roundCoord(-0.0000004); got != 0 || math.Signbit(got) {
		t.Errorf("roundCoord(-0.0000004) = %v (signbit=%v)，应为 +0", got, math.Signbit(got))
	}
	if got := ceilCoord(-0.0000004); got != 0 || math.Signbit(got) {
		t.Errorf("ceilCoord(-0.0000004) = %v (signbit=%v)，应为 +0", got, math.Signbit(got))
	}
}

// TestFloorCeilCoord 包围盒边界必须向外取整，且精度为 6 位小数。
func TestFloorCeilCoord(t *testing.T) {
	const v = 120.123456789
	if got, want := floorCoord(v), 120.123456; got != want {
		t.Errorf("floorCoord(%v) = %v, want %v", v, got, want)
	}
	if got, want := ceilCoord(v), 120.123457; got != want {
		t.Errorf("ceilCoord(%v) = %v, want %v", v, got, want)
	}
	if floorCoord(v) > v || ceilCoord(v) < v {
		t.Errorf("包围盒边界取整方向错误: floor=%v v=%v ceil=%v", floorCoord(v), v, ceilCoord(v))
	}
}

// TestRoundDistance 距离统一保留 3 位小数（1 mm）。
func TestRoundDistance(t *testing.T) {
	if got := roundDistance(1234.5678901234); got != 1234.568 {
		t.Errorf("roundDistance = %v, want 1234.568", got)
	}
	if got := roundDistance(0.0004); got != 0 {
		t.Errorf("roundDistance = %v, want 0", got)
	}
}

// TestDecodeH3_Precision 解码结果的中心点与边界顶点均为 6 位小数。
func TestDecodeH3_Precision(t *testing.T) {
	idx, err := EncodeH3Cell(&gis.Point{Lon: 120.0919826720555, Lat: 30.420827468961456}, 10)
	if err != nil {
		t.Fatalf("编码失败: %v", err)
	}

	l := NewDecodeH3Logic(t.Context(), nil)
	res, err := l.DecodeH3(&gis.DecodeH3Req{H3Index: idx.String()})
	if err != nil {
		t.Fatalf("解码失败: %v", err)
	}

	assertSixDecimals(t, "center.lon", res.Center.Lon)
	assertSixDecimals(t, "center.lat", res.Center.Lat)
	for i, p := range res.Boundary {
		assertSixDecimals(t, fmt.Sprintf("boundary[%d].lon", i), p.Lon)
		assertSixDecimals(t, fmt.Sprintf("boundary[%d].lat", i), p.Lat)
	}
}

// TestDecodeGeoHash_Precision 解码结果的中心点与包围盒边界均为 6 位小数，且包围盒只增不减。
func TestDecodeGeoHash_Precision(t *testing.T) {
	l := NewDecodeGeoHashLogic(t.Context(), nil)
	res, err := l.DecodeGeoHash(&gis.DecodeGeoHashReq{Geohash: "wtw3sjq"})
	if err != nil {
		t.Fatalf("解码失败: %v", err)
	}

	assertSixDecimals(t, "point.lon", res.Point.Lon)
	assertSixDecimals(t, "point.lat", res.Point.Lat)
	assertSixDecimals(t, "lonMin", res.LonMin)
	assertSixDecimals(t, "lonMax", res.LonMax)
	assertSixDecimals(t, "latMin", res.LatMin)
	assertSixDecimals(t, "latMax", res.LatMax)

	if res.LonMin > res.Point.Lon || res.LonMax < res.Point.Lon {
		t.Errorf("经度包围盒未覆盖中心点: [%v, %v] / %v", res.LonMin, res.LonMax, res.Point.Lon)
	}
	if res.LatMin > res.Point.Lat || res.LatMax < res.Point.Lat {
		t.Errorf("纬度包围盒未覆盖中心点: [%v, %v] / %v", res.LatMin, res.LatMax, res.Point.Lat)
	}
}

// TestDistance_Precision 距离计算结果保留 3 位小数（1 mm）。
func TestDistance_Precision(t *testing.T) {
	in := &gis.DistanceReq{
		A: &gis.Point{Lon: 120.0919826720555, Lat: 30.420827468961456},
		B: &gis.Point{Lon: 120.09252150158699, Lat: 30.420377510437717},
	}

	l := NewDistanceLogic(t.Context(), nil)
	res, err := l.Distance(in)
	if err != nil {
		t.Fatalf("计算失败: %v", err)
	}
	if res.Meters != roundDistance(res.Meters) {
		t.Errorf("meters = %v，未按 3 位小数取整", res.Meters)
	}
}

// TestRoutePoints_Precision 路径总距离保留 3 位小数。
func TestRoutePoints_Precision(t *testing.T) {
	l := NewRoutePointsLogic(t.Context(), nil)
	res, err := l.RoutePoints(&gis.RoutePointsReq{
		Start: &gis.Point{Lon: 120.0919, Lat: 30.4208},
		Points: []*gis.Point{
			{Lon: 120.0925, Lat: 30.4204},
			{Lon: 120.0921, Lat: 30.4214},
		},
	})
	if err != nil {
		t.Fatalf("计算失败: %v", err)
	}
	if res.TotalDistanceMeters != roundDistance(res.TotalDistanceMeters) {
		t.Errorf("totalDistanceMeters = %v，未按 3 位小数取整", res.TotalDistanceMeters)
	}
}

func abs(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}

// assertSixDecimals 断言坐标值已按 6 位小数取整。
func assertSixDecimals(t *testing.T, name string, v float64) {
	t.Helper()
	if v != roundCoord(v) {
		t.Errorf("%s = %v，未按 6 位小数取整", name, v)
	}
}
