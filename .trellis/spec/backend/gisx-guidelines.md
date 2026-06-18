# GIS 服务与 gisx 包规范

> 涵盖 `common/gisx/` 通用包和 `app/gis/` 服务的架构约定。

## common/gisx/ 包边界

| 可以放 | 不可以放 |
|--------|----------|
| 纯几何计算（坐标校验、线段相交、多边形相交） | 引用 `app/*/` 下的 pb 类型 |
| orb/H3 类型转换（OrbPolygonToH3GeoPolygon） | 引用 gRPC generated 代码 |
| `FenceStore` 接口定义 + `NoopFenceStore` | 具体 store 实现（放 `app/gis/model/`） |
| `FenceInfo` 通用结构体 | 业务错误码 (`extproto`) |

参考文件：
- `common/gisx/validate.go` — 坐标校验
- `common/gisx/intersect.go` — 几何相交判断（跨立实验算法）
- `common/gisx/gisx.go` — H3 多边形转换、ring 自动闭合
- `common/gisx/store.go` — FenceStore 接口定义

## 坐标系约定

**关键规则**：不同库的坐标参数顺序不同，混用是常见 bug 来源。

| 库/类型 | 顺序 | 示例 |
|---------|------|------|
| `orb.Point` | `[经度, 纬度]` (lon, lat) | `orb.Point{116.4, 39.9}` |
| `h3.LatLng` | `{纬度, 经度}` (lat, lng) | `h3.LatLng{Lat: 39.9, Lng: 116.4}` |
| `geohash.Encode` | `(纬度, 经度)` (lat, lon) | `geohash.EncodeWithPrecision(39.9, 116.4, 7)` |
| pb `Point` | `lat`, `lon` 独立字段 | `&gis.Point{Lat: 39.9, Lon: 116.4}` |

pb→orb 转换时必须翻转：`orb.Point{p.Lon, p.Lat}`。

## FenceStore 接口模式

所有 store 方法必须传 `context.Context` 作为第一个参数：

```go
type FenceStore interface {
    CreateFence(ctx context.Context, fenceId, name string, points []orb.Point, h3Resolution, geohashPrecision int, h3Cells, geohashCells []string) error
    LoadFencePolygon(ctx context.Context, fenceId string) ([]orb.Point, error)
    // ...
}
```

注入规则（`app/gis/internal/svc/servicecontext.go`）：
- 配置了 `DB.DataSource` → 使用 `model.NewGormFenceStore(db)`
- 未配置 → 使用 `&gisx.NoopFenceStore{}`

## app/gis/ 服务架构

### RPC 分类

| 类别 | 示例 | 特点 |
|------|------|------|
| 纯计算 | `Distance`, `EncodeH3`, `GenerateFenceCells`, `RoutePoints` | 无 DB 依赖，无状态 |
| 围栏 CRUD | `CreateFence`, `UpdateFence`, `DeleteFence`, `ListFences`, `GetFence` | 依赖 FenceStore |
| 围栏判断 | `PointInFence`, `PointInFences`, `NearbyFences` | 支持直传 polygon 或 fenceId 两种模式 |
| 坐标转换 | `TransformCoord`, `BatchTransformCoord` | WGS84/GCJ02/BD09 互转 |

### logic 层 helper 模式

`logic/helper.go` 存放 pb↔领域类型的转换和通用校验：
- `ValidatePoints` — pb Point 批量校验（非空、非 nil、经纬度范围）
- `pbPointToOrbPolygon` — pb Point 切片→orb.Polygon（单外环，自动闭合）

反模式：不要把 pb 类型的转换函数放到 `common/gisx/`。

### 批量接口校验规则

单点接口和批量接口必须保持相同的校验逻辑。每个入参 Point 都要经过 `ValidatePoints` 校验经纬度范围。

参考：`batchtransformcoordlogic.go`、`batchdistancelogic.go`。

### Model 层

```
app/gis/model/
├── gormmodel/fence.go    → GORM 模型定义
└── fencestore.go         → GormFenceStore 实现
```

模型主键约定：
- `gormx.LegacyIDMixin` — int64 自增主键（DB 性能）
- `FenceId string` uniqueIndex — 业务 UUID（API 暴露）

## 算法说明

### Geohash 网格扫描（GenerateFenceCells / computeGeohashCells）

算法：bbox 半步长网格扫描 + 双重过滤。

1. 计算多边形 bounding box
2. 以 `geohashCellSize / 2` 为步长遍历 bbox（半步长确保不遗漏边界格子）
3. 对每个采样点生成 geohash，构造格子矩形多边形
4. 精过滤：格子中心在围栏内 **或** 格子与围栏边界相交
5. 可选：扩展命中格子的 8 邻居

参考：`app/gis/internal/logic/generatefencecellslogic.go`。

### geohashCellSize 返回值约定

```go
func geohashCellSize(precision int, lat float64) (widthDeg, heightDeg float64)
```

- `widthDeg` = 经度方向跨度（lon step）
- `heightDeg` = 纬度方向跨度（lat step）
- 必须按 geohash 位划分公式精确计算角度跨度：`lonBits=(precision*5+1)/2`、`latBits=precision*5/2`、`widthDeg=360/2^lonBits`、`heightDeg=180/2^latBits`
- 不要用米制经验表再按纬度换算为度数；geohash 编码本身是在经纬度区间二分，格子角度跨度不依赖中心纬度

**调用方必须按 `lonStep, latStep := geohashCellSize(...)` 接收**，不要反写成 `latStep, lonStep`。

### Nearby geohash 查询

`FindNearbyFenceIds` 是 geohash 粗过滤，不是精确距离判断。查询时必须兼容入库 geohash 精度与本次查询精度不同的情况：

- 同精度：查询候选 geohash 和 8 邻居的 exact match
- 入库更粗：最多回退 2 级前缀 exact match（再粗则空间跨度过大，不适合作"附近"过滤）
- 入库更细：查询候选 geohash 的 `LIKE prefix%`

否则 `CreateFence` 默认 precision=7、`NearbyFences` 按 km 选择 precision=4/5/6 时会漏查候选围栏。

### 多边形相交判断（PolygonIntersect）

两阶段策略（`common/gisx/intersect.go`）：
1. 顶点包含：任一多边形的顶点在另一多边形内部（覆盖完全包含）
2. 边界相交：两外环的边存在线段交点（覆盖交叉穿越）

线段相交使用跨立实验（cross product straddle test），含退化情况（共线、端点接触）。

### 路径优化（RoutePoints）

近似求解开放式 TSP：
1. **最近邻贪心**（O(n²)）生成初始访问顺序
2. **2-opt 局部搜索** — 枚举所有可翻转子段，若缩短总距离则翻转

2-opt 边界规则：仅在 `j+1 < n` 时比较（开放路径末端无后继边）。

参考：`app/gis/internal/logic/routepointslogic.go`。

### H3 多边形覆盖（GenerateFenceH3Cells / CreateFence）

直接调用 `h3.PolygonToCellsExperimental` 计算所有与多边形重叠的六边形 cell。
需先通过 `gisx.OrbPolygonToH3GeoPolygon` 转换坐标格式。

## Proto 规范

- 字段命名统一 snake_case（`fence_id`, `h3_resolution`, `page_size`）
- 业务 ID 字段名为 `fence_id`（不用 `id`，避免与 DB 主键混淆）
- 精度/分辨率参数提供默认值说明：`uint32 h3_resolution = 3; // 默认 9`
- 时间字段用毫秒时间戳：`int64 created_at = 8;`
- 专有名词作为原子词：`geohashes`（不是 `geo_hashes`），`geohash_precision`

## 常见陷阱

| 陷阱 | 说明 | 参考文件 |
|------|------|----------|
| 坐标顺序混淆 | orb 用 [lon,lat]，H3 用 {lat,lng}，pb 用独立字段 | `logic/helper.go` |
| geohashCellSize 返回值 | 第一个是 widthDeg(lon)，第二个是 heightDeg(lat)，按 geohash 位数算角度跨度，不用米制经验表 | `logic/generatefencecellslogic.go` |
| Nearby geohash 精度不一致 | 查询必须兼容入库更粗/更细的 geohash cell（前缀最多回退 2 级），否则按 km 查询会漏候选 | `model/fencestore.go` |
| uint32 默认值判断 | `resolution == 0` 而非 `<= 0`（unsigned 不会负） | `logic/generatefenceh3cellslogic.go` |
| 批量接口漏校验 | BatchXxx 必须与单点版本保持相同的入参校验 | `logic/batchtransformcoordlogic.go` |
| 2-opt 开放路径边界 | 末端无后继边，`j+1 >= n` 时跳过 | `logic/routepointslogic.go` |
| polygon holes 死代码 | 当前 proto 只支持单环，OrbPolygonToH3GeoPolygon 的 holes 循环不会执行 | `common/gisx/gisx.go` |

## 单测覆盖

`common/gisx/gisx_test.go` 必须覆盖：
- `ValidateCoordinate` 边界值（±90/±180）
- `IsOrbPointsEqual` 浮点精度
- `OrbRingToH3LatLng` 自动闭合
- `OrbPolygonToH3GeoPolygon` 正常/带洞/异常
- `SegmentIntersect` 交叉/平行/共线/端点
- `RingIntersect` / `PolygonIntersect` 相交/包含/远离
