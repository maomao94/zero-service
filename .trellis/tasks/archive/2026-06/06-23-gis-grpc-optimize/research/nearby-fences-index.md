# Research: NearbyFences Spatial Index Implementation

- **Query**: How does FindNearbyFenceIds work? What is the recall strategy?
- **Scope**: internal
- **Date**: 2026-06-23

## Findings

### Constants (fencestore.go lines 19-26)

| Constant | Value | Purpose |
|---|---|---|
| `h3CellType` | `"h3"` | Tag for exact-precision H3 cells |
| `geohashCellType` | `"geohash"` | Tag for geohash cells |
| `h3RecallResolution` | `9` | Fixed resolution for nearby recall (tradeoff: cell edge ~0.2km) |
| `h3RecallCellType` | `"h3_r9"` | Tag for recall-only H3 cells (separate from exact cells) |
| `h3RecallAverageEdgeKm` | `0.2` | H3 resolution 9 average edge length for km-to-k conversion |
| `h3PolygonMaxCellBudget` | `1000` | Max cells for polygon-to-cells conversion budget |

### FindNearbyFenceIds Algorithm (lines 82-114)

1. **kmToH3RecallK(km)**: Converts search radius in km to H3 GridDisk ring count
2. **h3.LatLngToCell(origin, resolution=9)**: Encodes query point at fixed resolution 9
3. **h3.GridDisk(origin, k)**: Expands to candidate H3 cells within k rings
4. **DB query**: `WHERE cell_type = 'h3_r9' AND cell_id IN (candidate cells)` — looks up fences whose precomputed recall cells intersect the search area
5. **Deduplication**: Uses `map[string]struct{}` to deduplicate fence IDs

### Three-Tier Cell Storage (batchInsertCells, line 190)

Every created/updated fence stores THREE types of cells in `gis_fence_cells`:

| Type | Source | Use Case |
|---|---|---|
| `h3` | `computeFenceCells` (exact) | Direct fence-cell lookup |
| `geohash` | `computeFenceCells` (exact) | Geohash-based queries |
| `h3_r9` | `computeH3RecallCells` (resolution 9) | Nearby spatial index (coarse recall) |

### NearbyFences Full Pipeline (nearbyfenceslogic.go + fencestore.go)

```
Input: (lon, lat, km)
  ↓
1. kmToH3RecallK(km) → k (ring count)
2. h3.LatLngToCell(res=9) → origin cell
3. h3.GridDisk(origin, k) → candidate cells
4. DB: SELECT fence_id FROM cells WHERE cell_type='h3_r9' AND cell_id IN (candidates)
5. Deduplicate fence IDs
  ↓ (back in logic layer)
6. For each fence ID: LoadFencePolygon() → orb.Polygon
7. For each polygon: planar.PolygonContains(polygon, point)
8. Return hit fence IDs
```

## Patterns Worth Documenting in Spec

1. **Resolution 9 recall tradeoff**: Using a lower resolution (9) for recall means each cell is ~0.2km across. This creates a coarse but fast spatial index. The tradeoff is that nearby fences returned are a superset — exact filtering happens in logic layer with full polygon containment.

2. **Dual cell storage**: Each fence stores both exact-fidelity cells (at user-specified resolution) AND recall cells (at fixed resolution 9). This decouples query performance from fence creation parameters.

3. **GridDisk expansion**: `kmToH3RecallK(km)` converts a distance in km to a ring count. This is NOT a precise distance — GridDisk rings are topological, not metric. The comment `h3RecallAverageEdgeKm = 0.2` indicates the conversion approximation.

## Caveats

- `computeH3RecallCells` is not shown in the read range — it's likely defined after line 290 in fencestore.go
- The recall mechanism produces false positives (cells may overlap fences that are slightly beyond km range) — the exact polygon containment step (step 7) handles this
