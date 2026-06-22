package orbconv

import (
	"testing"

	"github.com/paulmach/orb"
)

var (
	op1 = orb.Polygon{orb.Ring{{0, 0}, {4, 0}, {4, 4}, {0, 4}, {0, 0}}}
	op2 = orb.Polygon{orb.Ring{{2, 2}, {6, 2}, {6, 6}, {2, 6}, {2, 2}}}
	op3 = orb.Polygon{orb.Ring{{10, 10}, {12, 10}, {12, 12}, {10, 12}, {10, 10}}}
	op4 = orb.Polygon{orb.Ring{{1, 1}, {3, 1}, {3, 3}, {1, 3}, {1, 1}}}
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
		if err != nil || p != nil {
			t.Error("nil 应返回 nil, nil")
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
