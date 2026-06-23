# Research: JSON Error Handling & Proto Comments

- **Query**: Read fencestore.go L240-290 for JSON error handling changes; read gis.proto L45-50 and L90-93 for Fence message and PointInFences comments
- **Scope**: internal
- **Date**: 2026-06-23

## Findings

### fencestore.go — JSON Error Handling Asymmetry

Two different JSON unmarshal error strategies in the same store:

#### ListFences (batch path, lines 247-249)

```go
if err := json.Unmarshal([]byte(f.Points), &poly); err != nil {
    logx.Errorf("解析围栏顶点JSON失败, fenceId=%s, err=%v", f.FenceId, err)
    // continues with nil polygon — does NOT return error
}
```

**Behavior**: Silent degradation. If a single fence's JSON is corrupt, that fence gets a nil polygon in the result list, but the batch operation continues. The caller gets a partial result set.

#### GetFence (single path, lines 273-275)

```go
if err := json.Unmarshal([]byte(fence.Points), &poly); err != nil {
    return nil, fmt.Errorf("解析围栏顶点JSON失败: %w", err)
    // returns error, wrapped for caller
}
```

**Behavior**: Hard failure. If the fence's JSON is corrupt, returns error immediately.

### gis.proto — Fence Message & Comments

#### Fence Message (lines 90-93)

```protobuf
message Fence {
  string fence_id = 1;          // 围栏 ID（从 store 查询判断时必填）
  repeated Point points = 2;    // 多边形顶点（主动判断时必填，至少 3 个点）
}
```

**Comment pattern**: Chinese comments describing the two-mode contract. `fence_id` is required for DB-lookup mode, `points` is required for inline-judgment mode. This exactly maps to the logic in `pointinfenceslogic.go` lines 46-60.

#### PointInFences RPC Comments (lines 46-48)

```protobuf
// 点是否命中电子围栏（多个围栏）。支持两种模式：上送 points 主动判断，或上送 fence_id 从 store 查询判断。
rpc PointInFences (PointInFencesReq) returns (PointInFencesRes);
```

The proto comment describes two modes:
1. "上送 points 主动判断" (send points for active judgment)
2. "上送 fence_id 从 store 查询判断" (send fence_id for store lookup judgment)

This is consistent with the implementation's `if len(fence.Points) > 0` / `else if fence.FenceId != ""` branching.

## Patterns Worth Documenting in Spec

1. **JSON error asymmetry is intentional**: Batch operations degrade gracefully (skip corrupt records), single-record operations fail fast. This is a common pattern in list-vs-get APIs.

2. **Proto two-mode contract**: The Fence message is designed for dual use (inline points OR DB reference). The comment convention uses Chinese with explicit "必填" (required) qualifiers to document mode-specific requirements.

3. **Proto comment language**: Chinese comments in proto files for field descriptions. This may be a project convention worth noting.

## Caveats

- The `FenceDetail` message (line 366-376) is a different structure from `Fence` — it always has both `points` and metadata fields (no two-mode ambiguity). `FenceDetail` is output-only.
- `Fence` (line 90) is input-only and supports the two-mode contract.
