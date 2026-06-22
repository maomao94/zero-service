# 引入 go-geos 并完善 common/gisx GEOS 工具层

## Goal

在 `common/gisx` 中引入 `github.com/twpayne/go-geos` 作为 GEOS C 库的 Go 绑定，构建一个完整、项目类型友好、安全的 GEOS 工具层，为后续 GIS 业务逻辑（如 `GenerateFenceCells`、`PointInFence`、`NearbyFences` 精判）迁移到 GEOS 打基础。

本任务**只做工具层和基础设施**，不迁移任何 `app/gis/internal/logic` 业务逻辑。业务迁移作为后续独立任务处理。

## Background

- 当前 `common/gisx/intersect.go` 使用纯 Go 跨立实验算法做 `PolygonIntersect`，`planar.PolygonContains` 做点包含。
- 未来围栏场景可能面对复杂多边形（自相交、带洞、大量顶点），纯 Go 实现在鲁棒性和性能上有瓶颈。
- GEOS 是成熟的开源几何库，`go-geos` 是其 Go 绑定，使用 `GEOS*_r` 线程安全 C API。
- `app/gis/Dockerfile` 已开启 `CGO_ENABLED=1`，但未安装 GEOS 开发库和运行时库。

## Requirements

### R1 依赖引入
- 新增 `github.com/twpayne/go-geos` 依赖到 `go.mod`。
- 不引入其他 GEOS Go 绑定（`paulsmith/gogeos` 已废弃，不采用）。

### R2 Docker / 部署
- `app/gis/Dockerfile` builder 阶段安装 `geos-dev` 和 `pkgconf`，并在构建时输出 GEOS 版本（`geos-config --version`）。
- `app/gis/Dockerfile` runtime 阶段安装 `geos` 动态库。
- 不影响其他服务 Dockerfile（GEOS 只在 gis 服务使用）。

### R3 工具层范围（完整，不分阶段）
`common/gisx` 新增 GEOS 工具，按 GEOS C API 能力分组：

- **版本信息**：`GEOSVersion()`、`GEOSVersionString()`
- **Context 管理**：内部 `*geos.Context`，对业务隐藏
- **几何构造**：从 `orb.Point`、`orb.Ring`、`orb.Polygon`、bbox 构造 GEOS geometry
- **格式转换**：WKT / WKB / GeoJSON 与 orb / GEOS 互转
- **基础谓词**：`Intersects`、`Contains`、`Covers`、`Within`、`Touches`、`Disjoint`、`Equals`、`Overlaps`、`Crosses`
- **Prepared Geometry**：`PreparedPolygon`，支持 `Intersects`、`Contains`、`ContainsPoint`、`Covers`、`CoversPoint`、`Disjoint`
- **Overlay 运算**：`Intersection`、`Union`、`Difference`、`SymDifference`
- **校验/修复**：`IsValid`、`IsValidReason`、`MakeValid`
- **简化/缓冲**：`Buffer`、`Simplify`、`TopologyPreserveSimplify`、`ConvexHull`、`ConcaveHull`
- **测量**：`Area`、`Length`、`Distance`、`Centroid`、`PointOnSurface`
- **Bounds / STRtree**：`Bounds`、`STRtree` 空间索引查询

### R4 安全封装
- `go-geos` 部分操作错误会 panic，工具层必须统一 `recover`，返回 `error`，不让 panic 泄漏到业务层。
- Context 生命周期管理：工具函数内部短生命周期；`PreparedPolygon` 等长生命周期对象提供显式 `Close()` 或依赖 GC cleanup。
- 坐标顺序约定：`orb.Point{lon, lat}` → GEOS `(x=lon, y=lat)`，与现有 `gisx-guidelines.md` 一致。

### R5 包边界
- `common/gisx` 对外仍以 `orb.*`、WKT、WKB、GeoJSON 为类型边界。
- 不让 `app/gis/internal/logic` 直接依赖 `github.com/twpayne/go-geos`。
- 不引用 `app/*/` 下的 pb 类型（符合现有 `gisx-guidelines.md` 包边界）。

### R6 测试覆盖
- `common/gisx` 新增 `_test.go`，覆盖：
  - 版本信息非空
  - 构造：point / ring / polygon / bbox
  - 格式转换：WKT / WKB / GeoJSON 往返
  - 谓词：相交 / 包含 / 覆盖 / 远离 / 边界接触 / 共线
  - `Contains` vs `Covers` 边界点语义差异
  - Prepared predicate 与普通 predicate 结果一致
  - Overlay：intersection / union / difference
  - 无效 polygon：`IsValid` / `IsValidReason` / `MakeValid`
  - Buffer / Simplify / ConvexHull
  - STRtree 查询命中和未命中
  - panic recover：传入无效几何不 panic，返回 error

## Acceptance Criteria

- [ ] `go.mod` 新增 `github.com/twpayne/go-geos`，`go mod tidy` 通过
- [ ] `app/gis/Dockerfile` builder 安装 `geos-dev` + `pkgconf`，构建时输出 GEOS 版本
- [ ] `app/gis/Dockerfile` runtime 安装 `geos`
- [ ] `common/gisx` 新增 GEOS 工具文件，覆盖 R3 全部能力分组
- [ ] 工具函数 panic 被 recover，返回 error（R4）
- [ ] `common/gisx` 对外不暴露 `github.com/twpayne/go-geos` 类型（R5）
- [ ] `common/gisx` 新增测试覆盖 R6 全部场景
- [ ] `go build ./common/gisx/...` 通过
- [ ] `go test ./common/gisx/... -v -count=1` 通过
- [ ] `go vet ./common/gisx/...` 通过
- [ ] `go build ./app/gis/...` 通过（未迁移业务逻辑，但验证不破坏现有编译）
- [ ] 本地 `app/gis` Docker 镜像构建通过（手动验证）

## Out of Scope

- 不迁移 `GenerateFenceCells`、`PointInFence`、`NearbyFences` 等业务逻辑到 GEOS
- 不修改 `common/gisx/intersect.go` 现有纯 Go 实现（保留共存）
- 不改其他服务 Dockerfile
- 不改 proto 定义
