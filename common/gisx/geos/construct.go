package geos

// construct.go — 几何对象构造函数
//
// 本文件提供从原始坐标创建 GEOS 几何对象的功能。
// 所有坐标约定：{x, y} = {经度(lon), 纬度(lat)}，与 orb.Point [lon, lat] 对齐。
//
// GEOS 几何类型体系：
//   - Point: 单个点，由 (x, y) 坐标定义
//   - LineString: 线段，由一系列坐标点连接而成，不要求闭合
// - LinearRing: 线性环，是一种特殊的 LineString，要求首尾坐标完全相同（闭合）
// - Polygon: 多边形，由一个外环（shell）和零或多个洞（holes）组成。每个环必须首尾闭合，不闭合返回 error。
//
// 所有函数通过 safeRun 包装，捕获 GEOS C 库的 panic 并转为 Go error。
// 底层调用 go-geos 的 Context 方法，使用包级默认 Context（sync.Once 单例）。

import (
	gogeos "github.com/twpayne/go-geos"
)

// NewPoint 创建一个 GEOS Point 几何对象。
//
// 参数：
//   - x: X 坐标（经度）
//   - y: Y 坐标（纬度）
//
// 返回的 *gogeos.Geom 类型为 TypeIDPoint(0)。
// 可以通过 g.X() 和 g.Y() 获取坐标值。
//
// 示例：
//
//	p, _ := geos.NewPoint(116.39, 39.9)  // 北京天安门附近
//	fmt.Println(p.X(), p.Y())            // 116.39 39.9
func NewPoint(x, y float64) (*gogeos.Geom, error) {
	return safeRun(func() (*gogeos.Geom, error) {
		// NewPointFromXY 是 go-geos 提供的便捷方法，内部创建 CoordSeq 再创建 Point
		return getDefaultContext().NewPointFromXY(x, y), nil
	})
}

// NewLineString 创建一个 GEOS LineString（线段）几何对象。
//
// 参数：
//   - coords: 坐标点序列，每个元素为 []float64{x, y}
//
// LineString 不要求首尾闭合（与 LinearRing 不同）。
// 至少需要 2 个坐标点。
//
// 示例：
//
//	ls, _ := geos.NewLineString([][]float64{{0, 0}, {3, 0}, {3, 3}})
//	// 创建一条折线：(0,0) → (3,0) → (3,3)
func NewLineString(coords [][]float64) (*gogeos.Geom, error) {
	return safeRun(func() (*gogeos.Geom, error) {
		return getDefaultContext().NewLineString(coords), nil
	})
}

// NewLinearRing 创建一个 GEOS LinearRing（线性环）几何对象。
//
// 参数：
//   - coords: 坐标点序列，每个元素为 []float64{x, y}
//
// LinearRing 是 Polygon 的组成部分，要求首尾坐标完全相同（闭合）。
// 未闭合环会导致 GEOS panic（被 safeRun 捕获为 error）。
//
// 示例：
//
//	lr, _ := geos.NewLinearRing([][]float64{{0, 0}, {4, 0}, {4, 4}, {0, 0}})
//	// 环必须首尾闭合
func NewLinearRing(coords [][]float64) (*gogeos.Geom, error) {
	return safeRun(func() (*gogeos.Geom, error) {
		return getDefaultContext().NewLinearRing(coords), nil
	})
}

// NewPolygon 创建一个 GEOS Polygon（多边形）几何对象。
//
// 参数：
//   - rings: 三维坐标数组，结构为 [][][]float64
//     - rings[0]: 外环（shell）的坐标，定义多边形的外边界
//     - rings[1:]: 洞（holes）的坐标，定义多边形内部的孔洞
//
// 每个 ring 的坐标格式为 [][]float64{{x1,y1}, {x2,y2}, ..., {x1,y1}}，首尾必须闭合。
//
// orb.Polygon 到本函数的映射：
//
//	orb.Polygon = []orb.Ring
//	  polygon[0] → rings[0]  （外环）
//	  polygon[1] → rings[1]  （第一个洞）
//	  polygon[2] → rings[2]  （第二个洞）
//	  ...
//
// 示例：
//
//	// 无洞多边形
//	poly, _ := geos.NewPolygon([][][]float64{
//	    {{0,0}, {4,0}, {4,4}, {0,4}, {0,0}},  // 外环
//	})
//
//	// 有洞多边形
//	poly, _ := geos.NewPolygon([][][]float64{
//	    {{0,0}, {10,0}, {10,10}, {0,10}, {0,0}},  // 外环
//	    {{2,2}, {8,2}, {8,8}, {2,8}, {2,2}},       // 洞
//	})
func NewPolygon(rings [][][]float64) (*gogeos.Geom, error) {
	return safeRun(func() (*gogeos.Geom, error) {
		if len(rings) == 0 {
			return nil, ErrEmptyOuterRing
		}
		return getDefaultContext().NewPolygon(rings), nil
	})
}

// --- 空几何构造函数 ---
//
// 空几何是有类型但没有坐标的几何对象。IsEmpty() 返回 true。
// 常见用途：
//   - 两个不相交几何的 Intersection 返回空几何（而非 nil）
//   - 初始化一个"无结果"的占位几何
//   - 遍历 Multi* 集合时作为空初始值

// NewEmptyPoint 创建一个空的 GEOS Point。
func NewEmptyPoint() (*gogeos.Geom, error) {
	return safeRun(func() (*gogeos.Geom, error) {
		return getDefaultContext().NewEmptyPoint(), nil
	})
}

// NewEmptyLineString 创建一个空的 GEOS LineString。
func NewEmptyLineString() (*gogeos.Geom, error) {
	return safeRun(func() (*gogeos.Geom, error) {
		return getDefaultContext().NewEmptyLineString(), nil
	})
}

// NewEmptyPolygon 创建一个空的 GEOS Polygon。
func NewEmptyPolygon() (*gogeos.Geom, error) {
	return safeRun(func() (*gogeos.Geom, error) {
		return getDefaultContext().NewEmptyPolygon(), nil
	})
}

// NewEmptyCollection 创建一个空的 GEOS 集合几何（指定子类型）。
//
// 参数 typeID 指定集合的子类型，如 TypeIDMultiPolygon, TypeIDGeometryCollection 等。
func NewEmptyCollection(typeID gogeos.TypeID) (*gogeos.Geom, error) {
	return safeRun(func() (*gogeos.Geom, error) {
		return getDefaultContext().NewEmptyCollection(typeID), nil
	})
}

// NewMultiPolygon 创建一个 GEOS MultiPolygon（多面）几何对象。
//
// 参数：
//   - geomss: 四维坐标数组，最外层表示多个独立多边形
//     - geomss[0]: 第一个多边形的坐标（包含外环和洞）
//     - geomss[1]: 第二个多边形的坐标
//     - ...
//
// 每个多边形的坐标格式为 [][][]float64：
//   - [0]: 外环坐标
//   - [1:]: 洞坐标
//
// MultiPolygon 中的每个多边形是独立的形状，彼此不连接。
// 如果只有一个多边形，直接返回 Polygon 即可。
//
// 示例：
//
//	// 两个分离的正方形
//	mp, _ := geos.NewMultiPolygon([][][][]float64{
//	    {{{0,0},{2,0},{2,2},{0,2},{0,0}}},
//	    {{{5,5},{7,5},{7,7},{5,7},{5,5}}},
//	})
func NewMultiPolygon(geomss [][][][]float64) (*gogeos.Geom, error) {
	return safeRun(func() (*gogeos.Geom, error) {
		geoms := make([]*gogeos.Geom, 0, len(geomss))
		for _, rings := range geomss {
			g, err := NewPolygon(rings)
			if err != nil {
				return nil, err
			}
			geoms = append(geoms, g)
		}
		return getDefaultContext().NewCollection(gogeos.TypeIDMultiPolygon, geoms), nil
	})
}

// NewMultiPolygonFromGeoms 从多个 GEOS Polygon 几何对象组装为 MultiPolygon。
//
// 参数：
//   - geoms: 已经构造好的 Polygon 几何对象切片
//
// 如果只有一个几何，直接返回该 Polygon（不包装为 MultiPolygon）。
// 这是内部函数，主要用于 orbconv 等需要先独立构造每个 Polygon 的场景。
func NewMultiPolygonFromGeoms(geoms []*gogeos.Geom) (*gogeos.Geom, error) {
	return NewCollectionFromGeoms(gogeos.TypeIDMultiPolygon, geoms)
}

// NewCollectionFromGeoms 从多个 GEOS 几何对象组装为指定类型的集合几何。
//
// 如果只有一个几何，直接返回该几何（不包装为 Multi*）。
// 支持的类型：TypeIDMultiPoint, TypeIDMultiLineString, TypeIDMultiPolygon。
// 不推荐 TypeIDGeometryCollection（异构集合，多数工具不兼容）。
func NewCollectionFromGeoms(typeID gogeos.TypeID, geoms []*gogeos.Geom) (*gogeos.Geom, error) {
	return safeRun(func() (*gogeos.Geom, error) {
		if len(geoms) == 0 {
			return nil, ErrEmptyGeoms
		}
		if len(geoms) == 1 {
			return geoms[0], nil
		}
		return getDefaultContext().NewCollection(typeID, geoms), nil
	})
}

// NewBoundsRect 从边界框（bbox）创建一个矩形 Polygon。
//
// 参数：
//   - minX, minY: 左下角坐标（最小 X, 最小 Y）
//   - maxX, maxY: 右上角坐标（最大 X, 最大 Y）
//
// 创建的矩形是一个 Polygon，边与坐标轴平行。
// 适用于快速创建查询范围框、裁剪区域等。
//
// 示例：
//
//	// 创建一个 1°×1° 的矩形
//	rect, _ := geos.NewBoundsRect(116.0, 39.0, 117.0, 40.0)
func NewBoundsRect(minX, minY, maxX, maxY float64) (*gogeos.Geom, error) {
	return safeRun(func() (*gogeos.Geom, error) {
		return getDefaultContext().NewGeomFromBounds(minX, minY, maxX, maxY), nil
	})
}
