# Research: Repository Date/Time Utility and Usage Inventory

- **Query**: Thoroughly inventory existing date/time utilities and formatting/parsing usage across the repository; classify standard seconds, date-only/time-only, subseconds, Unix timestamps, nullable/zero time, timezone-explicit, recurrence/scheduling, and serialization/database/protocol contracts.
- **Scope**: internal
- **Date**: 2026-08-13

## Findings

### Search Coverage

Repository-wide Go searches found 62 files using `carbon.*`, 43 files containing explicit parsing/formatting calls, 49 files using Unix timestamp conversions, 63 files containing `sql.NullTime`, `IsZero`, or the legacy epoch sentinel, and 71 files mentioning RRULE/CronJob/ScheduledTime. Generated Go/protobuf files and tests were included when identifying contracts, but representative source-of-truth `.proto`, model, and implementation files are cited below rather than enumerating every generated occurrence.

### Files Found

| File Path | Description |
|---|---|
| `common/carbonx/carbonx.go:7-15` | Process-global Carbon defaults: Carbon datetime layout, `Asia/Shanghai`, `zh-CN`, Monday week start, Saturday/Sunday weekend. |
| `common/tool/timeutil.go:9-31` | Reusable current/conversion helpers: second-normalized Carbon plus current Unix seconds, milliseconds, and microseconds. |
| `common/type.go:28-44` | `common.DateTime` JSON adapter; marshals microsecond datetime strings and parses strings through Carbon. |
| `common/copierx/type.go:12-27` | Shared copier conversion from `time.Time` to Carbon microsecond datetime string. |
| `common/tool/backoff.go:9-39` | Retry/backoff schedule computation and standard-second string wrapper. |
| `common/holiday/types.go:147-153` | Date-only key generation in an explicit caller-supplied location. |
| `common/imagex/exifx.go:116-128` | EXIF-specific parse contract (`2006:01:02 15:04:05`) and normalized output. |
| `common/rrulex/rrulex.go` | Complete RFC 5545 Set parsing/validation mechanism. |
| `common/rrulex/query.go` | Recurrence query shifting and occurrence iteration mechanism. |
| `common/rrulex/describe.go:475-504` | Location-aware recurrence rendering, including explicit numeric UTC offset for RDATE/EXDATE values. |
| `common/crontask/config.go` | Generic scheduler time state (`NextRun`, `ScheduledTime`, `LastRun`, effective bounds and durations). |
| `common/crontask/crontask.go:104,184,215,239,306` | Runtime scheduling timestamps; scheduled execution is normalized to second precision. |
| `app/trigger/internal/cronjob/schedule.go:17-221` | Trigger-owned strict Shanghai parsing, effective range normalization, exact time/date exclusion, and RRULE compilation. |
| `app/trigger/internal/cronjob/convert.go:237-257` | Trigger-local zero/NULL conversion and optional standard-second parser/formatter. |
| `app/trigger/internal/cronjob/handler.go:110-115` | Trigger-local zero-aware standard-second formatter for callback contracts. |
| `app/ispagent/internal/crontask/task_rule.go:270-294` | ISP-specific date + time-only composition and forgiving Shanghai parser returning zero on failure. |
| `app/ispagent/internal/handler/validate.go:137-170` | ISP contract validation for datetime strings and strict `HH:mm:ss` time-only input. |
| `app/djicloud/internal/hooks/store_helper.go:15-31` | DJI protocol millisecond timestamp fallback and local SQL-null constructor. |
| `app/djicloud/internal/logic/helper.go:50-68` | DJI API zero-aware millisecond and microsecond-string adapters. |
| `model/*model_gen.go` | Legacy generated models with `time.Time` audit/delete fields and Unix epoch delete sentinel. |
| `app/trigger/trigger.proto` | Trigger string date/datetime contracts and scheduling fields. |
| `facade/streamevent/streamevent.proto` | Stream-event microsecond send-time and standard scheduling callback string fields. |

### Existing Reusable Helpers

#### Carbon initialization

`common/carbonx/carbonx.go:7-15` is an initialization-only package:

```go
carbon.SetDefault(carbon.Default{
    Layout: carbon.DateTimeLayout,
    Timezone: carbon.Shanghai,
    Locale: "zh-CN",
    WeekStartsAt: carbon.Monday,
})
```

It is blank-imported by 14 service entry points, including `app/trigger/trigger.go:22`, `app/ispagent/ispagent.go:13`, `app/djicloud/djicloud.go:15`, and `facade/streamevent/streamevent.go:23`. Therefore Carbon calls made within those service processes inherit Shanghai unless the call supplies another location. Packages/tests/tools that do not enter through these binaries do not establish that default through their own imports.

#### General current-time and precision helpers

`common/tool/timeutil.go:9-31` contains:

```go
func NowStartOfSecond() *carbon.Carbon
func CarbonFromTimeStartOfSecond(t time.Time) *carbon.Carbon
func GenSecondTS() int64
func GenMilliTS() int64
func GenMicroTS() int64
```

The first two encode truncation to whole seconds. Trigger response assembly already reuses `CarbonFromTimeStartOfSecond`, for example `app/trigger/internal/logic/createcronjoblogic.go:54` and `previewcronjobschedulelogic.go:65`. Timestamp helpers expose current time only; they do not convert supplied values, handle zero/null, or state a protocol.

#### Serialization adapters

- `common.DateTime.MarshalJSON`, `common/type.go:31-44`, emits `ToDateTimeMicroString()` and `UnmarshalJSON` accepts Carbon-parseable strings.
- `common/copierx.Option`, `common/copierx/type.go:16-27`, maps any `time.Time` to `carbon.DateTimeMicroFormat` for copier-based DTO mapping.
- `common/tool/backoff.go:37-39` wraps a domain backoff calculation as a standard datetime string.

### Usage Classification

#### 1. Standard datetime, second precision

The dominant human/API representation is `yyyy-MM-dd HH:mm:ss` without an offset.

- Carbon output: `ToDateTimeString()` in file DTOs (`app/file/internal/logic/helper.go:52-53`), Trigger (`app/trigger/cron/cronservice.go:190-202`), podengine Docker normalization (`app/podengine/internal/logic/getpodlogic.go:64-114`), ISP (`app/ispagent/internal/logic/helper.go:50-53`), and task callbacks.
- Carbon format strings: `carbon.Now().Format("Y-m-d H:i:s")` in `zerorpc/internal/task/deferforwardtask.go:58,80`, `zerorpc/internal/logic/forwardtasklogic.go:78`, and `app/xfusionmock/internal/logic/pushterminalbindlogic.go:53`.
- Go layouts: Trigger declares `dateTimeLayout = "2006-01-02 15:04:05"` at `app/trigger/internal/cronjob/schedule.go:17`; the same literal appears in EXIF normalization (`common/imagex/exifx.go:125`), Docker utility output (`util/dockeru/main.go:122,230`), and CLI Docker display (`cli/dtui/internal/docker/container.go:66`).
- Precision normalization is semantically explicit in scheduling: `common/crontask/crontask.go:215` uses `carbon.Now().StartOfSecond()`, while Trigger's compiler normalizes current/start/end/exact values at `app/trigger/internal/cronjob/schedule.go:88,99,143,196,208`.

#### 2. Date-only and time-only

- Date-only canonical Go layout is used by holiday data: `common/holiday/json_source.go:75` validates `time.DateOnly`, `calendar.go:99` keys by it, and `types.go:147-148` applies a supplied location before formatting.
- Trigger exclude dates use Carbon `DateFormat` with explicit Shanghai at `app/trigger/internal/cronjob/schedule.go:81-94`; Trigger's general holiday helper uses `carbon.ParseByFormat(date, carbon.DateFormat)` at `app/trigger/internal/logic/helper.go:113`.
- Directory/object naming uses compact local dates `20060102`: `common/tool/ossutil.go:14`, `common/ossx/ossx.go:65`, `common/mediax/mediax.go:150`, gateway upload implementations at `gtw/internal/logic/common/mfsuploadfilelogic.go:61` and `gtw/internal/logic/file/putfilelogic.go:61`, and xfusion mock payload generation at `app/xfusionmock/internal/logic/pushalarmlogic.go:126`.
- ISP accepts strict time-only `HH:mm:ss`: manual length/digit/range checks are in `app/ispagent/internal/handler/validate.go:141-148`. It later combines `IntervalStartTime`'s date and `IntervalExecuteTime` at `app/ispagent/internal/crontask/task_rule.go:270-278`.
- UI-only time formats intentionally omit date/seconds: timeline `15:04:05` at `cli/uix/timeline.go:128`, deployment `01-02 15:04` at `cli/dtui/plugins/deploy/plugin.go:250`, and image display `2006-01-02 15:04` at `cli/dtui/internal/docker/image.go:75`.

#### 3. Subsecond / microsecond

- Microsecond text is the stream/event and detailed DTO convention: `app/bridgemqtt/internal/handler/mqttstreamhandler.go:83,104`, `app/bridgekafka/internal/handler/kafkastreamhandler.go:43`, `app/bridgedump/internal/svc/servicecontext.go:38-45`, `app/ieccaller/internal/svc/servicecontext.go:169`, and DJI output (`app/djicloud/internal/logic/helper.go:57-61,112,124,135,146,190`).
- `common.DateTime` JSON and `common/copierx.Option` also emit microseconds (`common/type.go:31-34`; `common/copierx/type.go:16-27`).
- Millisecond datetime text is used for DJI timestamp diagnostics: `common/djisdk/handler.go:24` converts a millisecond epoch to `ToDateTimeMilliString()`.
- Nanosecond precision appears as runtime state rather than display: `common/gnetx/session.go:72,82,137` stores last activity as Unix nanoseconds; `common/mqttx/topic_log.go:48` uses Unix nanoseconds in a log/topic identifier path.
- `time.RFC3339Nano` is specifically the Docker engine contract parser in `cli/dtui/internal/docker/inspect.go:138-147`; output is deliberately reduced to seconds.

#### 4. Unix timestamps

- Current timestamp helpers: `common/tool/timeutil.go:19-31` provides seconds/milliseconds/microseconds.
- Seconds are used by authentication/health/knowledge contracts: `zerorpc/internal/logic/generatetokenlogic.go:32`, `aiapp/aisolo/internal/logic/healthlogic.go:78`, `common/einox/knowledge/store_gorm.go:119,156,174,251`, and corresponding Redis/Milvus stores.
- Milliseconds are protocol-owned in DJI (`common/djisdk/protocol.go:167,178`, `protocol_drc.go:41`, `drc.go:283,432-455`), MCP event/state (`common/mcpx/memory_handler.go:47-126` and `common/mcpx/client.go:971-1156`), GIS response timestamps (`app/gis/internal/logic/listfenceslogic.go:55-56`), and xfusion mock payloads (`app/xfusionmock/internal/logic/pusheventlogic.go:47-48`).
- Nanoseconds are used for in-process atomic state and generated IDs: `common/gnetx/session.go:72-137`, `common/crontask/memory_store.go:144`.
- A field name alone does not establish unit. Evidence is at the conversion boundary: DJI `reportTime(ms int64)` calls `time.UnixMilli` (`app/djicloud/internal/hooks/store_helper.go:15-20`), whereas knowledge stores call `time.Unix(value, 0)`.

#### 5. Nullable and zero time

- Generic scheduler semantics use Go zero time for exhausted/no candidate and SQL NULL in adapters. Trigger and ISP duplicate an identical helper:

```go
// app/trigger/internal/cronjob/convert.go:237-239
// app/ispagent/internal/crontask/convert.go:71-73
return sql.NullTime{Time: value, Valid: !value.IsZero()}
```

- Trigger optional string conversion maps empty string to invalid `sql.NullTime`, and invalid `sql.NullTime` back to empty string (`app/trigger/internal/cronjob/convert.go:241-257`). Trigger callback formatting separately maps zero `time.Time` to empty string (`handler.go:110-115`).
- DJI API helpers map zero `time.Time` and invalid `sql.NullTime` to integer `0` or empty string (`app/djicloud/internal/logic/helper.go:50-68`). The hook ingestion path has different semantics: non-positive protocol milliseconds become `time.Now()` (`app/djicloud/internal/hooks/store_helper.go:15-20`).
- Legacy generated SQL models use Unix epoch as a soft-delete sentinel: `model/usermodel_gen.go:117,128`, `regionmodel_gen.go:128,139`, `ordertxnmodel_gen.go:152,163`, `hlstsfilesmodel_gen.go:121,132`. This differs from current `common/gormx`/scheduler SQL NULL semantics.
- Recurrence uses zero time to mean exhausted: `app/trigger/internal/cronjob/schedule.go:29-30,104-108`; the contract is also stated in `.trellis/spec/backend/crontask-guidelines.md:13-18`.

#### 6. Timezone-explicit

- `common/carbonx` sets Carbon's global timezone to `Asia/Shanghai`, but this only applies after package initialization (`common/carbonx/carbonx.go:7-15`).
- Trigger strict schedule inputs explicitly request `carbon.Shanghai` or load it: `app/trigger/internal/cronjob/schedule.go:82,99,127-136,196-210`.
- ISP recurrence parsing explicitly supplies `carbon.Shanghai`: `app/ispagent/internal/crontask/task_rule.go:273-274,286-294`.
- Asynq scheduler loads `Asia/Shanghai` at `zerorpc/internal/svc/asynqSchedulerServer.go:34-40`.
- Docker metadata is timezone-bearing input: `util/dockeru/main.go:117,225` parses numeric offset and MST; `cli/dtui/internal/docker/inspect.go:143` parses RFC3339Nano.
- RRULE description intentionally preserves timezone identity/offset: `common/rrulex/describe.go:484-503` emits location notice and `-07:00` offset.
- `time.Parse` without location is intentional for EXIF because EXIF input has no zone and is treated as a wall-clock metadata string (`common/imagex/exifx.go:116-128`).

#### 7. Recurrence and scheduling semantics

- Generic recurrence mechanism belongs to `common/rrulex`: complete Set validation, RDATE/EXDATE, safe query shifting, next occurrence, and description. Its public ownership is documented in `.trellis/spec/backend/rrulex-guidelines.md:9-25`.
- Generic execution-state semantics belong to `common/crontask`: `NextRun`, lease time, stable retry `ScheduledTime`, actual successful `LastRun`, and `LastScheduledRun` are distinct (`.trellis/spec/backend/crontask-guidelines.md:7-18`).
- Trigger owns business compilation: Shanghai parsing, Plan/CronJob range limits, PlanRule-to-RRULE mapping, exact specified/excluded times, exclude-date expansion, and `skipTimeFilter` (`app/trigger/internal/cronjob/schedule.go:35-221`).
- ISP owns its source fields and invalid-window predicate; `parseTime` deliberately maps missing/invalid text to zero (`app/ispagent/internal/crontask/task_rule.go:286-294`) and interval composition may advance by a wall-clock day (`270-281`).
- Zerorpc/asynq owns cron expression registration and minute-based delayed-task semantics (`zerorpc/internal/svc/asynqSchedulerServer.go:34-61`; `zerorpc/internal/logic/senddelaytasklogic.go:48`).

#### 8. Serialization, database, and protocol contracts

- Many RPC time fields are strings rather than protobuf Timestamp. Representative contracts: Trigger date/datetime and schedule fields in `app/trigger/trigger.proto:494-498,667,685,710-714,945-957,1366-1368`; StreamEvent send/schedule fields in `facade/streamevent/streamevent.proto:51,63,83,531-533,641`; Alarm comments specify second strings in `app/alarm/alarm.proto` (generated evidence at `app/alarm/alarm/alarm.pb.go:118`).
- Trigger CronJob `scheduled_time` is defined as original planned execution and formatted `yyyy-MM-dd HH:mm:ss`; this is a protocol/state contract, not presentation (`.trellis/spec/backend/trigger-guidelines.md:140-152,262-268`).
- Date-only Trigger inputs have protobuf length validation (`app/trigger/trigger.proto:597,667,685`) and are holiday/calendar concepts rather than generic instants.
- Database audit fields are commonly `time.Time`; optional business/scheduling fields are `sql.NullTime`, e.g. `model/ordertxnmodel_gen.go:57-78` and Trigger/ISP scheduling adapters.
- `common.DateTime` changes JSON shape from standard Go RFC3339 to a Carbon microsecond string (`common/type.go:28-44`).
- EXIF, Docker, DJI, IEC, RRULE, and ISP each carry external format/unit semantics; those conversions are protocol adapters rather than interchangeable display helpers.

### Duplicate Local Helpers and Repeated Shapes

| Repeated shape | Evidence | Behavioral distinction |
|---|---|---|
| `time.Time` -> `sql.NullTime`, invalid on zero | `app/trigger/internal/cronjob/convert.go:237-239`; `app/ispagent/internal/crontask/convert.go:71-73` | Exact duplicates in scheduler DB adapters. |
| Zero time -> empty standard-second string | `app/trigger/internal/cronjob/handler.go:110-115`; optional `sql.NullTime` variant at `convert.go:252-257` | One accepts `time.Time`; one accepts `sql.NullTime`. |
| Zero/nullable time -> epoch milliseconds | `app/djicloud/internal/logic/helper.go:50-68` | DJI API contract maps absence to `0`; not the same as scheduler SQL NULL. |
| Standard datetime literal | `app/trigger/internal/cronjob/schedule.go:17`; `common/imagex/exifx.go:125`; `util/dockeru/main.go:122,230`; CLI Docker files | Similar output text, but parsing strictness, timezone, and audience differ. |
| Compact date directory | `common/tool/ossutil.go:14`; `common/ossx/ossx.go:65`; `common/mediax/mediax.go:150`; two gateway upload logics at line 61 | Same `20060102` shape, but path construction packages own naming behavior. |
| Carbon standard datetime generation | `common/isp/client.go:411-416`; multiple Trigger, podengine, ISP, file call sites | Most are direct library calls rather than local wrappers. |
| Forgiving string parse to zero | `app/ispagent/internal/crontask/task_rule.go:286-294` | Invalid input is intentionally collapsed to zero after a separate validation boundary. |
| Strict exact second parse | `app/trigger/internal/cronjob/schedule.go:123-149` | Rejects malformed/noncanonical/out-of-range input and binds Shanghai. |

### Related Specs

- `.trellis/spec/backend/common-package-design.md:7-13` — cross-service mechanism belongs in common only with stable business-independent semantics; domain policy remains in the service.
- `.trellis/spec/backend/rrulex-guidelines.md:5-25` — defines ownership of RRULE parsing/query/description and leaves rule generation to Trigger/ISP.
- `.trellis/spec/backend/crontask-guidelines.md:7-18,63-80` — defines scheduling timestamp, zero/NULL, retry, preview, and occurrence semantics.
- `.trellis/spec/backend/trigger-guidelines.md:24-33,138-164` — defines Trigger-specific recurrence, exact-time, Shanghai, persistence, and callback contracts.

## Caveats / Not Found

- No repository-wide single datetime contract was found. The repository intentionally carries several representations: standard-second strings, microsecond strings, RFC3339/RFC5545 values, compact path dates, Unix values in multiple units, and protocol-specific wall-clock values.
- `model/constantKey.go:12-14` declares Carbon format-template variables (`Y-m-d H:i:s`, `Y-m-d`, `H:i:s`), but repository search found no cited consumers in the inspected results.
- Generated protobuf Go files multiply textual matches; `.proto` files are the contract source and should be used when classifying those fields.
- Bare `time.Now`, duration/timeouts, test fixture construction, and timestamps used only for elapsed-time measurement were not individually listed unless they established precision, serialization, scheduling, or protocol semantics.
