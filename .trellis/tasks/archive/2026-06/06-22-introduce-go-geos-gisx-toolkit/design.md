# 引入 go-geos 并完善 common/gisx GEOS 工具层

## 设计

### 1. 包结构与文件组织

`common/gisx` 新增 GEOS 工具文件，按 GEOS C API 能力分组，每个文件职责单一：

```
common/gisx/
├── intersect.go              # 现有纯 Go 相交判断（保留，不改动）
├── validate.go               # 现有坐标校验（保留）
├── gisx.go                   # 现有 H3 转换（保留）
├── store.go                  # 现有 FenceStore 接口（保留）
├── geos_context.go           # GEOS Context 管理 + 版本信息 + 安全执行封装
├── geos_construct.go         # 构造：Point/Ring/Polygon/Bounds 从 orb 类型
├── geos_convert.go           # 格式转换：WKT/WKB/GeoJSON 与 orb 互转
├── geos_predicate.go         # 基础谓词：Intersects/Contains/Covers/Within/...
├── geos_prepared.go          # PreparedPolygon + 谓词
├── geos_overlay.go           # Overlay：Intersection/Union/Difference/SymDifference
├── geos_valid.go             # IsValid/IsValidReason/MakeValid
├── geos_measure.go           # Area/Length/Distance/Centroid/PointOnSurface
├── geos_simplify.go          # Buffer/Simplify/ConvexHull/ConcaveHull
├── geos_strtree.go           # STRtree 空间索引
├── geos_test.go              # 测试（按能力分组子测试）
└── gisx_test.go              # 现有测试（保留）
```

### 2. Context 与生命周期

GEOS C API 以 `GEOSContextHandle_t` 为基础，`go-geos` 封装为 `*geos.Context`。工具层设计：

- 包级默认 Context：`defaultContext`，懒加载，`sync.Once` 保护。
- `withContext(c *geos.Context, fn func(*geos.Context) (T, error))` 泛型安全执行器，统一 `recover` panic → `error`。
- 短生命周期操作（一次谓词、一次构造）使用包级默认 Context。
- 长生命周期对象（`PreparedPolygon`、`STRtree`）持有自己的 `*geos.Context` 引用，避免被回收。

```go
var (
    defaultContext     *geos.Context
    defaultContextOnce sync.Once
)

func getDefaultContext() *geos.Context {
    defaultContextOnce.Do(func() {
        defaultContext = geos.NewContext()
    })
    return defaultContext
}

func safeRun[T any](fn func(*geos.Context) (T, error)) (result T, err error) {
    defer func() {
        if r := recover(); r != nil {
            err = fmt.Errorf("geos: %v", r)
            result = zero[T]()
        }
    }()
    return fn(getDefaultContext())
}
```

### 3. 坐标顺序约定

与 `gisx-guidelines.md` 一致：

| 来源 | 顺序 | GEOS 映射 |
|------|------|-----------|
| `orb.Point` | `[lon, lat]` | `x=lon, y=lat` |
| `orb.Ring` | `[]orb.Point{lon, lat}` | `[][]float64{{lon, lat}, ...}` |
| `orb.Polygon` | `[]orb.Ring` (外环 + 洞) | `[][][]float64` (外环 + 洞) |

构造函数内部做 orb → `[][]float64` 转换，不暴露 GEOS 坐标格式给业务。

### 4. 类型边界设计

工具层对外只暴露：

```go
// 几何结果类型（不暴露 *geos.Geom）
type Geometry struct {
    geom *geos.Geom
}

// Prepared Polygon
type PreparedPolygon struct {
    geom    *geos.Geom
    prep    *geos.PrepGeom
    context *geos.Context
}

// STRtree
type STRtree struct {
    tree    *geos.STRtree
    context *geos.Context
}
```

`Geometry` 提供 `Close()` 显式释放，也依赖 `runtime.AddCleanup`（go-geos 内部已做）。业务层通常不需要手动管理 `Geometry` 生命周期，因为谓词函数直接接受 `orb.Polygon`。

### 5. API 设计（按能力分组）

#### 5.1 版本与 Context（`geos_context.go`）

```go
func GEOSVersion() (major, minor, patch int)
func GEOSVersionString() string  // "3.12.1"
```

#### 5.2 构造（`geos_construct.go`）

```go
func NewPoint(p orb.Point) (*Geometry, error)
func NewLineString(ring orb.Ring) (*Geometry, error)
func NewLinearRing(ring orb.Ring) (*Geometry, error)
func NewPolygon(poly orb.Polygon) (*Geometry, error)
func NewBoundsRect(minX, minY, maxX, maxY float64) (*Geometry, error)
```

#### 5.3 格式转换（`geos_convert.go`）

```go
func FromWKT(wkt string) (*Geometry, error)
func (g *Geometry) ToWKT() (string, error)
func FromWKB(wkb []byte) (*Geometry, error)
func (g *Geometry) ToWKB() ([]byte, error)
func FromGeoJSON(geojson string) (*Geometry, error)
func (g *Geometry) ToGeoJSON(indent int) (string, error)

// orb 互转
func (g *Geometry) ToOrbPoint() (orb.Point, error)
func (g *Geometry) ToOrbRing() (orb.Ring, error)
func (g *Geometry) ToOrbPolygon() (orb.Polygon, error)
```

#### 5.4 基础谓词（`geos_predicate.go`）

便捷函数（直接接受 orb 类型）：

```go
func Intersects(a, b orb.Polygon) (bool, error)
func Contains(outer, inner orb.Polygon) (bool, error)
func Covers(outer, inner orb.Polygon) (bool, error)
func Within(inner, outer orb.Polygon) (bool, error)
func Touches(a, b orb.Polygon) (bool, error)
func Disjoint(a, b orb.Polygon) (bool, error)
func Equals(a, b orb.Polygon) (bool, error)
func Overlaps(a, b orb.Polygon) (bool, error)
func Crosses(a, b orb.Polygon) (bool, error)

// 点版本
func ContainsPoint(poly orb.Polygon, p orb.Point) (bool, error)
func CoversPoint(poly orb.Polygon, p orb.Point) (bool, error)
func IntersectsPoint(poly orb.Polygon, p orb.Point) (bool, error)
```

`Contains` vs `Covers` 语义：
- `Contains`：边界点不算包含（OGC 严格语义）
- `Covers`：边界点算包含（围栏命中场景用这个）

#### 5.5 Prepared Geometry（`geos_prepared.go`）

```go
func NewPreparedPolygon(poly orb.Polygon) (*PreparedPolygon, error)
func (p *PreparedPolygon) Intersects(other orb.Polygon) (bool, error)
func (p *PreparedPolygon) Contains(other orb.Polygon) (bool, error)
func (p *PreparedPolygon) ContainsPoint(pt orb.Point) (bool, error)
func (p *PreparedPolygon) Covers(other orb.Polygon) (bool, error)
func (p *PreparedPolygon) CoversPoint(pt orb.Point) (bool, error)
func (p *PreparedPolygon) Disjoint(other orb.Polygon) (bool, error)
func (p *PreparedPolygon) Close()
```

适用场景：一个围栏 polygon 对大量候选 geohash cell/center point 循环判定。

#### 5.6 Overlay（`geos_overlay.go`）

```go
func Intersection(a, b orb.Polygon) (orb.Polygon, error)
func Union(a, b orb.Polygon) (orb.Polygon, error)
func Difference(a, b orb.Polygon) (orb.Polygon, error)
func SymDifference(a, b orb.Polygon) (orb.Polygon, error)
```

返回 `orb.Polygon`，结果可能为空（返回空 polygon + nil error）。

#### 5.7 校验/修复（`geos_valid.go`）

```go
func IsValid(poly orb.Polygon) (bool, error)
func IsValidReason(poly orb.Polygon) (string, error)
func MakeValid(poly orb.Polygon) (orb.Polygon, error)
```

#### 5.8 简化/缓冲（`geos_simplify.go`）

```go
func Buffer(poly orb.Polygon, width float64, quadsegs int) (orb.Polygon, error)
func Simplify(poly orb.Polygon, tolerance float64) (orb.Polygon, error)
func TopologyPreserveSimplify(poly orb.Polygon, tolerance float64) (orb.Polygon, error)
func ConvexHull(poly orb.Polygon) (orb.Polygon, error)
func ConcaveHull(poly orb.Polygon, ratio float64, allowHoles bool) (orb.Polygon, error)
```

#### 5.9 测量（`geos_measure.go`）

```go
func Area(poly orb.Polygon) (float64, error)
func Length(poly orb.Polygon) (float64, error)
func Distance(a, b orb.Polygon) (float64, error)
func Centroid(poly orb.Polygon) (orb.Point, error)
func PointOnSurface(poly orb.Polygon) (orb.Point, error)
```

#### 5.10 STRtree（`geos_strtree.go`）

```go
type STRtree struct { ... }

func NewSTRtree(nodeCapacity int) *STRtree
func (t *STRtree) Insert(poly orb.Polygon, value interface{}) error
func (t *STRtree) Query(poly orb.Polygon) ([]interface{}, error)
func (t *STRtree) Close()
```

### 6. Dockerfile 变更

`app/gis/Dockerfile` 当前 builder 只装 `tzdata make gcc libtool musl-dev`，需要加 GEOS：

```dockerfile
# builder
RUN apk add --no-cache tzdata make gcc libtool musl-dev pkgconf geos-dev
RUN geos-config --version  # 构建日志输出 GEOS 版本

# runtime
FROM golang:1.26-alpine3.22
RUN apk add --no-cache geos
```

`geos-config --version` 输出到 build 日志，便于排查镜像 GEOS 版本与本地不一致问题。

### 7. 数据流

```
业务层 (app/gis/internal/logic)
  ↓ orb.Polygon / orb.Point
common/gisx (GEOS 工具层)
  ↓ safeRun → *geos.Context
github.com/twpayne/go-geos
  ↓ cgo GEOS*_r
libgeos_c.so (Docker: geos-dev build / geos runtime)
```

### 8. 错误处理策略

- `go-geos` 对无效几何或 API 误用会 `panic`，工具层 `safeRun` 统一 `recover` 转 `error`。
- 错误信息前缀 `geos: `，便于业务层识别来源。
- 解析错误（WKT/WKB/GeoJSON）本身返回 `error`，不会 panic，直接透传。
- `MakeValid` 可能返回 `MultiPolygon`（自相交修复后拆成多个），`ToOrbPolygon` 需要处理这种情况：若结果是 MultiPolygon，取所有子 polygon 合并为外环列表，或返回第一个 + warning。

### 9. 测试设计

`common/gisx/geos_test.go` 按能力分组：

```go
func TestGEOSVersion(t *testing.T)         // 版本非空
func TestConstruct(t *testing.T)           // Point/Ring/Polygon/Bounds
func TestWKTConvert(t *testing.T)          // WKT 往返
func TestWKBConvert(t *testing.T)          // WKB 往返
func TestGeoJSONConvert(t *testing.T)      // GeoJSON 往返
func TestPredicates(t *testing.T)          // 相交/包含/覆盖/远离/接触
func TestContainsVsCovers(t *testing.T)    // 边界点语义差异
func TestPreparedPolygon(t *testing.T)     // prepared 与普通结果一致
func TestOverlay(t *testing.T)             // intersection/union/difference
func TestValid(t *testing.T)               // IsValid/MakeValid 无效 polygon
func TestSimplify(t *testing.T)            // Buffer/Simplify/ConvexHull
func TestMeasure(t *testing.T)             // Area/Distance/Centroid
func TestSTRtree(t *testing.T)             // 插入/查询命中
func TestPanicRecover(t *testing.T)        // 无效输入不 panic
```

### 10. 兼容性

- 现有 `common/gisx/intersect.go` 的 `PolygonIntersect` 保留，不删除、不改动。
- 新增 GEOS 工具与现有纯 Go 工具共存，业务层可按需选择。
- `go.mod` 新增依赖不影响现有模块编译（`go build ./...` 验证）。

### 11. 风险与回滚

| 风险 | 缓解 |
|------|------|
| CGO 交叉编译复杂 | Dockerfile 已 `CGO_ENABLED=1`，Alpine `geos-dev` 提供 musl 兼容链接 |
| GEOS 版本差异 | `geos-config --version` 构建时输出；`GEOSVersion()` 运行时可查 |
| C heap 内存压力 | 短生命周期对象依赖 go-geos `runtime.AddCleanup`；长生命周期对象提供 `Close()` |
| panic 泄漏 | `safeRun` 统一 recover |
| MakeValid 返回 MultiPolygon | `ToOrbPolygon` 特殊处理，取子 polygon |
