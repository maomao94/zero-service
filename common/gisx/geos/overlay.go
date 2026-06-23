package geos

// overlay.go — 几何运算（Overlay / Valid / Measure / Simplify / Transform / Meta）
//
// 本文件包含 GEOS 几何引擎的核心运算函数，按功能分为以下几类：
//
// 1. Overlay（叠加运算）：两个几何之间的集合运算
//    - Intersection: 交集（公共区域）
//    - Union: 并集（合并区域）
//    - Difference: 差集（a 扣除与 b 重叠部分）
//    - SymDifference: 对称差集（并集减去交集）
//    - UnaryUnion: 单几何自合并（消除自重叠）
//
// 2. Valid（有效性）：检查和修复几何
//    - IsValid: 检查几何是否有效
//    - IsValidReason: 获取无效原因
//    - MakeValid: 修复无效几何
//
// 3. Measure（测量）：计算几何的度量属性
//    - Area: 面积
//    - Length: 周长/长度
//    - Distance: 两几何最小距离（平面欧几里得）
//    - Centroid: 质心
//    - PointOnSurface: 表面点（保证在多边形上）
//
// 4. Simplify（简化）：减少几何的顶点数
//    - Buffer: 缓冲区（外扩/内缩）
//    - Simplify: Douglas-Peucker 简化
//    - TopologyPreserveSimplify: 拓扑保持简化
//    - ConvexHull: 凸包
//    - ConcaveHull: 凹包
//
// 5. Transform（变换）：修改几何的形状
//    - Normalize: 规范化
//    - Reverse: 反转方向
//    - Snap: 顶点吸附
//    - ClipByRect: 矩形裁剪
//    - Densify: 密化
//    - OffsetCurve: 偏移曲线
//    - EndPoint/StartPoint: 线端点
//
// 6. Meta（元信息）：几何的附加属性
//    - MinimumClearance: 最窄宽度
//    - SRID/SetSRID: 空间参考 ID
//    - Precision: 精度模型
//    - FrechetDistance: 弗雷歇距离

import gogeos "github.com/twpayne/go-geos"

// --- 内部辅助函数 ---

// overlayTwo 对两个几何执行二元叠加运算。
// 如果任一几何为 nil，返回 errNil。
// fn 是实际的 GEOS 运算函数，由调用方提供。
func overlayTwo(a, b *gogeos.Geom, fn func() *gogeos.Geom) (*gogeos.Geom, error) {
	if a == nil || b == nil {
		return nil, errNil
	}
	return safeRun(func() (*gogeos.Geom, error) { return fn(), nil })
}

// transformOne 对单个几何执行变换运算。
// 如果几何为 nil，返回 errNil。
func transformOne(g *gogeos.Geom, fn func() *gogeos.Geom) (*gogeos.Geom, error) {
	if g == nil {
		return nil, errNil
	}
	return safeRun(func() (*gogeos.Geom, error) { return fn(), nil })
}

// --- Overlay（叠加运算）---

// Intersection 计算两个几何的交集（公共区域）。
//
// 返回 a 和 b 的公共部分。
// 如果 a 和 b 不相交，返回空几何（IsEmpty() == true），不会返回 nil error。
//
// 示例：
//
//	p1, _ := geos.NewPolygon([][][]float64{{{0,0},{4,0},{4,4},{0,4},{0,0}}})
//	p2, _ := geos.NewPolygon([][][]float64{{{2,2},{6,2},{6,6},{2,6},{2,2}}})
//	common, _ := geos.Intersection(p1, p2)
//	area, _ := geos.Area(common)  // 4.0（2×2 的正方形）
func Intersection(a, b *gogeos.Geom) (*gogeos.Geom, error) {
	return overlayTwo(a, b, func() *gogeos.Geom { return a.Intersection(b) })
}

// Union 计算两个几何的并集（合并区域）。
//
// 返回 a 和 b 合并后的几何。
// 如果 a 和 b 有重叠，重叠部分只保留一次。
//
// 示例：
//
//	p1, _ := geos.NewPolygon([][][]float64{{{0,0},{4,0},{4,4},{0,4},{0,0}}})
//	p2, _ := geos.NewPolygon([][][]float64{{{2,2},{6,2},{6,6},{2,6},{2,2}}})
//	merged, _ := geos.Union(p1, p2)
//	area, _ := geos.Area(merged)  // 28.0（16 + 16 - 4）
func Union(a, b *gogeos.Geom) (*gogeos.Geom, error) {
	return overlayTwo(a, b, func() *gogeos.Geom { return a.Union(b) })
}

// Difference 计算两个几何的差集（a - b）。
//
// 返回 a 中不与 b 重叠的部分。
// 注意参数顺序：Difference(a, b) = a 减去 b。
//
// 示例：
//
//	p1, _ := geos.NewPolygon([][][]float64{{{0,0},{4,0},{4,4},{0,4},{0,0}}})
//	p2, _ := geos.NewPolygon([][][]float64{{{2,2},{6,2},{6,6},{2,6},{2,2}}})
//	diff, _ := geos.Difference(p1, p2)
//	area, _ := geos.Area(diff)  // 12.0（16 - 4）
func Difference(a, b *gogeos.Geom) (*gogeos.Geom, error) {
	return overlayTwo(a, b, func() *gogeos.Geom { return a.Difference(b) })
}

// SymDifference 计算两个几何的对称差集。
//
// 返回 a 和 b 中不重叠的部分 = (a ∪ b) - (a ∩ b)。
// 等价于 Union(a,b) 减去 Intersection(a,b)。
//
// 示例：
//
//	p1, _ := geos.NewPolygon([][][]float64{{{0,0},{4,0},{4,4},{0,4},{0,0}}})
//	p2, _ := geos.NewPolygon([][][]float64{{{2,2},{6,2},{6,6},{2,6},{2,2}}})
//	sd, _ := geos.SymDifference(p1, p2)
//	area, _ := geos.Area(sd)  // 24.0（28 - 4）
func SymDifference(a, b *gogeos.Geom) (*gogeos.Geom, error) {
	return overlayTwo(a, b, func() *gogeos.Geom { return a.SymDifference(b) })
}

// UnaryUnion 对单个几何执行自合并。
//
// 主要用于：
//   - 消除 MultiPolygon 内部的自重叠边
//   - 合并 MultiPolygon 中相邻的多边形
//   - 修复自相交的几何
//
// 示例：
//
//	// 两个相邻多边形合并
//	mp, _ := geos.FromWKT("MULTIPOLYGON (((0 0, 2 0, 2 2, 0 2, 0 0)), ((2 0, 4 0, 4 2, 2 2, 2 0)))")
//	merged, _ := geos.UnaryUnion(mp)
//	// merged 是一个 4×2 的矩形
func UnaryUnion(g *gogeos.Geom) (*gogeos.Geom, error) {
	return transformOne(g, func() *gogeos.Geom { return g.UnaryUnion() })
}

// Envelope 计算几何的外包盒（Bounding Box）。
//
// 返回一个与坐标轴平行的最小矩形，包含几何的所有点。
// 返回值是 Polygon 类型。
//
// 示例：
//
//	g, _ := geos.NewPoint(3, 4)
//	env, _ := geos.Envelope(g)
//	// env 是一个退化的矩形（点）
func Envelope(g *gogeos.Geom) (*gogeos.Geom, error) {
	return transformOne(g, func() *gogeos.Geom { return g.Envelope() })
}

// Boundary 计算几何的边界。
//
// 不同类型几何的边界：
//   - Point: 空几何（点没有边界）
//   - LineString: 两个端点（MultiPoint）
//   - Polygon: 外环和洞的环（MultiLineString）
//
// 示例：
//
//	poly, _ := geos.NewPolygon([][][]float64{{{0,0},{4,0},{4,4},{0,4},{0,0}}})
//	boundary, _ := geos.Boundary(poly)
//	// boundary 是一个 LinearRing（4 条边组成的环）
func Boundary(g *gogeos.Geom) (*gogeos.Geom, error) {
	return transformOne(g, func() *gogeos.Geom { return g.Boundary() })
}

// BuildArea 从线几何构造面几何。
//
// 将一组线段视为面的边界，构造出对应的多边形。
// 假设嵌套的环是空洞（hole），而非额外的多边形。
//
// 示例：
//
//	lines, _ := geos.FromWKT("MULTILINESTRING ((0 0, 4 0), (4 0, 4 4), (4 4, 0 4), (0 4, 0 0))")
//	area, _ := geos.BuildArea(lines)
//	// area 是一个 4×4 的正方形 Polygon
func BuildArea(g *gogeos.Geom) (*gogeos.Geom, error) {
	return transformOne(g, func() *gogeos.Geom { return g.BuildArea() })
}

// LineMerge 合并相连的线段。
//
// 将 MultiLineString 中首尾相连的线段合并为更长的 LineString。
// 不相连的线段保持独立。
//
// 示例：
//
//	mls, _ := geos.FromWKT("MULTILINESTRING ((0 0, 1 1), (1 1, 2 2), (3 3, 4 4))")
//	merged, _ := geos.LineMerge(mls)
//	// merged 包含两条线：(0,0)→(2,2) 和 (3,3)→(4,4)
func LineMerge(g *gogeos.Geom) (*gogeos.Geom, error) {
	return transformOne(g, func() *gogeos.Geom { return g.LineMerge() })
}

// Node 在几何的所有边交点处分割线段。
//
// 确保所有线段在交叉点处被分割，便于后续的拓扑分析。
//
// 示例：
//
//	g, _ := geos.FromWKT("MULTILINESTRING ((0 0, 4 4), (0 4, 4 0))")
//	noded, _ := geos.Node(g)
//	// noded 在 (2,2) 处被分割为 4 条线段
func Node(g *gogeos.Geom) (*gogeos.Geom, error) {
	return transformOne(g, func() *gogeos.Geom { return g.Node() })
}

// MinimumRotatedRectangle 计算几何的最小外接旋转矩形。
//
// 与 Envelope 不同，这个矩形可以旋转，面积通常更小。
// 也称为 "最小面积外接矩形"（Minimum Area Bounding Rectangle）。
//
// 示例：
//
//	g, _ := geos.FromWKT("POLYGON ((0 0, 3 1, 2 3, -1 2, 0 0))")
//	rect, _ := geos.MinimumRotatedRectangle(g)
//	// rect 是一个面积最小的外接矩形，可能有旋转角度
func MinimumRotatedRectangle(g *gogeos.Geom) (*gogeos.Geom, error) {
	return transformOne(g, func() *gogeos.Geom { return g.MinimumRotatedRectangle() })
}

// --- Valid（有效性）---

// IsValid 判断几何是否有效。
//
// 有效的几何要求：
//   - 环是闭合的
//   - 环不自相交
//   - 洞完全在外环内部
//   - 洞之间不重叠
//   - 多边形的环方向正确（外环逆时针，洞顺时针）
//
// 常见的无效几何：
//   - "蝴蝶结" 形状（自相交的多边形）
//   - 洞延伸到外环外部
//
// 示例：
//
//	// 有效的正方形
//	p1, _ := geos.NewPolygon([][][]float64{{{0,0},{4,0},{4,4},{0,4},{0,0}}})
//	valid, _ := geos.IsValid(p1)  // true
//
//	// 无效的蝴蝶结
//	bowtie, _ := geos.NewPolygon([][][]float64{{{0,0},{4,0},{0,4},{4,4},{0,0}}})
//	valid, _ = geos.IsValid(bowtie)  // false
func IsValid(g *gogeos.Geom) (bool, error) {
	return oneAttr(g, func(gg *gogeos.Geom) bool { return gg.IsValid() })
}

// IsValidReason 返回几何无效的原因。
//
// 如果几何有效，返回空字符串。
// 如果几何无效，返回描述原因的字符串，如：
//   - "Self-intersection"
//   - "Hole lies outside shell"
//   - "Interior is disconnected"
//
// 适用于调试和日志记录。
//
// 示例：
//
//	bowtie, _ := geos.NewPolygon([][][]float64{{{0,0},{4,0},{0,4},{4,4},{0,0}}})
//	reason, _ := geos.IsValidReason(bowtie)
//	// reason = "Self-intersection [2 2]"
func IsValidReason(g *gogeos.Geom) (string, error) {
	if g == nil {
		return "", errNil
	}
	return safeRun(func() (string, error) { return g.IsValidReason(), nil })
}

// MakeValid 尝试修复无效的几何。
//
// 修复策略取决于 GEOS 版本和几何的无效类型：
//   - 自相交的多边形可能被拆分为 MultiPolygon（保留多个部分）
//   - 其他类型的无效几何可能被简化或重新构造
//   - 返回值直接透传 GEOS 结果，不做额外处理
//
// 示例：
//
//	bowtie, _ := geos.NewPolygon([][][]float64{{{0,0},{4,0},{0,4},{4,4},{0,0}}})
//	fixed, _ := geos.MakeValid(bowtie)
//	valid, _ := geos.IsValid(fixed)  // true
func MakeValid(g *gogeos.Geom) (*gogeos.Geom, error) {
	return transformOne(g, func() *gogeos.Geom { return g.MakeValid() })
}

// --- Measure（测量）---

// Area 计算几何的面积。
//
// 返回值的单位取决于输入坐标的单位：
//   - 如果坐标是经纬度（度），返回值是度²（无物理意义）
//   - 如果坐标是投影坐标（米），返回值是米²
//
// 仅适用于 Polygon 和 MultiPolygon。
// 对 Point 和 LineString 返回 0。
//
// 示例：
//
//	poly, _ := geos.NewPolygon([][][]float64{{{0,0},{4,0},{4,4},{0,4},{0,0}}})
//	area, _ := geos.Area(poly)  // 16.0
func Area(g *gogeos.Geom) (float64, error) {
	return oneAttr(g, func(gg *gogeos.Geom) float64 { return gg.Area() })
}

// Length 计算几何的长度或周长。
//
//   - 对 LineString：返回线的总长度
//   - 对 Polygon：返回外环和所有洞的周长之和
//   - 对 Point：返回 0
//
// 返回值的单位取决于输入坐标的单位。
//
// 示例：
//
//	poly, _ := geos.NewPolygon([][][]float64{{{0,0},{4,0},{4,4},{0,4},{0,0}}})
//	length, _ := geos.Length(poly)  // 16.0（周长 4×4）
func Length(g *gogeos.Geom) (float64, error) {
	return oneAttr(g, func(gg *gogeos.Geom) float64 { return gg.Length() })
}

// Distance 计算两个几何之间的最小平面距离。
//
// 返回的是欧几里得距离（直线距离）。
// 注意：如果输入是经纬度坐标，返回值的单位是"度"，不是米。
// 经纬度场景下的实际距离需要使用 Haversine 公式或投影坐标系。
//
// 示例：
//
//	p1, _ := geos.NewPoint(0, 0)
//	p2, _ := geos.NewPoint(3, 4)
//	dist, _ := geos.Distance(p1, p2)  // 5.0
func Distance(a, b *gogeos.Geom) (float64, error) {
	if a == nil || b == nil {
		return 0, errNil
	}
	return safeRun(func() (float64, error) { return a.Distance(b), nil })
}

// Centroid 计算几何的质心（重心）。
//
// 返回 (x, y, error)。
// 质心是几何的"平衡点"，对于均匀密度的几何，质心就是物理重心。
//
// 注意：凹多边形的质心可能在多边形外部！
// 如果需要保证返回的点在多边形内部，使用 PointOnSurface。
//
// 示例：
//
//	poly, _ := geos.NewPolygon([][][]float64{{{0,0},{4,0},{4,4},{0,4},{0,0}}})
//	x, y, _ := geos.Centroid(poly)  // (2.0, 2.0)
func Centroid(g *gogeos.Geom) (x, y float64, err error) {
	if g == nil {
		return 0, 0, errNil
	}
	v, e := safeRun(func() (pointPair, error) {
		c := g.Centroid()
		if c == nil || c.IsEmpty() {
			return pointPair{}, ErrEmptyResult
		}
		return pointPair{x1: c.X(), y1: c.Y()}, nil
	})
	if e != nil {
		return 0, 0, e
	}
	return v.x1, v.y1, nil
}

// PointOnSurface 计算几何表面上的一个点。
//
// 返回 (x, y, error)。
// 与 Centroid 不同，PointOnSurface 保证返回的点在几何内部或边界上。
// 适用于凹多边形，确保标注点在多边形内。
//
// 示例：
//
//	// L 形多边形
//	lshape, _ := geos.NewPolygon([][][]float64{{{0,0},{2,0},{2,1},{1,1},{1,2},{0,2},{0,0}}})
//	x, y, _ := geos.PointOnSurface(lshape)
//	// (x, y) 一定在 L 形内部
func PointOnSurface(g *gogeos.Geom) (x, y float64, err error) {
	if g == nil {
		return 0, 0, errNil
	}
	v, e := safeRun(func() (pointPair, error) {
		p := g.PointOnSurface()
		if p == nil || p.IsEmpty() {
			return pointPair{}, ErrEmptyResult
		}
		return pointPair{x1: p.X(), y1: p.Y()}, nil
	})
	if e != nil {
		return 0, 0, e
	}
	return v.x1, v.y1, nil
}

// --- Simplify & Buffer（简化和缓冲）---

// Buffer 创建几何的缓冲区（外扩或内缩）。
//
// 参数：
//   - width: 缓冲宽度，正值外扩，负值内缩
//   - quadsegs: 每个四分之一圆弧的线段数，越大越圆滑（推荐 8）
//
// 常见用途：
//   - 点缓冲：创建圆形区域
//   - 线缓冲：创建带状区域
//   - 面缓冲：外扩或内缩多边形
//   - 围栏扩展：扩大搜索范围
//
// 示例：
//
//	point, _ := geos.NewPoint(0, 0)
//	circle, _ := geos.Buffer(point, 100, 32)  // 半径 100 的圆（32 段近似）
//
//	poly, _ := geos.NewPolygon([][][]float64{{{0,0},{4,0},{4,4},{0,4},{0,0}}})
//	expanded, _ := geos.Buffer(poly, 1, 8)  // 外扩 1 单位
func Buffer(g *gogeos.Geom, width float64, quadsegs int) (*gogeos.Geom, error) {
	return transformOne(g, func() *gogeos.Geom { return g.Buffer(width, quadsegs) })
}

// Simplify 使用 Douglas-Peucker 算法简化几何。
//
// 参数：
//   - tolerance: 简化容差，越大简化越多（单位与坐标相同）
//
// Douglas-Peucker 算法通过移除距离简化线段小于容差的点来减少顶点数。
// 注意：Simplify 可能产生无效的几何（自相交），如需保证有效性使用 TopologyPreserveSimplify。
//
// 示例：
//
//	// 简化一个有很多点的多边形
//	g, _ := geos.FromWKT("POLYGON ((0 0, 1 0.1, 2 0, 3 0.1, 4 0, 4 4, 0 4, 0 0))")
//	simple, _ := geos.Simplify(g, 0.5)  // 移除偏离直线超过 0.5 的点
func Simplify(g *gogeos.Geom, tolerance float64) (*gogeos.Geom, error) {
	return transformOne(g, func() *gogeos.Geom { return g.Simplify(tolerance) })
}

// TopologyPreserveSimplify 使用拓扑保持的 Douglas-Peucker 算法简化几何。
//
// 与 Simplify 类似，但保证结果几何的有效性：
//   - 不会产生自相交
//   - 保持拓扑关系（包含、相交等）
//
// 适用于需要保证几何有效性的场景。
//
// 示例：
//
//	g, _ := geos.FromWKT("POLYGON ((0 0, 1 0.1, 2 0, 3 0.1, 4 0, 4 4, 0 4, 0 0))")
//	simple, _ := geos.TopologyPreserveSimplify(g, 0.5)
//	valid, _ := geos.IsValid(simple)  // true（保证有效）
func TopologyPreserveSimplify(g *gogeos.Geom, tolerance float64) (*gogeos.Geom, error) {
	return transformOne(g, func() *gogeos.Geom { return g.TopologyPreserveSimplify(tolerance) })
}

// ConvexHull 计算几何的凸包。
//
// 凸包是包含几何所有点的最小凸多边形。
// 可以想象成在几何周围拉一根橡皮筋，松开后形成的形状。
//
// 示例：
//
//	g, _ := geos.FromWKT("MULTIPOINT ((0 0), (4 0), (2 2), (4 4), (0 4))")
//	hull, _ := geos.ConvexHull(g)
//	// hull 是一个包含所有点的凸多边形
func ConvexHull(g *gogeos.Geom) (*gogeos.Geom, error) {
	return transformOne(g, func() *gogeos.Geom { return g.ConvexHull() })
}

// ConcaveHull 计算几何的凹包。
//
// 参数：
//   - ratio: 凹度比例，0~1 之间，越小越凹（0 最凹，1 等价于凸包）
//   - allowHoles: 是否允许结果中包含洞
//
// 凹包比凸包更贴合原始几何的形状，适用于创建更紧凑的边界。
//
// 示例：
//
//	g, _ := geos.FromWKT("MULTIPOINT ((0 0), (4 0), (2 2), (4 4), (0 4))")
//	concave, _ := geos.ConcaveHull(g, 0.5, false)  // 凹度 50%，不允许洞
func ConcaveHull(g *gogeos.Geom, ratio float64, allowHoles bool) (*gogeos.Geom, error) {
	var holes uint
	if allowHoles {
		holes = 1
	}
	return transformOne(g, func() *gogeos.Geom { return g.ConcaveHull(ratio, holes) })
}

// --- Transform（变换）---

// Normalize 规范化几何的坐标顺序。
//
// 将几何的环按统一规则排序，便于比较两个几何是否相等。
// 规范化后，相同形状的几何会有相同的坐标序列。
//
// 示例：
//
//	g1, _ := geos.FromWKT("POLYGON ((4 4, 0 4, 0 0, 4 0, 4 4))")
//	g2, _ := geos.FromWKT("POLYGON ((0 0, 4 0, 4 4, 0 4, 0 0))")
//	g1, _ = geos.Normalize(g1)
//	g2, _ = geos.Normalize(g2)
//	// 现在 g1 和 g2 的 WKT 输出相同
func Normalize(g *gogeos.Geom) (*gogeos.Geom, error) {
	return transformOne(g, func() *gogeos.Geom { return g.Normalize() })
}

// Reverse 反转几何的环方向。
//
// 将环的方向反转：顺时针变逆时针，逆时针变顺时针。
// GEOS 中外环通常为逆时针，洞为顺时针。
//
// 示例：
//
//	g, _ := geos.NewPolygon([][][]float64{{{0,0},{4,0},{4,4},{0,4},{0,0}}})
//	reversed, _ := geos.Reverse(g)  // 环方向反转
func Reverse(g *gogeos.Geom) (*gogeos.Geom, error) {
	return transformOne(g, func() *gogeos.Geom { return g.Reverse() })
}

// Snap 将 a 的顶点吸附到 b 的顶点。
//
// 参数：
//   - tolerance: 吸附距离，a 的顶点如果在 b 的顶点容差范围内，会被吸附到 b 的顶点
//
// 适用于修复两个几何之间的小间隙，使其在拓扑上正确连接。
//
// 示例：
//
//	a, _ := geos.FromWKT("LINESTRING (0 0, 1.01 0)")
//	b, _ := geos.FromWKT("LINESTRING (1 0, 2 0)")
//	snapped, _ := geos.Snap(a, b, 0.1)  // (1.01,0) 被吸附到 (1,0)
func Snap(a, b *gogeos.Geom, tolerance float64) (*gogeos.Geom, error) {
	return overlayTwo(a, b, func() *gogeos.Geom { return a.Snap(b, tolerance) })
}

// ClipByRect 用矩形裁剪几何。
//
// 参数：
//   - minX, minY, maxX, maxY: 裁剪矩形的边界
//
// 返回几何在矩形内的部分。
// 比 Intersection 更高效，因为矩形是特殊的简单几何。
//
// 示例：
//
//	g, _ := geos.NewPolygon([][][]float64{{{0,0},{4,0},{4,4},{0,4},{0,0}}})
//	clipped, _ := geos.ClipByRect(g, 1, 1, 3, 3)  // 裁剪到内部 2×2 区域
func ClipByRect(g *gogeos.Geom, minX, minY, maxX, maxY float64) (*gogeos.Geom, error) {
	return transformOne(g, func() *gogeos.Geom { return g.ClipByRect(minX, minY, maxX, maxY) })
}

// Densify 在几何的边上插入新顶点，使相邻顶点间距不超过 tolerance。
//
// 用于增加几何的顶点密度，使曲线更平滑。
// 常见用途：
//   - 投影转换前增加中间点，减少投影误差
//   - 简化后的几何重新密化
//
// 示例：
//
//	g, _ := geos.NewPolygon([][][]float64{{{0,0},{4,0},{4,4},{0,4},{0,0}}})
//	dense, _ := geos.Densify(g, 1.0)  // 每隔 1 单位插入一个顶点
func Densify(g *gogeos.Geom, tolerance float64) (*gogeos.Geom, error) {
	return transformOne(g, func() *gogeos.Geom { return g.Densify(tolerance) })
}

// OffsetCurve 计算线几何的偏移曲线。
//
// 参数：
//   - width: 偏移距离，正值向左偏移，负值向右偏移（沿线方向看）
//   - quadsegs: 圆角精度
//
// 与 Buffer 不同，OffsetCurve 只返回偏移后的线，不返回端盖。
// 适用于道路中心线偏移为车道线等场景。
//
// 示例：
//
//	road, _ := geos.NewLineString([][]float64{{0,0}, {100,0}})
//	lane, _ := geos.OffsetCurve(road, 3.75, 8)  // 向左偏移 3.75 米（一个车道宽度）
// OffsetCurve 计算线几何的偏移曲线。
//
// 参数：
//   - width: 偏移距离，正值向左偏移，负值向右偏移（沿线方向看）
//   - quadsegs: 圆角精度
//   - joinStyle: 拐角样式，默认 BufJoinStyleRound（圆角）
//   - mitreLimit: 尖角限制，仅 BufJoinStyleMitre 时有效，默认 5.0
//
// 与 Buffer 不同，OffsetCurve 只返回偏移后的线，不返回端盖。
// 适用于道路中心线偏移为车道线等场景。
func OffsetCurve(g *gogeos.Geom, width float64, quadsegs int, joinStyle gogeos.BufJoinStyle, mitreLimit float64) (*gogeos.Geom, error) {
	return transformOne(g, func() *gogeos.Geom { return g.OffsetCurve(width, quadsegs, joinStyle, mitreLimit) })
}

// EndPoint 获取线几何的终点。
//
// 仅适用于 LineString。
// 返回一个 Point 几何，表示线的最后一个坐标点。
//
// 示例：
//
//	ls, _ := geos.NewLineString([][]float64{{0,0}, {3,0}, {3,3}})
//	end, _ := geos.EndPoint(ls)
//	x, y := end.X(), end.Y()  // (3.0, 3.0)
func EndPoint(g *gogeos.Geom) (*gogeos.Geom, error) {
	return transformOne(g, func() *gogeos.Geom { return g.EndPoint() })
}

// StartPoint 获取线几何的起点。
//
// 仅适用于 LineString。
// 返回一个 Point 几何，表示线的第一个坐标点。
//
// 示例：
//
//	ls, _ := geos.NewLineString([][]float64{{0,0}, {3,0}, {3,3}})
//	start, _ := geos.StartPoint(ls)
//	x, y := start.X(), start.Y()  // (0.0, 0.0)
func StartPoint(g *gogeos.Geom) (*gogeos.Geom, error) {
	return transformOne(g, func() *gogeos.Geom { return g.StartPoint() })
}

// --- Meta（元信息）---

// MinimumClearance 计算几何的最窄宽度。
//
// 返回几何中任意两点之间的最小距离。
// 对于多边形，这表示最窄处的宽度（瓶颈）。
// 对于退化的几何（如线），返回 0。
//
// 示例：
//
//	// 一个哑铃形的多边形
//	g, _ := geos.FromWKT("POLYGON ((0 0, 2 0, 2 1, 4 1, 4 0, 6 0, 6 3, 4 3, 4 2, 2 2, 2 3, 0 3, 0 0))")
//	clearance, _ := geos.MinimumClearance(g)  // 中间连接处的宽度
func MinimumClearance(g *gogeos.Geom) (float64, error) {
	return oneAttr(g, func(gg *gogeos.Geom) float64 { return gg.MinimumClearance() })
}

// SRID 获取几何的空间参考 ID。
//
// SRID (Spatial Reference System Identifier) 标识几何使用的坐标系。
// 常见 SRID：
//   - 0: 未指定（默认）
//   - 4326: WGS 84（GPS 使用的经纬度坐标系）
//   - 3857: Web Mercator（Web 地图使用的投影坐标系）
//
// GEOS 不会根据 SRID 做投影转换，它只是一个标签。
//
// 示例：
//
//	g, _ := geos.NewPoint(116.39, 39.9)
//	srid, _ := geos.SRID(g)  // 0（默认未指定）
func SRID(g *gogeos.Geom) (int, error) {
	return oneAttr(g, func(gg *gogeos.Geom) int { return gg.SRID() })
}

// SetSRID 设置几何的空间参考 ID。
//
// 仅为几何设置 SRID 标签，不影响坐标值。
// 如果需要实际的坐标投影转换，需要使用 proj 等专门的投影库。
//
// 示例：
//
//	g, _ := geos.NewPoint(116.39, 39.9)
//	g, _ = geos.SetSRID(g, 4326)  // 标记为 WGS 84 坐标系
//	srid, _ := geos.SRID(g)       // 4326
func SetSRID(g *gogeos.Geom, srid int) (*gogeos.Geom, error) {
	return transformOne(g, func() *gogeos.Geom { return g.SetSRID(srid) })
}

// Precision 获取几何的精度模型值。
//
// 精度模型定义了坐标的精度网格。
// 值为 0 表示使用完整精度（默认）。
//
// 示例：
//
//	g, _ := geos.NewPoint(1.23456789, 2.34567890)
//	prec, _ := geos.Precision(g)  // 0（完整精度）
func Precision(g *gogeos.Geom) (float64, error) {
	return oneAttr(g, func(gg *gogeos.Geom) float64 { return gg.Precision() })
}

// FrechetDistance 计算两个几何之间的弗雷歇距离。
//
// 弗雷歇距离衡量两条曲线的相似程度，考虑了点的顺序。
// 可以理解为：一个人牵着狗在两条曲线上走，绳子需要的最小长度。
//
// 适用于比较轨迹、路线的相似性。
//
// 示例：
//
//	route1, _ := geos.NewLineString([][]float64{{0,0}, {1,0}, {2,0}})
//	route2, _ := geos.NewLineString([][]float64{{0,1}, {1,1}, {2,1}})
//	d, _ := geos.FrechetDistance(route1, route2)  // 1.0（两条平行线）
func FrechetDistance(a, b *gogeos.Geom) (float64, error) {
	if a == nil || b == nil {
		return 0, errNil
	}
	return safeRun(func() (float64, error) { return a.FrechetDistance(b), nil })
}

// --- 内部辅助函数 ---

// oneAttr 泛型辅助函数定义见 context.go。
