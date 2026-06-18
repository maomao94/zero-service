package gisx

import (
	"math"

	"github.com/paulmach/orb"
	"github.com/paulmach/orb/planar"
)

// SegmentIntersect 判断线段 a1-a2 与线段 b1-b2 是否相交。
// 算法：跨立实验（cross product straddle test），覆盖以下情况：
//   - 一般相交（两线段互相跨立）
//   - 端点接触（一端落在另一线段上）
//   - 共线重叠（部分或完全重合）
func SegmentIntersect(a1, a2, b1, b2 orb.Point) bool {
	// 第一步：bbox 快速排除 — 若两线段包围盒不重叠，必不相交
	if math.Max(a1.X(), a2.X()) < math.Min(b1.X(), b2.X()) ||
		math.Max(b1.X(), b2.X()) < math.Min(a1.X(), a2.X()) ||
		math.Max(a1.Y(), a2.Y()) < math.Min(b1.Y(), b2.Y()) ||
		math.Max(b1.Y(), b2.Y()) < math.Min(a1.Y(), a2.Y()) {
		return false
	}

	// 第二步：计算叉积判断各端点相对另一线段的方向
	d1 := cross(b1, b2, a1) // a1 相对 b1->b2 的方向
	d2 := cross(b1, b2, a2) // a2 相对 b1->b2 的方向
	d3 := cross(a1, a2, b1) // b1 相对 a1->a2 的方向
	d4 := cross(a1, a2, b2) // b2 相对 a1->a2 的方向

	// 第三步：跨立判定 — d1/d2 异号 且 d3/d4 异号 → 一般相交
	if (d1 > 0 && d2 < 0 || d1 < 0 && d2 > 0) &&
		(d3 > 0 && d4 < 0 || d3 < 0 && d4 > 0) {
		return true
	}

	// 第四步：退化情况 — 某端点与另一线段共线（叉积为 0），检查是否落在线段范围内
	if d1 == 0 && onSegment(b1, b2, a1) {
		return true
	}
	if d2 == 0 && onSegment(b1, b2, a2) {
		return true
	}
	if d3 == 0 && onSegment(a1, a2, b1) {
		return true
	}
	if d4 == 0 && onSegment(a1, a2, b2) {
		return true
	}

	return false
}

// cross 计算向量叉积 (b-a) × (c-a)。
// 返回值 > 0 表示 c 在 a→b 左侧，< 0 在右侧，= 0 三点共线。
func cross(a, b, c orb.Point) float64 {
	return (b.X()-a.X())*(c.Y()-a.Y()) - (b.Y()-a.Y())*(c.X()-a.X())
}

// onSegment 在已知 a、b、c 三点共线的前提下，判断 c 是否落在线段 ab 的 bbox 内。
func onSegment(a, b, c orb.Point) bool {
	return c.X() >= math.Min(a.X(), b.X()) &&
		c.X() <= math.Max(a.X(), b.X()) &&
		c.Y() >= math.Min(a.Y(), b.Y()) &&
		c.Y() <= math.Max(a.Y(), b.Y())
}

// RingIntersect 判断两个环的边界是否存在线段相交。
// 采用 O(n*m) 暴力枚举所有边对，适用于顶点数较少的场景。
func RingIntersect(r1, r2 orb.Ring) bool {
	n1 := len(r1)
	n2 := len(r2)

	for i := 0; i < n1; i++ {
		a1 := r1[i]
		a2 := r1[(i+1)%n1]

		for j := 0; j < n2; j++ {
			b1 := r2[j]
			b2 := r2[(j+1)%n2]

			if SegmentIntersect(a1, a2, b1, b2) {
				return true
			}
		}
	}
	return false
}

// PolygonIntersect 判断两个多边形是否相交（仅考虑外环 polygon[0]）。
// 检测策略分两步：
//  1. 顶点包含 — 任一多边形的顶点落在另一多边形内部（覆盖完全包含的情况）
//  2. 边界相交 — 两个外环的边存在线段交点（覆盖交叉穿越的情况）
func PolygonIntersect(p1, p2 orb.Polygon) bool {
	r1 := p1[0]
	r2 := p2[0]

	// 步骤一：顶点包含检测（处理一个多边形完全位于另一个内部的情况）
	for _, pt := range r1 {
		if planar.PolygonContains(p2, pt) {
			return true
		}
	}
	for _, pt := range r2 {
		if planar.PolygonContains(p1, pt) {
			return true
		}
	}

	// 步骤二：边界相交检测（处理边交叉但无顶点包含的情况）
	if RingIntersect(r1, r2) {
		return true
	}

	return false
}
