# geos — GEOS 几何引擎 Go 封装

基于 [go-geos v0.20.4](https://github.com/twpayne/go-geos) 的 Go 封装包，向上暴露 ~66 个函数，覆盖 GEOS C 库 80%+ 可实施 API。

## 设计原则

| 原则 | 说明 |
|------|------|
| **零 orb 依赖** | `geos` 包直接使用 `*gogeos.Geom`，不依赖 `orb` |
| **orb 转换独立** | `geos/orbconv` 子包负责 `orb` ↔ GEOS 互转 |
| **panic → error** | 统一通过 `safeRun`/`safeRunErr` 捕获 panic 并返回 error |
| **Context 私有** | `getDefaultContext()`（sync.Once 单例）包内私有，不对外暴露 |

## 包结构

```
common/gisx/geos/               ← 纯 GEOS 封装（零 orb 依赖）
├── context.go                  # GEOSVersion / safeRun panic recover
├── construct.go                # 5 几何构造（Point/Ring/Polygon/...）
├── convert.go                  # 6 格式互转（WKT/WKB/GeoJSON）
├── predicate.go                # 11 空间谓词
├── prepared.go                 # PreparedGeom + 12 加速谓词
├── overlay.go                  # Overlay / Valid / Measure / Simplify / Transform / Meta
├── relation.go                 # DE-9IM / 距离 / 最近点
├── introspect.go               # 6 几何内省（空/简/闭/环/Z）
├── strtree.go                  # STRtree R-Tree 空间索引
├── geos_test.go                # 45+ 测试用例
└── README.md

common/gisx/geos/orbconv/       ← orb 转换 + 便捷包装
├── orbconv.go                  # 6 转换 + 6 便捷谓词
└── orbconv_test.go             # 12+ 测试用例
```

## 快速开始

### 安装

```bash
# macOS
brew install geos pkg-config

# 验证 GEOS 版本
pkg-config --modversion geos
```

### 使用 orb 类型（推荐业务入口）

```go
import "zero-service/common/gisx/geos/orbconv"

fence := orb.Polygon{orb.Ring{{116.3, 39.9}, {116.4, 39.9}, ...}}

// 点命中（边界算）
hit, _ := orbconv.CoversPointOrb(fence, orb.Point{116.35, 39.95})

// 两围栏相交
ok, _ := orbconv.IntersectsOrb(fenceA, fenceB)
```

### 使用纯坐标

```go
import geos "zero-service/common/gisx/geos"

p, _ := geos.NewPolygon([][][]float64{{{0,0},{4,0},{4,4},{0,4},{0,0}}})
hit, _ := geos.Covers(p, geos.NewPoint(2, 2))
```

### Prepared Geometry（围栏固定 vs 大量候选）

```go
prep, _ := geos.NewPreparedGeom(fenceGeom)
defer prep.Close()
for _, cell := range cells {
    hit, _ := prep.IntersectsXY(cell.Lon, cell.Lat)
    // ...
}
```

---

## API 完整参考

### 版本信息

| 函数 | 功能 | 返回值 |
|------|------|--------|
| `GEOSVersion()` | 获取底层 GEOS C 库版本号 | `(major, minor, patch int)` |
| `GEOSVersionString()` | 获取版本字符串 | `"3.14.1"` |

### 几何构造

坐标约定：`{x, y}` = `{经度, 纬度}`，与 `orb.Point [lon, lat]` 对齐。

| 函数 | 功能 | 说明 |
|------|------|------|
| `NewPoint(x, y float64)` | 创建点 | 返回 `*gogeos.Geom`（Point 类型） |
| `NewLineString(coords [][]float64)` | 创建线 | 不要求闭合，至少 2 点 |
| `NewLinearRing(coords [][]float64)` | 创建环 | 自动闭合首尾，GEOS 要求闭合环 |
| `NewPolygon(coordss [][][]float64)` | 创建多边形 | `coordss[0]`=外环，`coordss[1:]`=洞 |
| `NewBoundsRect(minX, minY, maxX, maxY)` | 创建矩形 | 从 bbox 构造 Polygon |

### 格式转换

解析错误直接返回 `error`，不 panic。

| 函数 | 功能 | 说明 |
|------|------|------|
| `FromWKT(wkt string)` | WKT 文本 → 几何 | 如 `"POLYGON ((0 0, 4 0, 4 4, 0 4, 0 0))"` |
| `ToWKT(g)` | 几何 → WKT 文本 | |
| `FromWKB(wkb []byte)` | WKB 二进制 → 几何 | PostGIS 常用格式 |
| `ToWKB(g)` | 几何 → WKB 二进制 | |
| `FromGeoJSON(geojson string)` | GeoJSON 文本 → 几何 | |
| `ToGeoJSON(g, indent)` | 几何 → GeoJSON 文本 | indent=0 紧凑输出 |

### 空间谓词

所有谓词返回 `(bool, error)`。核心语义见下表。

| 函数 | 语义 | 边界点 | 围栏场景 |
|------|------|--------|----------|
| `Intersects(a, b)` | 两几何有任意公共点 | ✓ 算交集 | 两围栏是否重叠 |
| `Disjoint(a, b)` | 两几何完全无公共点 | ✗ 不相交 | 排除远离围栏 |
| **`Covers(a, b)`** | a 覆盖 b（b 所有点在 a 内或边上） | **✓ 算命中** | **围栏命中用这个** |
| `Contains(a, b)` | a 严格包含 b（b 所有点在 a 内部） | ✗ 不算 | 不推荐围栏命中 |
| `CoveredBy(a, b)` | a 被 b 覆盖（Covers 反向） | ✓ 算 | 自身是否在参照内 |
| `Within(a, b)` | a 在 b 内部（Contains 反向） | ✗ 不算 | |
| `Touches(a, b)` | 仅边界接触，内部无交集 | ✓ 接触 | |
| `Overlaps(a, b)` | 部分重叠（同维度交集） | ✓ 重叠 | |
| `Crosses(a, b)` | 穿越（不同维度交集） | ✓ 穿越 | 点多边形=交集 |
| `Equals(a, b)` | 拓扑相等（形状同） | - | 比较两个围栏是否等价 |
| `EqualsExact(a, b, tol)` | 坐标精确相等（±tolerance） | - | |

**关键语义**：`Covers(0,0)` 对边界点 `(0,0)` 返回 `true`，`Contains` 返回 `false`。围栏命中必须用 `Covers`。

### Prepared Geometry（加速谓词）

适用场景：**一个固定多边形对大量候选做循环判定**。预处理后谓词调用显著快于普通谓词。

`NewPreparedGeom(g)` 返回 `*PreparedGeom`。

| 方法 | 功能 | 等价普通谓词 |
|------|------|-------------|
| `.Intersects(other)` | 是否有交集 | Intersects |
| `.Contains(other)` | 严格包含 | Contains |
| `.ContainsXY(x, y float64)` | 严格包含点 | Contains + Point |
| `.Covers(other)` | 覆盖（边界算） | Covers |
| `.IntersectsXY(x, y float64)` | 点交集（等价 CoversPoint） | Covers + Point |
| `.Disjoint(other)` | 完全无交集 | Disjoint |
| `.CoveredBy(other)` | 被覆盖 | CoveredBy |
| `.Overlaps(other)` | 部分重叠 | Overlaps |
| `.Touches(other)` | 仅边界接触 | Touches |
| `.Within(other)` | 在内部 | Within |
| `.DistanceWithin(other, dist)` | 距离是否 ≤ dist | DistanceWithin |
| `.Close()` | 释放引用 | |

注意：`IntersectsXY` 等价于 `CoversPoint`（因 PrepGeom 无 `CoversXY`，语义相同——点落在几何上或内部）。

### Overlay 运算

返回值可能为空（`IsEmpty()==true`），不会返回 `nil` error。

| 函数 | 功能 | 说明 |
|------|------|------|
| `Intersection(a, b)` | 交集 | 公共区域 |
| `Union(a, b)` | 并集 | 合并区域 |
| `Difference(a, b)` | a - b 差集 | a 扣除与 b 重叠部分 |
| `SymDifference(a, b)` | 对称差集 | 并集 - 交集 |
| `UnaryUnion(g)` | 单几何自合并 | 消除自重叠内部边 |

### 校验

| 函数 | 功能 | 说明 |
|------|------|------|
| `IsValid(g)` | 是否有效几何 | 检查自相交、环方向等 |
| `IsValidReason(g)` | 无效原因 | 有效时返回空字符串 |
| `MakeValid(g)` | 修复无效几何 | 自相交可能拆为 MultiPolygon→只取第一个 |

### 简化 & 缓冲

| 函数 | 功能 | 说明 |
|------|------|------|
| `Buffer(g, width, quadsegs)` | 缓冲区 | width>0 外扩，<0 内缩；quadsegs 控制圆角精度 |
| `Simplify(g, tolerance)` | Douglas-Peucker 简化 | 可能产生无效拓扑 |
| `TopologyPreserveSimplify(g, tol)` | 拓扑保持简化 | 保证不产生新自相交 |
| `ConvexHull(g)` | 凸包 | 最小凸多边形包含所有点 |
| `ConcaveHull(g, ratio, allowHoles)` | 凹包 | ratio 0~1，越小越凹 |

### 测量（平面坐标距离）

| 函数 | 功能 | 说明 |
|------|------|------|
| `Area(g)` | 面积 | 平面坐标面积（度² 或 米²，取决于输入单位） |
| `Length(g)` | 周长/长度 | 平面坐标长度 |
| `Distance(a, b)` | 最小平面距离 | 平面欧几里得距离，**经纬度场景结果=度，无物理意义** |
| `Centroid(g)` | 质心 | 返回 `(x, y, error)`，凹多边形可能在外部 |
| `PointOnSurface(g)` | 表面点 | 返回 `(x, y, error)`，保证在多边形上 |
| `HausdorffDistance(a, b)` | 豪斯多夫距离 | 两个形状的最大偏差 |
| `FrechetDistance(a, b)` | 弗雷歇距离 | 曲线相似度 |
| `MinimumClearance(g)` | 最窄宽度 | 多边形瓶颈处最小宽度 |

### 变换

| 函数 | 功能 | 说明 |
|------|------|------|
| `Normalize(g)` | 规范化 | 环按统一规则排序 |
| `Reverse(g)` | 反转环方向 | 顺时针↔逆时针 |
| `Snap(a, b, tolerance)` | 顶点吸附 | a 顶点吸附到 b 顶点（容差内） |
| `Densify(g, tolerance)` | 密化 | 以最大间距插入顶点 |
| `ClipByRect(g, minX, minY, maxX, maxY)` | 矩形裁剪 | 裁剪到 bbox |
| `Envelope(g)` | 外包盒多边形 | 返回 bbox 的 Polygon |
| `Boundary(g)` | 边界几何 | Point→空，Line→端点，Polygon→环 |
| `BuildArea(g)` | 线→面 | 从线集合构造面 |
| `LineMerge(g)` | 合并线段 | 合并相连线段 |
| `Node(g)` | noding | 所有边交点被分割 |
| `MinimumRotatedRectangle(g)` | 最小外接旋转矩形 | 可旋转的定向外接矩形 |
| `OffsetCurve(g, width, quadsegs)` | 偏移曲线 | 线平移 |
| `EndPoint(g)` | 线终点 | 返回 `*gogeos.Geom` |
| `StartPoint(g)` | 线起点 | 返回 `*gogeos.Geom` |

### 空间索引 (STRtree)

| 构造 | 功能 |
|------|------|
| `NewSTRtree(nodeCapacity)` | 创建 R-Tree 索引（推荐 nodeCapacity=10） |

| 方法 | 功能 |
|------|------|
| `.Insert(g, value any)` | 插入几何+值，重复 value 报错 |
| `.Query(g) ([]any, error)` | 按范围查询所有相交值 |
| `.Iterate(func(any))` | 遍历索引中全部值 |
| `.Remove(g, value any) (bool, error)` | 移除（返回是否成功） |
| `.Close()` | 释放索引引用 |

**注意**：go-geos 标注 STRtree "currently broken"，`Nearest` 会 segfault。`Insert/Query/Iterate/Remove` 可用。

### DE-9IM 关系矩阵

| 函数 | 功能 | 说明 |
|------|------|------|
| `Relate(a, b)` | 交集矩阵字符串 | 如 `"212101212"` |
| `RelatePattern(a, b, pat)` | 模式匹配 | 如 `pattern="2********"` 匹配交集 |

### 内省 & 类型判断

| 函数 | 功能 | 说明 |
|------|------|------|
| `IsEmpty(g)` | 是否为空几何 | nil 几何视为空 |
| `IsSimple(g)` | 是否简单（无自相交） | |
| `IsClosed(g)` | 是否闭合 | 仅 Curve 类型（LineString/LinearRing） |
| `IsRing(g)` | 是否环形（闭合且简单） | 仅 Curve 类型 |
| `HasZ(g)` | 是否有 Z 坐标 | |

### 元信息

| 函数 | 功能 | 说明 |
|------|------|------|
| `SRID(g)` | 获取空间参考 ID | 默认 0，GEOS 不做投影运算 |
| `SetSRID(g, srid)` | 设置空间参考 ID | 仅为标签，不影响计算 |
| `Precision(g)` | 获取预设精度 | 精度模型值 |

### 高级距离函数

| 函数 | 功能 | 说明 |
|------|------|------|
| `DistanceWithin(a, b, dist)` | 距离是否在阈值内 | 平面距离，比 `Distance` 更高效 |
| `NearestPoints(a, b)` | 最近点对 | 返回 `(ax, ay, bx, by, err)` |

---

## orbconv 子包

```go
import "zero-service/common/gisx/geos/orbconv"
```

### 类型转换

| 函数 | 功能 |
|------|------|
| `PointToGeom(orb.Point)` | orb Point → GEOS Point |
| `RingToGeom(orb.Ring)` | orb Ring → GEOS LinearRing（自动闭合） |
| `PolygonToGeom(orb.Polygon)` | orb Polygon → GEOS Polygon |
| `GeomToRing(*gogeos.Geom)` | GEOS LineString/LinearRing → orb Ring |
| `GeomToPolygon(*gogeos.Geom)` | GEOS Polygon/MultiPolygon → orb Polygon |

### 便捷谓词（接受 orb 类型）

| 函数 | 功能 |
|------|------|
| `IntersectsOrb(a, b orb.Polygon)` | 是否有交集 |
| `ContainsOrb(outer, inner orb.Polygon)` | outer 是否包含 inner（边界不算） |
| `CoversOrb(outer, inner orb.Polygon)` | outer 是否覆盖 inner（边界算） |
| `CoversPointOrb(poly orb.Polygon, pt orb.Point)` | 点是否在围栏内或边上 |
| `ContainsPointOrb(poly orb.Polygon, pt orb.Point)` | 点是否严格在围栏内 |
| `ValidOrb(poly orb.Polygon)` | 是否有效几何 |

---

## 在项目中如何使用

### 围栏命中判断（核心用法）

```go
// 推荐：用 Covers 语义（边界算命中）
hit, err := orbconv.CoversPointOrb(fence, userPoint)

// Prepared 加速：一个围栏 vs 大量点
prep, _ := geos.NewPreparedGeom(geos.NewPolygon(fenceCoords))
for _, pt := range userPoints {
    prep.IntersectsXY(pt.Lon, pt.Lat)
}
```

### 围栏相交检查

```go
ok, _ := orbconv.IntersectsOrb(fenceA, fenceB)
```

### 围栏有效性校验

```go
if valid, _ := orbconv.ValidOrb(fence); !valid {
    // 修复自相交
    fixed := geos.MakeValid(g)
}
```

### 围栏简化

```go
// 简化（保留拓扑）
simplified := geos.TopologyPreserveSimplify(g, 0.001)
```

---

## 与 GEOS C API 的覆盖对照

已覆盖 66 个函数（go-geos Geom 100 个方法中可实施部分全覆盖）。

**未封装的 34 个 API 及原因**：

| 类别 | 未封装的方法 | 原因 |
|------|-------------|------|
| 访问器 | `X/Y/Type/TypeID/CoordSeq/Num*/ExteriorRing/InteriorRing/Geometry/Point/String/Bounds` | 直接通过 `*gogeos.Geom` 调用 |
| Precision 变体 | `IntersectionPrec/DifferencePrec/SymDifferencePrec/UnionPrec/UnaryUnionPrec/SetPrecision` | 低频、精度模型场景 |
| WithParams 变体 | `BufferWithParams/BufferWithStyle/MakeValidWithParams` | 复杂参数场景、低频 |
| 线性参考 | `Interpolate/InterpolateNormalized/Project/ProjectNormalized` | 极低频 |
| 高级运算 | `LargestEmptyCircle/MaximumInscribedCircle/MinimumClearanceLine/MinimumWidth` | 极低频 |
| 其他 | `Clone/CoverageUnion/DisjointSubsetUnion/PolygonizeFull/SharedPaths/ToEWKBWithSRID/SetUserData/UserData/RelateBoundaryNodeRule` | 低频或可通过其他方式实现 |

## 部署

### Docker（app/gis/Dockerfile）

```dockerfile
# 构建阶段
RUN apk add --no-cache pkgconf geos-dev
RUN geos-config --version   # 构建日志输出版本

# 运行阶段
RUN apk add --no-cache geos
```

### 本地开发

```bash
brew install geos pkg-config
go get github.com/twpayne/go-geos
```

## 安全控制

- 所有函数通过 `safeRun` `recover` go-geos panic → `error`
- 错误前缀 `geos: `，业务层可识别
- 解析错误（WKT/WKB/GeoJSON）直接透传
- PreparedGeom / STRtree 支持 `Close()`，也可依赖 GC
- nil 几何返回 `errNil` 错误

## 版本

go-geos v0.20.4 / GEOS C 3.14.1
