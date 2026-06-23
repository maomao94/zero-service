# Research: FenceStore Implementations

- **Query**: Find all FenceStore interface implementations, FenceInfo structs, and Points field usage
- **Scope**: internal
- **Date**: 2026-06-23

## Findings

### 1. FenceStore Interface Definition

**File**: `common/gisx/store.go`

**FenceInfo struct** (lines 14-24):
```go
type FenceInfo struct {
    FenceId          string
    Name             string
    Points           []orb.Point
    H3Resolution     int
    GeohashPrecision int
    H3Cells          []string
    Geohashes        []string
    CreatedAt        time.Time
    UpdatedAt        time.Time
}
```

**FenceStore interface** (lines 28-53):
```go
type FenceStore interface {
    CreateFence(ctx context.Context, fenceId, name string, points []orb.Point, h3Resolution, geohashPrecision int, h3Cells, geohashCells []string) error
    LoadFencePolygon(ctx context.Context, fenceId string) ([]orb.Point, error)
    FindNearbyFenceIds(ctx context.Context, lon, lat, km float64) ([]string, error)
    FindFenceIdsByCellIds(ctx context.Context, cellIDs []string) ([]string, error)
    UpdateFence(ctx context.Context, fenceId, name string, points []orb.Point, h3Resolution, geohashPrecision int, h3Cells, geohashCells []string) error
    RemoveFence(ctx context.Context, fenceId string) error
    ListFences(ctx context.Context, page, pageSize int64, name string) ([]FenceInfo, int64, error)
    GetFence(ctx context.Context, fenceId string) (*FenceInfo, error)
}
```

**NoopFenceStore** (lines 57-89): Empty implementation returning `ErrFenceStoreNotImplemented` for all 8 methods.

---

### 2. GormFenceStore — Production Implementation

**File**: `app/gis/model/fencestore.go`

**Struct** (lines 28-34):
```go
type GormFenceStore struct {
    db *gormx.DB
}
func NewGormFenceStore(db *gormx.DB) *GormFenceStore {
    return &GormFenceStore{db: db}
}
```

**Method implementations** (all 8 interface methods):

| Method | Line | Key Behavior |
|---|---|---|
| `CreateFence` | 36-66 | Marshal points→JSON, compute h3_r9 recall cells, transactional: insert `GisFence` + `GisFenceCell` batch |
| `LoadFencePolygon` | 68-79 | Select by fence_id, JSON unmarshal → `[]orb.Point` |
| `FindNearbyFenceIds` | 81-113 | H3 grid-disk at resolution 9 with radius km→k, query `cell_type = "h3_r9"` only |
| `FindFenceIdsByCellIds` | 115-133 | Query by `cell_id IN ?` across all cell_types, deduplicate fence IDs |
| `UpdateFence` | 135-171 | Transactional: update `GisFence` + delete old cells + re-insert new cells (incl. h3_r9) |
| `RemoveFence` | 173-187 | Transactional: delete `GisFenceCell` then `GisFence` by fence_id |
| `ListFences` | 209-261 | Paginated (default page=1, pageSize=20, max 200), optional name LIKE filter, batch-loads cells, assembles `[]gisx.FenceInfo` |
| `GetFence` | 263-285 | Single fence by ID, loads cells, assembles `*gisx.FenceInfo` |

**Helper methods** (internal, not exported):
- `batchInsertCells` (line 189) — inserts h3/geohash/h3_r9 cells into `GisFenceCell` table in batches of 500
- `batchLoadCells` (line 292) — loads cells for multiple fence IDs, splits by `h3` (exposed) vs `geohash` (exposed), filters out `h3_r9`
- `loadCellsByFenceId` (line 312) — loads cells for single fence, same filtering
- `computeH3RecallCells` (line 326) — computes h3_r9 cells from polygon for spatial recall
- `kmToH3RecallK` (line 342) — converts km radius to h3 grid-disk k value (~0.2 km/cell)
- `kmToGeohashPrecision` (line 350) — maps km range to geohash precision (unused in this file)

**Cell type constants** (lines 18-25):
```go
h3CellType             = "h3"       // business-facing H3 cells
geohashCellType        = "geohash"  // business-facing geohash cells
h3RecallResolution     = 9          // fixed for spatial recall
h3RecallCellType       = "h3_r9"    // internal only, not exposed in FenceInfo.H3Cells
h3RecallAverageEdgeKm  = 0.2
h3PolygonMaxCellBudget = 1000
```

---

### 3. Service Wiring

**File**: `app/gis/internal/svc/servicecontext.go` (lines 14-41)

```go
type ServiceContext struct {
    Config     config.Config
    FenceStore gisx.FenceStore  // line 16
}
```

Wiring logic (lines 19-41):
- Default: `&gisx.NoopFenceStore{}`
- If `c.DB.DataSource != ""`: `model.NewGormFenceStore(db)`, with auto-migration in DevMode/TestMode
- Logs `"[gis] FenceStore: GORM"` or `"[gis] FenceStore: Noop"`

---

### 4. GORM Models (Database Schema)

**File**: `app/gis/model/gormmodel/fence.go`

```go
// GisFence (lines 6-14)
type GisFence struct {
    gormx.LegacyIDMixin
    gormx.LegacyTimeMixin
    FenceId          string `gorm:"column:fence_id;type:varchar(36);uniqueIndex;not null"`
    Name             string `gorm:"column:name;type:varchar(255);not null;default:''"`
    Points           string `gorm:"column:points;type:text;not null;comment:多边形顶点JSON [[lon,lat],...]"`
    H3Resolution     int    `gorm:"column:h3_resolution;not null;default:9"`
    GeohashPrecision int    `gorm:"column:geohash_precision;not null;default:7"`
}
// Table: gis_fence

// GisFenceCell (lines 19-24)
type GisFenceCell struct {
    gormx.LegacyIDMixin
    FenceId  string `gorm:"column:fence_id;type:varchar(36);not null;index:idx_fence_cell_fence_id"`
    CellId   string `gorm:"column:cell_id;type:varchar(64);not null;index:idx_fence_cell_cell_id"`
    CellType string `gorm:"column:cell_type;type:varchar(10);not null"`
}
// Table: gis_fence_cell
```

**Important**: `GisFence.Points` is stored as a JSON `string` (not `[]orb.Point`). Marshalling/unmarshalling happens in `GormFenceStore`.

---

### 5. Logic Layer Consumers of FenceStore

All in `app/gis/internal/logic/`:

| Logic File | FenceStore Method Used | Line(s) |
|---|---|---|
| `createfencelogic.go` | `FenceStore.CreateFence()` | 86 |
| `updatefencelogic.go` | `FenceStore.UpdateFence()` | 84 |
| `deletefencelogic.go` | `FenceStore.RemoveFence()` | 34 |
| `listfenceslogic.go` | `FenceStore.ListFences()` | 29 |
| `getfencelogic.go` | `FenceStore.GetFence()` | 34 |
| `pointinfencelogic.go` | `FenceStore.LoadFencePolygon()` | 49 |
| `pointinfenceslogic.go` | `FenceStore.LoadFencePolygon()` | 52 |
| `nearbyfenceslogic.go` | `FenceStore.FindNearbyFenceIds()` / `LoadFencePolygon()` | 40, 49 |

**Unused methods**: `FindFenceIdsByCellIds` is defined but has no logic-layer consumer found.

---

### 6. Points Field Usage in FenceInfo (gisx)

**File**: `app/gis/internal/logic/listfenceslogic.go`, function `fenceInfoToDetail` (lines 45-62):
```go
func fenceInfoToDetail(f *gisx.FenceInfo) *gis.FenceDetail {
    points := make([]*gis.Point, len(f.Points))
    for i, p := range f.Points {
        points[i] = &gis.Point{Lat: p.Y(), Lon: p.X()}  // orb.Point: X=lon, Y=lat
    }
    return &gis.FenceDetail{
        FenceId:          f.FenceId,
        Name:             f.Name,
        Points:           points,
        H3Resolution:     uint32(f.H3Resolution),
        GeohashPrecision: uint32(f.GeohashPrecision),
        H3Cells:          f.H3Cells,
        Geohashes:        f.Geohashes,
        CreatedAt:        f.CreatedAt.UnixMilli(),
        UpdatedAt:        f.UpdatedAt.UnixMilli(),
    }
}
```

This is the **only place** where `gisx.FenceInfo.Points` is read and converted to the protobuf `*gis.Point` type. The `orb.Point` uses `X=Lon, Y=Lat` convention, consistent with the project-wide `{lon, lat}` ordering.

Also used by `getfencelogic.go` (line 40) which calls the same helper: `fenceInfoToDetail(info)`.

---

### 7. Other FenceInfo Structs (Different Packages, Different Semantics)

These are **NOT related** to `gisx.FenceInfo`. They share only the name.

#### 7a. `model/kafkamodel.go` (lines 95-99)
```go
// FenceInfo 围栏信息 (Kafka alarm domain)
type FenceInfo struct {
    FenceCode string `json:"fenceCode"` // 围栏code
    OrgCode   string `json:"orgCode"`
}
```
Used by `AlarmData.StartFences` and `EndFences` (lines 82-84).

#### 7b. `app/xfusionmock/xfusionmock/xfusionmock.pb.go` (lines 1063-1071)
```go
// FenceInfo 围栏信息 (protobuf generated for xfusionmock service)
type FenceInfo struct {
    state         protoimpl.MessageState
    FenceCode     string `protobuf:"bytes,1,opt,name=fenceCode,proto3" json:"fenceCode,omitempty"`
    OrgCode       string `protobuf:"bytes,7,opt,name=orgCode,proto3" json:"orgCode,omitempty"`
    unknownFields protoimpl.UnknownFields
    sizeCache     protoimpl.SizeCache
}
```
Used in `AlarmData.GetStartFences()` / `GetEndFences()`.

---

### 8. Spec Documentation Reference

**File**: `.trellis/spec/backend/gisx-guidelines.md` (lines 315-388)
Documents the FenceStore interface pattern, h3_r9 recall indexing rules, injection rules, app/gis service architecture, and model layer structure.

---

## Summary of Implementations

| Layer | File | Type | Purpose |
|---|---|---|---|
| Interface | `common/gisx/store.go` | `FenceStore` (interface) | Contract definition + `FenceInfo` struct |
| Interface | `common/gisx/store.go` | `NoopFenceStore` (struct) | Returns `ErrFenceStoreNotImplemented` for all methods |
| Store | `app/gis/model/fencestore.go` | `GormFenceStore` (struct) | GORM-backed production implementation |
| Wiring | `app/gis/internal/svc/servicecontext.go` | `ServiceContext.FenceStore` | Selects Noop or GORM based on DB config |
| DB Model | `app/gis/model/gormmodel/fence.go` | `GisFence` + `GisFenceCell` | GORM table models |

## Caveats / Not Found

- `FindFenceIdsByCellIds` has **no caller** in the logic layer — defined in interface and implemented in GORM, but apparently unused.
- `kmToGeohashPrecision` (line 350-362 in `fencestore.go`) is defined but **not called** anywhere in the same file.
- No test file found for `GormFenceStore` (no `fencestore_test.go` under `app/gis/model/`).
