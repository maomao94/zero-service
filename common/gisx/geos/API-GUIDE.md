# GEOS C 库与 go-geos 深度解析

基于 [libgeos.org](https://libgeos.org/usage/c_api/) 官方文档和 [go-geos](https://github.com/twpayne/go-geos) 源码。从 C 层讲起，到 Go 封装结束。

---

## 一、GEOS 是什么

GEOS（Geometry Engine - Open Source）是 PostGIS 的空间运算引擎，C++ 实现，对外暴露稳定 C API。

能做的：几何构造、空间谓词、叠加运算、缓冲简化、空间索引、WKT/WKB/GeoJSON 解析
不能做的：地图渲染、投影转换（PROJ）、地理测地距离计算、网络分析

当前封装基于 go-geos v0.20.4，底层链接 GEOS C 3.14.1。
go-geos 提供 `VersionCompare(major, minor, patch int) int` 用于运行时检测 GEOS 版本兼容性。

---

## 二、C API 两种风格

### 2.1 全局 API（已废弃）

```c
initGEOS(errorHandler, noticeHandler);   // 全局初始化
GEOSGeometry* g = GEOSGeomFromWKT("POINT(1 2)");
GEOSGeom_destroy(g);
finishGEOS();                              // 全局销毁
```

缺点：多线程不安全，全局共享状态。

### 2.2 Reentrant API（_r 后缀，推荐）

```c
GEOSContextHandle_t ctx = GEOS_init_r();  // 创建独立会话
GEOSGeometry* g = GEOSGeomFromWKT_r(ctx, "POINT(1 2)");
GEOSGeom_destroy_r(ctx, g);
GEOS_finish_r(ctx);                        // 销毁会话
```

每个 Context 独立拥有：内存分配池、错误消息缓冲区、WKT/WKB Reader/Writer 配置。官方文档建议 **"每个线程创建自己的 Context"**。

### 2.3 go-geos 全部使用 _r API

go-geos 只封装 `_r` 版本，不碰全局 API。每个 `*Context` 对应一个 `GEOSContextHandle_t`。

---

## 三、GEOS 核心对象模型

### 3.1 GEOSCoordSequence（坐标序列）

C 层坐标存储在连续的 `double` 数组中：

```
二维 (XY)：| X₀ | Y₀ | X₁ | Y₁ | X₂ | Y₂ | ...
三维 (XYZ)：| X₀ | Y₀ | Z₀ | X₁ | Y₁ | Z₁ | ...
四维 (XYZM)：| X₀ | Y₀ | Z₀ | M₀ | X₁ | Y₁ | Z₁ | M₁ | ...
```

创建方式（C API）：

```c
// 方式 1：从平面缓冲区创建（Go 最常用）
double buf[] = {1.0, 3.0, 2.0, 2.0, 3.0, 1.0};  // X₀,Y₀, X₁,Y₁, X₂,Y₂
GEOSCoordSequence* seq = GEOSCoordSeq_copyFromBuffer_r(ctx, buf, 3, 0, 0);

// 方式 2：从分离的数组创建
double x[] = {1.0, 2.0, 3.0};
double y[] = {3.0, 2.0, 1.0};
GEOSCoordSequence* seq = GEOSCoordSeq_copyFromArrays_r(ctx, x, y, NULL, NULL, 3);

// 方式 3：创建空序列后逐个设置
GEOSCoordSequence* seq = GEOSCoordSeq_create_r(ctx, 3, 2);  // 3点, 2维
GEOSCoordSeq_setXY_r(ctx, seq, 0, 1.0, 3.0);
```

关键规则：**创建 GEOSGeometry 时，CoordSeq 的所有权转移给 Geometry**。不需要单独销毁 CoordSeq。

拷贝回用户内存（C → 应用）：

```c
// 方式 1：拷贝到平面缓冲区
double buf[6];
GEOSCoordSeq_copyToBuffer_r(ctx, seq, buf, 0, 0);

// 方式 2：拷贝到分离数组（通常比逐个读取更快）
double x[3], y[3];
GEOSCoordSeq_copyToArrays_r(ctx, seq, x, y, NULL, NULL);
```

go-geos 的 `ToCoords()` 用的就是 `copyToBuffer_r`。

### 3.2 GEOSGeometry（几何对象）

GEOSGeometry 是泛型类型，`GEOSGeomTypeId_r()` 返回具体类型枚举。

创建示例（官方文档模式）：

```c
// Point
GEOSGeometry* pt = GEOSGeom_createPointFromXY_r(ctx, 1.0, 2.0);

// LineString（CoordSeq 所有权转移）
GEOSCoordSequence* seq = GEOSCoordSeq_copyFromBuffer_r(ctx, buf, 3, 0, 0);
GEOSGeometry* line = GEOSGeom_createLineString_r(ctx, seq);
// seq 不再需要销毁

// Polygon（需要外环 + 洞数组）
GEOSGeometry* shell = GEOSGeom_createLinearRing_r(ctx, shellSeq);
GEOSGeometry* holes[] = {hole1Geom, hole2Geom};
GEOSGeometry* poly = GEOSGeom_createPolygon_r(ctx, shell, holes, 2);
// shell 和 holes 中几何的所有权转移给 poly

// 集合（子几何数组，所有权转移）
GEOSGeometry* geoms[] = {poly1, poly2};
GEOSGeometry* mp = GEOSGeom_createCollection_r(ctx, GEOS_MULTIPOLYGON, geoms, 2);
// 注意：geoms 数组本身不会被接管，需要手动释放
free(geoms);

// 空几何（无参数的占位对象）
GEOSGeometry* empty = GEOSGeom_createEmptyPolygon_r(ctx);
GEOSisEmpty_r(ctx, empty);  // → 1 (true)
```

### 3.3 访问器：从 C 对象读取数据

```c
// 类型判断
int typeId = GEOSGeomTypeId_r(ctx, g);  // 0=Point, 1=LineString, ...

// 点坐标
double x, y;
GEOSGeomGetX_r(ctx, pt, &x);
GEOSGeomGetY_r(ctx, pt, &y);

// 坐标序列（Point/LineString/LinearRing 适用）
const GEOSCoordSequence* seq = GEOSGeom_getCoordSeq_r(ctx, g);

// 多边形：外环 + 洞
const GEOSGeometry* shell = GEOSGetExteriorRing_r(ctx, poly);
int nHoles = GEOSGetNumInteriorRings_r(ctx, poly);
const GEOSGeometry* hole0 = GEOSGetInteriorRingN_r(ctx, poly, 0);

// 集合：子几何
int nGeoms = GEOSGetNumGeometries_r(ctx, collection);
const GEOSGeometry* geom0 = GEOSGetGeometryN_r(ctx, collection, 0);
```

C 层这些访问器返回的是 **引用**（const 指针），不分配新内存，不能独立销毁。

---

## 四、go-geos 如何封装 C 对象

### 4.1 Context 结构

```go
// go-geos/context.go
func NewContext() *Context {
    cHandle := C.GEOS_init_r()                        // ① C 层分配句柄
    c := &Context{
        cHandle:  cHandle,
        refCount: &atomic.Int64{},
    }
    c.ref()                                             // ② 引用计数 = 1
    runtime.AddCleanup(c, func(h C.GEOSContextHandle_t) {
        if c.refCount.Add(-1) == 0 {
            C.finishGEOS_r(h)                          // ③ 计数归零时销毁
        }
    }, cHandle)
    // ④ 注册 C 错误回调 → 写入 Go error 指针
    c.errPHandle = cgo.NewHandle(&c.err)
    C.GEOSContext_setErrorMessageHandler_r(
        cHandle,
        C.GEOSMessageHandler_r(C.c_errorMessageHandler),
        unsafe.Pointer(&c.errPHandle),
    )
    return c
}
```

go-geos 官方建议：**每个 goroutine 用一个独立 Context 以获得最佳并发性能**。本项目使用单例（`sync.Once`），因为典型的请求-响应模式中，C 计算开销远大于锁竞争。如果你的场景需要对一个固定多边形做大量（10000+）谓词判定，考虑：
1. 优先使用 Prepared Geometry（预处理后的 R-Tree 内存在几何对象内部，不受 Context 锁影响）
2. 或为不同 goroutine 创建独立 Context

### 4.2 Geom 结构

```go
// go-geos/geom.go
type Geom struct {
    context          *Context              // 所属 Context
    cGeom            *C.struct_GEOSGeom_t  // C 层指针
    owner            *Geom                 // 父几何
    typeID           TypeID                // 缓存类型
    numGeometries    int                   // 缓存子几何数
    numInteriorRings int                   // 缓存洞数
    numPoints        int                   // 缓存点数
}
```

属性缓存：`typeID`、`numGeometries` 等在 `newNonNilGeom` 中一次性从 C 查询并缓存，后续调用零 cgo 开销。

### 4.3 所有权树

```
MultiPolygon (owner=nil)       ← 独立拥有 C 内存
├── Polygon[0]  (owner=↑)       ← 被 MultiPolygon 持有
│   ├── ExteriorRing (owner=↑) ← 被 Polygon[0] 持有
│   └── InteriorRing (owner=↑) ← 被 Polygon[0] 持有
└── Polygon[1]  (owner=↑)
```

规则（官方说明）：

> Returned sub-geometries (e.g. polygon rings or geometries in a collection) and coordinate sequences are owned by the geometry and are only valid for as long as the original geometry exists.

所以 `g.ExteriorRing()` 返回的子几何不能脱离父几何单独使用，否则 C 指针悬空。

### 4.4 跨 Context 克隆

go-geos 的做法：

```go
func (c *Context) Clone(g *Geom) *Geom {
    if g.context == c {
        return g.Clone()       // 同 Context：直接 GEOSGeom_clone_r
    }
    // 跨 Context：WKB 序列化中转
    clone, _ := c.NewGeomFromWKB(g.ToEWKBWithSRID())
    return clone
}
```

### 4.5 内存管理

go-geos README 原文：

> go-geos objects live mostly on the C heap. go-geos sets cleanup functions on the objects it creates that free the associated C memory. However, the C heap is not visible to the Go runtime. This can result in significant memory pressure as memory is consumed by large, un-freed geometries.

**C 堆对 Go GC 不可见**。大量未释放的几何对象会占用 C 内存而 Go GC 感知不到。对于长生命周期（如 PreparedGeom、STRtree），建议显式 `Close()`。

---

## 五、错误处理全链路

go-geos README 原文：

> go-geos panics whenever it encounters a GEOS return code indicating an error, rather than returning an error. Such panics will not occur if go-geos is used correctly. Panics will occur for invalid API calls, out-of-bounds access, or operations on invalid geometries.

所以 go-geos 的设计哲学是 **panic for programmer errors, not runtime errors**。我们的 `safeRun` 再把这些 panic 转回 error。

链路：

```
C 层出错
  → GEOS 调 errorMessageHandler 回调
  → 写入 Context.err
  → GEOS 函数返回 NULL/0
  → go-geos 检测到 → panic(Context.err)
  → safeRun recover → 转为 Go error（前缀 "geos: "）
```

---

## 六、GEOS C API 完整参考

### 6.1 初始化和版本

```c
GEOSContextHandle_t GEOS_init_r(void);
void GEOS_finish_r(GEOSContextHandle_t ctx);
const char* GEOSversion(void);
```

### 6.2 CoordSeq

```c
GEOSCoordSequence* GEOSCoordSeq_create_r(ctx, unsigned int size, unsigned int dims);
GEOSCoordSequence* GEOSCoordSeq_copyFromBuffer_r(ctx, const double* buf, unsigned int size, int hasZ, int hasM);
GEOSCoordSequence* GEOSCoordSeq_copyFromArrays_r(ctx, const double* x, const double* y, const double* z, const double* m, unsigned int size);
GEOSCoordSequence* GEOSCoordSeq_clone_r(ctx, const GEOSCoordSequence* s);

int GEOSCoordSeq_getSize_r(ctx, const GEOSCoordSequence* s, unsigned int* size);
int GEOSCoordSeq_getDimensions_r(ctx, const GEOSCoordSequence* s, unsigned int* dims);

int GEOSCoordSeq_getX_r(ctx, s, idx, &x);    int GEOSCoordSeq_setX_r(ctx, s, idx, x);
int GEOSCoordSeq_getY_r(ctx, s, idx, &y);    int GEOSCoordSeq_setY_r(ctx, s, idx, y);
int GEOSCoordSeq_getZ_r(ctx, s, idx, &z);    int GEOSCoordSeq_setZ_r(ctx, s, idx, z);

int GEOSCoordSeq_copyToBuffer_r(ctx, s, double* buf, int hasZ, int hasM);
int GEOSCoordSeq_copyToArrays_r(ctx, s, double* x, double* y, double* z, double* m);

int GEOSCoordSeq_isCCW_r(ctx, s, char* isCCW);
void GEOSCoordSeq_destroy_r(ctx, s);
```

### 6.3 几何构造（创建 C 对象）

```c
GEOSGeometry* GEOSGeom_createPoint_r(ctx, GEOSCoordSequence* s);
GEOSGeometry* GEOSGeom_createPointFromXY_r(ctx, double x, double y);
GEOSGeometry* GEOSGeom_createLineString_r(ctx, GEOSCoordSequence* s);
GEOSGeometry* GEOSGeom_createLinearRing_r(ctx, GEOSCoordSequence* s);
GEOSGeometry* GEOSGeom_createPolygon_r(ctx, GEOSGeometry* shell, GEOSGeometry** holes, unsigned int nholes);
GEOSGeometry* GEOSGeom_createCollection_r(ctx, int type, GEOSGeometry** geoms, unsigned int ngeoms);

GEOSGeometry* GEOSGeom_createEmptyPoint_r(ctx);
GEOSGeometry* GEOSGeom_createEmptyLineString_r(ctx);
GEOSGeometry* GEOSGeom_createEmptyPolygon_r(ctx);
GEOSGeometry* GEOSGeom_createEmptyCollection_r(ctx, int type);

GEOSGeometry* GEOSGeom_clone_r(ctx, const GEOSGeometry* g);
void GEOSGeom_destroy_r(ctx, GEOSGeometry* g);
```

### 6.4 几何访问器

```c
int GEOSGeomTypeId_r(ctx, const GEOSGeometry* g);

// 点
int GEOSGeomGetX_r(ctx, g, double* x);     int GEOSGeomGetY_r(ctx, g, double* y);

// 坐标序列
const GEOSCoordSequence* GEOSGeom_getCoordSeq_r(ctx, g);

// 多边形
const GEOSGeometry* GEOSGetExteriorRing_r(ctx, g);
int GEOSGetNumInteriorRings_r(ctx, g);
const GEOSGeometry* GEOSGetInteriorRingN_r(ctx, g, int n);

// 集合
int GEOSGetNumGeometries_r(ctx, g);
const GEOSGeometry* GEOSGetGeometryN_r(ctx, g, int n);

// 内省
char GEOSisEmpty_r(ctx, g);       char GEOSisSimple_r(ctx, g);
char GEOSisClosed_r(ctx, g);      char GEOSisRing_r(ctx, g);
char GEOSHasZ_r(ctx, g);
```

### 6.5 空间谓词（所有返回 char: 0/1，出错 2）

```c
char GEOSIntersects_r(ctx, a, b);     char GEOSDisjoint_r(ctx, a, b);
char GEOSTouches_r(ctx, a, b);        char GEOSCrosses_r(ctx, a, b);
char GEOSWithin_r(ctx, a, b);         char GEOSContains_r(ctx, a, b);
char GEOSOverlaps_r(ctx, a, b);       char GEOSCovers_r(ctx, a, b);
char GEOSCoveredBy_r(ctx, a, b);      char GEOSEquals_r(ctx, a, b);
char GEOSEqualsExact_r(ctx, a, b, double tolerance);
```

### 6.6 叠加运算（返回新 GEOSGeometry*）

```c
GEOSGeometry* GEOSIntersection_r(ctx, a, b);
GEOSGeometry* GEOSUnion_r(ctx, a, b);
GEOSGeometry* GEOSDifference_r(ctx, a, b);
GEOSGeometry* GEOSSymDifference_r(ctx, a, b);
GEOSGeometry* GEOSUnaryUnion_r(ctx, g);
GEOSGeometry* GEOSBoundary_r(ctx, g);
GEOSGeometry* GEOSEnvelope_r(ctx, g);
GEOSGeometry* GEOSConvexHull_r(ctx, g);
```

### 6.7 有效性

```c
char GEOSisValid_r(ctx, g);
const char* GEOSisValidReason_r(ctx, g);  // 有效时返回空字符串
GEOSGeometry* GEOSMakeValid_r(ctx, g);     // 可能返回 MultiPolygon
```

`GEOSisValidReason_r` 返回字符串如：
- `"Self-intersection at or near point 2 2"`
- `"Hole lies outside shell"`
- `"Interior is disconnected"`

### 6.8 缓冲、简化、变换

```c
GEOSGeometry* GEOSBuffer_r(ctx, g, double width, int quadsegs);
GEOSGeometry* GEOSSimplify_r(ctx, g, double tolerance);
GEOSGeometry* GEOSTopologyPreserveSimplify_r(ctx, g, double tolerance);
GEOSGeometry* GEOSConcaveHull_r(ctx, g, double ratio, unsigned int allowHoles);
GEOSGeometry* GEOSDensify_r(ctx, g, double tolerance);
GEOSGeometry* GEOSClipByRect_r(ctx, g, double xmin, double ymin, double xmax, double ymax);
GEOSGeometry* GEOSNode_r(ctx, g);
GEOSGeometry* GEOSSnap_r(ctx, a, b, double tolerance);
GEOSGeometry* GEOSOffsetCurve_r(ctx, g, double width, int quadsegs, int joinStyle, double mitreLimit);
GEOSGeometry* GEOSMinimumRotatedRectangle_r(ctx, g);

// 注意：Normalize 修改原几何！
GEOSGeometry* GEOSNormalize_r(ctx, g);  // in-place 修改
GEOSGeometry* GEOSReverse_r(ctx, g);
```

### 6.9 测量（结果通过指针输出）

```c
int GEOSArea_r(ctx, g, double* area);           // 返回 1=成功, 0=失败
int GEOSLength_r(ctx, g, double* length);
int GEOSDistance_r(ctx, a, b, double* dist);
int GEOSHausdorffDistance_r(ctx, a, b, double* dist);
int GEOSFrechetDistance_r(ctx, a, b, double* dist);
char GEOSDistanceWithin_r(ctx, a, b, double dist);  // 比 Distance 高效

GEOSGeometry* GEOSGetCentroid_r(ctx, g);        // 可能在凹多边形外部
GEOSGeometry* GEOSPointOnSurface_r(ctx, g);      // 保证在多边形上

int GEOSMinimumClearance_r(ctx, g, double* clearance);
```

### 6.10 DE-9IM

```c
char* GEOSRelate_r(ctx, a, b);              // 返回需 GEOSFree_r 释放
char GEOSRelatePattern_r(ctx, a, b, const char* pattern);
```

模式字符：`T`=非空, `F`=空, `0`=点, `1`=线, `2`=面, `*`=任意。

### 6.11 最近点

```c
GEOSCoordSequence* GEOSNearestPoints_r(ctx, a, b);
// 返回的 CoordSeq 包含 2 个点：[0]=a上最近点, [1]=b上最近点
```

### 6.12 WKT / WKB

```c
// WKT Reader
GEOSWKTReader* GEOSWKTReader_create_r(ctx);
GEOSGeometry* GEOSWKTReader_read_r(ctx, reader, const char* wkt);
void GEOSWKTReader_destroy_r(ctx, reader);

// WKT Writer
GEOSWKTWriter* GEOSWKTWriter_create_r(ctx);
void GEOSWKTWriter_setTrim_r(ctx, writer, char trim);  // 去掉末尾多余的 0
char* GEOSWKTWriter_write_r(ctx, writer, g);           // 需 GEOSFree_r 释放
void GEOSWKTWriter_destroy_r(ctx, writer);

// WKB Reader
GEOSWKBReader* GEOSWKBReader_create_r(ctx);
GEOSGeometry* GEOSWKBReader_read_r(ctx, reader, const unsigned char* wkb, size_t size);
void GEOSWKBReader_destroy_r(ctx, reader);

// WKB Writer
GEOSWKBWriter* GEOSWKBWriter_create_r(ctx);
unsigned char* GEOSWKBWriter_write_r(ctx, writer, g, size_t* size);  // 需 GEOSFree_r 释放
void GEOSWKBWriter_destroy_r(ctx, writer);

// GEOS 内部分配的内存统一释放
void GEOSFree_r(ctx, void* buffer);
```

### 6.13 Prepared Geometry

```c
const GEOSPreparedGeometry* GEOSPrepare_r(ctx, g);

char GEOSPreparedIntersects_r(ctx, prep, other);
char GEOSPreparedContains_r(ctx, prep, other);
char GEOSPreparedCovers_r(ctx, prep, other);
char GEOSPreparedCoveredBy_r(ctx, prep, other);
char GEOSPreparedDisjoint_r(ctx, prep, other);
char GEOSPreparedOverlaps_r(ctx, prep, other);
char GEOSPreparedTouches_r(ctx, prep, other);
char GEOSPreparedWithin_r(ctx, prep, other);

char GEOSPreparedContainsXY_r(ctx, prep, double x, double y);
char GEOSPreparedIntersectsXY_r(ctx, prep, double x, double y);
char GEOSPreparedDistanceWithin_r(ctx, prep, other, double dist);

const GEOSCoordSequence* GEOSPreparedNearestPoints_r(ctx, prep, other);

void GEOSPreparedGeom_destroy_r(ctx, prep);
```

### 6.14 STRtree 空间索引

```c
GEOSSTRtree* GEOSSTRtree_create_r(ctx, size_t nodeCapacity);  // 推荐 10

// 插入：树不接管几何或 item 的所有权，只存指针
void GEOSSTRtree_insert_r(ctx, tree, const GEOSGeometry* g, void* item);

// 查询：用回调通知每个命中的 item
void GEOSSTRtree_query_r(ctx, tree, const GEOSGeometry* g,
    void (*callback)(void* item, void* userdata), void* userdata);

// 遍历所有 item
void GEOSSTRtree_iterate_r(ctx, tree,
    void (*callback)(void* item, void* userdata), void* userdata);

char GEOSSTRtree_remove_r(ctx, tree, const GEOSGeometry* g, void* item);

// 最近邻（需自定义距离回调）
const void* GEOSSTRtree_nearest_generic_r(ctx, tree, const void* item,
    const GEOSGeometry* geom,
    int (*callback)(const void* a, const void* b, double* dist, void* userdata),
    void* userdata);

void GEOSSTRtree_destroy_r(ctx, tree);
```

官方重要提示：

> The tree doesn't take ownership of inputs, just holds references. So keep a list of the items you create to free them at the end.

树不接管插入的对象所有权！你需要自己管理 item 和 geometry 的生命周期。

### 6.15 辅助 API

```c
// SRID
int GEOSGetSRID_r(ctx, g);
void GEOSSetSRID_r(ctx, g, int srid);

// 外包盒（不创建对象，直接返回坐标范围）
void GEOSGeom_getExtent_r(ctx, g, &xmin, &ymin, &xmax, &ymax);

// 线段求交点
char GEOSSegmentIntersection_r(ctx, ax0,ay0, ax1,ay1, bx0,by0, bx1,by1, &cx, &cy);

// 三点方向
int GEOSOrientationIndex_r(ctx, ax,ay, bx,by, px,py);  // 0=共线, 1=CCW, -1=CW

// DE-9IM 模式匹配
char GEOSRelatePatternMatch_r(ctx, const char* mat, const char* pat);

// 从线构造面
GEOSGeometry* GEOSBuildArea_r(ctx, g);
GEOSGeometry* GEOSLineMerge_r(ctx, g);
GEOSGeometry* GEOSPolygonize_r(ctx, GEOSGeometry** geoms, unsigned int n);
```

### 6.16 版本比较

```c
int GEOSVersionCompare(int major, int minor, int patch);
// <0: 运行库版本低于指定版本; =0: 相等; >0: 运行库版本高于指定版本
```

---

## 七、go-geos 在 C 之上的 Go 层

| Go 层工作 | 说明 |
|-----------|------|
| GeoJSON 解析 | 纯 Go（`encoding/json`），不经过 C 的 GeoJSON Reader |
| WKT/WKB 解析 | 纯 Go 实现 |
| `runtime.AddCleanup` | 自动 C 内存管理 |
| 属性缓存 | `typeID`、`numGeometries` 创建时缓存 |
| `sync.OnceValue` | WKT/WKB/GeoJSON 的 Reader/Writer 懒加载 |
| `cgo.Handle` | 安全传递错误回调指针 |
| `geometry.Geometry`（子包） | 高级类型，实现 `sql.Scanner`/`json.Marshaler`/`gob.GobEncoder` 等 |

---

## 八、go-geos 未封装的 GEOS C API

| C API | 功能 | 原因 |
|-------|------|------|
| `GEOSVoronoiDiagram_r` | Voronoi 图 | 低频 |
| `GEOSDelaunayTriangulation_r` | Delaunay 三角网 | 低频 |
| `GEOSMaximumInscribedCircle_r` | 最大内切圆 | 低频（3.9+） |
| `GEOSLargestEmptyCircle_r` | 最大空圆 | 低频（3.9+） |
| `GEOSMinimumWidth_r` | 最小宽度 | 低频（3.9+） |
| `GEOSPolygonHullSimplify_r` | 多边形包简化 | 低频 |
| `GEOSCoverageUnion_r` | Coverage 合并 | 低频 |
| `GEOSSTRtree_nearest_generic_r` | R-Tree 最近邻 | ⚠️ go-geos 标注 broken |
| `GEOSCoordSeq_copyFromArrays_r` | 数组方式创建 CoordSeq | go-geos 用 buffer 方式 |

---

## 九、GEOS 官方关键设计要点

1. **所有权转移**：构造 Geometry 时 CoordSeq 和子几何的所有权转移给新 Geometry。但数组容器的所有权不转移（需要手动 free）。

2. **GEOSFree_r**：GEOS 内部分配的字符串（WKT、Relate 返回）必须用 `GEOSFree_r` 释放，不能用系统 `free()`（Windows 兼容性）。

3. **Normalize 是原地修改**：`GEOSNormalize_r` 修改输入几何本身，不创建新对象。

4. **STRtree 不接管所有权**：树只存指针，使用者负责管理插入对象的内存。

5. **空几何有意义**：不相交多边形的交集返回空 Polygon（非 NULL），`GEOSisEmpty_r` 返回 true。

6. **每个线程一个 Context**：官方推荐。go-geos 内部用 `sync.Mutex` 保证同一 Context 安全，但我们全局单例 Context 在并发场景下会有锁竞争。
