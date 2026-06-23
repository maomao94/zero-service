// Package orbconv provides conversion between orb types and go-geos geometries,
// plus convenience wrappers that accept orb types directly.
//
// This is the recommended entry point for application code that works with orb types.
// For code that doesn't use orb, use the parent geos package directly.
//
// orb 类型说明（github.com/paulmach/orb）：
//   - orb.Point: [2]float64，索引 0 = 经度(lon)，索引 1 = 纬度(lat)
//   - orb.Ring:  []orb.Point，闭合环，首尾点应相同
//   - orb.Polygon: []orb.Ring，第一个 ring 是外环，后续 ring 是洞（hole）
//
// GEOS Geom 类型说明（github.com/twpayne/go-geos）：
//   - Geom 是 GEOS 几何对象的 Go 封装，底层是 C 指针
//   - Geom 可以表示 Point、LineString、LinearRing、Polygon、MultiPolygon 等
//   - 通过 TypeID() 可以判断具体类型
//
// 环形闭合合约：
// orb 层：EnsureRingClosed / EnsurePolygonClosed 负责自动闭合（精确 ==），调用方应在传入 orbconv 前闭合
// orbconv 层：ringToCoords() 仅做纯数据转换，不校验也不修改闭合
// GEOS 层：要求首尾坐标完全相同（差 1e-10 就 panic），被 safeRun 捕获为 error
package orbconv

import (
	"fmt"

	"github.com/paulmach/orb"
	gogeos "github.com/twpayne/go-geos"

	"zero-service/common/gisx/geos"
)

// --- Conversion functions ---

// GeomToRing 从 GEOS LineString 或 LinearRing 中提取坐标，转换为 orb.Ring。
//
// 支持的输入类型：LineString 或 LinearRing。
// 如果输入是 Polygon，应先调用 g.ExteriorRing() 或 g.InteriorRing(i) 取出单个 ring 再传入。
//
// 核心流程：
//  1. g.CoordSeq() —— 从 Geom 中取出底层坐标序列（CoordSeq），GEOS 存储坐标的 C 数据结构
//  2. .ToCoords() —— 将 CoordSeq 拷贝为 Go 的 [][]float64，每个元素是 [x, y]
//  3. 遍历 coords，将每个 []float64{c[0], c[1]} 转为 orb.Point{lon, lat}
func GeomToRing(g *gogeos.Geom) (orb.Ring, error) {
	if g == nil {
		return nil, geos.ErrNil
	}
	// CoordSeq() 返回几何对象的坐标序列；ToCoords() 拷贝为 Go slice
	// 每个 []float64 是一个点，[0]=X(经度), [1]=Y(纬度)
	coords := g.CoordSeq().ToCoords()
	if len(coords) == 0 {
		return nil, geos.ErrEmptyRing
	}
	ring := make(orb.Ring, 0, len(coords))
	for _, c := range coords {
		// orb.Point 是 [2]float64，索引 0 = lon，索引 1 = lat
		ring = append(ring, orb.Point{c[0], c[1]})
	}
	return ring, nil
}

// GeomToPoint 从 GEOS Point 中提取坐标，转换为 orb.Point。
//
// 输入必须是 TypeIDPoint 类型的几何对象。nil 输入返回 error。
func GeomToPoint(g *gogeos.Geom) (orb.Point, error) {
	if g == nil {
		return orb.Point{}, geos.ErrNil
	}
	// g.X() = 经度, g.Y() = 纬度
	return orb.Point{g.X(), g.Y()}, nil
}

// GeomToLineString 从 GEOS LineString 中提取坐标，转换为 orb.LineString。
//
// 与 GeomToRing 的区别：
//   - GeomToRing 返回 orb.Ring（语义上暗示闭合）
//   - GeomToLineString 返回 orb.LineString（不暗示闭合）
//
// 两者底层实现相同，都是 CoordSeq().ToCoords() → []orb.Point。
func GeomToLineString(g *gogeos.Geom) (orb.LineString, error) {
	if g == nil {
		return nil, geos.ErrNil
	}
	coords := g.CoordSeq().ToCoords()
	if len(coords) == 0 {
		return nil, geos.ErrEmptyRing
	}
	ls := make(orb.LineString, 0, len(coords))
	for _, c := range coords {
		ls = append(ls, orb.Point{c[0], c[1]})
	}
	return ls, nil
}

// GeomToPolygon 从 GEOS 几何对象中提取多边形，转换为 orb.Polygon。
//
// 支持的输入类型：
//   - TypeIDPolygon: 单个多边形，直接提取（包括外环和所有洞）
//   - TypeIDMultiPolygon: 多个多边形的集合，只取第一个（索引 0），其余的丢弃
//
// 如果需要保留 MultiPolygon 中的所有多边形，请使用 GeomToMultiPolygon。
//
// orb.Polygon 的结构：[]orb.Ring，其中：
//   - polygon[0] = 外环（exterior ring），定义多边形的外边界
//   - polygon[1:] = 洞（hole rings），定义多边形内部的孔洞
//
// GEOS Polygon 的内部结构：
//   - ExteriorRing() —— 返回外环（LinearRing 类型的 Geom）
//   - NumInteriorRings() —— 返回洞的数量
//   - InteriorRing(i) —— 返回第 i 个洞（LinearRing 类型的 Geom）
func GeomToPolygon(g *gogeos.Geom) (orb.Polygon, error) {
	if g == nil {
		return nil, geos.ErrNil
	}
	// TypeID() 返回几何对象的类型标识：
	//   TypeIDPoint=0, TypeIDLineString=1, TypeIDLinearRing=2,
	//   TypeIDPolygon=3, TypeIDMultiPoint=4, TypeIDMultiLineString=5,
	//   TypeIDMultiPolygon=6, TypeIDGeometryCollection=7
	switch g.TypeID() {
	case gogeos.TypeIDPolygon:
		return singlePolyToOrb(g)
	case gogeos.TypeIDMultiPolygon:
		// NumGeometries() 返回集合中子几何对象的数量
		n := g.NumGeometries()
		if n == 0 {
			return orb.Polygon{}, nil
		}
		// Geometry(0) 返回集合中第 0 个子几何对象
		return singlePolyToOrb(g.Geometry(0))
	default:
		return nil, fmt.Errorf("GeomToPolygon 不支持的类型: %d", g.TypeID())
	}
}

// GeomToMultiPolygon 从 GEOS 几何对象中提取多个多边形，转换为 orb.MultiPolygon。
//
// 与 GeomToPolygon 的区别：
//   - GeomToPolygon 对 MultiPolygon 只取第一个，其余的丢弃
//   - GeomToMultiPolygon 保留 MultiPolygon 中的所有独立多边形
//
// 支持的输入类型：
//   - TypeIDPolygon: 单个多边形，包装为只含一个元素的 MultiPolygon
//   - TypeIDMultiPolygon: 遍历所有子几何，逐个转为 orb.Polygon
//     - 每个子几何是独立的 Polygon，可以有自己的外环和洞
//
// orb.MultiPolygon 是 []orb.Polygon（等价于 [][]orb.Ring）。
//
// GEOS 的类型层级：
//
//	MultiPolygon (多个独立形状，彼此分离)
//	├── Polygon[0] (有外环 + 洞)
//	├── Polygon[1] (有外环 + 洞)
//	└── Polygon[2] (有外环，无洞)
func GeomToMultiPolygon(g *gogeos.Geom) (orb.MultiPolygon, error) {
	if g == nil {
		return nil, geos.ErrNil
	}
	switch g.TypeID() {
	case gogeos.TypeIDPolygon:
		poly, err := singlePolyToOrb(g)
		if err != nil {
			return nil, err
		}
		return orb.MultiPolygon{poly}, nil
	case gogeos.TypeIDMultiPolygon:
		n := g.NumGeometries()
		if n == 0 {
			return orb.MultiPolygon{}, nil
		}
		result := make(orb.MultiPolygon, 0, n)
		for i := 0; i < n; i++ {
			// Geometry(i) 返回第 i 个子几何，类型为 Polygon
			poly, err := singlePolyToOrb(g.Geometry(i))
			if err != nil {
				return nil, err
			}
			result = append(result, poly)
		}
		return result, nil
	default:
		return nil, fmt.Errorf("GeomToMultiPolygon 不支持的类型: %d", g.TypeID())
	}
}

// GeomToMultiPoint 从 GEOS MultiPoint 几何对象中提取所有点，转换为 orb.MultiPoint。
//
// MultiPoint 是多个独立点的集合，每个点之间没有连接关系。
// 支持的输入类型：
//   - TypeIDPoint: 单个点，包装为只含一个元素的 MultiPoint
//   - TypeIDMultiPoint: 遍历所有子几何，逐个提取坐标
//
// orb.MultiPoint 是 []orb.Point。
func GeomToMultiPoint(g *gogeos.Geom) (orb.MultiPoint, error) {
	if g == nil {
		return nil, geos.ErrNil
	}
	switch g.TypeID() {
	case gogeos.TypeIDPoint:
		return orb.MultiPoint{{g.X(), g.Y()}}, nil
	case gogeos.TypeIDMultiPoint:
		n := g.NumGeometries()
		if n == 0 {
			return orb.MultiPoint{}, nil
		}
		result := make(orb.MultiPoint, 0, n)
		for i := 0; i < n; i++ {
			sub := g.Geometry(i)
			result = append(result, orb.Point{sub.X(), sub.Y()})
		}
		return result, nil
	default:
		return nil, fmt.Errorf("GeomToMultiPoint 不支持的类型: %d", g.TypeID())
	}
}

// GeomToMultiLineString 从 GEOS MultiLineString 几何对象中提取所有线，转换为 orb.MultiLineString。
//
// MultiLineString 是多个独立线段的集合。
// 支持的输入类型：
//   - TypeIDLineString: 单条线，包装为只含一个元素的 MultiLineString
//   - TypeIDMultiLineString: 遍历所有子几何，逐个提取坐标
//
// orb.MultiLineString 是 []orb.LineString。
func GeomToMultiLineString(g *gogeos.Geom) (orb.MultiLineString, error) {
	if g == nil {
		return nil, geos.ErrNil
	}
	switch g.TypeID() {
	case gogeos.TypeIDLineString:
		ls, err := GeomToLineString(g)
		if err != nil {
			return nil, err
		}
		return orb.MultiLineString{ls}, nil
	case gogeos.TypeIDMultiLineString:
		n := g.NumGeometries()
		if n == 0 {
			return orb.MultiLineString{}, nil
		}
		result := make(orb.MultiLineString, 0, n)
		for i := 0; i < n; i++ {
			ls, err := GeomToLineString(g.Geometry(i))
			if err != nil {
				return nil, err
			}
			result = append(result, ls)
		}
		return result, nil
	default:
		return nil, fmt.Errorf("GeomToMultiLineString 不支持的类型: %d", g.TypeID())
	}
}

// singlePolyToOrb 将单个 GEOS Polygon 转换为 orb.Polygon。
//
// 处理步骤：
//  1. ExteriorRing() —— 取出外环（返回一个 Geom，类型为 LinearRing）
//  2. GeomToRing() —— 将外环 Geom 转为 orb.Ring
//  3. NumInteriorRings() —— 获取洞的数量
//  4. InteriorRing(i) —— 逐个取出每个洞，转为 orb.Ring
//  5. 所有 ring 组合成 orb.Polygon
func singlePolyToOrb(g *gogeos.Geom) (orb.Polygon, error) {
	// ExteriorRing() 返回 Polygon 的外环，返回的 Geom 类型为 LinearRing
	extRing, err := GeomToRing(g.ExteriorRing())
	if err != nil {
		return nil, err
	}
	poly := orb.Polygon{extRing}
	// NumInteriorRings() 返回 Polygon 中洞（hole）的数量
	for i := 0; i < g.NumInteriorRings(); i++ {
		// InteriorRing(i) 返回第 i 个洞，返回的 Geom 类型为 LinearRing
		hole, err := GeomToRing(g.InteriorRing(i))
		if err != nil {
			return nil, err
		}
		poly = append(poly, hole)
	}
	return poly, nil
}

// RingToGeom 将 orb.Ring 转换为 GEOS LinearRing。
//
// 转换流程：
//  1. ringToCoords() —— 将 orb.Ring 转为 [][]float64（纯转换）
//  2. geos.NewLinearRing() —— 创建 GEOS 的 LinearRing，GEOS 校验闭合
//
// LinearRing 是 GEOS 中表示闭合线环的类型，要求首尾坐标完全相同。
// 如果 orb.Ring 的首尾不一致，返回 error。
func RingToGeom(ring orb.Ring) (*gogeos.Geom, error) {
	coords := ringToCoords(ring)
	if len(coords) == 0 {
		return nil, geos.ErrEmptyRing
	}
	return geos.NewLinearRing(coords)
}

// LineStringToGeom 将 orb.LineString 转换为 GEOS LineString。
//
// 与 RingToGeom 的区别：
//   - RingToGeom 创建的是 LinearRing（GEOS 要求闭合，不闭合报错）
//   - LineStringToGeom 不强制闭合，创建的是 LineString
//
// orb.LineString 是 []orb.Point，GEOS LineString 不要求首尾坐标相同。
func LineStringToGeom(ls orb.LineString) (*gogeos.Geom, error) {
	if len(ls) == 0 {
		return nil, geos.ErrEmptyRing
	}
	coords := make([][]float64, 0, len(ls))
	for _, p := range ls {
		coords = append(coords, []float64{p.Lon(), p.Lat()})
	}
	return geos.NewLineString(coords)
}

// PolygonToGeom 将 orb.Polygon 转换为 GEOS Polygon。
//
// orb.Polygon 结构：[]orb.Ring
//   - poly[0] = 外环
//   - poly[1:] = 洞
//
// GEOS Polygon 需要：
//   - 一个外环（shell）作为 LinearRing
//   - 零或多个洞（holes）作为 LinearRing
//
// 转换流程：
//  1. ringToCoords() 将外环转为 [][]float64（纯转换，不校验闭合）
//  2. 遍历洞，逐个转为 [][]float64
//  3. geos.NewPolygon() 用 coordss[0] 作外环，coordss[1:] 作洞
func PolygonToGeom(poly orb.Polygon) (*gogeos.Geom, error) {
	if len(poly) == 0 {
		return nil, geos.ErrEmptyOuterRing
	}
	// 外环必须存在且非空
	outerCoords := ringToCoords(poly[0])
	if len(outerCoords) == 0 {
		return nil, geos.ErrEmptyOuterRing
	}
	coordss := make([][][]float64, 0, len(poly))
	coordss = append(coordss, outerCoords)
	// 后续 ring 是洞，跳过空的
	for _, ring := range poly[1:] {
		coords := ringToCoords(ring)
		if len(coords) > 0 {
			coordss = append(coordss, coords)
		}
	}
	// geos.NewPolygon(coordss):
	//   - coordss[0] 作为外环(shell)
	//   - coordss[1:] 作为洞(holes)
	return geos.NewPolygon(coordss)
}

// MultiPolygonToGeom 将 orb.MultiPolygon 转换为 GEOS 几何对象。
//
// orb.MultiPolygon 是 []orb.Polygon，每个元素是一个独立的带洞多边形。
// 独立的形状之间不连接，不属于同一个 Polygon 的外环内洞关系。
//
// 转换流程：
//  1. 遍历 orb.MultiPolygon 中的每个 orb.Polygon
//  2. 对每个 orb.Polygon 调用 PolygonToGeom 转为 GEOS Polygon
//  3. geos.NewMultiPolygonFromGeoms() 组装为 MultiPolygon
//
// 特殊情况：
//   - 空切片：返回 error
//   - 单个多边形：返回 GEOS Polygon（不包装为 MultiPolygon）
//   - 多个多边形：返回 GEOS MultiPolygon
func MultiPolygonToGeom(mp orb.MultiPolygon) (*gogeos.Geom, error) {
	if len(mp) == 0 {
		return nil, geos.ErrEmptyOuterRing
	}
	geoms := make([]*gogeos.Geom, 0, len(mp))
	for _, poly := range mp {
		g, err := PolygonToGeom(poly)
		if err != nil {
			return nil, err
		}
		if g != nil {
			geoms = append(geoms, g)
		}
	}
	if len(geoms) == 0 {
		return nil, geos.ErrEmptyOuterRing
	}
	return geos.NewMultiPolygonFromGeoms(geoms)
}

// PointToGeom 将 orb.Point 转换为 GEOS Point。
//
// orb.Point 是 [2]float64，Lon()=索引 0，Lat()=索引 1
func PointToGeom(p orb.Point) (*gogeos.Geom, error) {
	return geos.NewPoint(p.Lon(), p.Lat())
}

// MultiPointToGeom 将 orb.MultiPoint 转换为 GEOS 几何对象。
//
// orb.MultiPoint 是 []orb.Point，每个元素是一个独立的点。
// 单个点直接返回 GEOS Point，多个点返回 GEOS MultiPoint。
func MultiPointToGeom(mp orb.MultiPoint) (*gogeos.Geom, error) {
	if len(mp) == 0 {
		return nil, geos.ErrEmptyOuterRing
	}
	geoms := make([]*gogeos.Geom, 0, len(mp))
	for _, pt := range mp {
		g, err := PointToGeom(pt)
		if err != nil {
			return nil, err
		}
		if g != nil {
			geoms = append(geoms, g)
		}
	}
	if len(geoms) == 0 {
		return nil, geos.ErrEmptyOuterRing
	}
	if len(geoms) == 1 {
		return geoms[0], nil
	}
	return geos.NewCollectionFromGeoms(gogeos.TypeIDMultiPoint, geoms)
}

// MultiLineStringToGeom 将 orb.MultiLineString 转换为 GEOS 几何对象。
//
// orb.MultiLineString 是 []orb.LineString，每个元素是一条独立的线。
// 单条线直接返回 GEOS LineString，多条线返回 GEOS MultiLineString。
func MultiLineStringToGeom(mls orb.MultiLineString) (*gogeos.Geom, error) {
	if len(mls) == 0 {
		return nil, geos.ErrEmptyOuterRing
	}
	geoms := make([]*gogeos.Geom, 0, len(mls))
	for _, ls := range mls {
		g, err := LineStringToGeom(ls)
		if err != nil {
			return nil, err
		}
		if g != nil {
			geoms = append(geoms, g)
		}
	}
	if len(geoms) == 0 {
		return nil, geos.ErrEmptyOuterRing
	}
	if len(geoms) == 1 {
		return geoms[0], nil
	}
	return geos.NewCollectionFromGeoms(gogeos.TypeIDMultiLineString, geoms)
}

// --- Convenience wrappers (orb types in, standard types out) ---

// IntersectsOrb 判断两 orb.Polygon 是否有交集。
// 返回 true 表示两个多边形有任何重叠（包括边相交、包含等）。
func IntersectsOrb(a, b orb.Polygon) (bool, error) {
	ag, bg, err := both(a, b)
	if err != nil {
		return false, err
	}
	return geos.Intersects(ag, bg)
}

// ContainsOrb 判断 outer 是否包含 inner（边界不算）。
// 即 inner 的所有点都严格在 outer 内部，不接触 outer 的边界。
func ContainsOrb(outer, inner orb.Polygon) (bool, error) {
	og, ig, err := both(outer, inner)
	if err != nil {
		return false, err
	}
	return geos.Contains(og, ig)
}

// CoversOrb 判断 outer 是否覆盖 inner（边界算，围栏命中场景）。
// 即 inner 的所有点都在 outer 内部或边界上。
// 围栏命中判断通常用 Covers 而非 Contains，因为点在边界上也算命中。
func CoversOrb(outer, inner orb.Polygon) (bool, error) {
	og, ig, err := both(outer, inner)
	if err != nil {
		return false, err
	}
	return geos.Covers(og, ig)
}

// CoversPointOrb 判断 poly 是否覆盖点 pt（边界算）。
// 即点在多边形内部或边界上都返回 true。
// 适用于围栏命中检测：用户站在围栏边界上也算命中。
//
// 实现流程：
//  1. PolygonToGeom(poly) —— 将 orb.Polygon 转为 GEOS Polygon
//  2. geos.NewPoint(pt.Lon(), pt.Lat()) —— 将 orb.Point 转为 GEOS Point
//  3. geos.Covers(g, pg) —— 判断多边形是否覆盖该点
func CoversPointOrb(poly orb.Polygon, pt orb.Point) (bool, error) {
	g, err := PolygonToGeom(poly)
	if err != nil || g == nil {
		return false, err
	}
	pg, err := geos.NewPoint(pt.Lon(), pt.Lat())
	if err != nil {
		return false, err
	}
	return geos.Covers(g, pg)
}

// ContainsPointOrb 判断 poly 是否包含点 pt（边界不算）。
// 即点严格在多边形内部，不接触边界才返回 true。
// 与 CoversPointOrb 的区别：Contains 不含边界，Covers 含边界。
func ContainsPointOrb(poly orb.Polygon, pt orb.Point) (bool, error) {
	g, err := PolygonToGeom(poly)
	if err != nil || g == nil {
		return false, err
	}
	pg, err := geos.NewPoint(pt.Lon(), pt.Lat())
	if err != nil {
		return false, err
	}
	return geos.Contains(g, pg)
}

// ValidOrb 判断 orb.Polygon 是否有效。
// 有效的多边形要求：
//   - 环是闭合的
//   - 环不自交
//   - 洞完全在外环内部
//   - 洞之间不重叠
func ValidOrb(poly orb.Polygon) (bool, error) {
	g, err := PolygonToGeom(poly)
	if err != nil || g == nil {
		return false, err
	}
	return geos.IsValid(g)
}

// MakeValidOrb 调用 GEOS MakeValid 修复无效多边形，返回有效的 orb.Polygon。
//
// GEOS MakeValid 对不同无效原因的 Polygon 返回不同类型（11 种场景实测）：
//   有效-无洞          → Polygon (3)   原样返回
//   有效-单洞          → Polygon (3)   原样返回
//   有效-多洞不重叠     → Polygon (3)   原样返回
//   无效-重叠洞         → MultiPolygon (6)  子0=外环不变+洞合并    取子0
//   无效-洞包含洞       → MultiPolygon (6)  子0=外环不变(无洞)     取子0
//   无效-洞超出外环     → MultiPolygon (6)  子0=外环重绘(掏了凹口)  取子0
//   无效-洞完全在外     → MultiPolygon (6)  子0=外环不变(无洞)     取子0
//   无效-自相交(蝴蝶结)  → MultiPolygon (6)  子0=三角形            取子0
//   无效-三洞重叠       → MultiPolygon (6)  子0=外环不变+洞合并    取子0
//   无效-洞碰外环边     → GeometryCollection (7)  拒绝
//   退化-三点共线       → MultiLineString (5)     拒绝
//
// 策略：GEOS 已经合法化了几何体，直接取子多边形 0 作为结果。
// 不跟原始多边形做外环点数或 bbox 比较——重绘本身就是合法的修复。
// 仅拒绝无法表示为 Polygon 的类型（GeometryCollection、退化类型）。
func MakeValidOrb(poly orb.Polygon) (orb.Polygon, error) {
	g, err := PolygonToGeom(poly)
	if err != nil || g == nil {
		return nil, err
	}
	fixed, err := geos.MakeValid(g)
	if err != nil {
		return nil, err
	}
	switch fixed.TypeID() {
	case gogeos.TypeIDPolygon:
		return GeomToPolygon(fixed)
	case gogeos.TypeIDMultiPolygon:
		sub0, err := GeomToPolygon(fixed.Geometry(0))
		if err != nil {
			return nil, err
		}
		if len(sub0) == 0 {
			return nil, fmt.Errorf("MakeValid 子多边形 0 为空")
		}
		return sub0, nil
	default:
		return nil, fmt.Errorf("MakeValid 返回不支持的类型: %d", fixed.TypeID())
	}
}

// both 将两个 orb.Polygon 分别转为 GEOS Geom，用于二元几何运算。
// 如果任一转换失败，返回错误。
func both(a, b orb.Polygon) (*gogeos.Geom, *gogeos.Geom, error) {
	ag, err := PolygonToGeom(a)
	if err != nil {
		return nil, nil, err
	}
	bg, err := PolygonToGeom(b)
	if err != nil {
		return nil, nil, err
	}
	return ag, bg, nil
}

// ringToCoords 将 orb.Ring 转为 [][]float64 坐标数组。
//
// orb.Ring 是 []orb.Point。
// 纯数据转换，不校验闭合（GEOS 层会校验）。
func ringToCoords(ring orb.Ring) [][]float64 {
	coords := make([][]float64, 0, len(ring))
	for _, p := range ring {
		coords = append(coords, []float64{p.Lon(), p.Lat()})
	}
	return coords
}
