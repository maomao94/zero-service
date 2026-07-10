# ISP interval-driven device reporting implementation plan

## Checklist

1. Read applicable specs before editing:
   - `.trellis/spec/guides/index.md`
   - `.trellis/spec/backend/isp-guidelines.md`
   - `.trellis/spec/backend/gnetx/index.md` if touching request/response behavior around `gnetx.Client`.
2. Add interval parsing/state:
   - Replace single `heartbeatFromItems` helper with parsing that extracts all known interval fields.
   - Keep `heart_beat_interval` separate from report intervals.
   - Preserve unsupported intervals in local state for future use.
3. Add report cache/category model:
   - Define categories for patrol device run data, status data, and coordinates.
   - Store latest code/items/update time/expired state per category.
   - Provide a client method for logic layer to update cache.
4. Add periodic report behavior:
   - Integrate into existing client loop or a clearly owned client goroutine using existing context cancellation.
   - Send only after registration.
   - Use `patroldevice_run_interval` for run data only; status/coordinates use the default 1 minute report-spec interval.
   - Skip empty or expired cache entries.
5. Change gRPC logic semantics:
   - `SendPatrolDeviceRunData` caches converted items and returns local success.
   - `SendPatrolDeviceStatusData` caches converted items and returns local success.
   - `SendPatrolDeviceCoordinates` caches converted items and returns local success.
   - Keep `ExecuteCommand` unchanged.
6. Add tests:
   - Interval parsing covers all known fields, invalid values, missing values.
   - Cache update clears expired state and records update time.
   - Freshness marking expires stale entries.
   - Reporter decision sends fresh due categories and skips expired/empty ones.
7. Update docs:
   - `docs/ispagent.md` explains interval fields, local cache semantics, periodic reporting, and stale skip behavior.
8. Run validation.

## Validation Commands

```bash
go test ./common/isp ./app/ispagent/internal/isp ./app/ispagent/internal/logic ./app/ispagent/internal/handler ./app/ispagent/internal/crontask
```

If package boundaries make the focused command insufficient, run:

```bash
go test ./app/ispagent/...
```

## Risk Areas

- Direct-send semantics changing to local acceptance may affect callers expecting upstream response fields.
- Reporter sends must not recurse through public `Execute` in a way that blocks registration/heartbeat progress unexpectedly.
- Holding locks during network sends can deadlock or block gRPC cache updates; snapshot cache before sending.
- `patroldevice_run_interval` must not accidentally drive status or coordinate categories; document that those categories use the default 1 minute interval unless overridden later.

## Review Gate Before Start

- PRD confirms run/status/coordinates are in scope.
- Design confirms no proto schema change is required.
- Implementation plan includes tests and docs.
