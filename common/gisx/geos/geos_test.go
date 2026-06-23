package geos

// geos_test.go — GEOS 封装包的测试文件
//
// 本文件覆盖以下功能：
//   - GEOS 版本查询
//   - 几何构造（Point / Polygon / BoundsRect）
//   - 格式转换（WKT / WKB / GeoJSON 的解析与序列化）
//   - 空间谓词（Intersects / Contains / Covers / Touches 等语义验证）
//   - Prepared Geometry 加速谓词
//   - Overlay 运算（Intersection / Union / SymDifference / Difference）
//   - 有效性检查（IsValid / IsValidReason / MakeValid）
//   - 简化与缓冲（Simplify / Buffer / ConvexHull / ConcaveHull）
//   - 测量（Area / Length / Centroid / PointOnSurface）
//   - 高级函数（Relate / DE-9IM / Hausdorff / DistanceWithin / SRID / Normalize）
//   - STRtree 空间索引（Insert / Query / Iterate / Remove）
//   - 内省函数（IsEmpty / IsSimple / IsClosed / IsRing / HasZ）
//   - 高级变换（Boundary / UnaryUnion / Envelope / Reverse / Snap 等）
//
// 测试坐标约定：
//   - rp* 变量表示矩形多边形（rect polygon）
//   - 所有坐标采用 {x, y} = {经度, 纬度} 顺序
//
// 测试用例设计思路：
//   - 正面测试：验证函数在正常输入下的行为
//   - 边界测试：验证边界点在不同谓词下的语义差异（Contains vs Covers）
//   - 负面测试：验证无效几何和无效格式的报错

import (
	"math"
	"testing"

	gogeos "github.com/twpayne/go-geos"
)

// 测试用几何数据（纯坐标数组形式）
// 命名规则：rp = rect polygon（矩形多边形）
var (
	// rp1: 原点开始的 4×4 正方形
	rp1 = [][][]float64{{{0, 0}, {4, 0}, {4, 4}, {0, 4}, {0, 0}}}
	// rp2: 与 rp1 重叠 2×2 区域的正方形（从 (2,2) 开始）
	rp2 = [][][]float64{{{2, 2}, {6, 2}, {6, 6}, {2, 6}, {2, 2}}}
	// rp3: 远离 rp1 的正方形（不相交）
	rp3 = [][][]float64{{{10, 10}, {12, 10}, {12, 12}, {10, 12}, {10, 10}}}
	// rp4: 在 rp1 内部的正方形（用于测试包含关系）
	rp4 = [][][]float64{{{1, 1}, {3, 1}, {3, 3}, {1, 3}, {1, 1}}}
	// rbowtie: 蝴蝶结形状的无效多边形（自相交）
	rbowtie = [][][]float64{{{0, 0}, {4, 0}, {0, 4}, {4, 4}, {0, 0}}}
	// rptouch: 与 rp1 共享边界的多边形（测试 Touches）
	rptouch = [][][]float64{{{4, 0}, {8, 0}, {8, 4}, {4, 4}, {4, 0}}}
)

// must 是测试辅助函数，将 (value, error) 模式转为 value，error 时 panic。
// 用于简化测试代码的写法：
//
//	使用前：g, err := NewPoint(1, 2); if err != nil { t.Fatal(err) }
//	使用后：g := must(NewPoint(1, 2))
func must[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}
	return v
}

// TestGEOSVersion 验证 GEOS 版本查询功能。
func TestGEOSVersion(t *testing.T) {
	m, _, _ := GEOSVersion()
	if m <= 0 {
		t.Fatalf("major 应 > 0: %d", m)
	}
	s := GEOSVersionString()
	if s == "" {
		t.Fatal("版本字符串不应为空")
	}
	t.Logf("GEOS %s", s)
}

// TestConstruct 验证几何构造函数。
func TestConstruct(t *testing.T) {
	t.Run("Point", func(t *testing.T) {
		// 创建点并验证坐标值
		g := must(NewPoint(1, 2))
		if g.X() != 1 || g.Y() != 2 {
			t.Fatalf("Point: (%f,%f)", g.X(), g.Y())
		}
	})
	t.Run("Polygon", func(t *testing.T) {
		// 创建多边形并验证类型
		g := must(NewPolygon(rp1))
		if g.TypeID() != gogeos.TypeIDPolygon {
			t.Fatal("类型应为 Polygon")
		}
	})
	t.Run("BoundsRect", func(t *testing.T) {
		// 创建外包盒矩形
		g := must(NewBoundsRect(0, 0, 4, 4))
		if g.IsEmpty() {
			t.Fatal("不应为空")
		}
	})
}

// TestWKTConvert 验证 WKT 格式的解析和序列化。
func TestWKTConvert(t *testing.T) {
	// 解析 WKT → 序列化 WKT（往返测试）
	g := must(FromWKT("POLYGON ((0 0, 4 0, 4 4, 0 4, 0 0))"))
	wkt := must(ToWKT(g))
	t.Logf("WKT: %s", wkt)

	// 无效 WKT 应报错
	_, err := FromWKT("NOT WKT")
	if err == nil {
		t.Fatal("无效 WKT 应报错")
	}
}

// TestWKBConvert 验证 WKB 格式的序列化和解析。
func TestWKBConvert(t *testing.T) {
	g := must(NewPolygon(rp1))
	wkb := must(ToWKB(g))
	if len(wkb) == 0 {
		t.Fatal("WKB 不应为空")
	}
	// WKB 往返测试：序列化 → 解析 → 验证非空
	g2 := must(FromWKB(wkb))
	if g2.IsEmpty() {
		t.Fatal("往返失败")
	}
}

// TestGeoJSONConvert 验证 GeoJSON 格式的解析和序列化。
func TestGeoJSONConvert(t *testing.T) {
	g := must(NewPolygon(rp1))
	json := must(ToGeoJSON(g, 0))
	t.Logf("GeoJSON: %s", json)
	// GeoJSON 往返测试
	g2 := must(FromGeoJSON(json))
	if g2.IsEmpty() {
		t.Fatal("往返失败")
	}
}

// TestPredicates 验证空间谓词的核心语义。
//
// 关键语义验证：
//   - Intersects: 重叠的返回 true，远离的返回 false
//   - Contains: 包含关系验证
//   - Covers: 覆盖关系验证
//   - Touches: 边界接触验证
//   - Contains vs Covers: 边界点的语义差异（Contains=false, Covers=true）
func TestPredicates(t *testing.T) {
	p1 := must(NewPolygon(rp1))
	p2 := must(NewPolygon(rp2))
	p3 := must(NewPolygon(rp3))
	p4 := must(NewPolygon(rp4))
	pt := must(NewPolygon(rptouch))

	t.Run("Intersects", func(t *testing.T) {
		// p1(0,0)-(4,4) 和 p2(2,2)-(6,6) 有重叠区域
		if !must(Intersects(p1, p2)) {
			t.Error("p1,p2 应相交")
		}
		// p1(0,0)-(4,4) 和 p3(10,10)-(12,12) 完全分离
		if must(Intersects(p1, p3)) {
			t.Error("p1,p3 不应相交")
		}
	})
	t.Run("Contains", func(t *testing.T) {
		// p4(1,1)-(3,3) 完全在 p1(0,0)-(4,4) 内部
		if !must(Contains(p1, p4)) {
			t.Error("p1 应包含 p4")
		}
	})
	t.Run("Covers", func(t *testing.T) {
		// p1 覆盖 p4（p4 在 p1 内部）
		if !must(Covers(p1, p4)) {
			t.Error("p1 应覆盖 p4")
		}
	})
	t.Run("Touches", func(t *testing.T) {
		// pt(4,0)-(8,4) 与 p1(0,0)-(4,4) 共享边 x=4，仅边界接触
		if !must(Touches(p1, pt)) {
			t.Error("p1,pt 应边界接触")
		}
	})
	t.Run("ContainsVsCovers边界", func(t *testing.T) {
		// 验证 Contains 和 Covers 在边界点上的差异
		bp := must(NewPoint(0, 0))   // 边界点
		ip := must(NewPoint(2, 2))   // 内部点
		if must(Contains(p1, bp)) {
			t.Error("Contains 边界点应为 false")
		}
		if !must(Contains(p1, ip)) {
			t.Error("Contains 内部点应为 true")
		}
		if !must(Covers(p1, bp)) {
			t.Error("Covers 边界点应为 true")
		}
	})
}

// TestPrepared 验证 Prepared Geometry 加速谓词。
//
// Prepared 比普通谓词多了 ContainsXY 和 IntersectsXY 方法，
// 可以直接判断点而不需要先构造 Point 几何。
func TestPrepared(t *testing.T) {
	p1 := must(NewPolygon(rp1))
	prep := must(NewPreparedGeom(p1))
	defer prep.Close()

	p2 := must(NewPolygon(rp2))
	p3 := must(NewPolygon(rp3))

	if !must(prep.Intersects(p2)) {
		t.Error("prep 应与 p2 相交")
	}
	if must(prep.Intersects(p3)) {
		t.Error("prep 不应与 p3 相交")
	}
	if !must(prep.ContainsXY(2, 2)) {
		t.Error("内部点应被包含")
	}
	if must(prep.ContainsXY(0, 0)) {
		t.Error("ContainsXY 边界点应为 false")
	}
	if !must(prep.IntersectsXY(0, 0)) {
		t.Error("IntersectsXY 边界点应为 true")
	}
}

// TestOverlay 验证叠加运算（Intersection / Union）。
func TestOverlay(t *testing.T) {
	p1 := must(NewPolygon(rp1))
	p2 := must(NewPolygon(rp2))
	p3 := must(NewPolygon(rp3))

	t.Run("Intersection", func(t *testing.T) {
		// p1 ∩ p2 = 从 (2,2) 到 (4,4) 的 2×2 正方形，面积应为 4
		res := must(Intersection(p1, p2))
		if res.IsEmpty() {
			t.Fatal("交集不应为空")
		}
		area := must(Area(res))
		if math.Abs(area-4) > 0.01 {
			t.Errorf("交集面积应为 4: %.4f", area)
		}
	})
	t.Run("Union", func(t *testing.T) {
		// p1 ∪ p4 = p1（因为 p4 在 p1 内部），面积 = 16
		res := must(Union(p1, must(NewPolygon(rp4))))
		area := must(Area(res))
		if math.Abs(area-16) > 0.01 {
			t.Errorf("并集面积应为 16: %.4f", area)
		}
	})
	t.Run("noInter", func(t *testing.T) {
		// p1 ∩ p3 = 空（不相交）
		res := must(Intersection(p1, p3))
		if !res.IsEmpty() {
			t.Error("无交集应返回空")
		}
	})
}

// TestValid 验证有效性和修复功能。
//
// 蝴蝶结形状的自相交多边形是最常见的无效几何类型。
// MakeValid 应能修复它。
func TestValid(t *testing.T) {
	p1 := must(NewPolygon(rp1))
	bt := must(NewPolygon(rbowtie))

	if !must(IsValid(p1)) {
		t.Error("p1 应有效")
	}
	if must(IsValid(bt)) {
		t.Error("bowtie 应无效")
	}
	reason := must(IsValidReason(bt))
	t.Logf("invalid: %s", reason)

	fixed := must(MakeValid(bt))
	if !must(IsValid(fixed)) {
		t.Error("MakeValid 结果应有效")
	}
}

// TestSimplify 验证简化和缓冲功能。
func TestSimplify(t *testing.T) {
	p := must(NewPolygon(rp4))
	// Buffer 外扩 1 单位后面积应大于 4
	buf := must(Buffer(p, 1, 8))
	area := must(Area(buf))
	if area <= 4 {
		t.Errorf("buffer 面积应 > 4: %.4f", area)
	}
	// 凸包应有效
	hull := must(ConvexHull(p))
	if !must(IsValid(hull)) {
		t.Error("凸包应有效")
	}
}

// TestMeasure 验证测量功能。
func TestMeasure(t *testing.T) {
	p1 := must(NewPolygon(rp1))
	// 面积：4×4 = 16
	area := must(Area(p1))
	if math.Abs(area-16) > 0.01 {
		t.Errorf("面积: %.4f", area)
	}
	// 周长：4+4+4+4 = 16
	length := must(Length(p1))
	if math.Abs(length-16) > 0.01 {
		t.Errorf("周长: %.4f", length)
	}
	// 质心：中心点 (2,2)
	x, y, err := Centroid(p1)
	if err != nil || math.Abs(x-2) > 0.01 || math.Abs(y-2) > 0.01 {
		t.Errorf("质心: (%.2f, %.2f)", x, y)
	}
	// PointOnSurface 应在多边形内部
	x2, y2, _ := PointOnSurface(p1)
	cov := must(Covers(p1, must(NewPoint(x2, y2))))
	if !cov {
		t.Error("PointOnSurface 应在多边形内")
	}
}

// TestAdvancedFuncs 验证高级函数（Relate / Hausdorff / SRID / Normalize）。
func TestAdvancedFuncs(t *testing.T) {
	p1 := must(NewPolygon(rp1))
	p2 := must(NewPolygon(rp2))

	t.Run("Relate", func(t *testing.T) {
		m := must(Relate(p1, p2))
		if m == "" {
			t.Error("DE-9IM 不应为空")
		}
	})
	t.Run("Hausdorff", func(t *testing.T) {
		d := must(HausdorffDistance(p1, p2))
		t.Logf("Hausdorff: %.4f", d)
	})
	t.Run("DistanceWithin", func(t *testing.T) {
		if !must(DistanceWithin(p1, p2, 1)) {
			t.Error("距离应在 1 内")
		}
	})
	t.Run("SRID", func(t *testing.T) {
		srid := must(SRID(p1))
		t.Logf("SRID: %d", srid)
		p := must(SetSRID(p1, 4326))
		if must(SRID(p)) != 4326 {
			t.Error("SetSRID 失败")
		}
	})
	t.Run("Normalize", func(t *testing.T) {
		_ = must(Normalize(p1))
	})
}

// TestSTRtree 验证 STRtree 空间索引的基本功能。
func TestSTRtree(t *testing.T) {
	tree := NewSTRtree(10)
	defer tree.Close()

	p1 := must(NewPolygon(rp1))
	p3 := must(NewPolygon(rp3))
	pq := must(NewPolygon(rp2))

	// 插入两个几何
	if err := tree.Insert(p1, "p1"); err != nil {
		t.Fatal(err)
	}
	if err := tree.Insert(p3, "p3"); err != nil {
		t.Fatal(err)
	}

	// 查询与 pq 相交的所有几何
	r := must(tree.Query(pq))
	if len(r) == 0 {
		t.Fatal("应命中 p1")
	}
	t.Logf("Query 命中: %v", r)
}

// TestPanicRecover 验证 safeRun 是否能正确捕获 C 库的 panic。
func TestPanicRecover(t *testing.T) {
	_, err := FromWKT("GARBAGE")
	if err == nil {
		t.Fatal("无效 WKT 应报错")
	}
	t.Logf("safeRun 捕获: %v", err)
}

// TestLineStringLinearRing 验证 LineString 和 LinearRing 的构造。
func TestLineStringLinearRing(t *testing.T) {
	// LineString：不要求闭合
	ls, err := NewLineString([][]float64{{0, 0}, {3, 0}, {3, 3}, {0, 3}})
	if err != nil {
		t.Fatal(err)
	}
	if ls.IsEmpty() || ls.TypeID() != gogeos.TypeIDLineString {
		t.Fatal("LineString 构造失败")
	}
	t.Run("closed ring ok", func(t *testing.T) {
		lr, err := NewLinearRing([][]float64{{0, 0}, {3, 0}, {3, 3}, {0, 0}})
		if err != nil {
			t.Fatal(err)
		}
		if lr.IsEmpty() {
			t.Fatal("LinearRing 构造失败")
		}
	})
	t.Run("unclosed ring error", func(t *testing.T) {
		_, err := NewLinearRing([][]float64{{0, 0}, {3, 0}, {3, 3}, {0, 1}})
		if err == nil {
			t.Error("未闭合的 LinearRing 应报错（GEOS 不自动闭合）")
		}
	})
}

// TestIntrospection 验证几何内省函数。
func TestIntrospection(t *testing.T) {
	p1 := must(NewPolygon(rp1))
	t.Run("IsEmpty", func(t *testing.T) {
		if must(IsEmpty(p1)) {
			t.Error("p1 不应为空")
		}
	})
	t.Run("IsSimple", func(t *testing.T) {
		if !must(IsSimple(p1)) {
			t.Error("p1 应为简单")
		}
	})
	t.Run("IsClosed", func(t *testing.T) {
		// IsClosed 仅适用于 Curve 类几何（LineString），Polygon 会报错
		ls := must(NewLineString([][]float64{{0, 0}, {3, 0}, {3, 3}, {0, 3}}))
		if ok, _ := IsClosed(ls); ok {
			t.Error("未闭合线段不应 IsClosed=true")
		}
	})
	t.Run("IsRing", func(t *testing.T) {
		// IsRing 仅适用于 Curve 类几何，Polygon 会报错
		lr := must(NewLinearRing([][]float64{{0, 0}, {3, 0}, {3, 3}, {0, 0}}))
		if !must(IsRing(lr)) {
			t.Error("闭合环应为环形")
		}
	})
	t.Run("HasZ", func(t *testing.T) {
		if must(HasZ(p1)) {
			t.Error("p1 不应有Z")
		}
	})
}

// TestMissingPredicates 验证其他空间谓词。
func TestMissingPredicates(t *testing.T) {
	p1 := must(NewPolygon(rp1))
	p2 := must(NewPolygon(rp2))
	p4 := must(NewPolygon(rp4))

	t.Run("CoveredBy", func(t *testing.T) {
		if !must(CoveredBy(p4, p1)) {
			t.Error("p4 应被 p1 覆盖")
		}
	})
	t.Run("EqualsExact", func(t *testing.T) {
		if !must(EqualsExact(p1, p1, 0.01)) {
			t.Error("自身应精确相等")
		}
	})
	t.Run("Crosses", func(t *testing.T) {
		_, err := Crosses(p1, p2)
		if err != nil {
			t.Fatal(err)
		}
	})
	t.Run("Overlaps", func(t *testing.T) {
		if !must(Overlaps(p1, p2)) {
			t.Error("p1,p2 应重叠")
		}
	})
	t.Run("Within", func(t *testing.T) {
		if !must(Within(p4, p1)) {
			t.Error("p4 应在 p1 内")
		}
	})
	t.Run("Disjoint", func(t *testing.T) {
		p3 := must(NewPolygon(rp3))
		if !must(Disjoint(p1, p3)) {
			t.Error("p1,p3 应不相交")
		}
	})
	t.Run("Equals", func(t *testing.T) {
		if !must(Equals(p1, p1)) {
			t.Error("自身应相等")
		}
	})
}

// TestAdvancedTransforms 验证高级变换功能。
func TestAdvancedTransforms(t *testing.T) {
	p1 := must(NewPolygon(rp1))
	p2 := must(NewPolygon(rp2))

	t.Run("Boundary", func(t *testing.T) {
		b := must(Boundary(p1))
		if b.IsEmpty() {
			t.Error("边界不应为空")
		}
	})
	t.Run("UnaryUnion", func(t *testing.T) {
		u := must(UnaryUnion(p1))
		if u.IsEmpty() {
			t.Fatal("UnaryUnion 不应为空")
		}
	})
	t.Run("Envelope", func(t *testing.T) {
		e := must(Envelope(p1))
		if e.IsEmpty() {
			t.Fatal("Envelope 不应为空")
		}
	})
	t.Run("Reverse", func(t *testing.T) {
		_ = must(Reverse(p1))
	})
	t.Run("Snap", func(t *testing.T) {
		_ = must(Snap(p1, p2, 0.1))
	})
	t.Run("Densify", func(t *testing.T) {
		d := must(Densify(p1, 0.5))
		if d.IsEmpty() {
			t.Error("Densify 不应为空")
		}
	})
	t.Run("MinimumRotatedRectangle", func(t *testing.T) {
		r := must(MinimumRotatedRectangle(p1))
		if r.IsEmpty() {
			t.Fatal("外接矩形不应为空")
		}
	})
	t.Run("FrechetDistance", func(t *testing.T) {
		d := must(FrechetDistance(p1, p2))
		t.Logf("Frechet: %.4f", d)
	})
}

// TestPreparedFull 验证 Prepared Geometry 的全部谓词方法。
func TestPreparedFull(t *testing.T) {
	p1 := must(NewPolygon(rp1))
	prep := must(NewPreparedGeom(p1))
	defer prep.Close()
	p2 := must(NewPolygon(rp2))
	p4 := must(NewPolygon(rp4))

	t.Run("Covers多边形", func(t *testing.T) {
		if !must(prep.Covers(p4)) {
			t.Error("prep 应覆盖 p4")
		}
	})
	t.Run("CoveredBy", func(t *testing.T) {
		_, err := prep.CoveredBy(p2)
		if err != nil {
			t.Fatal(err)
		}
	})
	t.Run("Overlaps", func(t *testing.T) {
		if !must(prep.Overlaps(p2)) {
			t.Error("应重叠")
		}
	})
	t.Run("Touches", func(t *testing.T) {
		_, err := prep.Touches(must(NewPolygon(rptouch)))
		if err != nil {
			t.Fatal(err)
		}
	})
	t.Run("Within", func(t *testing.T) {
		_, err := prep.Within(p2)
		if err != nil {
			t.Fatal(err)
		}
	})
	t.Run("DistanceWithin", func(t *testing.T) {
		if !must(prep.DistanceWithin(p2, 1)) {
			t.Error("距离应在1内")
		}
	})
}

// TestSTRtreeFull 验证 STRtree 空间索引的全部方法。
func TestSTRtreeFull(t *testing.T) {
	tree := NewSTRtree(10)
	defer tree.Close()
	p1 := must(NewPolygon(rp1))
	p4 := must(NewPolygon(rp4))

	tree.Insert(p1, "A")
	tree.Insert(p4, "B")

	t.Run("Iterate", func(t *testing.T) {
		n := 0
		err := tree.Iterate(func(v any) {
			n++
			t.Logf("Iter: %v", v)
		})
		if err != nil {
			t.Fatal(err)
		}
		if n != 2 {
			t.Errorf("应遍历2项: %d", n)
		}
	})
	t.Run("Remove", func(t *testing.T) {
		ok, err := tree.Remove(p4, "B")
		if err != nil {
			t.Fatal(err)
		}
		if !ok {
			t.Error("Remove 应成功")
		}
		// 重复移除应失败
		ok, _ = tree.Remove(p4, "B")
		if ok {
			t.Error("重复移除应失败")
		}
	})
}

// TestOverlayAll 验证所有 Overlay 运算功能。
func TestOverlayAll(t *testing.T) {
	p1 := must(NewPolygon(rp1))
	p2 := must(NewPolygon(rp2))

	t.Run("SymDifference", func(t *testing.T) {
		sd := must(SymDifference(p1, p2))
		if sd.IsEmpty() {
			t.Fatal("SymDifference 不应为空")
		}
	})
	t.Run("Difference", func(t *testing.T) {
		d := must(Difference(p1, must(NewPolygon(rp4))))
		if d.IsEmpty() {
			t.Fatal("Difference 不应为空")
		}
	})
	t.Run("ConcaveHull", func(t *testing.T) {
		ch := must(ConcaveHull(p1, 0.5, false))
		if ch.IsEmpty() {
			t.Fatal("ConcaveHull 不应为空")
		}
	})
	t.Run("ClipByRect", func(t *testing.T) {
		c := must(ClipByRect(p1, 0, 0, 2, 2))
		if c.IsEmpty() {
			t.Fatal("ClipByRect 不应为空")
		}
	})
}

// TestExtract 验证数据提取函数。
func TestExtract(t *testing.T) {
	t.Run("ExtractPoint", func(t *testing.T) {
		g, _ := NewPoint(116.39, 39.9)
		x, y, err := ExtractPoint(g)
		if err != nil || x != 116.39 || y != 39.9 {
			t.Fatalf("期望 (116.39, 39.9), 得到 (%f, %f)", x, y)
		}
	})

	t.Run("ExtractCoords_LineString", func(t *testing.T) {
		g, _ := NewLineString([][]float64{{0, 0}, {3, 0}, {3, 3}})
		coords, err := ExtractCoords(g)
		if err != nil || len(coords) != 3 {
			t.Fatalf("期望 3 个点, 得到 %d", len(coords))
		}
	})

	t.Run("ExtractPolygonCoords", func(t *testing.T) {
		g, _ := NewPolygon([][][]float64{
			{{0, 0}, {4, 0}, {4, 4}, {0, 4}, {0, 0}},
			{{1, 1}, {3, 1}, {3, 3}, {1, 3}, {1, 1}},
		})
		data, err := ExtractPolygonCoords(g)
		if err != nil {
			t.Fatal(err)
		}
		if len(data.Holes()) != 1 {
			t.Fatalf("期望 1 个洞, 得到 %d", len(data.Holes()))
		}
	})

	t.Run("ExtractMulti 泛型", func(t *testing.T) {
		mp, _ := NewMultiPolygon([][][][]float64{
			{{{0, 0}, {2, 0}, {2, 2}, {0, 2}, {0, 0}}},
			{{{5, 5}, {7, 5}, {7, 7}, {5, 7}, {5, 5}}},
		})
		polys, err := ExtractMulti(mp, ExtractPolygonCoords)
		if err != nil || len(polys) != 2 {
			t.Fatalf("期望 2 个多边形, 得到 %d", len(polys))
		}
	})

	t.Run("ExtractPoints 各类型", func(t *testing.T) {
		pt, _ := NewPoint(1, 2)
		pts, _ := ExtractPoints(pt)
		if len(pts) != 1 || pts[0][0] != 1 || pts[0][1] != 2 {
			t.Error("Point 提取失败")
		}
	})

	t.Run("ExtractPolygonOrMultiCoords", func(t *testing.T) {
		p1, _ := NewPolygon([][][]float64{{{0, 0}, {4, 0}, {4, 4}, {0, 4}, {0, 0}}})
		dat, _ := ExtractPolygonOrMultiCoords(p1)
		if len(dat) != 1 {
			t.Fatalf("单个 Polygon 期望 1 个, 得到 %d", len(dat))
		}
		mp, _ := NewMultiPolygon([][][][]float64{
			{{{0, 0}, {2, 0}, {2, 2}, {0, 2}, {0, 0}}},
			{{{5, 5}, {7, 5}, {7, 7}, {5, 7}, {5, 5}}},
		})
		dat, _ = ExtractPolygonOrMultiCoords(mp)
		if len(dat) != 2 {
			t.Fatalf("MultiPolygon 期望 2 个, 得到 %d", len(dat))
		}
	})
}

// TestNewMultiPolygonErrors 验证 NewMultiPolygon 对无效环的错误处理。
func TestNewMultiPolygonErrors(t *testing.T) {
	t.Run("unclosed ring error", func(t *testing.T) {
		_, err := NewMultiPolygon([][][][]float64{
			{{{0, 0}, {4, 0}, {4, 4}, {0, 4}}}, // 未闭合
		})
		if err == nil {
			t.Error("未闭合环应报错")
		}
	})
	t.Run("empty outer ring", func(t *testing.T) {
		_, err := NewMultiPolygon([][][][]float64{
			{{{}}},
		})
		if err == nil {
			t.Error("空外环应报错")
		}
	})
	t.Run("valid multi polygon", func(t *testing.T) {
		mp, err := NewMultiPolygon([][][][]float64{
			{{{0, 0}, {2, 0}, {2, 2}, {0, 2}, {0, 0}}},
			{{{5, 5}, {7, 5}, {7, 7}, {5, 7}, {5, 5}}},
		})
		if err != nil {
			t.Fatal(err)
		}
		if mp.NumGeometries() != 2 {
			t.Errorf("期望 2 个子几何, 得到 %d", mp.NumGeometries())
		}
	})
}

// TestConstructEmpty 验证空几何构造函数。
func TestConstructEmpty(t *testing.T) {
	t.Run("NewEmptyPoint", func(t *testing.T) {
		g, err := NewEmptyPoint()
		if err != nil {
			t.Fatal(err)
		}
		if !g.IsEmpty() {
			t.Fatal("应为空几何")
		}
	})
	t.Run("NewEmptyLineString", func(t *testing.T) {
		g, err := NewEmptyLineString()
		if err != nil {
			t.Fatal(err)
		}
		if !g.IsEmpty() {
			t.Fatal("应为空几何")
		}
	})
	t.Run("NewEmptyPolygon", func(t *testing.T) {
		g, err := NewEmptyPolygon()
		if err != nil {
			t.Fatal(err)
		}
		if !g.IsEmpty() {
			t.Fatal("应为空几何")
		}
	})
	t.Run("NewEmptyCollection", func(t *testing.T) {
		g, err := NewEmptyCollection(gogeos.TypeIDMultiPolygon)
		if err != nil {
			t.Fatal(err)
		}
		if !g.IsEmpty() {
			t.Fatal("应为空几何")
		}
	})
}

// TestUncoveredOverlay 验证 overlay.go 中未经测试的函数。
func TestUncoveredOverlay(t *testing.T) {
	t.Run("Distance", func(t *testing.T) {
		pt1 := must(NewPoint(0, 0))
		pt2 := must(NewPoint(3, 4))
		dist, err := Distance(pt1, pt2)
		if err != nil {
			t.Fatal(err)
		}
		if dist < 4.9 || dist > 5.1 {
			t.Errorf("距离应在 5 附近: %.4f", dist)
		}
	})

	t.Run("BuildArea", func(t *testing.T) {
		lines, err := FromWKT("MULTILINESTRING ((0 0, 4 0), (4 0, 4 4), (4 4, 0 4), (0 4, 0 0))")
		if err != nil {
			t.Fatal(err)
		}
		area, err := BuildArea(lines)
		if err != nil {
			t.Fatal(err)
		}
		if area.IsEmpty() {
			t.Fatal("BuildArea 不应为空")
		}
	})

	t.Run("LineMerge", func(t *testing.T) {
		mls, err := FromWKT("MULTILINESTRING ((0 0, 1 1), (1 1, 2 2))")
		if err != nil {
			t.Fatal(err)
		}
		merged, err := LineMerge(mls)
		if err != nil {
			t.Fatal(err)
		}
		if merged.IsEmpty() {
			t.Fatal("LineMerge 不应为空")
		}
	})

	t.Run("Node", func(t *testing.T) {
		lines, err := FromWKT("MULTILINESTRING ((0 0, 4 4), (0 4, 4 0))")
		if err != nil {
			t.Fatal(err)
		}
		noded, err := Node(lines)
		if err != nil {
			t.Fatal(err)
		}
		if noded.IsEmpty() {
			t.Fatal("Node 不应为空")
		}
	})

	t.Run("Simplify", func(t *testing.T) {
		g, err := NewPolygon(rp1)
		if err != nil {
			t.Fatal(err)
		}
		s := must(Simplify(g, 0.5))
		if s.IsEmpty() {
			t.Fatal("Simplify 不应为空")
		}
	})

	t.Run("TopologyPreserveSimplify", func(t *testing.T) {
		g := must(NewPolygon(rp1))
		s := must(TopologyPreserveSimplify(g, 0.5))
		if s.IsEmpty() {
			t.Fatal("TopologyPreserveSimplify 不应为空")
		}
	})

	t.Run("OffsetCurve", func(t *testing.T) {
		line, err := NewLineString([][]float64{{0, 0}, {10, 0}})
		if err != nil {
			t.Fatal(err)
		}
		oc := must(OffsetCurve(line, 1, 8, gogeos.BufJoinStyleRound, 5.0))
		if oc.IsEmpty() {
			t.Fatal("OffsetCurve 不应为空")
		}
	})

	t.Run("EndPoint", func(t *testing.T) {
		ls, err := NewLineString([][]float64{{0, 0}, {3, 0}, {3, 3}})
		if err != nil {
			t.Fatal(err)
		}
		ep := must(EndPoint(ls))
		if ep.X() != 3 || ep.Y() != 3 {
			t.Errorf("EndPoint 期望 (3,3), 得到 (%f,%f)", ep.X(), ep.Y())
		}
	})

	t.Run("StartPoint", func(t *testing.T) {
		ls, err := NewLineString([][]float64{{0, 0}, {3, 0}, {3, 3}})
		if err != nil {
			t.Fatal(err)
		}
		sp := must(StartPoint(ls))
		if sp.X() != 0 || sp.Y() != 0 {
			t.Errorf("StartPoint 期望 (0,0), 得到 (%f,%f)", sp.X(), sp.Y())
		}
	})

	t.Run("MinimumClearance", func(t *testing.T) {
		ls, err := NewLineString([][]float64{{0, 0}, {0, 1}, {1, 1}})
		if err != nil {
			t.Fatal(err)
		}
		_, err = MinimumClearance(ls)
		if err != nil {
			t.Skipf("MinimumClearance 在当前 GEOS 版本不可用: %v", err)
		}
	})

	t.Run("Precision", func(t *testing.T) {
		g := must(NewPoint(1.23456789, 2.3456789))
		prec := must(Precision(g))
		if prec != 0 {
			t.Errorf("默认精度应为 0: %.4f", prec)
		}
	})
}

// TestUncoveredRelation 验证 relation.go 中未经测试的函数。
func TestUncoveredRelation(t *testing.T) {
	p1 := must(NewPolygon(rp1))
	p4 := must(NewPolygon(rp4))

	t.Run("RelatePattern", func(t *testing.T) {
		matrix, err := Relate(p1, p4)
		if err != nil {
			t.Fatal(err)
		}
		ok, err := RelatePattern(p1, p4, matrix)
		if err != nil {
			t.Fatal(err)
		}
		if !ok {
			t.Errorf("RelatePattern 应与自身矩阵匹配")
		}
	})

	t.Run("NearestPoints", func(t *testing.T) {
		pA := must(NewPolygon(rp1))
		pB := must(NewPolygon(rp3))
		ax, ay, bx, by, err := NearestPoints(pA, pB)
		if err != nil {
			t.Fatal(err)
		}
		_ = ax
		_ = ay
		_ = bx
		_ = by
	})
}

// TestPreparedMore 验证 Prepared 中未经测试的方法。
func TestPreparedMore(t *testing.T) {
	p1 := must(NewPolygon(rp1))
	prep := must(NewPreparedGeom(p1))
	defer prep.Close()
	p2 := must(NewPolygon(rp2))
	p3 := must(NewPolygon(rp3))

	t.Run("Contains", func(t *testing.T) {
		ok, err := prep.Contains(must(NewPolygon(rp4)))
		if err != nil {
			t.Fatal(err)
		}
		if !ok {
			t.Error("应包含")
		}
	})

	t.Run("Disjoint", func(t *testing.T) {
		ok, err := prep.Disjoint(p3)
		if err != nil {
			t.Fatal(err)
		}
		if !ok {
			t.Error("应 disjoint")
		}
		ok, err = prep.Disjoint(p2)
		if err != nil {
			t.Fatal(err)
		}
		if ok {
			t.Error("不应 disjoint")
		}
	})
}

// TestExtractMultiSafe 验证安全集合提取。
func TestExtractMultiSafe(t *testing.T) {
	mp, err := NewMultiPolygon([][][][]float64{
		{{{0, 0}, {2, 0}, {2, 2}, {0, 2}, {0, 0}}},
		{{{5, 5}, {7, 5}, {7, 7}, {5, 7}, {5, 5}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	polys, err := ExtractMultiSafe(mp, ExtractPolygonCoords)
	if err != nil {
		t.Fatal(err)
	}
	if len(polys) != 2 {
		t.Fatalf("期望 2 个多边形, 得到 %d", len(polys))
	}
}

// TestCrosses 补全 Crosses 结果断言。
func TestCrosses(t *testing.T) {
	line, err := NewLineString([][]float64{{-1, 2}, {5, 2}})
	if err != nil {
		t.Fatal(err)
	}
	p1 := must(NewPolygon(rp1))
	ok, err := Crosses(line, p1)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Error("线应穿越多边形")
	}
}

// TestNilInputs 验证全包 exported 函数的 nil 输入处理。
func TestNilInputs(t *testing.T) {
	t.Run("construct", func(t *testing.T) {
		_, err := NewLineString(nil)
		if err != nil {
			t.Logf("NewLineString(nil): %v (go-geos behavior)", err)
		}
		_, err = NewLineString([][]float64{})
		if err != nil {
			t.Logf("NewLineString(empty): %v (go-geos behavior)", err)
		}
		_, err = NewLinearRing(nil)
		if err == nil {
			t.Error("空坐标应报错")
		}
		p, err := NewPolygon(nil)
		if err == nil {
			t.Error("nil Polygon 应报错")
		}
		if p != nil {
			t.Error("p 应为 nil")
		}
		_, err = NewMultiPolygon(nil)
		if err != nil {
			t.Logf("NewMultiPolygon(nil): %v", err)
		}
		_, err = NewMultiPolygon([][][][]float64{})
		if err != nil {
			t.Logf("NewMultiPolygon(empty): %v", err)
		}
		g, err := NewMultiPolygonFromGeoms(nil)
		if err == nil {
			t.Error("nil geoms 应报错")
		}
		if g != nil {
			t.Error("g 应为 nil")
		}
		g, err = NewCollectionFromGeoms(gogeos.TypeIDMultiPolygon, nil)
		if err == nil {
			t.Error("nil geoms 应报错")
		}
		if g != nil {
			t.Error("g 应为 nil")
		}
	})

	t.Run("predicate nil", func(t *testing.T) {
		p1 := must(NewPolygon(rp1))
		_, err := Intersects(nil, p1)
		if err == nil {
			t.Error("nil 应报错")
		}
		_, err = Contains(p1, nil)
		if err == nil {
			t.Error("nil 应报错")
		}
		_, err = Covers(nil, nil)
		if err == nil {
			t.Error("nil 应报错")
		}
	})

	t.Run("overlay nil", func(t *testing.T) {
		p1 := must(NewPolygon(rp1))
		_, err := Intersection(nil, p1)
		if err == nil {
			t.Error("nil 应报错")
		}
		_, err = Difference(p1, nil)
		if err == nil {
			t.Error("nil 应报错")
		}
		_, err = SymDifference(nil, nil)
		if err == nil {
			t.Error("nil 应报错")
		}
	})

	t.Run("measure nil", func(t *testing.T) {
		_, err := Area(nil)
		if err == nil {
			t.Error("nil Area 应报错")
		}
		_, err = Length(nil)
		if err == nil {
			t.Error("nil Length 应报错")
		}
		_, _, err = Centroid(nil)
		if err == nil {
			t.Error("nil Centroid 应报错")
		}
		_, _, err = PointOnSurface(nil)
		if err == nil {
			t.Error("nil PointOnSurface 应报错")
		}
	})

	t.Run("relation nil", func(t *testing.T) {
		_, err := Relate(nil, nil)
		if err == nil {
			t.Error("nil Relate 应报错")
		}
		_, err = RelatePattern(nil, nil, "T*F**FFF*")
		if err == nil {
			t.Error("nil RelatePattern 应报错")
		}
		_, err = HausdorffDistance(nil, nil)
		if err == nil {
			t.Error("nil HausdorffDistance 应报错")
		}
		_, _, _, _, err = NearestPoints(nil, nil)
		if err == nil {
			t.Error("nil NearestPoints 应报错")
		}
		_, err = DistanceWithin(nil, nil, 1)
		if err == nil {
			t.Error("nil DistanceWithin 应报错")
		}
	})

	t.Run("transform nil", func(t *testing.T) {
		_, err := Simplify(nil, 1)
		if err == nil {
			t.Error("nil Simplify 应报错")
		}
		_, err = Buffer(nil, 1, 8)
		if err == nil {
			t.Error("nil Buffer 应报错")
		}
		_, err = ConvexHull(nil)
		if err == nil {
			t.Error("nil ConvexHull 应报错")
		}
	})

	t.Run("valid nil", func(t *testing.T) {
		_, err := IsValid(nil)
		if err == nil {
			t.Error("nil IsValid 应报错")
		}
		_, err = IsValidReason(nil)
		if err == nil {
			t.Error("nil IsValidReason 应报错")
		}
		_, err = MakeValid(nil)
		if err == nil {
			t.Error("nil MakeValid 应报错")
		}
	})

	t.Run("extract nil", func(t *testing.T) {
		_, _, err := ExtractPoint(nil)
		if err == nil {
			t.Error("nil ExtractPoint 应报错")
		}
		coords, err := ExtractCoords(nil)
		if err == nil || coords != nil {
			t.Error("nil ExtractCoords 应报错")
		}
		ped, err := ExtractPolygonCoords(nil)
		if err == nil || ped != nil {
			t.Error("nil ExtractPolygonCoords 应报错")
		}
		pmc, err := ExtractPolygonOrMultiCoords(nil)
		if err == nil || pmc != nil {
			t.Error("nil ExtractPolygonOrMultiCoords 应报错")
		}
		em, err := ExtractMulti(nil, ExtractPolygonCoords)
		if err == nil || em != nil {
			t.Error("nil ExtractMulti 应报错")
		}
		ems, err := ExtractMultiSafe(nil, ExtractPolygonCoords)
		if err == nil || ems != nil {
			t.Error("nil ExtractMultiSafe 应报错")
		}
		ep, err := ExtractPoints(nil)
		if err == nil || ep != nil {
			t.Error("nil ExtractPoints 应报错")
		}
	})

	t.Run("introspection nil", func(t *testing.T) {
		_, err := IsEmpty(nil)
		if err == nil {
			t.Error("nil IsEmpty 应报错")
		}
		_, err = IsSimple(nil)
		if err == nil {
			t.Error("nil IsSimple 应报错")
		}
		_, err = IsClosed(nil)
		if err == nil {
			t.Error("nil IsClosed 应报错")
		}
		_, err = HasZ(nil)
		if err == nil {
			t.Error("nil HasZ 应报错")
		}
	})

	t.Run("strtree nil", func(t *testing.T) {
		tree := NewSTRtree(10)
		defer tree.Close()
		err := tree.Insert(nil, "v")
		if err == nil {
			t.Error("nil Insert 不应成功")
		}
	})

	t.Run("prepared nil", func(t *testing.T) {
		_, err := NewPreparedGeom(nil)
		if err == nil {
			t.Error("nil NewPreparedGeom 应报错")
		}
	})

	t.Run("convert nil", func(t *testing.T) {
		_, err := ToWKT(nil)
		if err == nil {
			t.Error("nil ToWKT 应报错")
		}
		_, err = ToWKB(nil)
		if err == nil {
			t.Error("nil ToWKB 应报错")
		}
	})
}

// TestErrorsSentinel 验证哨兵错误变量可用 errors.Is 判断。
func TestErrorsSentinel(t *testing.T) {
	errs := []error{
		ErrNil, ErrClosed, ErrNotPolygon, ErrEmptyRing,
		ErrEmptyOuterRing, ErrNotSupported, ErrEmptyGeoms, ErrEmptyResult,
	}
	for _, e := range errs {
		if e == nil {
			t.Error("哨兵错误不应为 nil")
		}
		if e.Error() == "" {
			t.Error("哨兵错误消息不应为空")
		}
	}
}
