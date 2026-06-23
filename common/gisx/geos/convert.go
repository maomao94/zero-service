package geos

// convert.go — 几何格式转换
//
// 本文件提供 GEOS 几何对象与常见文本/二进制格式之间的互转功能。
//
// 支持的格式：
//   - WKT (Well-Known Text): OGC 标准文本格式，人类可读
//     示例: "POLYGON ((0 0, 4 0, 4 4, 0 4, 0 0))"
//   - WKB (Well-Known Binary): OGC 标准二进制格式，PostGIS 存储格式
//     适合网络传输和数据库存储，体积小、解析快
//   - GeoJSON: JSON 格式的地理数据，Web 地图 API 通用格式
//     示例: {"type":"Polygon","coordinates":[[[0,0],[4,0],[4,4],[0,4],[0,0]]]}
//
// 所有解析函数（From*）在格式错误时返回 error，不会 panic。
// 所有序列化函数（To*）在几何为 nil 时可能 panic，由 safeRun 捕获。

import gogeos "github.com/twpayne/go-geos"

// FromWKT 将 WKT 文本解析为 GEOS 几何对象。
//
// WKT (Well-Known Text) 是 OGC 标准的几何文本表示格式。
// 常见格式：
//   - 点:      "POINT (116.39 39.9)"
//   - 线:      "LINESTRING (0 0, 3 0, 3 3)"
//   - 多边形:  "POLYGON ((0 0, 4 0, 4 4, 0 4, 0 0))"
//   - 带洞:    "POLYGON ((0 0, 10 0, 10 10, 0 10, 0 0), (2 2, 8 2, 8 8, 2 8, 2 2))"
//   - 多点:    "MULTIPOINT ((0 0), (1 1))"
//   - 多线:    "MULTILINESTRING ((0 0, 1 1), (2 2, 3 3))"
//   - 多面:    "MULTIPOLYGON (((0 0, 4 0, 4 4, 0 4, 0 0)))"
//
// WKT 坐标顺序：X Y，即 经度 纬度。
// 解析失败时返回 error（如格式不合法、括号不匹配等）。
//
// 示例：
//
//	g, err := geos.FromWKT("POLYGON ((0 0, 4 0, 4 4, 0 4, 0 0))")
//	if err != nil {
//	    log.Fatal(err)  // 格式错误
//	}
func FromWKT(wkt string) (*gogeos.Geom, error) {
	return safeRun(func() (*gogeos.Geom, error) {
		return getDefaultContext().NewGeomFromWKT(wkt)
	})
}

// ToWKT 将 GEOS 几何对象序列化为 WKT 文本。
//
// 返回的 WKT 字符串可用于日志输出、调试显示、或传递给其他系统。
// 如果需要控制输出精度，可使用 g.ToWKTWithPrecision(precision)。
//
// 示例：
//
//	g, _ := geos.NewPoint(116.39, 39.9)
//	wkt, _ := geos.ToWKT(g)
//	// wkt = "POINT (116.39 39.9)"
func ToWKT(g *gogeos.Geom) (string, error) {
	return safeRun(func() (string, error) { return g.ToWKT(), nil })
}

// FromWKB 将 WKB 二进制数据解析为 GEOS 几何对象。
//
// WKB (Well-Known Binary) 是 OGC 标准的几何二进制表示格式。
// PostGIS 的 ST_AsBinary() 函数输出的就是 WKB 格式。
// WKB 比 WKT 更紧凑，适合网络传输和数据库存储。
//
// 解析失败时返回 error（如数据截断、格式不合法等）。
//
// 示例：
//
//	// 从 PostGIS 查询结果解析
//	wkb, _ := hex.DecodeString("01030000000100000005000000...")
//	g, err := geos.FromWKB(wkb)
func FromWKB(wkb []byte) (*gogeos.Geom, error) {
	return safeRun(func() (*gogeos.Geom, error) {
		return getDefaultContext().NewGeomFromWKB(wkb)
	})
}

// ToWKB 将 GEOS 几何对象序列化为 WKB 二进制数据。
//
// 输出的 WKB 可直接存入 PostGIS 或通过网络传输。
// 如果需要带 SRID 的 WKB（EWKB），使用 g.ToEWKBWithSRID(srid)。
//
// 示例：
//
//	g, _ := geos.NewPolygon([][][]float64{{{0,0},{4,0},{4,4},{0,4},{0,0}}})
//	wkb, _ := geos.ToWKB(g)
//	// wkb 可直接写入数据库的 geometry 列
func ToWKB(g *gogeos.Geom) ([]byte, error) {
	return safeRun(func() ([]byte, error) { return g.ToWKB(), nil })
}

// FromGeoJSON 将 GeoJSON 文本解析为 GEOS 几何对象。
//
// GeoJSON 是基于 JSON 的地理数据格式，广泛用于 Web 地图 API（如 Mapbox、Leaflet）。
// 坐标顺序在 GeoJSON 标准中是 [经度, 纬度]，与 GEOS 的 X=经度, Y=纬度 一致。
//
// 支持的 GeoJSON 类型：
//   - Point: {"type":"Point","coordinates":[116.39,39.9]}
//   - LineString: {"type":"LineString","coordinates":[[0,0],[3,0],[3,3]]}
//   - Polygon: {"type":"Polygon","coordinates":[[[0,0],[4,0],[4,4],[0,4],[0,0]]]}
//   - MultiPoint, MultiLineString, MultiPolygon, GeometryCollection
//
// 注意：只接受 geometry 部分，不接受完整的 GeoJSON Feature 或 FeatureCollection。
// 解析失败时返回 error。
//
// 示例：
//
//	geojson := `{"type":"Polygon","coordinates":[[[0,0],[4,0],[4,4],[0,4],[0,0]]]}`
//	g, err := geos.FromGeoJSON(geojson)
func FromGeoJSON(geojson string) (*gogeos.Geom, error) {
	return safeRun(func() (*gogeos.Geom, error) {
		return getDefaultContext().NewGeomFromGeoJSON(geojson)
	})
}

// ToGeoJSON 将 GEOS 几何对象序列化为 GeoJSON 文本。
//
// 参数：
//   - indent: 缩进空格数，0 表示紧凑输出（无换行无缩进）
//
// 输出的 GeoJSON 可直接用于 Web API 响应或前端地图渲染。
//
// 示例：
//
//	g, _ := geos.NewPoint(116.39, 39.9)
//	geojson, _ := geos.ToGeoJSON(g, 0)
//	// geojson = {"type":"Point","coordinates":[116.39,39.9]}
//
//	geojson, _ = geos.ToGeoJSON(g, 2)
//	// 带缩进的美化输出
func ToGeoJSON(g *gogeos.Geom, indent int) (string, error) {
	return safeRun(func() (string, error) { return g.ToGeoJSON(indent), nil })
}
