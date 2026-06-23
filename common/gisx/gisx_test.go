package gisx

import (
	"errors"
	"testing"

	"github.com/paulmach/orb"
	"github.com/uber/h3-go/v4"
)

// TestValidateCoordinate 验证坐标校验（参数顺序 lon, lat）。
func TestValidateCoordinate(t *testing.T) {
	tests := []struct {
		name    string
		lon     float64
		lat     float64
		wantErr bool
	}{
		{"正常坐标-北京", 116.4074, 39.9042, false},
		{"正常坐标-南极", 0, -90, false},
		{"正常坐标-北极", 0, 90, false},
		{"正常坐标-日期变更线", 180, 0, false},
		{"纬度超上限", 116, 90.1, true},
		{"纬度超下限", 116, -90.1, true},
		{"经度超上限", 180.1, 39, true},
		{"经度超下限", -180.1, 39, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateCoordinate(tt.lon, tt.lat, 0)
			if tt.wantErr && err == nil {
				t.Error("期望返回错误，但得到 nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("期望无错误，但得到: %v", err)
			}
		})
	}
}


func TestOrbRingToH3LatLng_AutoClose(t *testing.T) {
	ring := orb.Ring{
		{116.0, 39.0},
		{117.0, 39.0},
		{117.0, 40.0},
		{116.0, 40.0},
	}
	result, err := OrbRingToH3LatLng(ring)
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 5 {
		t.Fatalf("未闭合 ring 应补一个点，期望 5 得到 %d", len(result))
	}
	if result[0].Lat != 39.0 || result[0].Lng != 116.0 {
		t.Errorf("坐标转换错误: got lat=%v lng=%v", result[0].Lat, result[0].Lng)
	}
	if result[4].Lat != 39.0 || result[4].Lng != 116.0 {
		t.Errorf("闭合点应与首点相同: got lat=%v lng=%v", result[4].Lat, result[4].Lng)
	}
	// 验证第二个点：orb (117,39) → H3 (lat=39, lng=117)
	if result[1].Lat != 39.0 || result[1].Lng != 117.0 {
		t.Errorf("第二个点转换错误: got lat=%v lng=%v", result[1].Lat, result[1].Lng)
	}
	// 验证第三个点：orb (117,40) → H3 (lat=40, lng=117)
	if result[2].Lat != 40.0 || result[2].Lng != 117.0 {
		t.Errorf("第三个点转换错误: got lat=%v lng=%v", result[2].Lat, result[2].Lng)
	}
	// 入参不变
	if len(ring) != 4 {
		t.Error("入参不应被修改")
	}
}

func TestOrbRingToH3LatLng_Empty(t *testing.T) {
	_, err := OrbRingToH3LatLng(orb.Ring{})
	if err == nil {
		t.Error("空 ring 应返回 error")
	}
}

func TestOrbRingToH3LatLng_TooFew(t *testing.T) {
	_, err := OrbRingToH3LatLng(orb.Ring{{116, 39}, {117, 39}})
	if err == nil {
		t.Error("不足 3 点应返回 error")
	}
}

func TestOrbRingToH3LatLng_AlreadyClosed(t *testing.T) {
	ring := orb.Ring{
		{116.0, 39.0},
		{117.0, 39.0},
		{117.0, 40.0},
		{116.0, 39.0},
	}
	result, err := OrbRingToH3LatLng(ring)
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 4 {
		t.Fatalf("已闭合 ring 不应重复加点，期望 4 得到 %d", len(result))
	}
}

func TestOrbPolygonToH3GeoPolygon(t *testing.T) {
	t.Run("未闭合外环自动闭合+坐标验证", func(t *testing.T) {
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
		// 验证 orb (116,39) → H3 (lat=39, lng=116)
		if gp.GeoLoop[0].Lat != 39.0 || gp.GeoLoop[0].Lng != 116.0 {
			t.Errorf("首点坐标: 期望 lat=39 lng=116, 得到 lat=%v lng=%v", gp.GeoLoop[0].Lat, gp.GeoLoop[0].Lng)
		}
	})
	t.Run("已闭合外环保持", func(t *testing.T) {
		polygon := orb.Polygon{
			orb.Ring{
				{116.0, 39.0},
				{117.0, 39.0},
				{117.0, 40.0},
				{116.0, 40.0},
				{116.0, 39.0},
			},
		}
		gp, err := OrbPolygonToH3GeoPolygon(polygon)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(gp.GeoLoop) != 5 {
			t.Errorf("已闭合外环应保持 5 点，得到 %d", len(gp.GeoLoop))
		}
	})
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
	if len(gp.Holes[0]) != 5 {
		t.Errorf("未闭合洞应自动闭合为 5 点，得到 %d", len(gp.Holes[0]))
	}
	// 验证洞坐标转换：orb (116.3, 39.3) → H3 (lat=39.3, lng=116.3)
	if gp.Holes[0][0].Lat != 39.3 || gp.Holes[0][0].Lng != 116.3 {
		t.Errorf("洞首点坐标: 期望 lat=39.3 lng=116.3, 得到 lat=%v lng=%v", gp.Holes[0][0].Lat, gp.Holes[0][0].Lng)
	}
}

func TestOrbPolygonToH3GeoPolygon_ClosedOuterUnclosedHole(t *testing.T) {
	polygon := orb.Polygon{
		orb.Ring{{116.0, 39.0}, {117.0, 39.0}, {117.0, 40.0}, {116.0, 40.0}, {116.0, 39.0}}, // 已闭合外环
		orb.Ring{{116.3, 39.3}, {116.7, 39.3}, {116.7, 39.7}, {116.3, 39.7}},                  // 未闭合洞
	}
	gp, err := OrbPolygonToH3GeoPolygon(polygon)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(gp.GeoLoop) != 5 {
		t.Errorf("已闭合外环应保持 5 点，得到 %d", len(gp.GeoLoop))
	}
	if len(gp.Holes) != 1 {
		t.Errorf("期望 1 个洞，得到 %d", len(gp.Holes))
	}
	if len(gp.Holes[0]) != 5 {
		t.Errorf("未闭合洞应自动闭合为 5 点，得到 %d", len(gp.Holes[0]))
	}
}

func TestOrbPolygonToH3GeoPolygon_Errors(t *testing.T) {
	t.Run("空多边形", func(t *testing.T) {
		_, err := OrbPolygonToH3GeoPolygon(orb.Polygon{})
		if err == nil {
			t.Error("空多边形应返回错误")
		}
	})
	t.Run("外环不足3点", func(t *testing.T) {
		_, err := OrbPolygonToH3GeoPolygon(orb.Polygon{orb.Ring{{1, 1}, {2, 2}}})
		if err == nil {
			t.Error("不足3点应返回错误")
		}
	})
	t.Run("无效洞被静默跳过", func(t *testing.T) {
		polygon := orb.Polygon{
			orb.Ring{{0, 0}, {4, 0}, {4, 4}, {0, 4}},       // 有效外环
			orb.Ring{{1, 1}, {2, 2}},                         // 无效洞（< 3 点，应跳过）
		}
		gp, err := OrbPolygonToH3GeoPolygon(polygon)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(gp.Holes) != 0 {
			t.Errorf("无效洞应被跳过，期望 0 个洞，得到 %d", len(gp.Holes))
		}
	})
}

func TestValidationError_ErrorsAs(t *testing.T) {
	err := ValidateCoordinate(200, 39, 0)
	if err == nil {
		t.Fatal("应报错")
	}
	var ve *ValidationError
	if !errors.As(err, &ve) {
		t.Error("errors.As 应能匹配 ValidationError")
	}
	if ve.Msg == "" {
		t.Error("消息不应为空")
	}
}

func TestOrbRingToH3LatLng_NoMutate(t *testing.T) {
	ring := orb.Ring{
		{116.0, 39.0},
		{117.0, 39.0},
		{117.0, 40.0},
		{116.0, 40.0},
	}
	originalLen := len(ring)
	_, _ = OrbRingToH3LatLng(ring)
	if len(ring) != originalLen {
		t.Errorf("入参 ring 长度被修改: 原 %d → %d", originalLen, len(ring))
	}
}

func TestIsRingClosed(t *testing.T) {
	tests := []struct {
		name   string
		ring   orb.Ring
		closed bool
	}{
		{"闭合 5 点", orb.Ring{{0, 0}, {4, 0}, {4, 4}, {0, 4}, {0, 0}}, true},
		{"未闭合 4 点", orb.Ring{{0, 0}, {4, 0}, {4, 4}, {0, 4}}, false},
		{"闭合 3 点", orb.Ring{{0, 0}, {4, 0}, {0, 0}}, true},
		{"微小偏差不闭合", orb.Ring{{0, 0}, {4, 0}, {4, 4}, {0, 4}, {1e-10, 0}}, false},
		{"不足 3 点", orb.Ring{{0, 0}, {1, 1}}, false},
		{"空 ring", orb.Ring{}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if IsRingClosed(tt.ring) != tt.closed {
				t.Errorf("期望 %v, 得到 %v", tt.closed, !tt.closed)
			}
		})
	}
}

func TestEnsureRingClosed(t *testing.T) {
	t.Run("已闭合不修改", func(t *testing.T) {
		ring := orb.Ring{{0, 0}, {4, 0}, {4, 4}, {0, 4}, {0, 0}}
		result, err := EnsureRingClosed(ring)
		if err != nil {
			t.Fatal(err)
		}
		if len(result) != 5 {
			t.Errorf("期望 5 点, 得到 %d", len(result))
		}
	})
	t.Run("未闭合自动追加", func(t *testing.T) {
		ring := orb.Ring{{0, 0}, {4, 0}, {4, 4}, {0, 4}}
		result, err := EnsureRingClosed(ring)
		if err != nil {
			t.Fatal(err)
		}
		if len(result) != 5 {
			t.Fatalf("期望 5 点, 得到 %d", len(result))
		}
		// 首尾精确闭合
		if result[0].Lon() != result[4].Lon() || result[0].Lat() != result[4].Lat() {
			t.Error("首尾应精确闭合")
		}
		// 追回的是首点副本
		if result[4].Lon() != 0 || result[4].Lat() != 0 {
			t.Errorf("闭合点应为 (0,0), 得到 (%.1f, %.1f)", result[4].Lon(), result[4].Lat())
		}
		// 入参不变
		if len(ring) != 4 {
			t.Error("入参不应被修改")
		}
	})
	t.Run("3点未闭合追加", func(t *testing.T) {
		ring := orb.Ring{{0, 0}, {4, 0}, {4, 4}}
		result, err := EnsureRingClosed(ring)
		if err != nil {
			t.Fatal(err)
		}
		if len(result) != 4 {
			t.Fatalf("期望 4 点, 得到 %d", len(result))
		}
		if result[0].Lon() != result[3].Lon() || result[0].Lat() != result[3].Lat() {
			t.Error("首尾应精确闭合")
		}
	})
	t.Run("不足3点报错", func(t *testing.T) {
		_, err := EnsureRingClosed(orb.Ring{{0, 0}, {1, 1}})
		if err == nil {
			t.Error("不足 3 点应报错")
		}
	})
	t.Run("空 ring 报错", func(t *testing.T) {
		_, err := EnsureRingClosed(orb.Ring{})
		if err == nil {
			t.Error("空 ring 应报错")
		}
	})
}

func TestEnsurePolygonClosed(t *testing.T) {
	t.Run("外环未闭合+洞已闭合", func(t *testing.T) {
		poly := orb.Polygon{
			orb.Ring{{0, 0}, {10, 0}, {10, 10}, {0, 10}},           // 未闭合外环
			orb.Ring{{2, 2}, {8, 2}, {8, 8}, {2, 8}, {2, 2}},       // 已闭合洞
		}
		result, err := EnsurePolygonClosed(poly)
		if err != nil {
			t.Fatal(err)
		}
		if len(result) != 2 {
			t.Fatalf("期望 2 个 ring, 得到 %d", len(result))
		}
		if len(result[0]) != 5 {
			t.Errorf("外环未闭合→应追加, 得到 %d 点", len(result[0]))
		}
		if len(result[1]) != 5 {
			t.Errorf("洞已闭合→不应追加, 得到 %d 点", len(result[1]))
		}
		if len(poly[0]) != 4 {
			t.Error("入参不应被修改")
		}
	})
	t.Run("外环和洞都未闭合", func(t *testing.T) {
		poly := orb.Polygon{
			orb.Ring{{0, 0}, {10, 0}, {10, 10}, {0, 10}},           // 未闭合外环
			orb.Ring{{2, 2}, {8, 2}, {8, 8}, {2, 8}},                // 未闭合洞
		}
		result, err := EnsurePolygonClosed(poly)
		if err != nil {
			t.Fatal(err)
		}
		if len(result) != 2 {
			t.Fatalf("期望 2 个 ring, 得到 %d", len(result))
		}
		if len(result[0]) != 5 {
			t.Errorf("外环未闭合→应追加, 得到 %d 点", len(result[0]))
		}
		if len(result[1]) != 5 {
			t.Errorf("洞未闭合→应追加, 得到 %d 点", len(result[1]))
		}
	})
}

func TestEnsurePolygonClosed_Errors(t *testing.T) {
	t.Run("空 polygon", func(t *testing.T) {
		result, err := EnsurePolygonClosed(orb.Polygon{})
		if err != nil {
			t.Errorf("空 polygon 不应报错: %v", err)
		}
		if len(result) != 0 {
			t.Errorf("期望空结果, 得到 %d ring", len(result))
		}
	})
	t.Run("洞不足3点", func(t *testing.T) {
		poly := orb.Polygon{
			orb.Ring{{0, 0}, {4, 0}, {4, 4}, {0, 4}, {0, 0}}, // 有效外环
			orb.Ring{{1, 1}, {2, 2}}, // 不足 3 点的洞
		}
		_, err := EnsurePolygonClosed(poly)
		if err == nil {
			t.Error("洞不足 3 点应报错")
		}
	})
}

func TestH3LatLngsToOrbRing(t *testing.T) {
	ring := H3LatLngsToOrbRing([]h3.LatLng{
		{Lat: 39.0, Lng: 116.0},
		{Lat: 40.0, Lng: 117.0},
		{Lat: 39.0, Lng: 116.0},
	})
	if len(ring) != 3 {
		t.Fatalf("期望 3 个点, 得到 %d", len(ring))
	}
	if ring[0].Lon() != 116.0 || ring[0].Lat() != 39.0 {
		t.Errorf("首点坐标错误: (%f, %f)", ring[0].Lon(), ring[0].Lat())
	}
	if ring[1].Lon() != 117.0 || ring[1].Lat() != 40.0 {
		t.Errorf("第二点坐标错误: (%f, %f)", ring[1].Lon(), ring[1].Lat())
	}
}

func TestH3LatLngsToOrbPolygon(t *testing.T) {
	gp := h3.GeoPolygon{
		GeoLoop: []h3.LatLng{
			{Lat: 39.0, Lng: 116.0},
			{Lat: 39.0, Lng: 117.0},
			{Lat: 40.0, Lng: 117.0},
			{Lat: 40.0, Lng: 116.0},
			{Lat: 39.0, Lng: 116.0},
		},
	}
	poly := H3LatLngsToOrbPolygon(gp)
	if len(poly) != 1 {
		t.Fatalf("期望 1 个 ring（外环）, 得到 %d", len(poly))
	}
	if len(poly[0]) != 5 {
		t.Fatalf("期望 5 个点, 得到 %d", len(poly[0]))
	}
}

func TestH3Roundtrip(t *testing.T) {
	original := orb.Ring{
		{116.0, 39.0},
		{117.0, 39.0},
		{117.0, 40.0},
		{116.0, 40.0},
		{116.0, 39.0},
	}
	h3LatLngs, err := OrbRingToH3LatLng(original)
	if err != nil {
		t.Fatal(err)
	}
	roundtripped := H3LatLngsToOrbRing(h3LatLngs)
	if len(roundtripped) != len(original) {
		t.Fatalf("长度不匹配: %d vs %d", len(roundtripped), len(original))
	}
	for i := range original {
		if original[i].Lon() != roundtripped[i].Lon() || original[i].Lat() != roundtripped[i].Lat() {
			t.Errorf("点[%d] 不匹配: (%f,%f) vs (%f,%f)",
				i, original[i].Lon(), original[i].Lat(), roundtripped[i].Lon(), roundtripped[i].Lat())
		}
	}
}
