package geos

// extract.go — 几何数据提取工具
//
// 提供不依赖 orb 的通用数据提取函数，直接从 *gogeos.Geom 中获取原始坐标数据。
// 适用于不想引入 orb 类型依赖的场景。
//
// 核心设计：
//   - 每个 type 一个提取函数，返回简单的 Go 原生类型
//   - ExtractMulti 用泛型统一处理所有集合类型
//
// 坐标约定：始终 {X, Y} = {经度, 纬度}，不做任何重排。

import (
	"fmt"

	gogeos "github.com/twpayne/go-geos"
)

// --- 基础提取器 ---

// ExtractPoint 从 GEOS Point 中提取 (x, y) 坐标。
//
// 适用于 TypeIDPoint。非 Point 类型会 panic（被 safeRun 捕获）。
func ExtractPoint(g *gogeos.Geom) (x, y float64, err error) {
	if g == nil {
		return 0, 0, errNil
	}
	type xy struct{ x, y float64 }
	r, e := safeRun(func() (xy, error) {
		return xy{g.X(), g.Y()}, nil
	})
	if e != nil {
		return 0, 0, e
	}
	return r.x, r.y, nil
}

// ExtractCoords 从 GEOS LineString / LinearRing / Point 中提取坐标序列。
//
// 返回 [][]float64{{x1,y1}, {x2,y2}, ...}。
// 这是最通用的提取函数，通过 CoordSeq().ToCoords() 获取数据。
func ExtractCoords(g *gogeos.Geom) ([][]float64, error) {
	if g == nil {
		return nil, errNil
	}
	return safeRun(func() ([][]float64, error) {
		coords := g.CoordSeq().ToCoords()
		if coords == nil {
			return nil, nil
		}
		return coords, nil
	})
}

// ExtractPolygonCoords 从 GEOS Polygon 中提取所有环的坐标。
//
// 返回 PolygonData，与 orb.Polygon 约定一致：
//
//	data[0]  = 外环坐标 [][]float64{{x1,y1}, {x2,y2}, ...}
//	data[1:] = 洞坐标（每个洞是 [][]float64），无洞时为空
//
// 示例：
//
//	data, _ := geos.ExtractPolygonCoords(poly)
//	outer := data.Outer()   // 外环
//	for _, h := range data.Holes() { ... }  // 遍历洞
func ExtractPolygonCoords(g *gogeos.Geom) (PolygonData, error) {
	if g == nil {
		return nil, errNil
	}
	return safeRun(func() (PolygonData, error) {
		outer := g.ExteriorRing().CoordSeq().ToCoords()
		n := g.NumInteriorRings()
		data := make(PolygonData, 0, 1+n)
		data = append(data, outer)
		for i := 0; i < n; i++ {
			data = append(data, g.InteriorRing(i).CoordSeq().ToCoords())
		}
		return data, nil
	})
}

// ExtractPolygonOrMultiCoords 自动判断类型，提取多边形坐标。
//
// 如果是 Polygon：返回一个元素的切片 [{outer: ..., holes: ...}]
// 如果是 MultiPolygon：返回多个
// 其他类型：返回 nil
//
// 这是最方便的"不知道是不是 Multi"场景的提取函数。
func ExtractPolygonOrMultiCoords(g *gogeos.Geom) ([]PolygonData, error) {
	if g == nil {
		return nil, errNil
	}
	switch g.TypeID() {
	case gogeos.TypeIDPolygon:
		data, err := ExtractPolygonCoords(g)
		if err != nil {
			return nil, err
		}
		return []PolygonData{data}, nil
	case gogeos.TypeIDMultiPolygon:
		return ExtractMulti(g, ExtractPolygonCoords)
	}
	return nil, fmt.Errorf("%w: %d", ErrNotSupported, g.TypeID())
}

// PolygonData 表示一个多边形（外环 + 洞）的坐标数据。
// 与 orb.Polygon 约定一致：data[0] 是外环，data[1:] 是洞（可为空）。
//
// 示例：
//
//	无洞多边形:  [][]float64{外环}                                len = 1
//	带 1 个洞:   [][]float64{外环, 洞}                             len = 2
//	带 2 个洞:   [][]float64{外环, 洞1, 洞2}                       len = 3
type PolygonData [][][]float64

// Outer 返回外环坐标。
func (d PolygonData) Outer() [][]float64 {
	if len(d) == 0 {
		return nil
	}
	return d[0]
}

// Holes 返回洞坐标列表（可能为空）。
func (d PolygonData) Holes() [][][]float64 {
	if len(d) <= 1 {
		return nil
	}
	return [][][]float64(d[1:])
}

// --- 泛型集合提取器 ---

// ExtractMulti 是泛型集合提取器，遍历 GEOS Multi*/GeometryCollection 的所有子几何。
//
// 类型参数 T 是每个子几何的提取结果类型。
// fn 是子几何提取函数，如 ExtractPoint, ExtractCoords 等。
//
// 对于 TypeIDMultiPoint, TypeIDMultiLineString, TypeIDMultiPolygon 可靠。
// 对于 TypeIDGeometryCollection 直接返回 error，提示使用 ExtractMultiSafe。
//
// 使用示例：
//
//	lines, _ := geos.ExtractMulti(multiLineStr, geos.ExtractCoords)
func ExtractMulti[T any](g *gogeos.Geom, fn func(*gogeos.Geom) (T, error)) ([]T, error) {
	if g == nil {
		return nil, errNil
	}
	if g.TypeID() == gogeos.TypeIDGeometryCollection {
		return nil, fmt.Errorf("GeometryCollection 不推荐，子类型不确定，请使用 ExtractMultiSafe")
	}
	n := g.NumGeometries()
	if n == 0 {
		return nil, nil
	}
	result := make([]T, 0, n)
	for i := 0; i < n; i++ {
		sub := g.Geometry(i)
		v, err := fn(sub)
		if err != nil {
			return nil, fmt.Errorf("ExtractMulti[%d]: %w", i, err)
		}
		result = append(result, v)
	}
	return result, nil
}

// ExtractMultiSafe 安全遍历集合几何，跳过类型不匹配的子几何（仅警告不报错）。
//
// 与 ExtractMulti 的区别：
//   - ExtractMulti 遇到类型不匹配会立即返回 error
//   - ExtractMultiSafe 跳过不匹配的子几何，继续处理下一个
//
// 适用于 GeometryCollection 中混合不同类型的场景。
func ExtractMultiSafe[T any](g *gogeos.Geom, fn func(*gogeos.Geom) (T, error)) ([]T, error) {
	if g == nil {
		return nil, errNil
	}
	n := g.NumGeometries()
	if n == 0 {
		return nil, nil
	}
	result := make([]T, 0, n)
	for i := 0; i < n; i++ {
		sub := g.Geometry(i)
		v, err := fn(sub)
		if err != nil {
			continue // 跳过类型不匹配的子几何
		}
		result = append(result, v)
	}
	return result, nil
}

// --- 帮助函数：提取点并解析为浮点数对 ---

// ExtractPoints 从 GEOS Point, MultiPoint, LineString 中提取所有点。
//
// 返回 [][]float64{{x1,y1}, {x2,y2}, ...}。
// 自动处理：Point → 单元素，MultiPoint/LineString → 多元素。
func ExtractPoints(g *gogeos.Geom) ([][]float64, error) {
	if g == nil {
		return nil, errNil
	}
	switch g.TypeID() {
	case gogeos.TypeIDPoint:
		x, y, err := ExtractPoint(g)
		if err != nil {
			return nil, err
		}
		return [][]float64{{x, y}}, nil
	case gogeos.TypeIDLineString, gogeos.TypeIDLinearRing:
		return ExtractCoords(g)
	case gogeos.TypeIDMultiPoint:
		return ExtractMulti(g, func(sub *gogeos.Geom) ([]float64, error) {
			x, y, err := ExtractPoint(sub)
			if err != nil {
				return nil, err
			}
			return []float64{x, y}, nil
		})
	default:
		return nil, nil
	}
}
