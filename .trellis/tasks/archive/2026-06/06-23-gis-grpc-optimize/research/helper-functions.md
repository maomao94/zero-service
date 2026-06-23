# Research: Helper Functions Inventory

- **Query**: List ALL exported and unexported helper functions in `app/gis/internal/logic/helper.go`
- **Scope**: internal
- **Date**: 2026-06-23

## Findings

### Exported Functions (4)

| Signature | Line | Description |
|---|---|---|
| `ValidateH3Resolution(resolution uint32) (int, error)` | 20 | Validates H3 resolution 0-15, returns int or error |
| `ValidateGeoHashPrecision(precision uint32) (int, error)` | 29 | Validates geohash precision 1-12, returns int or error |
| `EncodeH3Cell(point *gis.Point, resolution int) (h3.Cell, error)` | 170 | Encodes lat/lon to H3 cell |
| `ValidatePoints(points ...*gis.Point) error` | 176 | Batch validates pb Point list (non-empty, non-nil, coordinate range) |

### Unexported Functions (8)

| Signature | Line | Description |
|---|---|---|
| `resolveH3Resolution(r uint32) (int, error)` | 40 | Validate H3 resolution; 0→default 9, >15→error. Has caveat comment about proto3 zero-value ambiguity. |
| `resolveGeohashPrecision(p uint32) (int, error)` | 51 | Validate geohash precision; ≤0→default 7, >12→error |
| `computeFenceCells(polygon orb.Polygon, h3Resolution, geohashPrecision int) (h3CellStrings []string, geohashes []string, err error)` | 62 | Computes both H3 cells and geohash cells covering a polygon in one call |
| `scanGeohashCells(polygon orb.Polygon, precision int, includeNeighbors bool) (map[string]struct{}, error)` | 81 | Core geohash scan algorithm: walks polygon bbox at precision-level steps, collects covered cells via center-point check + GEOS intersect |
| `computeGeohashCells(polygon orb.Polygon, precision int) []string` | 138 | Wraps scanGeohashCells(..., false); **silently swallows errors** (returns nil) |
| `geohashCellSize(precision int, _ float64) (widthDeg, heightDeg float64)` | 152 | Calculates lon/lat step size from geohash precision bits |
| `validateCoordType(t gis.CoordType) error` | 160 | Validates coord type is 1=WGS84, 2=GCJ02, or 3=BD09 |
| `pbPointToOrbPolygon(points []*gis.Point) (orb.Polygon, error)` | 193 | Converts pb Point slice to orb.Polygon: validates ≥3 points, validates each coordinate, auto-closes ring via gisx.EnsurePolygonClosed |

### Dependencies

- `zero-service/common/gisx` — ValidateCoordinate, EnsurePolygonClosed, OrbPolygonToH3GeoPolygon
- `zero-service/common/gisx/geos/orbconv` — IntersectsOrb
- `zero-service/common/tool` — NewErrorByPbCode
- `zero-service/third_party/extproto` — error codes (Code__1_01_PARAM, Code__1_01_PARAM_MISSING, Code__1_01_PARAM_INVALID)
- `github.com/mmcloughlin/geohash` — geohash encode/decode
- `github.com/paulmach/orb` — orb.Polygon, orb.Point, orb.Ring
- `github.com/uber/h3-go/v4` — H3 cell operations

## Caveats / Not Found

- `computeGeohashCells` silently swallows errors from `scanGeohashCells` — intentionally returns nil on failure, no error propagation
- `geohashCellSize` has an unused `_ float64` parameter (originally for latitude-dependent sizing but simplified)
