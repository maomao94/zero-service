# Research: Validation/Computation Flow Differences

- **Query**: Compare current logic files against expected patterns from old spec descriptions; note where inline code duplicates helper functions
- **Scope**: internal
- **Date**: 2026-06-23

## Findings

### 1. `createfencelogic.go` — Clean Delegation (Reference Pattern)

Flow: `pbPointToOrbPolygon` → `resolveH3Resolution` → `resolveGeohashPrecision` → `computeFenceCells` → `FenceStore.CreateFence`

- **Correctly** delegates ALL validation/resolution to helper functions
- No inline duplication of default values or range checks
- This is the cleanest logic file and should serve as the spec reference pattern

### 2. `generatefencecellslogic.go` — Inline Duplication

**Duplication**: Lines 33-37 duplicate `resolveGeohashPrecision` logic:

```go
precision := int(in.Precision)
if precision <= 0 {
    precision = 7
} else if precision > 12 {
    return nil, tool.NewErrorByPbCode(extproto.Code__1_01_PARAM, "geohash精度最大为12")
}
```

This is functionally identical to `resolveGeohashPrecision` (helper.go:51-58) but:
- Uses `Code__1_01_PARAM` instead of `Code__1_01_PARAM` (same code, different error message wording)
- Does NOT call the helper function
- Only difference: helper says "geohash精度最大为12", inline says "geohash精度最大为12"

**Recommendation for spec**: Either call `resolveGeohashPrecision` or explain why the inline version is needed.

### 3. `generatefenceh3cellslogic.go` — Inline Duplication

**Duplication**: Lines 40-46 duplicate `resolveH3Resolution` logic:

```go
resolution := in.Resolution
if resolution == 0 {
    resolution = 9
}
if resolution > 15 {
    return nil, tool.NewErrorByPbCode(extproto.Code__1_01_PARAM, "H3分辨率必须在0-15之间")
}
```

This is functionally identical to `resolveH3Resolution` (helper.go:40-48) but:
- Does NOT call the helper function
- Misses the doc comment about proto3 zero-value ambiguity
- Uses `Code__1_01_PARAM` vs helper which uses `Code__1_01_PARAM` (same)
- Importantly: `in.Resolution` is `uint32` — the inline code does `if resolution == 0` which can't distinguish "not set" from "0" (same as helper's caveat)

**Recommendation for spec**: Document that this file should call `resolveH3Resolution` to stay DRY.

### 4. `pointinfenceslogic.go` — Two-Mode Logic

Two dispatch modes in the loop (lines 46-60):

| Mode | Condition | Action |
|---|---|---|
| Inline points | `len(fence.Points) > 0` | `pbPointToOrbPolygon(fence.Points)` |
| DB lookup | `fence.FenceId != ""` | `FenceStore.LoadFencePolygon()` |
| Skip | Both empty | `continue` |

**Notable**: The FenceId filter on line 62 (`fence.FenceId != ""`) means inline-points-only fences with no ID are evaluated but never included in results.

### 5. `nearbyfenceslogic.go` — Error Handling Strategy

Two different error handling strategies in the SAME function:

| Step | Error Handling | Rationale |
|---|---|---|
| `FindNearbyFenceIds` (line 40) | **Returns error immediately** | Fatal: spatial index lookup failed |
| `LoadFencePolygon` (line 49) | **Logs and continues** | Non-fatal: one polygon may be corrupt, try others |

This is a good pattern to document: spatial index failures are fatal (no results possible), individual polygon load failures are non-fatal (continue with remaining candidates).

## Key Flow Differences Summary

| Logic File | Uses Helpers? | Inline Duplication? |
|---|---|---|
| `createfencelogic.go` | YES (clean) | No |
| `generatefencecellslogic.go` | Partial (uses `pbPointToOrbPolygon`, `scanGeohashCells`) | YES (precision validation at L33-37) |
| `generatefenceh3cellslogic.go` | Partial (uses `pbPointToOrbPolygon`) | YES (resolution validation at L40-46) |
| `pointinfenceslogic.go` | Partial (uses `ValidatePoints`, `pbPointToOrbPolygon`) | No (no resolution/precision needed) |
| `nearbyfenceslogic.go` | Partial (uses `ValidatePoints`) | No (no resolution/precision needed) |

## Caveats

- `computeFenceCells` could theoretically be used by `generatefenceh3cellslogic.go` but isn't — the logic file does its own H3 conversion and cell computation inline
- The `generatefencecellslogic.go` uses `scanGeohashCells` directly (which is correct), but duplicates the precision default logic
