package geos

// prepared.go — Prepared Geometry（预处理几何加速谓词）
//
// Prepared Geometry 是 GEOS 的性能优化机制。
// 适用于：一个固定几何对大量候选做循环判定的场景。
//
// 为什么需要 Prepared Geometry？
// 普通的 GEOS 谓词每次调用都要重新计算几何的内部索引（R-Tree）。
// 如果同一个几何要和成千上万个候选做判定，每次都重新计算就很浪费。
// Prepared Geometry 在创建时就预计算好内部索引，后续调用直接使用缓存。
//
// 性能对比：
//   - 普通 Contains(polygon, point) 调用 10000 次：每次都要遍历多边形的所有边
//   - PreparedGeom.Contains(point) 调用 10000 次：第一次建立 R-Tree，后续直接查询
//   - 性能差距可达 10-100 倍（取决于几何复杂度和候选数量）
//
// 使用场景：
//   - 围栏命中判断：一个围栏 vs 大量用户位置
//   - 空间查询：一个查询范围 vs 大量候选几何
//   - 批量处理：同一个几何反复用于判断
//
// 使用方式：
//
//	// 1. 创建 Prepared Geometry
//	prep, _ := geos.NewPreparedGeom(fenceGeom)
//	defer prep.Close()
//
//	// 2. 循环判断
//	for _, point := range points {
//	    hit, _ := prep.IntersectsXY(point.Lon, point.Lat)
//	    if hit { ... }
//	}
//
// 注意事项：
//   - 传入的几何必须由 geos 包构造（使用同一个默认 Context）
//   - 否则 GEOS 会 panic（已被 safeRun 捕获为 error）
//   - PreparedGeom 支持 Close() 释放引用，也可依赖 GC 自动回收

import (
	"fmt"

	gogeos "github.com/twpayne/go-geos"
)

// PreparedGeom 是 Prepared Geometry 的封装。
//
// 内部持有两个引用：
//   - geom: 原始几何对象
//   - prep: 预处理后的几何对象（包含缓存的 R-Tree 索引）
//
// 所有谓词方法都通过 prepRun 包装，统一处理 nil 检查和 panic 捕获。
type PreparedGeom struct {
	geom *gogeos.Geom
	prep *gogeos.PrepGeom
}

// NewPreparedGeom 从 *gogeos.Geom 构造 PreparedGeom。
//
// 创建时会预计算几何的内部索引，这个操作可能比较耗时（对于复杂几何）。
// 但后续的谓词调用会显著更快。
//
// 参数：
//   - g: 原始几何对象，必须由 geos 包构造
//
// 示例：
//
//	fence, _ := geos.NewPolygon([][][]float64{{{0,0},{100,0},{100,100},{0,100},{0,0}}})
//	prep, _ := geos.NewPreparedGeom(fence)
//	defer prep.Close()
func NewPreparedGeom(g *gogeos.Geom) (*PreparedGeom, error) {
	if g == nil {
		return nil, errNil
	}
	return safeRun(func() (*PreparedGeom, error) {
		return &PreparedGeom{geom: g, prep: g.Prepare()}, nil
	})
}

// Close 释放底层引用。重复调用安全。
//
// go-geos 通过 runtime.AddCleanup 自动管理 C 内存，此处仅置空 Go 引用帮助 GC。
// 可以不调用 Close()，依赖 GC 自动回收，但显式调用是好习惯。
//
// 示例：
//
//	prep, _ := geos.NewPreparedGeom(g)
//	defer prep.Close()  // 函数结束时释放
func (p *PreparedGeom) Close() {
	if p != nil {
		p.prep = nil
		p.geom = nil
	}
}

// prepRun 是内部辅助函数，用于 PreparedGeom 的谓词调用。
// 统一处理 nil 检查和 panic 捕获。
func (p *PreparedGeom) prepRun(fn func() bool) (bool, error) {
	if p == nil || p.prep == nil {
		return false, fmt.Errorf("PreparedGeom 已关闭或未初始化")
	}
	return safeRun(func() (bool, error) { return fn(), nil })
}

// Intersects 判断 prepared 几何与 other 是否有交集。
//
// 等价于 geos.Intersects(p.geom, other)，但使用预处理索引加速。
//
// 示例：
//
//	prep, _ := geos.NewPreparedGeom(fence)
//	point, _ := geos.NewPoint(50, 50)
//	hit, _ := prep.Intersects(point)  // true
func (p *PreparedGeom) Intersects(other *gogeos.Geom) (bool, error) {
	return p.prepRun(func() bool { return p.prep.Intersects(other) })
}

// Contains 判断 prepared 几何是否严格包含 other（边界不算）。
//
// 等价于 geos.Contains(p.geom, other)，但使用预处理索引加速。
// 注意：边界点不算包含，围栏命中用 Covers。
//
// 示例：
//
//	prep, _ := geos.NewPreparedGeom(fence)
//	inner, _ := geos.NewPolygon(...)
//	ok, _ := prep.Contains(inner)
func (p *PreparedGeom) Contains(other *gogeos.Geom) (bool, error) {
	return p.prepRun(func() bool { return p.prep.Contains(other) })
}

// ContainsXY 判断 prepared 几何是否严格包含点 (x, y)。
//
// 这是包含点的高效版本，不需要先创建 Point 几何对象。
// 注意：边界点不算包含，围栏命中用 IntersectsXY。
//
// 参数：
//   - x: X 坐标（经度）
//   - y: Y 坐标（纬度）
//
// 示例：
//
//	prep, _ := geos.NewPreparedGeom(fence)
//	ok, _ := prep.ContainsXY(50, 50)  // 内部点 → true
//	ok, _ = prep.ContainsXY(0, 0)     // 边界点 → false
func (p *PreparedGeom) ContainsXY(x, y float64) (bool, error) {
	return p.prepRun(func() bool { return p.prep.ContainsXY(x, y) })
}

// Covers 判断 prepared 几何是否覆盖 other（边界算）。
//
// 等价于 geos.Covers(p.geom, other)，但使用预处理索引加速。
// 边界点也算覆盖，适用于围栏命中判断。
//
// 示例：
//
//	prep, _ := geos.NewPreparedGeom(fence)
//	inner, _ := geos.NewPolygon(...)
//	ok, _ := prep.Covers(inner)
func (p *PreparedGeom) Covers(other *gogeos.Geom) (bool, error) {
	return p.prepRun(func() bool { return p.prep.Covers(other) })
}

// IntersectsXY 判断 prepared 几何是否与点 (x, y) 相交。
//
// 这是判断点是否在几何内部或边界上的高效版本。
// 等价于 CoversPoint（点在几何上或内部都返回 true）。
//
// 围栏命中的推荐用法：
//
//	prep, _ := geos.NewPreparedGeom(fence)
//	hit, _ := prep.IntersectsXY(userLon, userLat)
//
// 参数：
//   - x: X 坐标（经度）
//   - y: Y 坐标（纬度）
//
// 示例：
//
//	prep, _ := geos.NewPreparedGeom(fence)
//	hit, _ := prep.IntersectsXY(0, 0)     // 边界点 → true
//	hit, _ = prep.IntersectsXY(50, 50)    // 内部点 → true
//	hit, _ = prep.IntersectsXY(200, 200)  // 外部点 → false
func (p *PreparedGeom) IntersectsXY(x, y float64) (bool, error) {
	return p.prepRun(func() bool { return p.prep.IntersectsXY(x, y) })
}

// Disjoint 判断 prepared 几何是否与 other 完全无交集。
//
// 等价于 geos.Disjoint(p.geom, other)，但使用预处理索引加速。
//
// 示例：
//
//	prep, _ := geos.NewPreparedGeom(fence)
//	faraway, _ := geos.NewPoint(1000, 1000)
//	ok, _ := prep.Disjoint(faraway)  // true
func (p *PreparedGeom) Disjoint(other *gogeos.Geom) (bool, error) {
	return p.prepRun(func() bool { return p.prep.Disjoint(other) })
}

// CoveredBy 判断 prepared 几何是否被 other 覆盖。
//
// 等价于 geos.CoveredBy(p.geom, other)，但使用预处理索引加速。
//
// 示例：
//
//	prep, _ := geos.NewPreparedGeom(innerFence)
//	ok, _ := prep.CoveredBy(outerFence)  // inner 是否在 outer 内
func (p *PreparedGeom) CoveredBy(other *gogeos.Geom) (bool, error) {
	return p.prepRun(func() bool { return p.prep.CoveredBy(other) })
}

// Overlaps 判断是否部分重叠。
//
// 等价于 geos.Overlaps(p.geom, other)，但使用预处理索引加速。
//
// 示例：
//
//	prep, _ := geos.NewPreparedGeom(fenceA)
//	ok, _ := prep.Overlaps(fenceB)  // 两个围栏是否重叠
func (p *PreparedGeom) Overlaps(other *gogeos.Geom) (bool, error) {
	return p.prepRun(func() bool { return p.prep.Overlaps(other) })
}

// Touches 判断是否仅边界接触。
//
// 等价于 geos.Touches(p.geom, other)，但使用预处理索引加速。
//
// 示例：
//
//	prep, _ := geos.NewPreparedGeom(fenceA)
//	ok, _ := prep.Touches(fenceB)  // 两个围栏是否仅边界接触
func (p *PreparedGeom) Touches(other *gogeos.Geom) (bool, error) {
	return p.prepRun(func() bool { return p.prep.Touches(other) })
}

// Within 判断是否在 other 内部。
//
// 等价于 geos.Within(p.geom, other)，但使用预处理索引加速。
//
// 示例：
//
//	prep, _ := geos.NewPreparedGeom(innerFence)
//	ok, _ := prep.Within(outerFence)  // inner 是否在 outer 内
func (p *PreparedGeom) Within(other *gogeos.Geom) (bool, error) {
	return p.prepRun(func() bool { return p.prep.Within(other) })
}

// DistanceWithin 判断与 other 的距离是否在 dist 范围内。
//
// 比调用 Distance() 再比较更高效，因为可以提前终止计算。
//
// 参数：
//   - other: 另一个几何
//   - dist: 距离阈值
//
// 示例：
//
//	prep, _ := geos.NewPreparedGeom(fence)
//	nearby, _ := geos.NewPoint(105, 50)
//	ok, _ := prep.DistanceWithin(nearby, 10)  // 距离是否在 10 以内
func (p *PreparedGeom) DistanceWithin(other *gogeos.Geom, dist float64) (bool, error) {
	return p.prepRun(func() bool { return p.prep.DistanceWithin(other, dist) })
}
