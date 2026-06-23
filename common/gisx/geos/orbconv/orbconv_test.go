package orbconv

import (
	"testing"

	"github.com/paulmach/orb"
	gogeos "github.com/twpayne/go-geos"

	"zero-service/common/gisx/geos"
)

var (
	op1 = orb.Polygon{orb.Ring{{0, 0}, {4, 0}, {4, 4}, {0, 4}, {0, 0}}}
	op2 = orb.Polygon{orb.Ring{{2, 2}, {6, 2}, {6, 6}, {2, 6}, {2, 2}}}
	op3 = orb.Polygon{orb.Ring{{10, 10}, {12, 10}, {12, 12}, {10, 12}, {10, 10}}}
	op4 = orb.Polygon{orb.Ring{{1, 1}, {3, 1}, {3, 3}, {1, 3}, {1, 1}}}

	// 带洞多边形：外环 10×10，中间一个 4×4 的洞
	opWithHole = orb.Polygon{
		orb.Ring{{0, 0}, {10, 0}, {10, 10}, {0, 10}, {0, 0}}, // 外环
		orb.Ring{{3, 3}, {7, 3}, {7, 7}, {3, 7}, {3, 3}},       // 洞
	}
)

func TestConversion(t *testing.T) {
	t.Run("PolygonToGeom", func(t *testing.T) {
		g, err := PolygonToGeom(op1)
		if err != nil {
			t.Fatal(err)
		}
		if g == nil || g.IsEmpty() {
			t.Fatal("PolygonToGeom 失败")
		}
	})
	t.Run("GeomToPolygon", func(t *testing.T) {
		g, _ := PolygonToGeom(op1)
		p, err := GeomToPolygon(g)
		if err != nil {
			t.Fatal(err)
		}
		if len(p) != 1 {
			t.Fatal("外环丢失")
		}
	})
	t.Run("PointToGeom", func(t *testing.T) {
		g, err := PointToGeom(orb.Point{1, 2})
		if err != nil {
			t.Fatal(err)
		}
		if g.X() != 1 || g.Y() != 2 {
			t.Error("点坐标不匹配")
		}
	})
	t.Run("RingToGeom", func(t *testing.T) {
		g, err := RingToGeom(op1[0])
		if err != nil {
			t.Fatal(err)
		}
		r, _ := GeomToRing(g)
		if len(r) < 4 {
			t.Fatal("环转换失败")
		}
	})
	t.Run("nilInput", func(t *testing.T) {
		p, err := GeomToPolygon(nil)
		if p != nil || err == nil {
			t.Error("nil 应返回 error")
		}
	})
}

func TestPredicates(t *testing.T) {
	t.Run("IntersectsOrb", func(t *testing.T) {
		ok, err := IntersectsOrb(op1, op2)
		if err != nil || !ok {
			t.Error("重叠应相交")
		}
		ok, err = IntersectsOrb(op1, op3)
		if err != nil || ok {
			t.Error("远离不应相交")
		}
	})
	t.Run("ContainsOrb", func(t *testing.T) {
		ok, _ := ContainsOrb(op1, op4)
		if !ok {
			t.Error("应包含")
		}
	})
	t.Run("CoversOrb", func(t *testing.T) {
		ok, _ := CoversOrb(op1, op4)
		if !ok {
			t.Error("应覆盖")
		}
	})
	t.Run("CoversPointOrb边界", func(t *testing.T) {
		ok, _ := CoversPointOrb(op1, orb.Point{0, 0})
		if !ok {
			t.Error("边界点应被覆盖")
		}
	})
	t.Run("ContainsPointOrb边界", func(t *testing.T) {
		ok, _ := ContainsPointOrb(op1, orb.Point{0, 0})
		if ok {
			t.Error("Contains 边界点应为 false")
		}
	})
	t.Run("ValidOrb", func(t *testing.T) {
		ok, _ := ValidOrb(op1)
		if !ok {
			t.Error("简单方形应有效")
		}
	})
}

// TestMultiPolygonConversion 验证 MultiPolygon 的往返转换。
func TestMultiPolygonConversion(t *testing.T) {
	t.Run("多个独立多边形 ↔ GEOS MultiPolygon", func(t *testing.T) {
		// orb.MultiPolygon = 两个分离的方形
		orbMP := orb.MultiPolygon{
			orb.Polygon{orb.Ring{{0, 0}, {2, 0}, {2, 2}, {0, 2}, {0, 0}}},
			orb.Polygon{orb.Ring{{5, 5}, {7, 5}, {7, 7}, {5, 7}, {5, 5}}},
		}

		// orb → GEOS
		g, err := MultiPolygonToGeom(orbMP)
		if err != nil {
			t.Fatal(err)
		}
		if g.TypeID() != gogeos.TypeIDMultiPolygon {
			t.Errorf("期望 MultiPolygon，实际 TypeID=%d", g.TypeID())
		}
		if g.NumGeometries() != 2 {
			t.Errorf("期望 2 个子几何，实际 %d", g.NumGeometries())
		}

		// GEOS → orb 往返
		result, err := GeomToMultiPolygon(g)
		if err != nil {
			t.Fatal(err)
		}
		if len(result) != 2 {
			t.Fatalf("期望 2 个多边形，实际 %d", len(result))
		}
		// 验证每个多边形有一个外环
		for i, poly := range result {
			if len(poly) != 1 {
				t.Errorf("多边形[%d] 期望 1 个外环，实际 %d 个 ring", i, len(poly))
			}
		}
	})

	t.Run("单个多边形 → GeomToMultiPolygon 返回单元素", func(t *testing.T) {
		g, _ := PolygonToGeom(op1)
		mp, err := GeomToMultiPolygon(g)
		if err != nil {
			t.Fatal(err)
		}
		if len(mp) != 1 {
			t.Fatalf("期望 1 个多边形，实际 %d", len(mp))
		}
		if len(mp[0]) != 1 {
			t.Error("外环丢失")
		}
	})

	t.Run("带洞多边形往返", func(t *testing.T) {
		g, err := PolygonToGeom(opWithHole)
		if err != nil {
			t.Fatal(err)
		}
		// 验证 GEOS 多边形有 1 个洞
		if g.NumInteriorRings() != 1 {
			t.Errorf("期望 1 个洞，实际 %d", g.NumInteriorRings())
		}

		// GEOS → orb 往返，洞应保留
		poly, err := GeomToPolygon(g)
		if err != nil {
			t.Fatal(err)
		}
		if len(poly) != 2 {
			t.Fatalf("期望 2 个 ring（外环+洞），实际 %d", len(poly))
		}
		// poly[0] = 外环，poly[1] = 洞
	})

	t.Run("MultiPolygon 中包含带洞的多边形", func(t *testing.T) {
		// 构造一个 MultiPolygon：第一个有洞，第二个无洞
		mp, _ := geos.NewMultiPolygon([][][][]float64{
			{{{0, 0}, {10, 0}, {10, 10}, {0, 10}, {0, 0}}, {{3, 3}, {7, 3}, {7, 7}, {3, 7}, {3, 3}}},
			{{{20, 20}, {22, 20}, {22, 22}, {20, 22}, {20, 20}}},
		})

		orbMP, err := GeomToMultiPolygon(mp)
		if err != nil {
			t.Fatal(err)
		}
		if len(orbMP) != 2 {
			t.Fatalf("期望 2 个多边形，实际 %d", len(orbMP))
		}
		// 第一个多边形：外环 + 1 个洞
		if len(orbMP[0]) != 2 {
			t.Errorf("多边形[0] 期望 2 个 ring（外环+洞），实际 %d", len(orbMP[0]))
		}
		// 第二个多边形：只有外环
		if len(orbMP[1]) != 1 {
			t.Errorf("多边形[1] 期望 1 个 ring（仅外环），实际 %d", len(orbMP[1]))
		}
	})

	t.Run("空/nil 边界", func(t *testing.T) {
		_, err := GeomToMultiPolygon(nil)
		if err == nil {
			t.Error("nil 应返回 error")
		}
		_, err = MultiPolygonToGeom(nil)
		if err == nil {
			t.Error("nil MultiPolygon 应返回 error")
		}
		_, err = MultiPolygonToGeom(orb.MultiPolygon{})
		if err == nil {
			t.Error("空切片应返回 error")
		}
	})

	t.Run("GeomToPolygon 取 MultiPolygon 第一个", func(t *testing.T) {
		mp, _ := geos.NewMultiPolygon([][][][]float64{
			{{{0, 0}, {2, 0}, {2, 2}, {0, 2}, {0, 0}}},
			{{{5, 5}, {7, 5}, {7, 7}, {5, 7}, {5, 5}}},
		})
		poly, err := GeomToPolygon(mp)
		if err != nil {
			t.Fatal(err)
		}
		// 应该只拿到第一个多边形
		if len(poly) != 1 {
			t.Errorf("期望 1 个外环，实际 %d 个 ring", len(poly))
		}
	})
}

// TestMoreConversions 验证 orbconv 中未经测试的转换函数。
func TestMoreConversions(t *testing.T) {
	t.Run("GeomToPoint", func(t *testing.T) {
		g := mustGeos(geos.NewPoint(116.39, 39.9))
		pt, err := GeomToPoint(g)
		if err != nil {
			t.Fatal(err)
		}
		if pt.Lon() != 116.39 || pt.Lat() != 39.9 {
			t.Errorf("坐标不匹配: (%f,%f)", pt.Lon(), pt.Lat())
		}
	})

	t.Run("GeomToLineString", func(t *testing.T) {
		g := mustGeos(geos.NewLineString([][]float64{{0, 0}, {3, 0}, {3, 3}}))
		ls, err := GeomToLineString(g)
		if err != nil {
			t.Fatal(err)
		}
		if len(ls) != 3 {
			t.Fatalf("期望 3 个点, 得到 %d", len(ls))
		}
	})

	t.Run("GeomToMultiPoint", func(t *testing.T) {
		pt, err := geos.NewPoint(1, 2)
		if err != nil {
			t.Fatal(err)
		}
		mp, err := GeomToMultiPoint(pt)
		if err != nil {
			t.Fatal(err)
		}
		if len(mp) != 1 || mp[0].Lon() != 1 || mp[0].Lat() != 2 {
			t.Error("单个点应返回单元素 MultiPoint")
		}
	})

	t.Run("GeomToMultiLineString", func(t *testing.T) {
		ls, err := geos.NewLineString([][]float64{{0, 0}, {3, 0}})
		if err != nil {
			t.Fatal(err)
		}
		mls, err := GeomToMultiLineString(ls)
		if err != nil {
			t.Fatal(err)
		}
		if len(mls) != 1 {
			t.Fatalf("单线期望返回 1 元素, 得到 %d", len(mls))
		}
	})

	t.Run("LineStringToGeom", func(t *testing.T) {
		ls := orb.LineString{orb.Point{0, 0}, orb.Point{3, 0}, orb.Point{3, 3}}
		g, err := LineStringToGeom(ls)
		if err != nil {
			t.Fatal(err)
		}
		if g.TypeID() != gogeos.TypeIDLineString {
			t.Error("应为 LineString 类型")
		}
	})

	t.Run("MultiPointToGeom", func(t *testing.T) {
		mp := orb.MultiPoint{{1, 2}, {3, 4}}
		g, err := MultiPointToGeom(mp)
		if err != nil {
			t.Fatal(err)
		}
		if g.TypeID() != gogeos.TypeIDMultiPoint {
			t.Errorf("期望 MultiPoint, 得到 %d", g.TypeID())
		}
	})

	t.Run("MultiLineStringToGeom", func(t *testing.T) {
		mls := orb.MultiLineString{
			orb.LineString{orb.Point{0, 0}, orb.Point{3, 0}},
			orb.LineString{orb.Point{3, 0}, orb.Point{3, 3}},
		}
		g, err := MultiLineStringToGeom(mls)
		if err != nil {
			t.Fatal(err)
		}
		if g.TypeID() != gogeos.TypeIDMultiLineString {
			t.Errorf("期望 MultiLineString, 得到 %d", g.TypeID())
		}
	})
}

// TestConversionNilInputs 验证所有转换函数的 nil 输入。
func TestConversionNilInputs(t *testing.T) {
	t.Run("GeomTo* nil", func(t *testing.T) {
		_, err := GeomToRing(nil)
		if err == nil {
			t.Error("nil GeomToRing 应报错")
		}
		_, err = GeomToPoint(nil)
		if err == nil {
			t.Error("nil GeomToPoint 应报错")
		}
		_, err = GeomToLineString(nil)
		if err == nil {
			t.Error("nil GeomToLineString 应报错")
		}
		_, err = GeomToPolygon(nil)
		if err == nil {
			t.Error("nil GeomToPolygon 应报错")
		}
		_, err = GeomToMultiPolygon(nil)
		if err == nil {
			t.Error("nil GeomToMultiPolygon 应报错")
		}
		_, err = GeomToMultiPoint(nil)
		if err == nil {
			t.Error("nil GeomToMultiPoint 应报错")
		}
		_, err = GeomToMultiLineString(nil)
		if err == nil {
			t.Error("nil GeomToMultiLineString 应报错")
		}
	})
	t.Run("*ToGeom nil/empty", func(t *testing.T) {
		_, err := RingToGeom(nil)
		if err == nil {
			t.Error("nil Ring 应报错")
		}
		_, err = LineStringToGeom(nil)
		if err == nil {
			t.Error("nil LineString 应报错")
		}
		_, err = PolygonToGeom(nil)
		if err == nil {
			t.Error("nil Polygon 应报错")
		}
		_, err = MultiPolygonToGeom(nil)
		if err == nil {
			t.Error("nil MultiPolygon 应报错")
		}
	})
}

// TestUnsupportedTypes 验证 GeomTo* 函数对不支持的几何类型的错误处理。
func TestUnsupportedTypes(t *testing.T) {
	line, err := geos.NewLineString([][]float64{{0, 0}, {3, 0}})
	if err != nil {
		t.Fatal(err)
	}
	pt, err := geos.NewPoint(1, 2)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("GeomToPolygon unsupported", func(t *testing.T) {
		_, err := GeomToPolygon(line)
		if err == nil {
			t.Error("LineString 不应被 GeomToPolygon 接受")
		}
	})
	t.Run("GeomToMultiPolygon unsupported", func(t *testing.T) {
		_, err := GeomToMultiPolygon(pt)
		if err == nil {
			t.Error("Point 不应被 GeomToMultiPolygon 接受")
		}
	})
	t.Run("GeomToMultiPoint unsupported", func(t *testing.T) {
		_, err := GeomToMultiPoint(line)
		if err == nil {
			t.Error("LineString 不应被 GeomToMultiPoint 接受")
		}
	})
	t.Run("GeomToMultiLineString unsupported", func(t *testing.T) {
		_, err := GeomToMultiLineString(pt)
		if err == nil {
			t.Error("Point 不应被 GeomToMultiLineString 接受")
		}
	})
}

// TestErrorPaths 验证便捷包装器的错误路径。
func TestErrorPaths(t *testing.T) {
	t.Run("ValidOrb invalid", func(t *testing.T) {
		bowtie := orb.Polygon{orb.Ring{{0, 0}, {4, 0}, {0, 4}, {4, 4}, {0, 0}}}
		ok, err := ValidOrb(bowtie)
		if err != nil {
			t.Fatal(err)
		}
		if ok {
			t.Error("bowtie 应无效")
		}
	})
	t.Run("PolygonToGeom empty outer", func(t *testing.T) {
		_, err := PolygonToGeom(orb.Polygon{orb.Ring{}})
		if err == nil {
			t.Error("空外环应报错")
		}
	})
	t.Run("CoversOrb nil", func(t *testing.T) {
		_, err := CoversOrb(nil, op1)
		if err == nil {
			t.Error("nil 应报错")
		}
	})
	t.Run("ContainsOrb nil", func(t *testing.T) {
		_, err := ContainsOrb(nil, op1)
		if err == nil {
			t.Error("nil 应报错")
		}
	})
	t.Run("IntersectsOrb nil", func(t *testing.T) {
		_, err := IntersectsOrb(nil, op1)
		if err == nil {
			t.Error("nil 应报错")
		}
	})
	t.Run("CoversPointOrb nil polygon", func(t *testing.T) {
		_, err := CoversPointOrb(nil, orb.Point{1, 2})
		if err == nil {
			t.Error("nil polygon 应报错")
		}
	})
	t.Run("ContainsPointOrb nil polygon", func(t *testing.T) {
		_, err := ContainsPointOrb(nil, orb.Point{1, 2})
		if err == nil {
			t.Error("nil polygon 应报错")
		}
	})
	t.Run("MultiPointToGeom single", func(t *testing.T) {
		g, err := MultiPointToGeom(orb.MultiPoint{{1, 2}})
		if err != nil {
			t.Fatal(err)
		}
		if g.TypeID() != gogeos.TypeIDPoint {
			t.Errorf("单点应返回 Point, 得到 %d", g.TypeID())
		}
	})
	t.Run("MultiLineStringToGeom single", func(t *testing.T) {
		g, err := MultiLineStringToGeom(orb.MultiLineString{{orb.Point{0, 0}, orb.Point{3, 0}}})
		if err != nil {
			t.Fatal(err)
		}
		if g.TypeID() != gogeos.TypeIDLineString {
			t.Errorf("单线应返回 LineString, 得到 %d", g.TypeID())
		}
	})
}

// mustGeos 是 orbconv 测试的辅助函数，将 geos 包返回的 (T, error) 转为 T。
func mustGeos[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}
	return v
}
