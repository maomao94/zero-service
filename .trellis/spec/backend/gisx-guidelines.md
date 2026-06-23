# GIS 服务与 gisx 包规范

> 涵盖 `common/gisx/` 通用包和 `app/gis/` 服务的架构约定。

## common/gisx/ 包边界

 | 可以放 | 不可以放 |
 |--------|----------|
 | 纯几何计算（坐标校验、线段相交、多边形相交） | 引用 `app/*/` 下的 pb 类型 |
 | orb/H3 类型转换（OrbPolygonToH3GeoPolygon） | 引用 gRPC generated 代码 |
 | GEOS 工具层（go-geos 封装，见下文 GEOS 工具层章节，位于 `common/gisx/geos` 子包） | 让 `app/gis/internal/logic` 直接依赖 `github.com/twpayne/go-geos` |
 | `FenceStore` 接口定义 + `NoopFenceStore` | 具体 store 实现（放 `app/gis/model/`） |
 | `FenceInfo` 通用结构体 | 业务错误码 (`extproto`) |

 参考文件：
  - `common/gisx/validate.go` — 坐标校验（参数顺序 lon, lat）
  - `common/gisx/gisx.go` — H3 转换、闭合检测/自动闭合（`IsRingClosed` / `EnsureRingClosed` / `EnsurePolygonClosed`）
  - `common/gisx/store.go` — FenceStore 接口定义
  - `common/gisx/doc.go` — 包级文档
  - `common/gisx/geos/doc.go` — GEOS 子包文档
  - `common/gisx/geos/errors.go` — 哨兵错误定义
  - `common/gisx/geos/extract.go` — 数据提取函数（泛型 ExtractMulti[T]）
  - `common/gisx/geos/*.go` — GEOS 工具层（独立子包 `geos`，零 orb 依赖）
  - `common/gisx/geos/orbconv/*.go` — orb 类型转换层

## 坐标系约定

**关键规则**：不同库的坐标参数顺序不同，混用是常见 bug 来源。

| 库/类型 | 顺序 | 示例 |
|---------|------|------|
| `orb.Point` | `[经度, 纬度]` (lon, lat) | `orb.Point{116.4, 39.9}` |
| `h3.LatLng` | `{纬度, 经度}` (lat, lng) | `h3.LatLng{Lat: 39.9, Lng: 116.4}` |
| `geohash.Encode` | `(纬度, 经度)` (lat, lon) | `geohash.EncodeWithPrecision(39.9, 116.4, 7)` |
| pb `Point` | `lat`, `lon` 独立字段 | `&gis.Point{Lat: 39.9, Lon: 116.4}` |

pb→orb 转换时必须翻转：`orb.Point{p.Lon, p.Lat}`。

## GEOS 工具层约定

`common/gisx/geos` 子包（包名 `geos`）薄封装 `github.com/twpayne/go-geos`。GEOS 是 CGO 动态库依赖，不是纯 Go 包。

### 架构

```
common/gisx/geos/               ← 零 orb 依赖，直接使用 *gogeos.Geom
├── context.go                  # GEOSVersion / safeRun / errNil
├── construct.go                # NewPoint/LineString/LinearRing/Polygon/Multi*/空几何
├── convert.go                  # WKT/WKB/GeoJSON 互转
├── extract.go                  # 7 数据提取函数 + PolygonData 类型 + 泛型 ExtractMulti[T]
├── predicate.go                # 11 谓词（Intersects/Contains/Covers/...）
├── prepared.go                 # PreparedGeom + 11 方法 + Close
├── overlay.go                  # Overlay/Valid/Measure/Simplify/Transform/Meta（33 函数）
├── relation.go                 # DE-9IM/Hausdorff/NearestPoints
├── introspect.go               # IsEmpty/Simple/Closed/Ring/HasZ
├── strtree.go                  # STRtree R-Tree
├── errors.go                   # 哨兵错误定义（ErrNil/ErrClosed/ErrNotPolygon/...）
├── doc.go                      # 包文档
├── API-GUIDE.md                # GEOS C 库原理与实战指南
├── geos_test.go                # 55+ 测试
└── README.md                   # API 完整参考

common/gisx/geos/orbconv/       ← orb 转换 + 便捷包装
├── orbconv.go                  # 14 转换 + 6 便捷谓词
└── orbconv_test.go             # 40+ 测试
```

### 核心原则

- **Context 私有**：`getDefaultContext()`（sync.Once 单例）包内私有，业务层不感知
- **panic → error**：所有 GEOS 调用经过 `safeRun`/`safeRunErr`，统一 recover
- **无包装层**：直接使用 `*gogeos.Geom`，不自建 `Geometry` 包装类型
- **无冗余字段**：`PreparedGeom`、`STRtree` 不存储 Context 引用
- **环形闭合：三层设计**：
  1. **orb 应用层**：`EnsureRingClosed`/`EnsurePolygonClosed` 自动闭合（精确 `==` 比较，首尾不同时追加首点），调用方应在传入 orbconv 前闭合
  2. **orbconv 转换层**：`ringToCoords()` 纯数据转换，不校验不修改闭合
  3. **GEOS C 层**：原生校验，首尾坐标不完全相同（差 1e-10 就 panic `Points of LinearRing do not form a closed linestring`），`safeRun` 捕获 panic 转 error

### 环形闭合跨层合约

| 层 | 职责 | 实现 |
|----|------|------|
| orb 应用层 (`gisx.go`) | **自动闭合**（精确 `==`，未闭合追加首点） | `IsRingClosed` / `EnsureRingClosed` / `EnsurePolygonClosed` |
| orbconv 转换层 | **纯数据转换**，不校验不修改闭合 | `ringToCoords()` → `[][]float64` |
| GEOS C 层 (`construct.go`) | **原生校验**，不闭合 panic → `safeRun` 捕获转 error | `NewLinearRing` / `NewPolygon` 直接透传坐标 |

**调用链**：`pbPointToOrbPolygon` → `EnsurePolygonClosed`（自动闭合）→ `PolygonToGeom` / `RingToGeom` → `ringToCoords`（纯转换）→ `NewLinearRing`（GEOS 校验）。

**关键规则**：GEOS 要求首尾精确相等（`==` 比较，不容差），`IsRingClosed`/`EnsureRingClosed` 统一使用精确 `==`，确保通过 GEOS 校验。

### 错误处理约定

`geos/errors.go` 定义 8 个导出哨兵错误，调用方可通过 `errors.Is` 区分：

| 错误 | 含义 |
|------|------|
| `ErrNil` | 几何对象为 nil |
| `ErrClosed` | 对象已关闭或未初始化 |
| `ErrNotPolygon` | 几何非 Polygon 类型 |
| `ErrEmptyRing` | 环坐标为空 |
| `ErrEmptyOuterRing` | 外环为空 |
| `ErrNotSupported` | 不支持的操作或类型 |
| `ErrEmptyGeoms` | 几何对象切片为空，无法构造集合 |
| `ErrEmptyResult` | 运算结果为空几何（如不相交的交集、退化几何的质心） |

所有 nil 输入统一返回 `ErrNil`。`Centroid`/`PointOnSurface` 运算结果为空时返回 `ErrEmptyResult`（区分"输入 nil"和"结果空"）。

### Docker / CGO 镜像构建契约

#### 1. Scope / Trigger
- 修改 `app/gis/Dockerfile`、`.dockerignore`、`app/gis/deploy.sh`
- 升级 `geos` 版本、切换 Go 版本、新增 C 依赖
- 增加多架构构建（amd64/arm64）
- 构建耗时异常（>60s 无缓存命中）

#### 2. 签名（关键文件路径）
| 文件 | 职责 |
|------|------|
| `app/gis/Dockerfile` | 两阶段构建（builder + runtime） |
| `.dockerignore`（仓库根） | 控制 Docker build context |
| `app/gis/deploy.sh` | 构建 → save → scp → load 部署链路 |

#### 3. 契约

**builder 阶段**
```dockerfile
# builder — 编译环境
FROM golang:1.26-alpine3.22 AS builder        # musl libc
ENV CGO_ENABLED=1                              # GEOS 必须
ARG GOPROXY=https://goproxy.cn,direct
ENV GOPROXY=$GOPROXY                           # 必须 ENV 导出，ARG 不持久

RUN apk add --no-cache \
    tzdata ca-certificates make gcc musl-dev pkgconf geos-dev \
    && ln -snf /usr/share/zoneinfo/$TZ /etc/localtime \  # apk add tzdata 之后
    && geos-config --version                              # 校验头文件

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download

COPY . .
ARG TARGETARCH  # BuildKit 内置：amd64 / arm64
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,id=go-build-${TARGETARCH},target=/root/.cache/go-build \
    go build -trimpath -ldflags="-s -w" -o /out/gis ./app/gis
```

**runtime 阶段**
```dockerfile
FROM alpine:3.22  # 与 builder 同 musl libc，~7MB
RUN apk add --no-cache geos  # 只装运行库
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=builder /usr/share/zoneinfo/Asia/Shanghai /usr/share/zoneinfo/Asia/Shanghai
```

**构建环境**
- `deploy.sh` 必须 `DOCKER_BUILDKIT=1`，否则 cache mount 不生效
- `.dockerignore` 必配，最少排除 `.git`、`logs`、`*.tar`、`env/`、`.trellis`、`.opencode`

**多架构**
```bash
# 单架构（deploy.sh 内置支持）: ./deploy.sh dev linux/arm64
# 多架构（需手动 buildx --push）:
docker buildx build --platform linux/amd64,linux/arm64 \
    -t registry.example.com/zero-service-gis:latest -f app/gis/Dockerfile --push .
```

#### 4. 校验矩阵
| 检查项 | 通过条件 | 检测方法 |
|--------|---------|---------|
| CGO 头文件 | `geos-config --version` 输出有效版本 | 构建日志 |
| GEOS 运行库 | 容器内 `ldd /app/gis \| grep geos` 有匹配 | `docker run --rm <img> ldd /app/gis` |
| 时区 | `date` 输出 CST | `docker run --rm <img> date` |
| CA 证书 | HTTPS 请求不报证书错误 | 服务启动后对外请求正常 |
| musl 兼容 | `ldd` 无 "not found" | `docker run --rm <img> ldd /app/gis` |
| BuildKit | `RUN --mount` 不报错 | 构建日志无 mount 错误 |
| GEOS 版本 | 启动日志输出版本 | `docker logs <container> \| grep "GEOS Version"` |

#### 5. Good / Base / Bad

**Good（当前实现）**
```dockerfile
# builder: golang:1.26-alpine3.22, CGO_ENABLED=1, geos-dev
# runtime: alpine:3.22, 只装 geos
# cache: id=go-build-${TARGETARCH} 多架构隔离
# binary: -trimpath -ldflags="-s -w"
```

**Base（可用但不优）**
```dockerfile
# builder 和 runtime 都用 golang:alpine（共享层，镜像大）
# cache mount 不用 id 区分架构（单架构可用，多架构冲突）
```

**Bad（反模式）**

| 反模式 | 后果 |
|--------|------|
| `ENV GOARCH=amd64` 写死 | ARM 构建失败 |
| runtime 用 `golang:1.26-alpine3.22` | 镜像膨胀（~350MB vs ~10MB） |
| runtime 用 `debian/ubuntu` | musl vs glibc 不兼容 |
| builder 只装 `geos` 没 `geos-dev` | 编译缺头文件 |
| runtime 装 `geos-dev` | 带入 gcc/musl-dev，增大攻击面 |
| `COPY --from=builder /build/gis`（路径含空格等） | 用 `/out/` 目录隔离，避免路径穿越 |
| `GOPROXY` 只设 `ARG` 不设 `ENV` | go 命令用不到代理 |
| 没 `.dockerignore` | `.git`（数百 MB）进 context |
| 没 `DOCKER_BUILDKIT=1` | cache mount 静默失败 |
| `ln -snf` 在 `apk add tzdata` 前 | 冷构建 zoneinfo 不存在 |
| runtime 重装 `tzdata ca-certificates` | 应用从 builder 复制，减少安装 |

#### 6. 测试验证

- [ ] 本地 `bash -n app/gis/deploy.sh` 语法检查
- [ ] `go build ./app/gis/... && go vet ./app/gis/...` 编译+静态检查
- [ ] `DOCKER_BUILDKIT=1 docker build -f app/gis/Dockerfile .` 构建成功
- [ ] `docker run --rm <img> sh -c "./gis -f etc/gis.yaml" 2>&1 | head -5` 输出 GEOS 版本
- [ ] `docker run --rm <img> ldd /app/gis | grep -i geos` GEOS 动态库可链接
- [ ] `docker run --rm <img> date` 输出 CST 时区
- [ ] 二次构建耗时 <30s（cache mount 命中）

#### 7. Wrong vs Correct

**Wrong — runtime 用 debian**
```dockerfile
FROM debian:bookworm-slim       # glibc
RUN apt-get install -y libgeos3 # 动态库路径不同
```
→ 二进制链接 musl，debian 无 musl，无法运行

**Correct — runtime 用 alpine**
```dockerfile
FROM alpine:3.22
RUN apk add --no-cache geos
```

**Wrong — 时区写在 apk add 前**
```dockerfile
ENV TZ=Asia/Shanghai
RUN ln -snf /usr/share/zoneinfo/$TZ /etc/localtime  # 文件还不存在
RUN apk add --no-cache tzdata
```

**Correct — 时区写在 apk add tzdata 后**
```dockerfile
RUN apk add --no-cache tzdata ... \
    && ln -snf /usr/share/zoneinfo/$TZ /etc/localtime
```

### 对外 API 边界

- 业务层推荐通过 `orbconv` 使用（接受 `orb` 类型）
- 纯坐标场景直接使用 `geos.NewPoint/NewPolygon` 等
- `app/gis/internal/logic` **不直接** import `github.com/twpayne/go-geos`
- `Close()` 置空 Go 引用帮助 GC，go-geos 通过 `runtime.AddCleanup` 管理 C 内存

### orbconv 转换矩阵

| GEOS 类型 | orb 类型 | GEOS→orb | orb→GEOS |
|-----------|----------|----------|----------|
| Point | `orb.Point` | `GeomToPoint` | `PointToGeom` |
| LineString | `orb.LineString` | `GeomToLineString` | `LineStringToGeom` |
| LinearRing | `orb.Ring` | `GeomToRing` | `RingToGeom` |
| Polygon | `orb.Polygon` | `GeomToPolygon` | `PolygonToGeom` |
| MultiPoint | `orb.MultiPoint` | `GeomToMultiPoint` | `MultiPointToGeom` |
| MultiLineString | `orb.MultiLineString` | `GeomToMultiLineString` | `MultiLineStringToGeom` |
| MultiPolygon | `orb.MultiPolygon` | `GeomToMultiPolygon` | `MultiPolygonToGeom` |

### extract 提取器（零 orb 依赖）

| 函数 | 输入 | 输出 |
|------|------|------|
| `ExtractPoint(g)` | Point | `(x,y float64, error)` |
| `ExtractCoords(g)` | LineString/Ring/Point | `[][]float64` |
| `ExtractPolygonCoords(g)` | Polygon | `PolygonData`（`[0]`=外环, `[1:]`=洞） |
| `ExtractPoints(g)` | Point/LineString/MultiPoint | `[][]float64` |
| `ExtractPolygonOrMultiCoords(g)` | Polygon/MultiPolygon | `[]PolygonData` |
| `ExtractMulti[T](g, fn)` | 任意 Multi* | `[]T`（泛型，拒绝 GeometryCollection） |
| `ExtractMultiSafe[T](g, fn)` | 含 GeometryCollection | `[]T`（跳过不匹配的子几何） |

### safeRun 机制

所有公开函数必须通过 `safeRun(func() (T, error))` 执行业务逻辑。go-geos 非法调用会 panic，`safeRun` 统一 recover 转 error，前缀 `geos: `。

构造函数（需 `*gogeos.Context`）在 `safeRun` 闭包内调用 `getDefaultContext()`：
```go
func NewPoint(x, y float64) (*gogeos.Geom, error) {
    return safeRun(func() (*gogeos.Geom, error) {
        return getDefaultContext().NewPointFromXY(x, y), nil
    })
}
```

### oneAttr 泛型辅助模式

`context.go` 提供泛型辅助函数 `oneAttr[T]`，统一处理单几何属性查询（替代原有的 `oneBool`/`oneFloat`/`oneInt`）：

```go
func oneAttr[T any](g *gogeos.Geom, fn func(*gogeos.Geom) T) (T, error) {
    if g == nil {
        var zero T
        return zero, errNil
    }
    return safeRun(func() (T, error) { return fn(g), nil })
}
```

用于 `IsEmpty`/`IsSimple`/`Area`/`Length`/`SRID` 等，自动处理 nil 和 panic 捕获。

### 谓词语义

- `Covers`/`IntersectsXY`（PreparedGeom）：**边界点算命中**，围栏场景用这个
- `Contains`/`ContainsXY`：OGC 严格语义，边界点不算
- `Intersects`：任意公共点（含边界接触）

### PreparedGeom 使用

```go
prep, _ := geos.NewPreparedGeom(fenceGeom)
defer prep.Close()
for _, pt := range points {
    hit, _ := prep.IntersectsXY(pt.Lon, pt.Lat)
}
```

单次判断用普通谓词，避免预处理开销。

### STRtree

go-geos 标注 STRtree "currently broken"（`Nearest` segfault）。本封装只暴露 Insert/Query/Iterate/Remove。查询只做空间过滤，业务层需精判。

### Context 并发模型

项目使用全局单例 Context（`sync.Once`）。go-geos 内部对每个 Context 有 `sync.Mutex`，同一 Context 下并发调用串行执行。高并发场景优先使用 Prepared Geometry（预处理索引在几何对象内部，不受 Context 锁影响）。

### 距离计算

GEOS `Distance` 是**平面笛卡尔距离**（单位=坐标单位）。经纬度场景结果无物理意义（度），应使用 `orb/geo.Distance`（Haversine 球面距离，米）。

`HausdorffDistance`/`FrechetDistance` 用于形状比较（不依赖球面）。

## FenceStore 接口模式

所有 store 方法必须传 `context.Context` 作为第一个参数。参数顺序与全项目坐标约定 `{lon, lat}` 一致：

```go
type FenceStore interface {
    CreateFence(ctx context.Context, fenceId, name string, polygon orb.Polygon, h3Resolution, geohashPrecision int, h3Cells, geohashCells []string) error
    LoadFencePolygon(ctx context.Context, fenceId string) (orb.Polygon, error)
    FindNearbyFenceIds(ctx context.Context, lon, lat, km float64) ([]string, error)
    // ...
}
```

`FenceInfo.Polygon` 类型为 `orb.Polygon`（`[]orb.Ring`），`polygon[0]`=外环，`polygon[1:]`=洞。围栏多边形支持外环 + 多个洞。

### 召回索引自动生成规则

store 层实现（`GormFenceStore`）在 `CreateFence` / `UpdateFence` 内部自动做以下工作：

- 根据 `points` 和 `h3RecallResolution=9` 生成 `h3_r9` 召回 cells
- 与业务 `h3` / `geohash` cells 一起写入 `gis_fence_cell` 表
- `h3_r9` cells 不暴露给 logic 层，不出现在 `FenceInfo.H3Cells` 中
- `FindNearbyFenceIds` 固定查询 `cell_type = "h3_r9"`，不和 `h3` 混查

注入规则（`app/gis/internal/svc/servicecontext.go`）：
- 配置了 `DB.DataSource` → 使用 `model.NewGormFenceStore(db)`
- 未配置 → 使用 `&gisx.NoopFenceStore{}`

## app/gis/ 服务架构

### RPC 分类

 | 类别 | 示例 | 特点 |
 |------|------|------|
 | 纯计算 | `Distance`, `EncodeH3`, `EncodeGeoHash`, `GenerateFenceCells`, `RoutePoints` | 无 DB 依赖，无状态 |
 | 多精度编码 | `EncodeH3Multi`, `EncodeGeoHashMulti` | 单点 multi-resolution 编码，repeated 入参 |
 | GridDisk 邻域查询 | `GridDisk` (h3_index), `GridDiskByPoint` (经纬度) | 两个独立 RPC 分别处理 origin 输入形式 |
 | 围栏 CRUD | `CreateFence`, `UpdateFence`, `DeleteFence`, `ListFences`, `GetFence` | 依赖 FenceStore |
 | 围栏判断 | `PointInFence`, `PointInFences`, `NearbyFences` | 支持两种模式：上送 points 主动判断，或上送 fence_id 从 store 加载判断；`PointInFences` 命中但 fence_id 为空时不加入结果；`NearbyFences` 使用 H3 召回 + polygon 精判，store 错误直接返回 |
 | 坐标转换 | `TransformCoord`, `BatchTransformCoord` | WGS84/GCJ02/BD09 互转 |

### logic 层 helper 模式

`logic/helper.go` 存放 pb↔领域类型的转换和通用校验：

**导出函数（供外部 RPC 调用方使用）：**
- `ValidatePoints` — pb Point 批量校验（非空、非 nil、经纬度范围）
- `ValidateH3Resolution` — 校验 H3 分辨率 0-15，返回 int。**不设默认值**，传入 0 合法返回 0
- `ValidateGeoHashPrecision` — 校验 geohash 精度 1-12，返回 int
- `EncodeH3Cell` — 将 pb Point 编码为 H3 cell

**内部辅助函数（供 logic 内复用，消除重复校验/计算）：**
- `resolveH3Resolution` — 校验 H3 分辨率，0 时默认 9。注释注明 proto3 零值歧义
- `resolveGeohashPrecision` — 校验 geohash 精度，0 时默认 7
- `computeFenceCells` — 计算多边形覆盖的 H3 cells + geohash cells（CreateFence/UpdateFence 共用）
- `scanGeohashCells` — geohash bbox 扫描核心算法（GenerateFenceCells 和 computeGeohashCells 共用）。支持 `includeNeighbors` 控制邻居扩展，GEOS 错误返回 error
- `computeGeohashCells` — 调用 `scanGeohashCells`，不含邻居扩展。**GEOS 错误静默返回 nil**（向后兼容）
- `geohashCellSize` — 按 geohash 位划分计算格子经纬度跨度
- `validateCoordType` — 校验坐标系类型 1-3（TransformCoord 和 BatchTransformCoord 共用）
- `pbPointToOrbPolygon` — pb Point 切片→orb.Polygon（含点数校验 + 坐标校验 + 自动闭合）

**反模式：**
- 不要在 logic 文件内联解析/校验 H3 分辨率或 geohash 精度；应使用 `resolveH3Resolution` / `resolveGeohashPrecision`
- 不要让创建/更新围栏重复写 H3 cells + geohash cells 计算；使用 `computeFenceCells`
- 不要在 inline 做 geohash bbox 扫描；调用 `scanGeohashCells`
- 不要把 pb 类型的转换函数放到 `common/gisx/`

### 批量接口校验规则

单点接口和批量接口必须保持相同的校验逻辑。每个入参 Point 都要经过 `ValidatePoints` 校验经纬度范围。批量坐标转换的 `source_type`/`target_type` 校验与单点版本一致，使用共享的 `validateCoordType`。

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

> GEOS 工具层 API 和约定见上方 [GEOS 工具层约定](#geos-工具层约定) 章节。以下为 geohash/H3 算法说明。

### Geohash 网格扫描（scanGeohashCells / computeGeohashCells / GenerateFenceCells）

核心算法集中在 `helper.go` 的 `scanGeohashCells`，`computeGeohashCells`（不带邻居）和 `GenerateFenceCells`（支持 `includeNeighbors`）均委托它。

算法：bbox 半步长网格扫描 + 双重过滤。本接口为纯计算，不再支持按 `fence_id` 从 store 加载围栏。

1. 计算多边形 bounding box
2. 以 `geohashCellSize / 2` 为步长遍历 bbox（半步长确保不遗漏边界格子）
3. 对每个采样点生成 geohash，构造格子矩形多边形
4. 精过滤：格子中心在围栏内 **或** 格子与围栏边界相交（相交判断已迁移到 `orbconv.IntersectsOrb`）
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

### H3 召回索引（推荐，替代 Nearby geohash 查询）

`FindNearbyFenceIds` 使用固定的 `h3_r9` 召回索引做粗过滤。设计要点：

- 召回索引精度固定为 **H3 resolution = 9**（`cell_type = "h3_r9"`），不按 `km` 动态选择。
- `CreateFence` / `UpdateFence` 在 store 层根据多边形自动生成 `h3_r9` 召回 cells，logic 层不感知内部召回索引。
- 查询链路：`LatLngToCell(point, 9)` → `GridDisk(origin, k)` → `cell_type = "h3_r9" AND cell_id IN ?`。
- `km` 只换算 H3 网格圈数 `k`：`k = ceil(km / 0.2)`（基于 res=9 平均边长约 200m），最小 1。
- `FenceInfo.H3Cells` 业务上暴露 `cell_type = "h3"`，不暴露内部 `h3_r9`。
- 未来升级召回精度时，新增 `cell_type = "h3_r10"` 重建索引，再切换查询条件；不改主表结构。

参考文件：`app/gis/model/fencestore.go`，常量定义：

```go
const (
    h3RecallResolution    = 9
    h3RecallCellType      = "h3_r9"
    h3RecallAverageEdgeKm = 0.2  // res=9 平均边长 km
)
```

优势：
- 查询 resolution 和入库 resolution 始终对齐，不会因精度不一致漏查候选。
- 按 km 变 resolution 的老方案不准确且不可靠，已在本轮废弃。

### 老方案（已废弃）：Nearby geohash 查询

> 该方案已被 H3 召回索引替代，旧代码保留但不应在新路径中使用。

`FindNearbyFenceIds` 曾使用 geohash 粗过滤，查询时必须兼容入库 geohash 精度与本次查询精度不同的情况：

- 同精度：查询候选 geohash 和 8 邻居的 exact match
- 入库更粗：最多回退 2 级前缀 exact match（再粗则空间跨度过大，不适合作"附近"过滤）
- 入库更细：查询候选 geohash 的 `LIKE prefix%`

否则 `CreateFence` 默认 precision=7、`NearbyFences` 按 km 选择 precision=4/5/6 时会漏查候选围栏。

### NearbyFences 完整链路（H3 召回 + 多边形精判）

```mermaid
graph LR
  A[point + km] --> B[H3 LatLngToCell res=9]
  B --> C[GridDisk origin_k]
  C --> D[查 cell_type=h3_r9 AND cell_id IN ...]
  D --> E[候选 fence_id 列表]
  E --> F[LoadFencePolygon 加载各候选多边形]
  F --> G[planar.PolygonContains 精判]
  G --> H[返回真正命中的 fence_id 列表]
```

注意：
- `km` 只控制 H3 召回圈的广度，不控制精判后的结果范围。
- 如果围栏入库时没有生成 `h3_r9` cells（例如为回填的老数据），`NearbyFences` 查不到该围栏。
- store 错误直接返回给调用方（不再静默返回空结果），调用方可区分"附近无围栏"和"查询失败"。

### GridDisk 圈层查询

两个独立 RPC 分别处理不同的 origin 输入形式，不在单个 request 中塞多个主输入字段：

```proto
message GridDiskReq {
  string h3_index = 1;     // H3 origin index
  uint32 k = 2;            // 周围圈数，默认 1；0 表示只返回 origin
}

message GridDiskByPointReq {
  Point point = 1;
  uint32 resolution = 2;   // H3 分辨率 0-15，默认 9
  uint32 k = 3;            // 周围圈数，默认 1；0 表示只返回 origin
}

message GridDiskRes {
  string origin = 1;
  repeated GridDiskCell cells = 2;
}

message GridDiskCell {
  string h3_index = 1;
  uint32 ring = 2;         // H3 圈数：0=origin，1=第一圈，依此类推；不是米级距离
}
```

响应字段用 `ring` 而非 `distance`，H3 网格圈数不是米级距离，调用方不会误解。

k=0 传透到 `GridDiskDistances(origin, 0)` 只返回 origin；不做 `if k <= 0 { k = 1 }` 覆盖。

### 多精度编码（EncodeH3Multi / EncodeGeoHashMulti）

单点 multi-resolution 编码模式：

```proto
message EncodeH3MultiReq {
  Point point = 1;
  repeated uint32 resolutions = 2; // H3 分辨率 0-15，必填
}

message EncodeH3MultiRes {
  repeated H3Index h3_indexes = 1; // 按 resolutions 顺序对齐
}
```

- `resolutions` / `precisions` 必填，为空时返回参数缺失错误
- 返回顺序与请求顺序保持一致，不去重
- 响应结构复用 `H3Index` / `GeoHashIndex`（resolution + value）

### PointsWithinRadius 响应精简

```proto
message RadiusHit {
  int32 index = 1;
  double distance_meters = 2;
}

message PointsWithinRadiusRes {
  repeated RadiusHit hits = 1;
}
```

不再返回 Point 坐标，避免响应体膨胀。用 orb `geo.Distance` 算精确球面距离。

### 路径优化（RoutePoints）

近似求解开放式 TSP：
1. **最近邻贪心**（O(n²)）生成初始访问顺序
2. **2-opt 局部搜索** — 枚举所有可翻转子段，若缩短总距离则翻转

2-opt 边界规则：仅在 `j+1 < n` 时比较（开放路径末端无后继边）。

参考：`app/gis/internal/logic/routepointslogic.go`。

### H3 多边形覆盖（GenerateFenceH3Cells / CreateFence）

直接调用 `h3.PolygonToCellsExperimental` 计算所有与多边形重叠的六边形 cell。
需先通过 `gisx.OrbPolygonToH3GeoPolygon` 转换坐标格式。
`GenerateFenceH3Cells` 为纯计算接口，不支持按 `fence_id` 从 store 加载围栏。

## Proto 规范

- 字段命名统一 snake_case（`fence_id`, `h3_resolution`, `page_size`）
- 业务 ID 字段名为 `fence_id`（不用 `id`，避免与 DB 主键混淆）
- 精度/分辨率参数提供默认值说明：`uint32 h3_resolution = 3; // 默认 9`
- 时间字段用毫秒时间戳：`int64 created_at = 8;`
- 专有名词作为原子词：`geohashes`（不是 `geo_hashes`），`geohash_precision`
- Circle 语义不叫 `distance`：H3 GridDisk 返回的层数用 `ring` 表达，`distance` 只用于米级球面距离（`distance_meters`）

## 常见陷阱

| 陷阱 | 说明 | 参考文件 |
|------|------|----------|
| 坐标顺序混淆 | orb 用 [lon,lat]，H3 用 {lat,lng}，pb 用独立字段 | `logic/helper.go` |
| geohashCellSize 返回值 | 第一个是 widthDeg(lon)，第二个是 heightDeg(lat)，按 geohash 位数算角度跨度，不用米制经验表 | `logic/helper.go` |
| H3 召回精度不一致 | 查询 resolution 必须和入库 `h3_r9` 一致，否则 `GridDisk` 查不到对应 cells；已按固定 res=9 消除了此问题 | `model/fencestore.go` |
| 老数据缺少 h3_r9 cells | 未回填的旧围栏没有 `h3_r9` 行，`NearbyFences` 查不到；旧代码保留的 geohash 查询也不应该再被用作唯一召回路径 | `model/fencestore.go` |
| km 只控制候选广度 | `NearbyFences` 的 `km` 只影响 H3 候选集大小，不决定最终结果；polygon 精判后只返回真正命中的围栏 | `logic/nearbyfenceslogic.go` |
| 批量接口漏校验 | BatchXxx 必须与单点版本保持相同的入参校验，包括 coord_type 等 | `logic/batchtransformcoordlogic.go` |
| 2-opt 开放路径边界 | 末端无后继边，`j+1 >= n` 时跳过 | `logic/routepointslogic.go` |
| polygon holes | `FenceInfo.Polygon` 使用 `orb.Polygon`（`polygon[0]`=外环, `polygon[1:]`=洞），`OrbPolygonToH3GeoPolygon` 的 holes 循环对洞生效 | `common/gisx/store.go`, `common/gisx/gisx.go` |
| `h3.CellFromString` 无 error | `CellFromString` 返回单值（非 `Cell, error`），需用 `.IsValid()` 检查无效 index | `logic/griddisklogic.go` |
| `h3.GridDiskDistances` 返回 `[][]Cell` | 返回值是 `[][]Cell`（按环分层），不是 `[]DistanceEntry`。用 `ringNum, ringCells := range result` 遍历 | `logic/griddisklogic.go` |
| `resolutions` 必填不默认 | `EncodeH3Multi` 的 `resolutions` 空时返回参数错误，不设默认值；单精度由 `EncodeH3` 承担 | `logic/encodeh3multilogic.go` |
| **GEOS 闭合三层设计** | orb 应用层 `EnsureRingClosed`/`EnsurePolygonClosed` 自动闭合（精确 ==），orbconv 层 `ringToCoords` 纯转换不校验，GEOS 层原生校验不闭合就 panic。调用方应确保传入前已闭合 | `common/gisx/gisx.go`, `common/gisx/geos/orbconv/orbconv.go` |
| JSON 反序列化错误 | `ListFences` 遇到坏 JSON 仍保留记录（空多边形 + 记日志），`GetFence` 直接返回 error；不可静默丢弃两条链路行为不一致 | `model/fencestore.go` |
| polygon 校验重复 | `pbPointToOrbPolygon` 已含点数 + 坐标双重校验，调用方不应再单独做 `len(points) < 3` 或 `ValidatePoints` | `logic/helper.go` |
| H3 resolution / geohash precision 内联 | 不要在 logic 文件内联默认值和上限校验，使用 `resolveH3Resolution` / `resolveGeohashPrecision` | `logic/helper.go` |

## 单测覆盖

`common/gisx/gisx_test.go` 必须覆盖：
- `ValidateCoordinate` 边界值（±90/±180）
- `IsRingClosed` 精确闭合语义（闭合/未闭合/3点/微小偏差不闭合/空）
- `EnsureRingClosed` 自动追加 + 不修改入参 + error（<3点/空/3点追加）
- `EnsurePolygonClosed` 外环未闭合+洞已闭合/全未闭合 + error（空/洞不足3点）
- `OrbRingToH3LatLng` 自动闭合 + 不修改入参 + 坐标转换验证（lon→lng, lat→lat）+ error（空/<3点）
- `H3LatLngsToOrbRing` 反向转换（纬度在前 → orb.Ring 经度在前）
- `H3LatLngsToOrbPolygon` 将 H3 GeoPolygon（含洞）转为 orb.Polygon
- `OrbPolygonToH3GeoPolygon` 正常/带洞/异常 + 坐标转换验证 + 无效洞静默跳过
- `ValidationError` `errors.As` 断言
- `NoopFenceStore` 全部 8 个方法 + `ErrFenceStoreNotImplemented`
- `FenceInfo.Polygon` 带洞结构

`common/gisx/geos/geos_test.go` 必须覆盖：
- GEOSVersion 非空
- 构造：Point/Polygon/BoundsRect/LineString/LinearRing + 4 个 NewEmpty*
- WKT/WKB/GeoJSON 往返 + 无效输入
- 全部 11 谓词 + Contains vs Covers 边界语义
- PreparedGeom 全部 11 方法 + Close
- Overlay（Intersection/Union/Difference/SymDifference）面积校验
- Valid/IsValidReason/MakeValid（bowtie 自相交场景）
- Buffer/Simplify/TopologyPreserveSimplify/ConvexHull/ConcaveHull
- Area/Length/Distance/Centroid/PointOnSurface 精度校验
- BuildArea/LineMerge/Node/OffsetCurve/EndPoint/StartPoint
- Relate/RelatePattern/Hausdorff/Frechet/DistanceWithin/NearestPoints
- SRID/SetSRID/Precision/Normalize/Reverse/MinimumClearance 基本功能
- STRtree Insert/Query/Iterate/Remove
- safeRun panic recover（无效 WKT→error）
- 全包 nil 输入错误返回（construct/predicate/overlay/measure/transform/valid/extract/introspect/relation）
- 哨兵错误 `errors.Is` 验证（ErrNil/ErrClosed/ErrEmptyRing 等 8 个）

`common/gisx/geos/orbconv/orbconv_test.go` 必须覆盖：
- 全部 7 个双向转换往返（Point/LineString/Ring/Polygon/MultiPolygon/MultiPoint/MultiLineString）
- IntersectsOrb/ContainsOrb/CoversOrb 重叠+远离
- CoversPointOrb/ContainsPointOrb 边界语义
- ValidOrb 有效+无效多边形
- nil 输入返回 error
- 不支持类型错误路径
- 便捷包装器 nil polygon 错误路径
