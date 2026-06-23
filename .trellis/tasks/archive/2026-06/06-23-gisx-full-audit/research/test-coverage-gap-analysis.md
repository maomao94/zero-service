# Research: Test Coverage Gap Analysis for `common/gisx/`

- **Query**: Thorough test coverage gap analysis for the `common/gisx/` package (all 15 files)
- **Scope**: internal
- **Date**: 2026-06-23

## Findings

### Legend

- **COVERED**: Has at least one test exercising the function with assertions on the result
- **MISSING**: No test at all
- **PARTIAL**: Test exists but only exercises happy path; error paths, nil inputs, edge cases not covered
- **INDIRECT**: Only tested as a dependency of another tested function (no dedicated test)

---

### 1. `common/gisx/gisx.go` (3 exported functions)

| Function | Status | Notes |
|---|---|---|
| `OrbPolygonToH3GeoPolygon` | COVERED | Tests: happy path, with hole, empty polygon error, <3 points error |
| `IsOrbPointsEqual` | COVERED | Tests: equal, within epsilon, different |
| `OrbRingToH3LatLng` | COVERED | Tests: auto-close, empty ring, already closed |

**Missing edge cases:**
- `OrbPolygonToH3GeoPolygon`: Hole with <3 points (skipped silently per code, not validated in test)
- No nil/negative resolution test for h3 functions

---

### 2. `common/gisx/validate.go` (1 exported function)

| Function | Status | Notes |
|---|---|---|
| `ValidateCoordinate` | COVERED | Tests: valid coords, lat out of range (±90.1), lon out of range (±180.1) |

**Missing edge cases:**
- `idx` parameter always passed as `0` in tests; error message format with different `idx` values not validated
- `validationError` type is unexported but not tested for `errors.As` compatibility

---

### 3. `common/gisx/store.go` (1 interface + 1 struct + 1 var)

| Item | Type | Status | Notes |
|---|---|---|---|
| `FenceInfo` | struct | INDIRECT | Data struct, no independent test |
| `FenceStore` | interface (9 methods) | N/A | Interface definition; implementation testing needed |
| `NoopFenceStore.CreateFence` | method | MISSING | |
| `NoopFenceStore.LoadFencePolygon` | method | MISSING | |
| `NoopFenceStore.FindNearbyFenceIds` | method | MISSING | |
| `NoopFenceStore.FindFenceIdsByCellIds` | method | MISSING | |
| `NoopFenceStore.UpdateFence` | method | MISSING | |
| `NoopFenceStore.RemoveFence` | method | MISSING | |
| `NoopFenceStore.ListFences` | method | MISSING | |
| `NoopFenceStore.GetFence` | method | MISSING | |
| `ErrFenceStoreNotImplemented` | var | MISSING | Only used by NoopFenceStore (untested) |

**Why significant:** All 8 `NoopFenceStore` methods are completely untested. Each should return `ErrFenceStoreNotImplemented`, but no test verifies this contract.

---

### 4. `common/gisx/geos/construct.go` (12 exported functions)

| Function | Status | Notes |
|---|---|---|
| `NewPoint` | COVERED | TestConstruct/Point |
| `NewLineString` | COVERED | TestLineStringLinearRing |
| `NewLinearRing` | COVERED | TestLineStringLinearRing (auto-close tested implicitly via NewPolygon) |
| `NewPolygon` | COVERED | TestConstruct/Polygon, used pervasively |
| `NewEmptyPoint` | MISSING | No test |
| `NewEmptyLineString` | MISSING | No test |
| `NewEmptyPolygon` | MISSING | No test |
| `NewEmptyCollection` | MISSING | No test |
| `NewMultiPolygon` | COVERED | TestExtract/ExtractMulti, TestExtract/ExtractPolygonOrMultiCoords |
| `NewMultiPolygonFromGeoms` | INDIRECT | Tested via orbconv MultiPolygonToGeom |
| `NewCollectionFromGeoms` | INDIRECT | Tested via NewMultiPolygonFromGeoms wrapper |
| `NewBoundsRect` | COVERED | TestConstruct/BoundsRect |

**Missing:**
- All 4 `NewEmpty*` functions completely untested
- `NewMultiPolygonFromGeoms`: Single-geom path (should return Polygon directly) not explicitly tested
- `NewCollectionFromGeoms`: Nil/empty geoms slice error path not tested
- `NewBoundsRect`: Degenerate rect (min==max) not tested

---

### 5. `common/gisx/geos/convert.go` (6 exported functions)

| Function | Status | Notes |
|---|---|---|
| `FromWKT` | COVERED | TestWKTConvert (valid + invalid) |
| `ToWKT` | COVERED | TestWKTConvert (round-trip) |
| `FromWKB` | COVERED | TestWKBConvert (round-trip) |
| `ToWKB` | COVERED | TestWKBConvert |
| `FromGeoJSON` | COVERED | TestGeoJSONConvert (round-trip) |
| `ToGeoJSON` | COVERED | TestGeoJSONConvert (indent=0 only) |

**Missing edge cases:**
- `FromGeoJSON`: Invalid/malformed GeoJSON not tested
- `ToGeoJSON`: Non-zero indent not tested; nil geometry not tested
- `FromWKB`: Invalid/corrupt binary data not tested
- `ToWKT`/`ToWKB`: nil geometry panic recovery not tested

---

### 6. `common/gisx/geos/extract.go` (10 exported functions/types)

| Function | Status | Notes |
|---|---|---|
| `ExtractPoint` | COVERED | TestExtract/ExtractPoint |
| `ExtractCoords` | COVERED | TestExtract/ExtractCoords_LineString |
| `ExtractPolygonCoords` | COVERED | TestExtract/ExtractPolygonCoords |
| `ExtractPolygonOrMultiCoords` | COVERED | TestExtract/ExtractPolygonOrMultiCoords |
| `PolygonData.Outer()` | COVERED | (indirect via ExtractPolygonCoords) |
| `PolygonData.Holes()` | COVERED | (indirect via ExtractPolygonCoords hole count assertion) |
| `ExtractMulti[T]` | COVERED | TestExtract/ExtractMulti (with PolygonData) |
| `ExtractMultiSafe[T]` | MISSING | No test at all |
| `ExtractPoints` | COVERED | TestExtract/ExtractPoints (Point type only) |

**Missing edge cases:**
- `ExtractPoint`: nil input (code handles it, but not tested)
- `ExtractCoords`: nil input (returns nil, nil — not tested)
- `ExtractPolygonCoords`: nil input (returns errNil — not tested)
- `ExtractMulti`: nil input, GeometryCollection rejection not tested
- `ExtractMultiSafe`: Completely untested (nil input, mixed-type GeometryCollection)
- `ExtractPoints`: MultiPoint, LineString type paths not tested; nil input not tested
- `ExtractPolygonOrMultiCoords`: Unsupported type (non-Polygon, non-MultiPolygon) error path not tested

---

### 7. `common/gisx/geos/predicate.go` (11 exported functions)

| Function | Status | Notes |
|---|---|---|
| `Intersects` | COVERED | TestPredicates/Intersects (overlapping + disjoint) |
| `Contains` | COVERED | TestPredicates/Contains, ContainsVsCovers (boundary vs interior) |
| `Covers` | COVERED | TestPredicates/Covers, ContainsVsCovers |
| `Within` | COVERED | TestMissingPredicates/Within |
| `Touches` | COVERED | TestPredicates/Touches |
| `Disjoint` | COVERED | TestMissingPredicates/Disjoint |
| `Equals` | COVERED | TestMissingPredicates/Equals (self-equality only) |
| `Overlaps` | COVERED | TestMissingPredicates/Overlaps |
| `Crosses` | PARTIAL | TestMissingPredicates/Crosses — only checks err != nil, not the boolean result |
| `CoveredBy` | COVERED | TestMissingPredicates/CoveredBy |
| `EqualsExact` | COVERED | TestMissingPredicates/EqualsExact (self-equality with tolerance) |

**Missing edge cases:**
- ALL predicates: No nil-input tests (code handles via `predicateTwo`, but never tested)
- `Equals`: Different-but-same-shape geometries not tested (only self-equality)
- `Crosses`: Only error check, no assertion on expected boolean value (line-crossing-polygon semantic not validated)
- `EqualsExact`: Points that differ just past tolerance not tested

---

### 8. `common/gisx/geos/prepared.go` (1 type + 13 methods)

| Function/Method | Status | Notes |
|---|---|---|
| `NewPreparedGeom` | COVERED | TestPrepared, TestPreparedFull |
| `Close()` | COVERED | (defer in tests) |
| `Intersects` | COVERED | TestPrepared (overlapping + disjoint) |
| `Contains` | MISSING | Only `ContainsXY` tested; `Contains(other Geom)` not tested |
| `ContainsXY` | COVERED | TestPrepared (interior + boundary semantics) |
| `Covers` | COVERED | TestPreparedFull/Covers |
| `IntersectsXY` | COVERED | TestPrepared |
| `Disjoint` | MISSING | No test at all |
| `CoveredBy` | PARTIAL | TestPreparedFull/CoveredBy — only checks err != nil, not the result |
| `Overlaps` | COVERED | TestPreparedFull/Overlaps |
| `Touches` | PARTIAL | TestPreparedFull/Touches — only checks err != nil, not the result |
| `Within` | PARTIAL | TestPreparedFull/Within — only checks err != nil, not the result |
| `DistanceWithin` | COVERED | TestPreparedFull/DistanceWithin |

**Missing edge cases:**
- `NewPreparedGeom`: nil input not tested
- `Close()`: Double-close safety not tested
- `Contains(other Geom)`: Completely untested (only ContainsXY tested)
- `Disjoint`: Completely untested
- Any method called on a closed/nil PreparedGeom not tested for error handling

---

### 9. `common/gisx/geos/overlay.go` (37 exported functions)

| Function | Status | Notes |
|---|---|---|
| `Intersection` | COVERED | TestOverlay/Intersection (overlapping + no-intersection) |
| `Union` | COVERED | TestOverlay/Union (subsumed geometry) |
| `Difference` | COVERED | TestOverlayAll/Difference |
| `SymDifference` | COVERED | TestOverlayAll/SymDifference |
| `UnaryUnion` | COVERED | TestAdvancedTransforms/UnaryUnion |
| `Envelope` | COVERED | TestAdvancedTransforms/Envelope |
| `Boundary` | COVERED | TestAdvancedTransforms/Boundary |
| `BuildArea` | MISSING | No test |
| `LineMerge` | MISSING | No test |
| `Node` | MISSING | No test |
| `MinimumRotatedRectangle` | COVERED | TestAdvancedTransforms/MinimumRotatedRectangle |
| `IsValid` | COVERED | TestValid (valid + invalid) |
| `IsValidReason` | COVERED | TestValid |
| `MakeValid` | COVERED | TestValid |
| `Area` | COVERED | TestMeasure (polygon) |
| `Length` | COVERED | TestMeasure (polygon) |
| `Distance` | MISSING | Not tested directly (only DistanceWithin and FrechetDistance are tested) |
| `Centroid` | COVERED | TestMeasure |
| `PointOnSurface` | COVERED | TestMeasure |
| `Buffer` | COVERED | TestSimplify |
| `Simplify` | MISSING | Function exists but not tested (TestSimplify only tests Buffer + ConvexHull) |
| `TopologyPreserveSimplify` | MISSING | No test |
| `ConvexHull` | COVERED | TestSimplify |
| `ConcaveHull` | COVERED | TestOverlayAll/ConcaveHull |
| `Normalize` | PARTIAL | TestAdvancedFuncs/Normalize — result discarded (`_ = must(...)`) |
| `Reverse` | PARTIAL | TestAdvancedTransforms/Reverse — result discarded |
| `Snap` | PARTIAL | TestAdvancedTransforms/Snap — result discarded |
| `ClipByRect` | COVERED | TestOverlayAll/ClipByRect |
| `Densify` | COVERED | TestAdvancedTransforms/Densify |
| `OffsetCurve` | MISSING | No test |
| `EndPoint` | MISSING | No test |
| `StartPoint` | MISSING | No test |
| `MinimumClearance` | MISSING | No test |
| `SRID` | COVERED | TestAdvancedFuncs/SRID |
| `SetSRID` | COVERED | TestAdvancedFuncs/SRID |
| `Precision` | MISSING | No test |
| `FrechetDistance` | COVERED | TestAdvancedTransforms/FrechetDistance |

**Missing edge cases (for COVERED functions):**
- `Intersection`: nil input not tested
- `Union`: nil input not tested
- `IsValid`: nil input not tested
- `Area`/`Length`: Point, LineString types not tested; nil input not tested
- `ConcaveHull`: `allowHoles=true` not tested (only tested with `false`)
- `Densify`: zero tolerance not tested
- All overlay functions: nil geometry inputs not tested

---

### 10. `common/gisx/geos/relation.go` (5 exported functions)

| Function | Status | Notes |
|---|---|---|
| `Relate` | COVERED | TestAdvancedFuncs/Relate (non-empty string check) |
| `RelatePattern` | MISSING | No test |
| `DistanceWithin` | COVERED | TestAdvancedFuncs/DistanceWithin |
| `HausdorffDistance` | COVERED | TestAdvancedFuncs/Hausdorff (just logs value) |
| `NearestPoints` | MISSING | No test |

**Missing edge cases:**
- `Relate`: nil input not tested; specific DE-9IM matrix value not asserted (only checks non-empty)
- `RelatePattern`: Completely untested
- `DistanceWithin`: nil input not tested; threshold exactly at distance not tested
- `HausdorffDistance`: Expected value never asserted (only logged); nil input not tested
- `NearestPoints`: Completely untested

---

### 11. `common/gisx/geos/introspect.go` (5 exported functions)

| Function | Status | Notes |
|---|---|---|
| `IsEmpty` | COVERED | TestIntrospection/IsEmpty (non-empty polygon) |
| `IsSimple` | COVERED | TestIntrospection/IsSimple (non-empty polygon → true) |
| `IsClosed` | COVERED | TestIntrospection/IsClosed (non-closed LineString → false) |
| `IsRing` | COVERED | TestIntrospection/IsRing (LinearRing → true) |
| `HasZ` | COVERED | TestIntrospection/HasZ (2D polygon → false) |

**Missing edge cases:**
- `IsEmpty`: nil input (returns true — not tested); empty geometry (not tested)
- `IsSimple`: Self-intersecting LineString not tested (only polygon tested)
- `IsClosed`: nil input not tested; Closed LineString not tested; Polygon input (should error) not tested
- `IsRing`: nil input not tested; Non-ring LineString not tested
- `HasZ`: 3D geometry (should return true) not tested; nil input not tested

---

### 12. `common/gisx/geos/strtree.go` (1 type + 6 methods)

| Function/Method | Status | Notes |
|---|---|---|
| `NewSTRtree` | COVERED | TestSTRtree, TestSTRtreeFull |
| `Insert` | COVERED | TestSTRtree, TestSTRtreeFull |
| `Query` | COVERED | TestSTRtree (positive + no-match) |
| `Iterate` | COVERED | TestSTRtreeFull/Iterate (count=2) |
| `Remove` | COVERED | TestSTRtreeFull/Remove (success + double-remove=false) |
| `Close()` | COVERED | (defer in tests) |

**Missing edge cases:**
- `Insert`/`Query`/`Iterate`/`Remove`: Called on nil/closed tree (code handles, not tested)
- `Query`: Query with no inserted geometries (empty tree)
- `Iterate`: Empty tree (should return nil error with zero iterations)
- `Remove`: Removing non-existent item in a non-empty tree (only tested after first remove)
- `NewSTRtree`: Unusual nodeCapacity values (0, 1, negative) not tested

---

### 13. `common/gisx/geos/context.go` (2 exported functions)

| Function | Status | Notes |
|---|---|---|
| `GEOSVersion` | COVERED | TestGEOSVersion (major > 0) |
| `GEOSVersionString` | COVERED | TestGEOSVersion (non-empty) |

Unexported helpers (`safeRun`, `safeRunErr`, `zeroValue`, `getDefaultContext`) tested indirectly pervasively.

---

### 14. `common/gisx/geos/errors.go` (7 exported vars)

| Item | Status | Notes |
|---|---|---|
| `ErrNil` | INDIRECT | Used pervasively; tested indirectly via nil-input tests on extract.go |
| `ErrClosed` | MISSING | No test exercises any code path returning ErrClosed |
| `ErrNotPolygon` | MISSING | No test verifies ErrNotPolygon is returned |
| `ErrEmptyRing` | INDIRECT | Tested via orbconv nil tests |
| `ErrEmptyOuterRing` | INDIRECT | Tested via orbconv empty polygon tests |
| `ErrNotSupported` | MISSING | No test exercises any code path returning ErrNotSupported |
| `ErrEmptyGeoms` | MISSING | No test exercises error path in NewCollectionFromGeoms with empty slice |

**Note:** These are sentinel error variables. Coverage depends on whether the code paths that return them are exercised. `ErrClosed`, `ErrNotPolygon`, `ErrNotSupported`, `ErrEmptyGeoms` are defined but have no test coverage.

---

### 15. `common/gisx/geos/orbconv/orbconv.go` (20 exported functions)

#### Conversion: orb → GEOS (7 functions)

| Function | Status | Notes |
|---|---|---|
| `PointToGeom` | COVERED | TestConversion/PointToGeom |
| `RingToGeom` | COVERED | TestConversion/RingToGeom |
| `LineStringToGeom` | MISSING | No test |
| `PolygonToGeom` | COVERED | TestConversion/PolygonToGeom, TestMultiPolygonConversion |
| `MultiPolygonToGeom` | COVERED | TestMultiPolygonConversion (2 polys + nil/empty errors) |
| `MultiPointToGeom` | MISSING | No test |
| `MultiLineStringToGeom` | MISSING | No test |

#### Conversion: GEOS → orb (7 functions)

| Function | Status | Notes |
|---|---|---|
| `GeomToPoint` | MISSING | No test |
| `GeomToRing` | COVERED | TestConversion/RingToGeom (round-trip) |
| `GeomToLineString` | MISSING | No test |
| `GeomToPolygon` | COVERED | TestConversion/GeomToPolygon, TestMultiPolygonConversion (MultiPolygon takes first) |
| `GeomToMultiPolygon` | COVERED | TestMultiPolygonConversion (single polygon, multi polygon, with holes) |
| `GeomToMultiPoint` | MISSING | No test |
| `GeomToMultiLineString` | MISSING | No test |

#### Convenience wrappers (6 functions)

| Function | Status | Notes |
|---|---|---|
| `IntersectsOrb` | COVERED | TestPredicates/IntersectsOrb (overlapping + disjoint) |
| `ContainsOrb` | COVERED | TestPredicates/ContainsOrb |
| `CoversOrb` | COVERED | TestPredicates/CoversOrb |
| `CoversPointOrb` | COVERED | TestPredicates/CoversPointOrb (boundary point → true) |
| `ContainsPointOrb` | COVERED | TestPredicates/ContainsPointOrb (boundary point → false) |
| `ValidOrb` | COVERED | TestPredicates/ValidOrb |

**Missing edge cases:**
- `GeomToPolygon`: Non-Polygon/MultiPolygon type error path not tested
- `GeomToMultiPolygon`: Non-Polygon/MultiPolygon type error path not tested
- `GeomToMultiPoint`: Completely untested
- `GeomToMultiLineString`: Completely untested
- `GeomToPoint`: Completely untested (even though GeomToRing is tested)
- `GeomToLineString`: Completely untested
- `LineStringToGeom`: Completely untested (empty LineString error not tested)
- `MultiPointToGeom`: Completely untested
- `MultiLineStringToGeom`: Completely untested
- `PolygonToGeom`: Invalid ring (unclosed 3-point ring) not tested
- All convenience wrappers: nil/empty input error handling not tested

---

## Summary Statistics

| File | Total Exported | Covered | Missing | Partial |
|---|---|---|---|---|
| `gisx.go` | 3 | 3 | 0 | 0 |
| `validate.go` | 1 | 1 | 0 | 0 |
| `store.go` (impl) | 8 methods | 0 | 8 | 0 |
| `construct.go` | 12 | 8 | 4 | 0 |
| `convert.go` | 6 | 6 | 0 | 0 |
| `extract.go` | 10 | 9 | 1 | 0 |
| `predicate.go` | 11 | 10 | 0 | 1 |
| `prepared.go` | 14 | 9 | 2 | 3 |
| `overlay.go` | 37 | 27 | 9 | 1 |
| `relation.go` | 5 | 3 | 2 | 0 |
| `introspect.go` | 5 | 5 | 0 | 0 |
| `strtree.go` | 7 | 7 | 0 | 0 |
| `context.go` | 2 | 2 | 0 | 0 |
| `errors.go` | 7 vars | 3 | 4 | 0 |
| `orbconv.go` | 20 | 12 | 8 | 0 |
| **TOTAL** | **148** | **105** | **38** | **5** |

**Coverage rate: 105/148 = 70.9% of exported API items have direct test coverage.**

## Critical Gaps Summary

**Untested files/areas (highest priority):**

1. **`store.go` — ALL 8 NoopFenceStore methods MISSING** (0% coverage)
2. **`overlay.go` — 9 untested functions:** BuildArea, LineMerge, Node, Distance, Simplify, TopologyPreserveSimplify, OffsetCurve, EndPoint, StartPoint, MinimumClearance, Precision
3. **`orbconv.go` — 8 untested conversion functions:** GeomToPoint, GeomToLineString, GeomToMultiPoint, GeomToMultiLineString, LineStringToGeom, MultiPointToGeom, MultiLineStringToGeom
4. **`construct.go` — 4 untested empty constructors:** NewEmptyPoint/LineString/Polygon/Collection
5. **`prepared.go` — 2 completely untested methods** (Contains on Geom, Disjoint); 3 partial

**Nil/error edge case pattern (across ALL files):** Most functions have internal nil checks but zero nil-input tests. This is a systemic gap.

**Functions with result assertions missing (partial):** Crosses, Normalize, Reverse, Snap, CoveredBy (prepared), Touches (prepared), Within (prepared)

## Related Specs

- `.trellis/spec/` — project coding guidelines (none specific to gisx testing)

## Caveats / Not Found

- The `doc.go` files are package documentation only (no functions to test)
- The `geos/doc.go` file is package documentation
- All analysis covers only **exported** API surface; unexported helpers (`safeRun`, `predicateTwo`, `overlayTwo`, `transformOne`, `oneBool`, `oneFloat`, `oneInt`, `ensureRingClosed`, `ringToCoords`, `singlePolyToOrb`, `both`) are tested indirectly
- GEOS C library panic recovery (`safeRun`) is tested once via `TestPanicRecover`, but only for WKT parsing — not for other scenarios like nil geometry operations
- No integration-level tests exist that compose multiple operations (e.g., Construct → Buffer → Intersects → Extract → Convert)
