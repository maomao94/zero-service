// Package gisx 提供 GIS（地理信息系统）相关的通用工具和接口。
//
// # 坐标约定
//
// 全包统一采用 {经度, 纬度} 顺序，即 {Lon, Lat} = {X, Y}。
// 这与 orb.Point [lon, lat] 和 GEOS 的 X/Y 一致。
// 唯一需要反转的地方是 H3 相关函数（H3 的 LatLng 结构要求纬度在前），
// 由 OrbRingToH3LatLng 显式处理。
//
// # 包内分工
//
//   validate.go  — 坐标合法性校验（ValidateCoordinate），错误类型 ValidationError 支持 errors.As
//   gisx.go      — orb ↔ H3 格式互转（OrbRingToH3LatLng / H3LatLngsToOrbRing）+ 闭合检测/自动闭合（IsRingClosed / EnsureRingClosed / EnsurePolygonClosed）
//   store.go     — 围栏数据存取接口（FenceStore）+ 空实现（NoopFenceStore），围栏多边形使用 orb.Polygon（外环 + 洞）
//
// # 子包 geos/
//
// GEOS C 几何引擎的全功能 Go 封装，详见 common/gisx/geos/doc.go。
// 提供几何构造、空间谓词、叠加运算、格式转换等 96+ 函数。
// 推荐通过 orbconv 子包使用 orb 类型，或直接用 extract.go 获取原生坐标数据。
//
// # 快速入口
//
//	// 坐标校验（参数顺序 lon, lat）
//	err := gisx.ValidateCoordinate(116.39, 39.9, 0)
//
//	// orb 多边形转 H3 格式
//	gp, err := gisx.OrbPolygonToH3GeoPolygon(orbPolygon)
//
//	// H3 格式转 orb 多边形（反向）
//	poly := gisx.H3LatLngsToOrbPolygon(gp)
//
//	// GEOS 几何操作（推荐通过 orbconv）
//	hit, _ := orbconv.CoversPointOrb(fence, userPoint)
package gisx
