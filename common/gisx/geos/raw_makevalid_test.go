// Package geos_test — GEOS MakeValid 行为全覆盖实测
//
// 本文件保留所有测试数据用于：
//  1. GEOS 版本升级后回归验证 MakeValid 行为是否变化
//  2. 排查无效多边形问题时的快速参考
//  3. MakeValidOrb 实现正确性的验证基准
//
// 运行：go test -count=1 -run "TestMakeValid_Raw|TestSub1Analysis" -v ./common/gisx/geos/
package geos_test

import (
	"fmt"
	"testing"

	gogeos "github.com/twpayne/go-geos"
	"zero-service/common/gisx/geos"
)

func bbox(pts [][]float64) (xMin, xMax, yMin, yMax float64) {
	xMin, yMin = pts[0][0], pts[0][1]
	xMax, yMax = xMin, yMin
	for _, p := range pts {
		if p[0] < xMin {
			xMin = p[0]
		}
		if p[0] > xMax {
			xMax = p[0]
		}
		if p[1] < yMin {
			yMin = p[1]
		}
		if p[1] > yMax {
			yMax = p[1]
		}
	}
	return
}

// TestMakeValid_Raw 覆盖 11 个 Polygon 场景，打印 GEOS MakeValid 原始输出
func TestMakeValid_Raw(t *testing.T) {
	outer := [][]float64{{0, 0}, {10, 0}, {10, 10}, {0, 10}, {0, 0}}

	type tc struct {
		idx   int
		label string
		rings [][][]float64
	}
	cases := []tc{
		{1, "正常无洞", [][][]float64{outer}},
		{2, "有效单洞", [][][]float64{outer, {{2, 2}, {4, 2}, {4, 4}, {2, 4}, {2, 2}}}},
		{3, "有效多洞", [][][]float64{outer, {{2, 2}, {3, 2}, {3, 3}, {2, 3}, {2, 2}}, {{5, 5}, {6, 5}, {6, 6}, {5, 6}, {5, 5}}}},
		{4, "重叠洞", [][][]float64{outer, {{2, 2}, {6, 2}, {6, 6}, {2, 6}, {2, 2}}, {{4, 4}, {8, 4}, {8, 8}, {4, 8}, {4, 4}}}},
		{5, "洞包含洞", [][][]float64{outer, {{2, 2}, {8, 2}, {8, 8}, {2, 8}, {2, 2}}, {{3, 3}, {5, 3}, {5, 5}, {3, 5}, {3, 3}}}},
		{6, "洞超出外环", [][][]float64{outer, {{5, 2}, {15, 2}, {15, 8}, {5, 8}, {5, 2}}}},
		{7, "洞完全在外", [][][]float64{outer, {{15, 15}, {20, 15}, {20, 20}, {15, 20}, {15, 15}}}},
		{8, "自相交蝴蝶结", [][][]float64{{{0, 0}, {4, 0}, {0, 4}, {4, 4}, {0, 0}}}},
		{9, "退化三点共线", [][][]float64{{{0, 0}, {5, 0}, {10, 0}, {0, 0}}}},
		{10, "洞碰外环边", [][][]float64{outer, {{0, 0}, {5, 0}, {5, 5}, {0, 5}, {0, 0}}}},
		{11, "三洞链式重叠", [][][]float64{outer, {{2, 2}, {5, 2}, {5, 5}, {2, 5}, {2, 2}}, {{3, 3}, {6, 3}, {6, 6}, {3, 6}, {3, 3}}, {{4, 4}, {7, 4}, {7, 7}, {4, 7}, {4, 4}}}},
	}

	for _, c := range cases {
		p, err := geos.NewPolygon(c.rings)
		if err != nil {
			fmt.Printf("%2d. %-16s → NewPolygon失败: %v\n", c.idx, c.label, err)
			continue
		}
		valid, _ := geos.IsValid(p)
		reason, _ := geos.IsValidReason(p)
		fixed, err := geos.MakeValid(p)
		if err != nil {
			fmt.Printf("%2d. %-16s → MakeValid失败: %v\n", c.idx, c.label, err)
			continue
		}

		fmt.Printf("%2d. %-16s valid=%-5v reason=%s\n", c.idx, c.label, valid, reason)
		n := fixed.NumGeometries()
		for i := 0; i < n; i++ {
			sub := fixed.Geometry(i)
			switch sub.TypeID() {
			case gogeos.TypeIDPolygon:
				e := sub.ExteriorRing().CoordSeq().ToCoords()
				xMin, xMax, yMin, yMax := bbox(e)
				nr := sub.NumInteriorRings()
				fmt.Printf("    sub[%d] Polygon 外环%d点 x[%.0f-%.0f] y[%.0f-%.0f] 洞数=%d\n",
					i, len(e), xMin, xMax, yMin, yMax, nr)
			case gogeos.TypeIDLineString:
				fmt.Printf("    sub[%d] LineString\n", i)
			case gogeos.TypeIDPoint:
				fmt.Printf("    sub[%d] Point\n", i)
			default:
				fmt.Printf("    sub[%d] TypeID=%d\n", i, sub.TypeID())
			}
		}
	}
}

// TestSub1Analysis 分析 MultiPolygon 场景下 sub[1] 的真实数据
// 验证"取 sub[0] 丢弃 sub[1]"的语义正确性
func TestSub1Analysis(t *testing.T) {
	outer := [][]float64{{0, 0}, {10, 0}, {10, 10}, {0, 10}, {0, 0}}

	type tc struct {
		label    string
		rings    [][][]float64
		sub1Desc string
	}
	cases := []tc{
		{"重叠洞", [][][]float64{outer,
			{{2, 2}, {6, 2}, {6, 6}, {2, 6}, {2, 2}},
			{{4, 4}, {8, 4}, {8, 8}, {4, 8}, {4, 4}}},
			"重叠区域方块, 已被 sub[0] 合并洞排除, 丢弃无意义"},
		{"洞超出外环", [][][]float64{outer,
			{{5, 2}, {15, 2}, {15, 8}, {5, 8}, {5, 2}}},
			"越界部分, sub[0] 已通过凹口绕过, 丢弃无意义"},
		{"洞完全在外", [][][]float64{outer,
			{{15, 15}, {20, 15}, {20, 20}, {15, 20}, {15, 15}}},
			"洞本身在原始bbox外, 丢弃无意义"},
		{"自相交蝴蝶结", [][][]float64{
			{{0, 0}, {4, 0}, {0, 4}, {4, 4}, {0, 0}}},
			"下半三角, 原始无效无法保留全量"},
	}

	for _, c := range cases {
		p, _ := geos.NewPolygon(c.rings)
		f, _ := geos.MakeValid(p)
		n := f.NumGeometries()
		if n < 2 {
			fmt.Printf("%-16s sub<2, 跳过\n", c.label)
			continue
		}
		e0 := f.Geometry(0).ExteriorRing().CoordSeq().ToCoords()
		x0, x1, y0, y1 := bbox(e0)
		nr0 := f.Geometry(0).NumInteriorRings()

		e1 := f.Geometry(1).ExteriorRing().CoordSeq().ToCoords()
		x10, x11, y10, y11 := bbox(e1)
		nr1 := f.Geometry(1).NumInteriorRings()

		fmt.Printf("%-16s sub[0]=外环%d点 x[%.0f-%.0f] y[%.0f-%.0f] 洞%d | sub[1]=外环%d点 x[%.0f-%.0f] y[%.0f-%.0f] 洞%d | %s\n",
			c.label, len(e0), x0, x1, y0, y1, nr0, len(e1), x10, x11, y10, y11, nr1, c.sub1Desc)
	}
}
