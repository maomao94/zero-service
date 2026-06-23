package geos

// predicate.go — 空间谓词（Spatial Predicates）
//
// 空间谓词用于判断两个几何对象之间的空间关系。
// 所有谓词返回 (bool, error)。
//
// 核心概念：DE-9IM 模型
// GEOS 的空间谓词基于 DE-9IM（Dimensionally Extended 9-Intersection Model），
// 这是一个数学模型，用于描述两个几何之间的拓扑关系。
//
// 重点理解：Contains vs Covers
//   - Contains: 严格包含，边界点不算。点 (0,0) 对多边形 (0,0)-(4,0)-(4,4)-(0,4) Contains = false
//   - Covers: 覆盖，边界点算。点 (0,0) 对同一多边形 Covers = true
//
// 围栏命中判断必须用 Covers，不要用 Contains！
// 因为用户站在围栏边界上也算命中。
//
// 所有谓词通过 predicateTwo 辅助函数统一处理 nil 检查和 panic 捕获。

import (
	gogeos "github.com/twpayne/go-geos"
)

// predicateTwo 是内部辅助函数，用于二元谓词（两个几何之间的关系判断）。
// 统一处理 nil 检查和 panic 捕获。
func predicateTwo(a, b *gogeos.Geom, fn func() bool) (bool, error) {
	if a == nil || b == nil {
		return false, errNil
	}
	return safeRun(func() (bool, error) { return fn(), nil })
}

// Intersects 判断两个几何是否有任意公共点（含边界接触）。
//
// 这是最常用的空间谓词，判断两个几何是否有任何交集。
// 包括：重叠、包含、边界接触等各种情况。
//
// 示例：
//
//	p1, _ := geos.NewPolygon([][][]float64{{{0,0},{4,0},{4,4},{0,4},{0,0}}})
//	p2, _ := geos.NewPolygon([][][]float64{{{2,2},{6,2},{6,6},{2,6},{2,2}}})
//	p3, _ := geos.NewPolygon([][][]float64{{{10,10},{12,10},{12,12},{10,12},{10,10}}})
//
//	ok, _ := geos.Intersects(p1, p2)  // true（有重叠区域）
//	ok, _ = geos.Intersects(p1, p3)   // false（完全分离）
func Intersects(a, b *gogeos.Geom) (bool, error) {
	return predicateTwo(a, b, func() bool { return a.Intersects(b) })
}

// Contains 判断 a 是否严格包含 b（边界点不算，OGC 语义）。
//
// b 的所有点都必须严格在 a 的内部，不能接触 a 的边界。
//
// Contains vs Covers 的关键区别：
//   - Contains(point(0,0), polygon(0,0)-(4,0)-(4,4)-(0,4)) = false（点在边界上）
//   - Covers(point(0,0), polygon(0,0)-(4,0)-(4,4)-(0,4))  = true（边界算覆盖）
//
// 围栏命中场景不要用 Contains，用 Covers！
//
// 示例：
//
//	outer, _ := geos.NewPolygon([][][]float64{{{0,0},{10,0},{10,10},{0,10},{0,0}}})
//	inner, _ := geos.NewPolygon([][][]float64{{{2,2},{8,2},{8,8},{2,8},{2,2}}})
//	boundary, _ := geos.NewPoint(0, 0)
//
//	ok, _ := geos.Contains(outer, inner)    // true（内部多边形）
//	ok, _ = geos.Contains(outer, boundary)  // false（边界点不算）
func Contains(a, b *gogeos.Geom) (bool, error) {
	return predicateTwo(a, b, func() bool { return a.Contains(b) })
}

// Covers 判断 a 是否覆盖 b（边界点算，围栏命中场景）。
//
// b 的所有点都在 a 的内部或边界上。
//
// 这是围栏命中判断的正确选择：
//   - 用户在围栏内部 → Covers = true
//   - 用户在围栏边界上 → Covers = true
//   - 用户在围栏外部 → Covers = false
//
// 示例：
//
//	fence, _ := geos.NewPolygon([][][]float64{{{0,0},{10,0},{10,10},{0,10},{0,0}}})
//	point, _ := geos.NewPoint(0, 0)  // 边界点
//
//	ok, _ := geos.Covers(fence, point)   // true（边界算覆盖）
//	ok, _ = geos.Contains(fence, point)  // false（边界不算包含）
func Covers(a, b *gogeos.Geom) (bool, error) {
	return predicateTwo(a, b, func() bool { return a.Covers(b) })
}

// Within 判断 a 是否在 b 内部（Contains 的反向）。
//
// Within(a, b) 等价于 Contains(b, a)。
// 即 a 的所有点都在 b 的内部。
//
// 示例：
//
//	inner, _ := geos.NewPolygon([][][]float64{{{2,2},{8,2},{8,8},{2,8},{2,2}}})
//	outer, _ := geos.NewPolygon([][][]float64{{{0,0},{10,0},{10,10},{0,10},{0,0}}})
//
//	ok, _ := geos.Within(inner, outer)  // true
func Within(a, b *gogeos.Geom) (bool, error) {
	return predicateTwo(a, b, func() bool { return a.Within(b) })
}

// Touches 判断两个几何是否仅边界接触（内部无交集）。
//
// 两个几何只在边界上接触，内部没有重叠。
// 例如：相邻的两个正方形共享一条边。
//
// 示例：
//
//	p1, _ := geos.NewPolygon([][][]float64{{{0,0},{4,0},{4,4},{0,4},{0,0}}})
//	p2, _ := geos.NewPolygon([][][]float64{{{4,0},{8,0},{8,4},{4,4},{4,0}}})
//
//	ok, _ := geos.Touches(p1, p2)  // true（共享右边和左边）
func Touches(a, b *gogeos.Geom) (bool, error) {
	return predicateTwo(a, b, func() bool { return a.Touches(b) })
}

// Disjoint 判断两个几何是否完全无交集。
//
// 两个几何没有任何公共点，包括边界。
// Disjoint 是 Intersects 的完全反义。
//
// 示例：
//
//	p1, _ := geos.NewPolygon([][][]float64{{{0,0},{4,0},{4,4},{0,4},{0,0}}})
//	p2, _ := geos.NewPolygon([][][]float64{{{10,10},{12,10},{12,12},{10,12},{10,10}}})
//
//	ok, _ := geos.Disjoint(p1, p2)  // true（完全分离）
func Disjoint(a, b *gogeos.Geom) (bool, error) {
	return predicateTwo(a, b, func() bool { return a.Disjoint(b) })
}

// Equals 判断两个几何在拓扑上是否相等。
//
// 拓扑相等意味着两个几何有相同的形状和位置，即使坐标顺序不同。
// 例如：同一个多边形从不同顶点开始遍历，拓扑上仍然相等。
//
// 示例：
//
//	g1, _ := geos.FromWKT("POLYGON ((0 0, 4 0, 4 4, 0 4, 0 0))")
//	g2, _ := geos.FromWKT("POLYGON ((4 4, 0 4, 0 0, 4 0, 4 4))")
//
//	ok, _ := geos.Equals(g1, g2)  // true（形状相同，只是起点不同）
func Equals(a, b *gogeos.Geom) (bool, error) {
	return predicateTwo(a, b, func() bool { return a.Equals(b) })
}

// Overlaps 判断两个几何是否部分重叠。
//
// 两个几何有重叠区域，但都不完全包含对方。
// 要求两个几何是同维度的（点-点、线-线、面-面）。
//
// 示例：
//
//	p1, _ := geos.NewPolygon([][][]float64{{{0,0},{4,0},{4,4},{0,4},{0,0}}})
//	p2, _ := geos.NewPolygon([][][]float64{{{2,2},{6,2},{6,6},{2,6},{2,2}}})
//
//	ok, _ := geos.Overlaps(p1, p2)  // true（部分重叠）
func Overlaps(a, b *gogeos.Geom) (bool, error) {
	return predicateTwo(a, b, func() bool { return a.Overlaps(b) })
}

// Crosses 判断两个几何是否穿越。
//
// 两个几何的交集维度低于它们本身的维度。
// 例如：
//   - 线穿越多边形（1维穿越2维）
//   - 线与线交叉（1维交叉1维）
//
// 示例：
//
//	line, _ := geos.NewLineString([][]float64{{-1,2}, {5,2}})
//	poly, _ := geos.NewPolygon([][][]float64{{{0,0},{4,0},{4,4},{0,4},{0,0}}})
//
//	ok, _ := geos.Crosses(line, poly)  // true（线穿过面）
func Crosses(a, b *gogeos.Geom) (bool, error) {
	return predicateTwo(a, b, func() bool { return a.Crosses(b) })
}

// CoveredBy 判断 a 是否被 b 覆盖（Covers 的反向）。
//
// CoveredBy(a, b) 等价于 Covers(b, a)。
// 即 a 的所有点都在 b 的内部或边界上。
//
// 示例：
//
//	inner, _ := geos.NewPolygon([][][]float64{{{2,2},{8,2},{8,8},{2,8},{2,2}}})
//	outer, _ := geos.NewPolygon([][][]float64{{{0,0},{10,0},{10,10},{0,10},{0,0}}})
//
//	ok, _ := geos.CoveredBy(inner, outer)  // true
func CoveredBy(a, b *gogeos.Geom) (bool, error) {
	return predicateTwo(a, b, func() bool { return a.CoveredBy(b) })
}

// EqualsExact 判断两个几何在容差范围内是否精确相等。
//
// 与 Equals 不同，EqualsExact 逐点比较坐标值。
// tolerance 是容差，两个坐标的差值小于容差就认为相等。
//
// 适用于需要精确比较坐标的场景，如序列化/反序列化后的验证。
//
// 示例：
//
//	g1, _ := geos.NewPoint(1.0, 2.0)
//	g2, _ := geos.NewPoint(1.0000001, 2.0)
//
//	ok, _ := geos.EqualsExact(g1, g2, 0.001)  // true（在容差 0.001 内）
//	ok, _ = geos.EqualsExact(g1, g2, 0.00000001)  // false（超出容差）
func EqualsExact(a, b *gogeos.Geom, tolerance float64) (bool, error) {
	return predicateTwo(a, b, func() bool { return a.EqualsExact(b, tolerance) })
}
