# Research: Date/Time Ownership and Replacement Boundaries

- **Query**: Identify what belongs in `common/carbonx` versus what remains domain-specific, and identify unsafe mechanical replacements.
- **Scope**: internal
- **Date**: 2026-08-13

## Findings

### Files Found

| File Path | Description |
|---|---|
| `common/carbonx/carbonx.go:7-15` | Existing Carbon-wide defaults only. |
| `common/tool/timeutil.go:9-31` | Existing cross-service precision/timestamp mechanisms currently outside `carbonx`. |
| `common/type.go:28-44` | JSON wire type with microsecond contract. |
| `common/copierx/type.go:12-27` | Framework-specific DTO conversion. |
| `common/holiday/types.go:147-153` | Calendar date key tied to supplied location. |
| `common/rrulex/` | Shared RFC 5545 mechanism with independent semantic ownership. |
| `common/crontask/` | Shared scheduler state machine with independent semantic ownership. |
| `app/trigger/internal/cronjob/schedule.go:17-221` | Trigger business validation and recurrence compilation. |
| `app/ispagent/internal/crontask/task_rule.go:270-294` | ISP source protocol and schedule mapping. |
| `app/djicloud/internal/hooks/store_helper.go:15-31` | DJI timestamp-unit/fallback ingestion policy. |
| `common/imagex/exifx.go:116-128` | EXIF syntax adapter. |

### Evidence-Based Ownership Classification

The repository's common-package rule is explicit: a common capability must have cross-service demand, stable business-independent inputs/outputs, and must not absorb domain policy (`.trellis/spec/backend/common-package-design.md:7-13`). Applying that existing rule yields the following boundary map.

#### `common/carbonx` scope represented by existing evidence

`common/carbonx` currently owns only process-global Carbon configuration (`common/carbonx/carbonx.go:7-15`). The reusable shapes adjacent to that responsibility are:

- Canonical Carbon defaults and named canonical Carbon layouts/timezone already supplied by Carbon.
- Whole-second normalization of current/supplied instants, currently implemented in `common/tool/timeutil.go:9-17` and already reused by Trigger.
- Unit-explicit current Unix timestamps, currently implemented in `common/tool/timeutil.go:19-31`; these are generic clock operations, although call sites with protocol fields still own the choice of unit.
- Pure, zero-aware or nullable formatting/conversion primitives only where absence semantics are explicit in the function name/type. The exact duplicate `toNullTime` in Trigger and ISP (`app/trigger/internal/cronjob/convert.go:237-239`; `app/ispagent/internal/crontask/convert.go:71-73`) is a cross-adapter mechanism candidate by shape, while its SQL dependency means ownership could also remain with a database-focused common package rather than Carbon configuration.

This classification does not make `carbonx` the owner of all uses of `time.Time`: `common/rrulex`, `common/crontask`, `common/holiday`, protocol packages, JSON types, and UI formatting each already have a narrower semantic owner.

#### Existing common mechanisms that remain separate from `carbonx`

- RRULE parsing/querying/description remains `common/rrulex`; `.trellis/spec/backend/rrulex-guidelines.md:5-25` explicitly assigns that ownership and forbids duplicating it in schedulers.
- Scheduler timestamps, leases, retries, scheduled-vs-actual execution, and exhausted zero values remain `common/crontask`; `.trellis/spec/backend/crontask-guidelines.md:7-18` defines these as state-machine semantics.
- Calendar date keys/weekend/holiday behavior remains `common/holiday`; `dateKey` requires an explicit location (`common/holiday/types.go:147-153`).
- JSON behavior remains with `common.DateTime` (`common/type.go:28-44`) because changing its format changes wire serialization.
- Copier behavior remains `common/copierx` (`common/copierx/type.go:12-27`) because it is tied to `jinzhu/copier` converter registration.

#### Domain/protocol semantics that remain with their owners

- Trigger strict canonical validation, inclusive effective ranges, 3/100-year limits, RDATE/EXDATE expansion, and `skipTimeFilter` (`app/trigger/internal/cronjob/schedule.go:35-221`).
- ISP source-message validation, `HH:mm:ss`, forgiving parse-to-zero behavior, date/time composition, and invalid execution windows (`app/ispagent/internal/handler/validate.go:137-170`; `app/ispagent/internal/crontask/task_rule.go:270-294,352-362`).
- DJI millisecond epochs, non-positive-report fallback to now, and absent API output as zero (`app/djicloud/internal/hooks/store_helper.go:15-20`; `app/djicloud/internal/logic/helper.go:50-68`).
- EXIF's colon-separated wall-clock syntax and fallback to raw metadata (`common/imagex/exifx.go:116-128`).
- Docker RFC3339Nano or offset/MST parsing (`cli/dtui/internal/docker/inspect.go:138-147`; `util/dockeru/main.go:117-122,225-230`).
- Compact date/file/ID naming (`common/tool/idutil.go:38-40`, `common/ossx/ossx.go:65`, `common/mediax/mediax.go:150`), because those formats are naming schemes rather than a canonical API datetime.
- CLI display-only formats (`cli/uix/timeline.go:128`, `cli/dtui/plugins/deploy/plugin.go:250`, `cli/dtui/internal/docker/image.go:75`).

### Unsafe Mechanical Replacements

The following replacements are not semantics-preserving based on existing contracts:

1. **`time.Now()` -> `carbon.Now()` globally**: Carbon's location is configured by blank-import side effect in only 14 service binaries (`common/carbonx/carbonx.go:7-15` plus entry-point imports). Go `time.Now()` returns an instant in `time.Local`; Carbon may use Shanghai. Tests, tools, and packages may execute without the initializer.

2. **`time.Parse` -> `carbon.Parse` globally**: `time.Parse` is strict to a supplied layout and defaults location differently. Carbon parsing may accept broader input and inherit defaults. Trigger exact times deliberately perform length, strict parse, round-trip canonicality, range, and Shanghai checks (`app/trigger/internal/cronjob/schedule.go:123-149`). EXIF deliberately parses `2006:01:02 15:04:05` (`common/imagex/exifx.go:116-128`). Docker consumes zone-bearing formats.

3. **Any `yyyy-MM-dd HH:mm:ss` formatter -> one universal formatter**: identical text can represent local Shanghai schedule time, a zone-less EXIF wall clock, Docker metadata converted to local display, callback protocol time, or UI output. Caller location and zero/null behavior differ.

4. **`ToDateTimeString()` <-> `ToDateTimeMicroString()`**: these are distinct wire precision contracts. Stream/event/DJI detailed responses use microseconds, while Trigger scheduling and callbacks are second-normalized. `common.DateTime.MarshalJSON` currently emits microseconds despite its comment saying seconds (`common/type.go:31-34`).

5. **Unix second/millisecond/microsecond calls -> a single timestamp helper**: units are protocol and storage contracts. DJI/MCP/GIS use milliseconds; health/auth/knowledge use seconds; in-process activity uses nanoseconds. Field names such as `timestamp` do not reliably encode unit.

6. **Non-positive Unix input -> zero time**: DJI ingestion maps `ms <= 0` to current time (`app/djicloud/internal/hooks/store_helper.go:15-20`), while other APIs use integer zero to represent absence (`app/djicloud/internal/logic/helper.go:50-68`).

7. **Epoch sentinel -> SQL NULL/Go zero mechanically**: legacy generated models explicitly write `time.Unix(0, 0)` as undeleted state (`model/*model_gen.go` cited in the inventory), while modern scheduler adapters use invalid `sql.NullTime`. Database queries and generated update code encode the sentinel.

8. **`sql.NullTime` -> `time.Time` or vice versa mechanically**: Trigger and ISP use `Valid` to preserve SQL NULL and scheduler exhaustion. DJI's `sqlNullTime` always marks valid (`app/djicloud/internal/hooks/store_helper.go:30-31`), reflecting a different ingestion guarantee.

9. **Zero time -> empty string globally**: Trigger callback formatting uses empty (`app/trigger/internal/cronjob/handler.go:110-115`); recurrence zero means exhausted; legacy models use epoch; JSON `common.DateTime` currently formats whatever value it holds without a zero special case.

10. **`time.Local` -> `carbon.Shanghai` mechanically**: Trigger `parseOptionalTime` currently uses `time.Local` (`app/trigger/internal/cronjob/convert.go:241-249`) while strict compilation uses Shanghai. These coincide only when process local timezone is Shanghai; changing one can alter persisted instants.

11. **`Add(24*time.Hour)` -> calendar `AddDay`, or the reverse**: ISP interval composition currently uses absolute 24 hours (`app/ispagent/internal/crontask/task_rule.go:270-277`). RRULE guidelines distinguish wall-clock calendar frequencies from durations and explicitly account for DST (`.trellis/spec/backend/rrulex-guidelines.md:44-52`).

12. **Formatting recurrence values without offset**: RRULE descriptions explicitly render `2006-01-02 15:04:05 -07:00` and include timezone notice (`common/rrulex/describe.go:484-503`) because DST folds can create duplicate local wall-clock text.

13. **Replacing domain recurrence code with a generic datetime parser**: Trigger compilation and ISP mapping construct RFC 5545 semantics, not merely parse datetimes. RDATE union, EXDATE subtraction, DTSTART anchoring, inclusive query behavior, zero/exhaustion, and predicate filtering are owned by `rrulex`/domain compilers.

14. **Replacing compact date formats (`20060102`) with standard date output**: object paths, backup names, and IDs would change externally visible naming and storage locations (`common/ossx/ossx.go:65`; `common/tool/idutil.go:38-40`; `cli/dtui/plugins/deploy/plugin.go:486`).

15. **Changing protobuf string times to Go/protobuf native time through helper migration**: the `.proto` string shape, JSON names, validation lengths, and comments are protocol contracts. Generated files are not the source to edit; representative source fields are in `app/trigger/trigger.proto` and `facade/streamevent/streamevent.proto`.

### Related Specs

- `.trellis/spec/backend/common-package-design.md` — common/domain ownership test.
- `.trellis/spec/backend/rrulex-guidelines.md` — RFC 5545 and timezone/DST boundaries.
- `.trellis/spec/backend/crontask-guidelines.md` — scheduler state and SQL NULL boundaries.
- `.trellis/spec/backend/trigger-guidelines.md` — Trigger exact-time and callback contracts.

## Caveats / Not Found

- The active task PRD is still a placeholder (`.trellis/tasks/08-13-unify-datetime-tools/prd.md:1-18`), so there is no task-level accepted target API or migration scope against which to narrow ownership further.
- `common/carbonx` currently has no public helper API; all helper placement statements above are ownership classification from existing code/spec evidence, not evidence that such APIs already exist.
- Exact duplicate code does not by itself establish that the existing spec permits a new shared API; `common-package-design.md:9-12` also requires clear cross-service demand and stable semantics.
