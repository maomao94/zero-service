package geos

// introspect.go — 几何内省（属性查询）
//
// 本文件提供查询几何对象自身属性的函数。
// 这些函数不修改几何对象，只读取其状态。
//
// GEOS 几何对象的属性包括：
//   - 是否为空（IsEmpty）：没有坐标的几何
//   - 是否简单（IsSimple）：没有自相交
//   - 是否闭合（IsClosed）：首尾坐标相同（仅适用于 Curve 类型）
//   - 是否为环（IsRing）：闭合且简单（仅适用于 Curve 类型）
//   - 是否有 Z 坐标（HasZ）：三维几何
//
// 注意：IsClosed 和 IsRing 仅适用于 Curve 类型（LineString、LinearRing），
// 对 Polygon 调用会 panic（被 safeRun 捕获为 error）。

import gogeos "github.com/twpayne/go-geos"

// IsEmpty 判断几何是否为空。
//
// 空几何是指没有坐标数据的几何对象。
// nil 几何返回 error（与 IsSimple/IsClosed/IsRing/HasZ 行为一致）。
//
// 以下情况会产生空几何：
//   - 从 WKT 解析 "POINT EMPTY"
//   - 两个不相交多边形的 Intersection 结果
//   - 显式创建的空几何
//
// 示例：
//
//	g, _ := geos.FromWKT("POINT EMPTY")
//	empty, _ := geos.IsEmpty(g)  // true
func IsEmpty(g *gogeos.Geom) (bool, error) {
	return oneAttr(g, func(gg *gogeos.Geom) bool { return gg.IsEmpty() })
}

// IsSimple 判断几何是否简单（无自相交）。
//
// 简单几何的定义：
//   - Point: 总是简单
//   - LineString: 不自相交（线段不与自身交叉）
//   - Polygon: 总是简单（如果有效的话）
//
// 一个 "8" 字形的 LineString 就不是简单的，因为中间交叉了。
//
// 示例：
//
//	// 简单线段
//	ls1, _ := geos.NewLineString([][]float64{{0,0}, {3,0}, {3,3}})
//	simple, _ := geos.IsSimple(ls1)  // true
//
//	// 自相交线段（8 字形）
//	ls2, _ := geos.NewLineString([][]float64{{0,0}, {3,3}, {0,3}, {3,0}})
//	simple, _ = geos.IsSimple(ls2)   // false
func IsSimple(g *gogeos.Geom) (bool, error) {
	return oneAttr(g, func(gg *gogeos.Geom) bool { return gg.IsSimple() })
}

// IsClosed 判断几何的首尾是否闭合。
//
// 仅适用于 Curve 类型（LineString、LinearRing）。
// 对 Polygon 调用会返回 error（GEOS 不支持对 Polygon 调用此方法）。
//
// LinearRing 在构造时就要求闭合，所以 IsClosed 总是 true。
// LineString 如果首尾坐标相同，IsClosed 也为 true。
//
// 示例：
//
//	// 未闭合的线段
//	ls, _ := geos.NewLineString([][]float64{{0,0}, {3,0}, {3,3}})
//	closed, _ := geos.IsClosed(ls)  // false
//
//	// 闭合的环
//	lr, _ := geos.NewLinearRing([][]float64{{0,0}, {3,0}, {3,3}, {0,0}})
//	closed, _ = geos.IsClosed(lr)   // true
func IsClosed(g *gogeos.Geom) (bool, error) {
	return oneAttr(g, func(gg *gogeos.Geom) bool { return gg.IsClosed() })
}

// IsRing 判断几何是否为环形（闭合且简单）。
//
// 仅适用于 Curve 类型（LineString、LinearRing）。
// 环形 = IsClosed && IsSimple，即首尾闭合且不自相交。
//
// LinearRing 在构造时就满足这两个条件，所以 IsRing 总是 true。
//
// 示例：
//
//	lr, _ := geos.NewLinearRing([][]float64{{0,0}, {3,0}, {3,3}, {0,0}})
//	ring, _ := geos.IsRing(lr)  // true
func IsRing(g *gogeos.Geom) (bool, error) {
	return oneAttr(g, func(gg *gogeos.Geom) bool { return gg.IsRing() })
}

// HasZ 判断几何是否有 Z 坐标（三维几何）。
//
// 标准的二维几何只有 X, Y 坐标。
// 如果几何包含 Z 坐标（高程），HasZ 返回 true。
//
// 从 WKT 解析 "POINT Z (1 2 3)" 会创建带 Z 的点。
// 从 GeoJSON 解析带第三个坐标的点也会有 Z。
//
// 示例：
//
//	g2d, _ := geos.NewPoint(1, 2)
//	hasZ, _ := geos.HasZ(g2d)  // false
//
//	g3d, _ := geos.FromWKT("POINT Z (1 2 3)")
//	hasZ, _ = geos.HasZ(g3d)   // true
func HasZ(g *gogeos.Geom) (bool, error) {
	return oneAttr(g, func(gg *gogeos.Geom) bool { return gg.HasZ() })
}

// oneAttr 泛型辅助函数定义见 context.go，统一处理 nil 检查和 panic 捕获。
