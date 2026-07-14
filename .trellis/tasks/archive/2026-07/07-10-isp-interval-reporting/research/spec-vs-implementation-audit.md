# Research: Spec vs Implementation Audit (isp-guidelines.md correctness)

- **Query**: Compare `.trellis/spec/backend/isp-guidelines.md` against current `reporting.go` + `client.go` + `ispagent.proto`
- **Scope**: internal
- **Date**: 2026-07-14

## Summary

The spec is approximately 85% accurate. The core "巡视装置上报缓存" section (lines 88-111) is remarkably precise and aligns perfectly with the implementation. The gaps are mostly **omissions of new patterns** rather than incorrect statements.

---

## Section 1: ACCURATE — No Changes Needed

These sections still match the implementation:

| Spec Lines | Section | Verdict |
|---|---|---|
| 1-23 | Protocol overview, frame format, messageId | ✅ Accurate |
| 24-36 | Package structure (`common/isp/`) | ✅ Accurate |
| 37-48 | Encoding/decoding, gnetx Codec | ✅ Accurate |
| 49-56 | Message model (Identifiable, Correlatable, SendSeq/RecvSeq) | ✅ Accurate |
| 57-60 | Item convention (`map[string]string`) | ✅ Accurate |
| 72-73 | Heartbeat ≠ report interval ≠ cache freshness | ✅ Accurate |
| 74-77 | Response convention (251-3, responseWithCode) | ✅ Accurate |
| 78-85 | Common errors table | ✅ Accurate |
| 88-102 | **巡视装置上报缓存** — entire section | ✅ Accurate (see Section 3 for minor terminology gap) |
| 103-110 | 巡视上报缓存测试契约 | ✅ Accurate |
| 112-119 | Client lifecycle (shutdown listener, serviceGroup vs Client) | ✅ Accurate |
| 121-123 | Source file references (ispagent.go, servicecontext.go) | ✅ Accurate |
| 125-133 | 汉化映射 (names.go) | ✅ Accurate |
| 138-146 | 巡视任务持久化 (FirstOrCreate + Assign) | ✅ Accurate |
| 148-151 | carbon 时间格式化 | ✅ Accurate |

---

## Section 2: OUTDATED — Needs Updates

### 2a. Registration response table (lines 66-71)

**Current spec text:**

```
| patroldevice_run_interval | 巡视装置运行数据周期上报间隔 |
| nest_run_interval | 无人机机巢运行数据间隔，未实现具体上报时也应保留在本地间隔状态 |
| weather_interval | PatrolDevice / PatrolHost | 微气象数据间隔，未实现具体上报时也应保留在本地间隔状态 |
```

**Problem:** The table is factually correct but doesn't document the **code-level category mapping** now implemented in `applyRegistrationIntervals()` (reporting.go:202-223):

- `patroldevice_run_interval` → `ReportCategoryPatrolDeviceRunData`
- `nest_run_interval` → `ReportCategoryDroneNestRunData`
- `weather_interval` → `ReportCategoryEnvData`

The current implementation **does** send drone nest and environment data upstream (the spec says these are "未实现具体上报时也应保留"), so those caveats are now stale.

**Recommended replacement text:**

```markdown
| 字段 | 用途 | 映射到的 ReportCategory |
|------|------|------------------------|
| `heart_beat_interval` | 只覆盖系统心跳间隔 | (none — heartbeat is not a report category) |
| `patroldevice_run_interval` | 巡视装置运行数据刷新间隔 | `ReportCategoryPatrolDeviceRunData` (2-0) |
| `nest_run_interval` | 无人机机巢运行数据刷新间隔 | `ReportCategoryDroneNestRunData` (10004-0) |
| `weather_interval` | 环境/微气象数据刷新间隔 | `ReportCategoryEnvData` (21-0) |

解析入口：`reportManager.applyRegistrationIntervals()`，字段缺失或非法时保持当前值不变。
解析后**全部缓存的 `lastSent` 置零**，使注册成功后立即触发一次完整上报。
```

### 2b. Line 94 — Category list is incomplete

**Current text:**

> `patroldevice_run_interval` 只驱动巡视装置运行数据；状态数据、坐标/经纬度属于独立 ISP 上报类别，注册协议未给出对应频率时使用 report spec 中定义的默认 1 分钟间隔。

**Problem:** Only mentions run/status/coord categories. Now there are 5.

**Recommended addition after line 94:**

```markdown
- 当前共 5 个上报类别（见 `reportManager.cache`）：巡视装置运行数据（2-0）、巡视装置状态数据（1-0）、巡视装置坐标（3-0）、无人机机巢运行数据（10004-0）、环境/微气象数据（21-0）。
- `nest_run_interval` 驱动无人机机巢运行数据；`weather_interval` 驱动环境/微气象数据。注册协议未给出对应频率的类别（状态数据）使用默认 1 分钟间隔。
```

### 2c. Line 96 — keyAttrs list incomplete

**Current text:**

> 当前巡视设备类：运行/状态使用 `patroldevice_code + type`，坐标使用 `patroldevice_code`

**Problem:** Missing drone nest (`nest_code + type`) and env data (`patroldevice_code + type`).

**Recommended replacement:**

```markdown
当前 keyAttrs 定义（见 `keyAttrsByCategory`）：
- 运行/状态：`patroldevice_code + type`
- 坐标：`patroldevice_code`
- 无人机机巢：`nest_code + type`
- 环境/微气象：`patroldevice_code + type`
```

---

## Section 3: NEW PATTERNS — Not Documented in Spec

These implementation details exist in the code but are entirely absent from the spec. They should be added as new subsections under "巡视装置上报缓存".

### 3a. ReportCategory types (reporting.go:18-26)

```go
type ReportCategory int

const (
    ReportCategoryPatrolDeviceRunData     = ReportCategory(isp.MessageIDPatrolDeviceRunData)     // 2-0
    ReportCategoryPatrolDeviceStatusData  = ReportCategory(isp.MessageIDPatrolDeviceStatusData)  // 1-0
    ReportCategoryPatrolDeviceCoordinates = ReportCategory(isp.MessageIDPatrolDeviceCoordinates) // 3-0
    ReportCategoryDroneNestRunData        = ReportCategory(isp.MessageIDDroneNestRunData)        // 10004-0
    ReportCategoryEnvData                 = ReportCategory(isp.MessageIDEnvData)                 // 21-0
)
```

`ReportCategory` is `(Type<<16)|Command` from `common/isp` constants, so it directly maps to ISP messageId.

**Spec text to add:**

```markdown
### ReportCategory 定义

5 个内置上报类别，值等于 `common/isp` 中的 `MessageID` 常量（`(Type<<16)|Command`）：
- `ReportCategoryPatrolDeviceRunData` — 巡视装置运行数据（2-0）
- `ReportCategoryPatrolDeviceStatusData` — 巡视装置状态数据（1-0）
- `ReportCategoryPatrolDeviceCoordinates` — 巡视装置坐标（3-0）
- `ReportCategoryDroneNestRunData` — 无人机机巢运行数据（10004-0）
- `ReportCategoryEnvData` — 环境/微气象数据（21-0）
```

### 3b. keyAttrsByCategory mapping (reporting.go:37-43)

```go
var keyAttrsByCategory = map[ReportCategory][]string{
    ReportCategoryPatrolDeviceRunData:     {"patroldevice_code", "type"},
    ReportCategoryPatrolDeviceStatusData:  {"patroldevice_code", "type"},
    ReportCategoryPatrolDeviceCoordinates: {"patroldevice_code"},
    ReportCategoryDroneNestRunData:        {"nest_code", "type"},
    ReportCategoryEnvData:                 {"patroldevice_code", "type"},
}
```

This map controls how `itemKey()` builds unique keys from Item attributes. Key construction falls back to `item_index=N` if all attrs are empty and appends `item_index=N` if any attrs are missing (but not all).

### 3c. ReportManagerOptions functional options pattern (reporting.go:91-125)

```go
type ReportManagerOptions struct {
    RunDataInterval    time.Duration
    StatusDataInterval time.Duration
    CoordInterval      time.Duration
    NestRunInterval    time.Duration
    EnvDataInterval    time.Duration
}

type ReportManagerOption func(*ReportManagerOptions)

func WithRunDataInterval(d time.Duration) ReportManagerOption { ... }
func WithStatusDataInterval(d time.Duration) ReportManagerOption { ... }
// ... etc
```

`newReportManager(opts ...ReportManagerOption)` applies user overrides; zero values fall back to defaults. This allows tests to inject custom intervals. Currently `NewClient` calls `newReportManager()` with no options (all defaults).

### 3d. Default intervals (reporting.go:30-33, 144-166)

```go
const (
    defaultReportInterval = time.Minute   // 1 min — general
    defaultCoordInterval  = 2 * time.Second // 2s — coordinates
)
```

| Category | Default Interval | noFreshCheck |
|---|---|---|
| PatrolDeviceRunData | 1 min | false |
| PatrolDeviceStatusData | 1 min | false |
| PatrolDeviceCoordinates | **2 seconds** | **true** |
| DroneNestRunData | 1 min | false |
| EnvData | 1 min | false |

Coordinates use 2s because position data is high-frequency and stale data is better than no data (noFreshCheck=true). All others default to 1 min.

### 3e. applyRegistrationIntervals pattern (reporting.go:202-223)

Registration field → ReportCategory mapping:

```
patroldevice_run_interval → ReportCategoryPatrolDeviceRunData
nest_run_interval         → ReportCategoryDroneNestRunData
weather_interval          → ReportCategoryEnvData
```

After applying intervals, **all cached `lastSent` timestamps are reset to zero**, ensuring immediate first report after registration. This is critical: without it, a re-registration (e.g., after reconnect) would wait for the old interval to expire before reporting.

### 3f. dueReports two-phase locking (reporting.go:234-280)

The spec says "释放读锁后短写锁删除" but the actual implementation deserves explicit documentation:

```
1. RLock — scan all category→code→items
   - Check shouldReport: lastSent.IsZero() || elapsed >= interval
   - For noFreshCheck categories: cloneAll if shouldReport
   - For normal categories: call freshItems() → (fresh, expired)
     - Append expired items to expired slice
     - If shouldReport, use fresh as snapshot items
   - Collect empty refs (cachedReport with zero items)
2. RUnlock — release read lock
3. Lock — deleteExpired(expired, empty)
   - Delete empty cachedReport refs (with re-check for concurrent refill)
   - Delete expired items (with updatedAt.Equal check to avoid race)
4. Return snapshots
```

### 3g. markSent snapLastSent race protection (reporting.go:317-326)

```go
func (r *reportManager) markSent(category ReportCategory, code string, sentAt time.Time, snapLastSent time.Time) {
    r.mu.Lock()
    defer r.mu.Unlock()
    if report := r.cache[category][code]; report != nil {
        if !snapLastSent.IsZero() && report.lastSent.IsZero() {
            return  // lastSent was reset by a concurrent update — skip
        }
        report.lastSent = sentAt
    }
}
```

When `markSent` is called after a successful send, it checks:
- `snapLastSent` is the `lastSent` value captured at snapshot time
- If `snapLastSent` was non-zero (meaning "not a first report") but current `lastSent` is now zero, a concurrent `update()` reset it (new itemKey arrived), so we **skip** the mark. This prevents markSent from overwriting the zero that was intentionally set to trigger immediate next report.

### 3h. freshItems returning (items, expired) tuple (reporting.go:364-380)

```go
func freshItems(items map[string]*cachedItem, code string, now time.Time, timeout time.Duration) ([]isp.Item, []expiredReportItem)
```

Returns both fresh items (cloned) and expired keys for deferred cleanup. Expired items are logged with Debug level and include `updated_at`, `now`, and `timeout` for diagnostics.

### 3i. CacheReport validation (reporting.go:479-484)

```go
func (c *Client) CacheReport(ctx context.Context, category ReportCategory, code string, items []isp.Item) error {
    if err := c.reports.update(category, code, items, time.Now()); err != nil {
        return err  // unknown category → error
    }
    ...
}
```

`CacheReport` returns an error if the category is not in `keyAttrsByCategory`. The logic layer (e.g., `SendDroneNestRunDataLogic`) returns this error to the gRPC caller. Known categories silently succeed.

### 3j. SetInterval runtime override (reporting.go:446-448)

```go
func (c *Client) SetInterval(category ReportCategory, d time.Duration) {
    c.reports.setInterval(category, d)
}
```

Exposed on `*Client` for runtime category interval changes (e.g., admin debug). Only accepts known categories and positive durations.

### 3k. freshnessTimeout formula (reporting.go:345-352)

```go
func freshnessTimeout(interval time.Duration) time.Duration {
    twice := interval * 2
    plus := interval + 10*time.Second
    if twice > plus { return twice }
    return plus
}
```

Formula: `max(interval×2, interval+10s)`.

- For 1 min interval → `max(2min, 1min10s)` = **2 min**
- For 2s interval → `max(4s, 12s)` = **12s** (but coordinates are noFreshCheck, so irrelevant)
- For short intervals (<10s): timeout = interval+10s (gives grace period)
- For long intervals (≥10s): timeout = interval×2

### 3l. itemKey fallback logic (reporting.go:385-403)

When all keyAttrs are empty in the Item, falls back to `item_index=N` (using the slice index). When some attrs are present but not all, appends `item_index=N` as a disambiguator.

### 3m. ReportCategoryInfo proto message (ispagent.proto:215-224)

```protobuf
message ReportCategoryInfo {
  int32 category = 1;            // 上报类别 messageId
  string name = 2;               // 中文名称
  int64 interval_seconds = 3;    // 当前间隔（秒）
  bool no_fresh_check = 4;       // 是否跳过新鲜度检查
  int32 type = 5;                // ISP Type
  int32 command = 6;             // ISP Command
  repeated string key_attrs = 7; // 缓存 key 属性列表
}
```

---

## Section 4: Proto Conventions Assessment

### Existing proto coverage in spec: NONE

The spec has **zero** documentation about the proto messages or RPC conventions. All proto knowledge is implicit in the file itself (`ispagent.proto`).

### What should be documented

**New RPCs added for interval reporting:**

| RPC | Request | Response | Purpose |
|---|---|---|---|
| `SendDroneNestRunData` | `SendDroneNestRunDataReq` | `CommandRes` | Drone nest run data cache update |
| `SendEnvData` | `SendEnvDataReq` | `CommandRes` | Env/micro-weather data cache update |
| `ListReportIntervals` | `ListReportIntervalsReq` (empty) | `ListReportIntervalsRes` | Diagnostic: query all category intervals |

**Existing RPCs with changed semantics:**

| RPC | Old behavior | New behavior |
|---|---|---|
| `SendPatrolDeviceRunData` | Synchronous ISP upstream send | Cache update → immediate local ack |
| `SendPatrolDeviceStatusData` | Synchronous ISP upstream send | Cache update → immediate local ack |
| `SendPatrolDeviceCoordinates` | Synchronous ISP upstream send | Cache update → immediate local ack |

**New proto messages:**

- `DroneNestRunData` (line 56-64): nested under `SendDroneNestRunDataReq`
- `EnvData` (line 67-75): nested under `SendEnvDataReq`
- `ReportCategoryInfo` (line 215-224): returned by `ListReportIntervalsRes`
- `ListReportIntervalsReq` / `ListReportIntervalsRes` (lines 226-232)

All new RPCs return `CommandRes` (the same response type used by existing RPCs), maintaining consistency.

**Convention to document:**

```markdown
### Proto Conventions for Report RPCs

- All gRPC report RPCs write to in-memory cache via `Client.CacheReport()` and return `CommandRes{Success: true, Code: "100"}` immediately.
- RPC names follow pattern `Send<ReportType>` with corresponding `Send<ReportType>Req` containing `code` + repeated typed items.
- Each proto data message maps to an ISP protocol table (e.g., `DroneNestRunData` maps to table O.40).
- Logic layer converts proto messages to `[]isp.Item` (typed fields → string map) before calling `CacheReport`.
- `ListReportIntervals` is a read-only diagnostic RPC with empty request, returning full category metadata.
```

---

## Section 5: Client Public API Surface (Undocumented)

The following public methods/helpers on `*Client` are not mentioned in the spec at all:

| Method | Purpose |
|---|---|
| `Client.CacheReport(ctx, category, code, items)` | gRPC entry point for cache update |
| `Client.ReportIntervals()` | Returns `map[ReportCategory]time.Duration` |
| `Client.SetInterval(category, duration)` | Runtime interval override |
| `Client.SetNoFreshCheck(category, bool)` | Toggle fresh check per category |
| `Client.CategoryNoFreshCheck(category)` | Query fresh check flag |
| `CategoryMessageName(category)` (package-level) | Returns Chinese name for a category |
| `CategoryKeyAttrs(category)` (package-level) | Returns key attrs for a category |

---

## Section 6: Summary of Recommendations

### Must-fix outdated content:

1. **Lines 66-71**: Registration table — add explicit ReportCategory mappings, remove "未实现" caveats since drone nest and env are implemented.
2. **Line 94**: Add all 5 categories, not just run/status/coord.
3. **Line 96**: Add keyAttrs for drone nest (`nest_code + type`) and env data (`patroldevice_code + type`).

### Strongly recommended additions:

4. New subsection: "ReportCategory 定义" (5 types with messageId mapping)
5. New subsection: "keyAttrsByCategory" (per-category item key construction)
6. New subsection: "默认上报间隔" (1min general, 2s coord with noFreshCheck)
7. New subsection: "注册间隔应用 (applyRegistrationIntervals)" (field→category mapping + lastSent reset)
8. New subsection: "freshnessTimeout 公式" (`max(interval*2, interval+10s)`)
9. New subsection: "markSent snapLastSent 竞态保护" (concurrent update detection)
10. New subsection: "Proto Conventions for Report RPCs" (send→cache semantics, new messages, diagnostic RPC)
11. Add `Client` public API surface (CacheReport, SetInterval, ReportIntervals, etc.)

### No action needed:

- All of lines 1-63 (protocol/package/message model) — unchanged
- All of lines 72-85 (heartbeat, response, common errors) — unchanged
- Lines 88-102 core caching logic — remarkably accurate
- Lines 103-110 test contracts — unchanged
- Lines 112-151 (lifecycle, names, persistence, carbon) — unchanged
