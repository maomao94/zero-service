// Package geos 是对 GEOS C 几何引擎库的 Go 封装。
//
// # 设计原则
//
//   - 零 orb 依赖：直接使用 *gogeos.Geom，不依赖 github.com/paulmach/orb
//   - orb 转换独立：geos/orbconv 子包负责 orb ↔ GEOS 互转
//   - panic → error：统一通过 safeRun/safeRunErr 捕获 GEOS C 库 panic 转为 Go error
//   - Context 私有：getDefaultContext()（sync.Once 单例）包内私有，不对外暴露
//
// # 坐标约定
//
// 项目全部采用 {经度, 纬度} 顺序，即 {X, Y} = {Lon, Lat}。
// 这是 GeoJSON 标准和 orb 库的默认顺序，也是 GEOS 库 X/Y 的原生顺序。
//
// 所有需要反转的地方（如 H3 的 Lat/Lng）都在调用侧显式处理，不在 geos 包内。
// 详见 common/gisx/gisx.go 中的 OrbRingToH3LatLng。
//
// # 包内分工
//
//   construct.go    — 12 个几何构造（Point / LineString / Ring / Polygon / MultiPolygon / MultiPolygonFromGeoms / CollectionFromGeoms / BoundsRect / 4 个空几何）
//   convert.go      — 6 个格式转换（WKT / WKB / GeoJSON 的解析与序列化）
//   extract.go      — 7 个数据提取函数 + PolygonData 类型 + 泛型 ExtractMulti[T]
//   predicate.go    — 11 个空间谓词（Intersects / Contains / Covers / Touches 等）
//   prepared.go     — Prepared Geometry + 11 个加速谓词
//   overlay.go      — Overlay / Valid / Measure / Simplify / Transform / Meta 共 30+ 函数
//   relation.go     — DE-9IM 关系矩阵 / 距离 / 最近点
//   introspect.go   — 5 个几何内省函数（IsEmpty / IsSimple / IsClosed / IsRing / HasZ）
//   strtree.go      — STRtree R-Tree 空间索引
//   context.go      — GEOS 版本 / safeRun panic 捕获 / 默认 Context / 泛型辅助函数 oneAttr
//
// # GEOS 版本信息
//
// 当前封装基于 go-geos v0.20.4，底层链接 GEOS C 3.14.1。
// 已覆盖 GEOS C 库 80%+ 可实施 API（96+ 函数）。
//
// # 几何对象生命周期
//
// go-geos 通过 runtime.AddCleanup 自动管理 C 内存，无需手动释放。
// 但以下情况建议显式 Close：
//   - PreparedGeom：加速谓词用完后 Close 帮助 GC
//   - STRtree：索引用完后 Close 释放引用
//
// # 线程安全
//
// 所有函数都是线程安全的。go-geos 的每个 Context 内部持有 sync.Mutex，
// 任何对 C 指针的操作都经过加锁。同一 Context 下的并发调用串行执行。
//
// # 错误处理
//
// 所有函数通过 safeRun recover go-geos 可能发生的 panic 并转为 error。
// panic 恢复的错误前缀为 "geos: "；哨兵错误（如 ErrNil "geom 为 nil"）无此前缀。
// nil 几何参数返回 ErrNil 错误。
//
// # 快速开始
//
// 详见同目录 README.md，或从以下入口开始：
//
//	// 使用 orb 类型（推荐）
//	import "zero-service/common/gisx/geos/orbconv"
//	hit, _ := orbconv.CoversPointOrb(fence, userPoint)
//
//	// 使用纯坐标（不依赖 orb）
//	import geos "zero-service/common/gisx/geos"
//	g, _ := geos.NewPoint(116.39, 39.9)
//	wkt, _ := geos.ToWKT(g)
package geos
