# GIS 地理信息服务

`gis.rpc` 是地理信息服务（默认端口 25006），提供 GeoHash/H3 编码解码、电子围栏管理与命中判断、坐标转换、距离计算和路径优化能力。几何精确计算基于 GEOS（CGO），H3/GeoHash 用于索引与粗召回。

## 服务职责

- **GeoHash 编码/解码**：单点编码、多精度编码（1-12）、解码返回中心点与格网边界。
- **H3 编码/解码**：单分辨率编码、多分辨率编码（0-15）、解码返回中心点与边界多边形；`GridDisk` 按圈数获取周围 cells。
- **电子围栏**：围栏 CRUD（多边形支持洞），创建/更新时计算并持久化覆盖的 H3/geohash cells；命中判断同时支持上送 polygon 主动判断与上送 `fence_id` 从 store 查询判断。
- **空间计算**：点是否在半径内、附近围栏粗过滤、两点/批量球面距离（米）。
- **坐标转换**：WGS84 / GCJ02 / BD09 三种坐标系互转，支持批量。
- **路径优化**：计算点集合的近似最优访问路径（开放式 TSP），返回访问顺序与总距离。

## 配置

配置文件：`app/gis/etc/gis.yaml`。关键项：

| 配置项 | 说明 | 默认值 |
| --- | --- | --- |
| `ListenOn` | gRPC 监听地址 | `0.0.0.0:25006` |
| `Timeout` | 单次调用上限（毫秒） | `10000` |
| `Middlewares.StatConf.IgnoreContentMethods` | 不计入统计的方法（如 `PointsWithinRadius`） | 空 |
| `DB.DataSource` | 可选。配置后启用 GORM FenceStore 持久化围栏（Dev/Test 自动迁移 `GisFence`、`GisFenceCell` 表） | 空（纯计算模式） |
| `NacosConfig.IsRegister` | 是否注册到 Nacos（可选） | `false` |

未配置 `DB` 时使用 NoopFenceStore，围栏 CRUD 与 `fence_id` 查询判断不可用，其余纯计算能力不受影响。

## 关键能力

完整 RPC 定义见 [`app/gis/gis.proto`](../../app/gis/gis.proto)（`service Gis`），字段与校验以 proto 为权威。

| 分组 | RPC | 说明 |
| --- | --- | --- |
| GeoHash | `EncodeGeoHash` / `EncodeGeoHashMulti` / `DecodeGeoHash` | 编码、多精度编码、解码 |
| H3 | `EncodeH3` / `EncodeH3Multi` / `DecodeH3` / `GridDisk` / `GridDiskByPoint` | 编码、多分辨率编码、解码、邻域查询 |
| 围栏 cells 计算 | `GenerateFenceCells` / `GenerateFenceH3Cells` | 多边形覆盖的 geohash/H3 cells（纯计算，不持久化） |
| 命中判断 | `PointInFence` / `PointInFences` / `PointsWithinRadius` / `NearbyFences` | 围栏命中、半径内点、附近围栏粗过滤 |
| 距离 | `Distance` / `BatchDistance` | 两点球面距离（米） |
| 坐标转换 | `TransformCoord` / `BatchTransformCoord` | WGS84/GCJ02/BD09 互转 |
| 路径优化 | `RoutePoints` | 开放式 TSP 近似最优路径 |
| 围栏 CRUD | `CreateFence` / `UpdateFence` / `DeleteFence` / `ListFences` / `GetFence` | 围栏生命周期管理与查询 |

## 关键约定

- 领域坐标统一为 `longitude, latitude`（proto 中 `Point.lon` / `Point.lat`）。
- H3/geohash cells 仅用于粗召回，最终命中由多边形/圆的精确谓词判断。
- 围栏主体与 H3 cells 索引在同一事务写入，失败回滚并清理旧索引。
- H3 默认分辨率 9，geohash 默认精度 7（逻辑层默认值，proto 无默认）。

## 部署

- 依赖 GEOS C 库（CGO 编译）。Docker 部署在运行镜像中安装 `geos`，本地开发执行 `brew install geos pkg-config`；启动日志打印 GEOS 版本。
- 标准 go-zero zRPC 服务，启动方式：

```bash
./gis -f etc/gis.yaml
```

Dev/Test 模式注册 gRPC reflection。
- GEOS 封装层说明见 [`common/gisx/geos/README.md`](../../common/gisx/geos/README.md)。

## 权威契约

- RPC 契约：[`app/gis/gis.proto`](../../app/gis/gis.proto)
- 服务配置：`app/gis/etc/gis.yaml`
- 公共库：`common/gisx`（围栏 store 接口与几何适配）、`common/gisx/geos`（GEOS 封装）
