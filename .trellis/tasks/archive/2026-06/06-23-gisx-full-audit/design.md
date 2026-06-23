# design.md — gisx 整体优化技术设计

## 1. FenceInfo.Points → orb.Polygon

### 变更范围

```
FenceInfo.Points   []orb.Point  →  orb.Polygon

FenceStore 接口:
  CreateFence(..., points []orb.Point, ...)          → points orb.Polygon
  LoadFencePolygon(...) ([]orb.Point, error)          → (orb.Polygon, error)
  UpdateFence(..., points []orb.Point, ...)           → points orb.Polygon

调用方:
  app/gis/model/fencestore.go (GormFenceStore)  — 实现适配
  app/gis/internal/logic/listfenceslogic.go     — orb.Point→pb 转换适配
```

### orb.Polygon 语义

```
orb.Polygon = []orb.Ring
  polygon[0] = 外环 (exterior ring)
  polygon[1:] = 洞 (hole rings)

无洞: len(polygon) = 1
有 1 个洞: len(polygon) = 2
有 N 个洞: len(polygon) = 1 + N
```

与包内已有函数一致：`OrbPolygonToH3GeoPolygon`, `PolygonToGeom`, `ExtractPolygonCoords`

### 向后兼容策略

不做兼容层，直接 breaking change。影响面小（1 个实现 + 1 个 logic），一次性改完。

---

## 2. OrbRingToH3LatLng 修复

**问题**: `ring = append(ring, ring[0])` 当 ring 底层数组有剩余 capacity 时，会原地修改调用方的数据。

**修复**: 先分配结果 slice，再判断是否需要追加闭合点（不修改入参）。

```go
func OrbRingToH3LatLng(ring orb.Ring) []h3.LatLng {
    if len(ring) == 0 {
        return nil
    }
    needClose := !IsOrbPointsEqual(ring[0], ring[len(ring)-1])
    n := len(ring)
    if needClose {
        n++
    }
    res := make([]h3.LatLng, n)
    for i, pt := range ring {
        res[i] = h3.LatLng{Lat: pt[1], Lng: pt[0]}
    }
    if needClose {
        res[len(ring)] = res[0]
    }
    return res
}
```

---

## 3. validationError 导出

```go
// 旧
type validationError struct { msg string }

// 新
type ValidationError struct { Msg string }
func (e *ValidationError) Error() string { return e.Msg }
```

---

## 4. Centroid 错误信息

```go
// errors.go 新增
var ErrEmptyResult = fmt.Errorf("运算结果为空几何")

// overlay.go Centroid
func Centroid(g *gogeos.Geom) (x, y float64, err error) {
    if g == nil {
        return 0, 0, errNil
    }
    v, e := safeRun(func() (pointPair, error) {
        c := g.Centroid()
        if c == nil || c.IsEmpty() {
            return pointPair{}, ErrEmptyResult  // 而非 errNil
        }
        return pointPair{x1: c.X(), y1: c.Y()}, nil
    })
    ...
}
```

---

## 5. IsEmpty nil 行为统一

```go
func IsEmpty(g *gogeos.Geom) (bool, error) {
    if g == nil {
        return false, errNil  // 旧: return true, nil
    }
    return oneBool(g, func(gg *gogeos.Geom) bool { return gg.IsEmpty() })
}
```

---

## 6. oneBool / predicateTwo nil 错误统一

```go
// predicateTwo: 改为复用 errNil
func predicateTwo(a, b *gogeos.Geom, fn func() bool) (bool, error) {
    if a == nil || b == nil {
        return false, errNil  // 旧: fmt.Errorf("geometry 为 nil")
    }
    return safeRun(...)
}
```

同样修改 `Relate`, `RelatePattern`, `HausdorffDistance`, `NearestPoints` 中的内联 nil 检查。
