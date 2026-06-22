package geos

import (
	"math"
	"testing"

	gogeos "github.com/twpayne/go-geos"
)

// 测试用几何（纯坐标）
var (
	rp1 = [][][]float64{{{0, 0}, {4, 0}, {4, 4}, {0, 4}, {0, 0}}}
	rp2 = [][][]float64{{{2, 2}, {6, 2}, {6, 6}, {2, 6}, {2, 2}}}
	rp3 = [][][]float64{{{10, 10}, {12, 10}, {12, 12}, {10, 12}, {10, 10}}}
	rp4 = [][][]float64{{{1, 1}, {3, 1}, {3, 3}, {1, 3}, {1, 1}}}
	rbowtie = [][][]float64{{{0, 0}, {4, 0}, {0, 4}, {4, 4}, {0, 0}}}
	rptouch = [][][]float64{{{4, 0}, {8, 0}, {8, 4}, {4, 4}, {4, 0}}}
)

func must[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}
	return v
}

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

func TestConstruct(t *testing.T) {
	t.Run("Point", func(t *testing.T) {
		g := must(NewPoint(1, 2))
		if g.X() != 1 || g.Y() != 2 {
			t.Fatalf("Point: (%f,%f)", g.X(), g.Y())
		}
	})
	t.Run("Polygon", func(t *testing.T) {
		g := must(NewPolygon(rp1))
		if g.TypeID() != gogeos.TypeIDPolygon {
			t.Fatal("类型应为 Polygon")
		}
	})
	t.Run("BoundsRect", func(t *testing.T) {
		g := must(NewBoundsRect(0, 0, 4, 4))
		if g.IsEmpty() {
			t.Fatal("不应为空")
		}
	})
}

func TestWKTConvert(t *testing.T) {
	g := must(FromWKT("POLYGON ((0 0, 4 0, 4 4, 0 4, 0 0))"))
	wkt := must(ToWKT(g))
	t.Logf("WKT: %s", wkt)

	_, err := FromWKT("NOT WKT")
	if err == nil {
		t.Fatal("无效 WKT 应报错")
	}
}

func TestWKBConvert(t *testing.T) {
	g := must(NewPolygon(rp1))
	wkb := must(ToWKB(g))
	if len(wkb) == 0 {
		t.Fatal("WKB 不应为空")
	}
	g2 := must(FromWKB(wkb))
	if g2.IsEmpty() {
		t.Fatal("往返失败")
	}
}

func TestGeoJSONConvert(t *testing.T) {
	g := must(NewPolygon(rp1))
	json := must(ToGeoJSON(g, 0))
	t.Logf("GeoJSON: %s", json)
	g2 := must(FromGeoJSON(json))
	if g2.IsEmpty() {
		t.Fatal("往返失败")
	}
}

func TestPredicates(t *testing.T) {
	p1 := must(NewPolygon(rp1))
	p2 := must(NewPolygon(rp2))
	p3 := must(NewPolygon(rp3))
	p4 := must(NewPolygon(rp4))
	pt := must(NewPolygon(rptouch))

	t.Run("Intersects", func(t *testing.T) {
		if !must(Intersects(p1, p2)) {
			t.Error("p1,p2 应相交")
		}
		if must(Intersects(p1, p3)) {
			t.Error("p1,p3 不应相交")
		}
	})
	t.Run("Contains", func(t *testing.T) {
		if !must(Contains(p1, p4)) {
			t.Error("p1 应包含 p4")
		}
	})
	t.Run("Covers", func(t *testing.T) {
		if !must(Covers(p1, p4)) {
			t.Error("p1 应覆盖 p4")
		}
	})
	t.Run("Touches", func(t *testing.T) {
		if !must(Touches(p1, pt)) {
			t.Error("p1,pt 应边界接触")
		}
	})
	t.Run("ContainsVsCovers边界", func(t *testing.T) {
		bp := must(NewPoint(0, 0))
		ip := must(NewPoint(2, 2))
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

func TestOverlay(t *testing.T) {
	p1 := must(NewPolygon(rp1))
	p2 := must(NewPolygon(rp2))
	p3 := must(NewPolygon(rp3))

	t.Run("Intersection", func(t *testing.T) {
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
		res := must(Union(p1, must(NewPolygon(rp4))))
		area := must(Area(res))
		if math.Abs(area-16) > 0.01 {
			t.Errorf("并集面积应为 16: %.4f", area)
		}
	})
	t.Run("noInter", func(t *testing.T) {
		res := must(Intersection(p1, p3))
		if !res.IsEmpty() {
			t.Error("无交集应返回空")
		}
	})
}

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

func TestSimplify(t *testing.T) {
	p := must(NewPolygon(rp4))
	buf := must(Buffer(p, 1, 8))
	area := must(Area(buf))
	if area <= 4 {
		t.Errorf("buffer 面积应 > 4: %.4f", area)
	}
	hull := must(ConvexHull(p))
	if !must(IsValid(hull)) {
		t.Error("凸包应有效")
	}
}

func TestMeasure(t *testing.T) {
	p1 := must(NewPolygon(rp1))
	area := must(Area(p1))
	if math.Abs(area-16) > 0.01 {
		t.Errorf("面积: %.4f", area)
	}
	length := must(Length(p1))
	if math.Abs(length-16) > 0.01 {
		t.Errorf("周长: %.4f", length)
	}
	x, y, err := Centroid(p1)
	if err != nil || math.Abs(x-2) > 0.01 || math.Abs(y-2) > 0.01 {
		t.Errorf("质心: (%.2f, %.2f)", x, y)
	}
	x2, y2, _ := PointOnSurface(p1)
	cov := must(Covers(p1, must(NewPoint(x2, y2))))
	if !cov {
		t.Error("PointOnSurface 应在多边形内")
	}
}

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

func TestSTRtree(t *testing.T) {
	tree := NewSTRtree(10)
	defer tree.Close()

	p1 := must(NewPolygon(rp1))
	p3 := must(NewPolygon(rp3))
	pq := must(NewPolygon(rp2))

	if err := tree.Insert(p1, "p1"); err != nil {
		t.Fatal(err)
	}
	if err := tree.Insert(p3, "p3"); err != nil {
		t.Fatal(err)
	}

	r := must(tree.Query(pq))
	if len(r) == 0 {
		t.Fatal("应命中 p1")
	}
	t.Logf("Query 命中: %v", r)
}

func TestPanicRecover(t *testing.T) {
	_, err := FromWKT("GARBAGE")
	if err == nil {
		t.Fatal("无效 WKT 应报错")
	}
	t.Logf("safeRun 捕获: %v", err)
}

func TestLineStringLinearRing(t *testing.T) {
	ls, err := NewLineString([][]float64{{0, 0}, {3, 0}, {3, 3}, {0, 3}})
	if err != nil {
		t.Fatal(err)
	}
	if ls.IsEmpty() || ls.TypeID() != gogeos.TypeIDLineString {
		t.Fatal("LineString 构造失败")
	}
	lr, err := NewLinearRing([][]float64{{0, 0}, {3, 0}, {3, 3}, {0, 0}})
	if err != nil {
		t.Fatal(err)
	}
	if lr.IsEmpty() {
		t.Fatal("LinearRing 构造失败")
	}
}

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

func TestSTRtreeFull(t *testing.T) {
	tree := NewSTRtree(10)
	defer tree.Close()
	p1 := must(NewPolygon(rp1))
	p4 := must(NewPolygon(rp4))

	tree.Insert(p1, "A")
	tree.Insert(p4, "B")

	t.Run("Iterate", func(t *testing.T) {
		n := 0
		tree.Iterate(func(v any) {
			n++
			t.Logf("Iter: %v", v)
		})
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
		ok, _ = tree.Remove(p4, "B")
		if ok {
			t.Error("重复移除应失败")
		}
	})
}

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
