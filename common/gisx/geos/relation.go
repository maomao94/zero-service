package geos

// relation.go — DE-9IM 关系矩阵、距离函数、最近点
//
// 本文件提供高级空间关系分析功能：
//
// 1. DE-9IM（Dimensionally Extended 9-Intersection Model）
//    - Relate: 计算两个几何的 DE-9IM 矩阵字符串
//    - RelatePattern: 用模式匹配 DE-9IM 矩阵
//
// 2. 距离函数
//    - DistanceWithin: 距离阈值判断（比 Distance 更高效）
//    - HausdorffDistance: 豪斯多夫距离（两个形状的最大偏差）
//
// 3. 最近点
//    - NearestPoints: 计算两个几何之间的最近点对
//
// DE-9IM 是什么？
// DE-9IM 是一个 3×3 的矩阵，描述两个几何在 9 个维度上的交集关系。
// 矩阵的每个元素表示两个几何在该维度上的交集的维度（-1, 0, 1, 2）。
//
// 例如，对于两个多边形：
//
//	矩阵字符串 "212101212" 表示：
//	┌─────────────────────────────────────────────────────────┐
//	│               │ Interior(geom.B) │ Boundary(geom.B) │ Exterior(geom.B) │
//	├─────────────────────────────────────────────────────────┤
//	│ Interior(A)  │        2        │        1        │        2        │
//	│ Boundary(A)  │        1        │        0        │        1        │
//	│ Exterior(A)  │        2        │        1        │        2        │
//	└─────────────────────────────────────────────────────────┘
//
// 其中维度值：-1=空, 0=点, 1=线, 2=面

import (
	"fmt"

	gogeos "github.com/twpayne/go-geos"
)

// Relate 计算两个几何的 DE-9IM 关系矩阵字符串。
//
// 返回一个 9 个字符的字符串，如 "212101212"。
// 每个字符表示两个几何在对应维度上的交集维度。
//
// DE-9IM 矩字符串的含义：
//   - 第 1-3 个字符：A 的 Interior 与 B 的 Interior/Boundary/Exterior 的交集维度
//   - 第 4-6 个字符：A 的 Boundary 与 B 的 Interior/Boundary/Exterior 的交集维度
//   - 第 7-9 个字符：A 的 Exterior 与 B 的 Interior/Boundary/Exterior 的交集维度
//
// 维度值：-1(空), 0(点), 1(线), 2(面)
//
// 常见矩阵：
//   - "212101212": 两个面有重叠
//   - "FF2FF1212": 两个面仅边界接触
//   - "0F1FF0212": 点在面内部
//
// 示例：
//
//	p1, _ := geos.NewPolygon([][][]float64{{{0,0},{4,0},{4,4},{0,4},{0,0}}})
//	p2, _ := geos.NewPolygon([][][]float64{{{2,2},{6,2},{6,6},{2,6},{2,2}}})
//	matrix, _ := geos.Relate(p1, p2)  // "212101212"
func Relate(a, b *gogeos.Geom) (string, error) {
	if a == nil || b == nil {
		return "", errNil
	}
	return safeRun(func() (string, error) { return a.Relate(b), nil })
}

// RelatePattern 用模式匹配 DE-9IM 矩阵。
//
// 参数：
//   - a, b: 两个几何
//   - pattern: DE-9IM 模式字符串，每个字符可以是：
//   - 'T': 任意值（非空，即维度 ≥ 0）
//   - 'F': 必须为空（维度 = -1）
//   - '0': 必须是点（维度 = 0）
//   - '1': 必须是线（维度 = 1）
//   - '2': 必须是面（维度 = 2）
//   - '*': 任意值（包括空）
//
// 常见模式：
//   - "T*F**FFF*": 判断是否 Contains（严格包含）
//   - "T*F**F*FF": 判断是否 Within（在内部）
//   - "FF*FF****": 判断是否 Disjoint（完全分离）
//
// 示例：
//
//	p1, _ := geos.NewPolygon(...)
//	p2, _ := geos.NewPolygon(...)
//	ok, _ := geos.RelatePattern(p1, p2, "T*F**FFF*")  // p1 是否包含 p2
func RelatePattern(a, b *gogeos.Geom, pattern string) (bool, error) {
	if a == nil || b == nil {
		return false, errNil
	}
	return safeRun(func() (bool, error) { return a.RelatePattern(b, pattern), nil })
}

// DistanceWithin 判断两个几何的距离是否在阈值内。
//
// 比调用 Distance() 再比较更高效，因为：
//   - DistanceWithin 内部可以提前终止计算
//   - 不需要计算精确距离值
//
// 参数：
//   - dist: 距离阈值
//
// 示例：
//
//	p1, _ := geos.NewPoint(0, 0)
//	p2, _ := geos.NewPoint(3, 4)
//	ok, _ := geos.DistanceWithin(p1, p2, 5)   // true（距离=5）
//	ok, _ = geos.DistanceWithin(p1, p2, 4)    // false（距离=5 > 4）
func DistanceWithin(a, b *gogeos.Geom, dist float64) (bool, error) {
	return predicateTwo(a, b, func() bool { return a.DistanceWithin(b, dist) })
}

// HausdorffDistance 计算两个几何之间的豪斯多夫距离。
//
// 豪斯多夫距离衡量两个形状之间的最大偏差。
// 定义：H(A,B) = max( h(A,B), h(B,A) )
// 其中 h(A,B) = max_{a∈A} min_{b∈B} d(a,b)
//
// 直观理解：A 中每个点到 B 的最近距离的最大值。
// 适用于衡量两个形状的相似程度。
//
// 示例：
//
//	route1, _ := geos.NewLineString([][]float64{{0,0}, {10,0}})
//	route2, _ := geos.NewLineString([][]float64{{0,1}, {10,1}})
//	d, _ := geos.HausdorffDistance(route1, route2)  // 1.0（最大偏差）
func HausdorffDistance(a, b *gogeos.Geom) (float64, error) {
	if a == nil || b == nil {
		return 0, errNil
	}
	return safeRun(func() (float64, error) { return a.HausdorffDistance(b), nil })
}

// pointPair 是内部辅助结构，用于存储两个点的坐标。
type pointPair struct{ x1, y1, x2, y2 float64 }

// NearestPoints 计算两个几何之间的最近点对。
//
// 返回 (ax, ay, bx, by, err)，其中：
//   - (ax, ay): 几何 a 上距离 b 最近的点
//   - (bx, by): 几何 b 上距离 a 最近的点
//
// 适用于：
//   - 计算两个几何之间的最短连接线
//   - 找到两个围栏之间最近的边界点
//   - 计算精确的最短距离（结合 Distance）
//
// 示例：
//
//	p1, _ := geos.NewPolygon([][][]float64{{{0,0},{4,0},{4,4},{0,4},{0,0}}})
//	p2, _ := geos.NewPolygon([][][]float64{{{10,10},{14,10},{14,14},{10,14},{10,10}}})
//	ax, ay, bx, by, _ := geos.NearestPoints(p1, p2)
//	// (ax,ay) = (4,4)（p1 上距离 p2 最近的点）
//	// (bx,by) = (10,10)（p2 上距离 p1 最近的点）
func NearestPoints(a, b *gogeos.Geom) (ax, ay, bx, by float64, err error) {
	if a == nil || b == nil {
		return 0, 0, 0, 0, errNil
	}
	result, err := safeRun(func() (pointPair, error) {
		coords := a.NearestPoints(b)
		if len(coords) != 2 {
			return pointPair{}, fmt.Errorf("NearestPoints 返回异常坐标数: %d", len(coords))
		}
		return pointPair{coords[0][0], coords[0][1], coords[1][0], coords[1][1]}, nil
	})
	if err != nil {
		return 0, 0, 0, 0, err
	}
	return result.x1, result.y1, result.x2, result.y2, nil
}
