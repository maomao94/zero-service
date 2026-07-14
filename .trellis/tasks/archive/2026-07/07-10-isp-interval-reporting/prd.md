# ISP interval-driven device reporting

## Goal

Optimize `ispagent` device-reporting behavior for ISP protocol scenarios where registration responses return different reporting intervals. Downstream gRPC reports should update local in-memory state first; `ispagent` should then periodically report cached device data to the upstream ISP system according to intervals learned from registration responses, while independently detecting stale downstream data.

## Confirmed Facts

- Current ISP client registration only reads `heart_beat_interval` from `251-4` response items and uses it as the heartbeat send interval.
- Current `SendPatrolDeviceRunData`, `SendPatrolDeviceStatusData`, and `SendPatrolDeviceCoordinates` RPC logic directly calls `IspClient.Execute(...)`, so RPC input is immediately sent upstream.
- `common/isp` already defines message constants for patrol device run/status/coordinate data, environment data, drone nest status, and drone nest run data.
- `ispagent.proto` currently exposes specific RPCs for patrol device run data, status data, and coordinates, but not for environment/weather or drone nest run data.
- XML root name is configurable and currently supports `PatrolDevice` and `PatrolHost` in `common/isp`.
- The user wants to treat local heartbeat checking and periodic device reporting as separate concepts.

## Protocol Scenarios

Registration response intervals differ by root tag and integration scenario:

| Root tag | Scenario | Registration response item fields |
|----------|----------|-----------------------------------|
| `PatrolDevice` | Robot/drone reconnects to edge node and sends registration; edge node replies | `heart_beat_interval`, `patroldevice_run_interval`, `nest_run_interval`, `weather_interval` |
| `PatrolHost` | Edge node reconnects to regional inspection host and sends registration; regional host replies | `heart_beat_interval`, `patroldevice_run_interval`, `nest_run_interval`, `weather_interval` |
| `PatrolHost` | Regional inspection host reconnects to upper system and sends registration; upper system replies | `heart_beat_interval`, `patroldevice_run_interval`, `nest_run_interval`, `weather_interval` |
| `CloudHost` | Regional host sends registration/heartbeat/system messages to upper algorithm/cloud system | `heart_beat_interval`, `run_params_interval`; likely out of scope for now |

## Requirements

- Parse registration response interval items beyond `heart_beat_interval`.
- Reserve known protocol interval fields even if this task does not yet send every corresponding report type.
- Preserve heartbeat behavior as its own interval-driven system message flow.
- Introduce local in-memory cache for downstream report data received via proto/gRPC.
- Change relevant proto/gRPC report semantics so downstream reports update local memory instead of synchronously sending ISP messages upstream.
- Provide a generic reporting mode that future report types can reuse:
  - Registration parsing stores interval values by report category.
  - Downstream proto/gRPC input writes into a category-specific local cache.
  - Periodic reporters read the cache and send the mapped ISP Type/Command report.
  - Freshness checks mark each category independently as expired.
  - Adding a new report later should not require inventing a new lifecycle model.
- Limit concrete reporting implementation in this task to the currently selected proto-provided report types; unsupported interval fields are parsed/reserved but do not require full upstream reporting yet.
- Add periodic reporting from local memory to upstream ISP according to registration response intervals:
  - `patroldevice_run_interval` for patrol device run data only.
  - Current proto-provided patrol device status data and coordinates are separate report categories; registration does not define their reporting frequency, so they use default 1 minute report intervals and must not reuse heartbeat or `patroldevice_run_interval`.
  - `nest_run_interval` for drone nest run data when supported.
  - `weather_interval` for micro-weather/environment data.
- Add stale-data tracking separate from upstream reporting intervals:
  - Each downstream cache update records last update time.
  - A local check loop detects when downstream data has not been refreshed within the expected freshness window.
  - Expired local data is marked in memory so reporting logic can avoid treating stale input as fresh.
  - Periodic reporting skips expired data instead of sending stale values upstream.
- Keep interval units as seconds, matching protocol item definitions.
- Treat missing or invalid interval items as fallback/default behavior rather than crashing registration.
- Exclude `CloudHost` / `run_params_interval` from implementation unless later explicitly requested.

## Acceptance Criteria

- [ ] Registration response parsing recognizes `heart_beat_interval`, `patroldevice_run_interval`, `nest_run_interval`, and `weather_interval` as second-based durations.
- [ ] Heartbeat send interval remains independent from report intervals and stale-data checks.
- [ ] Registration interval fields that are not implemented as upstream reports yet are still preserved in the local interval state for future use.
- [ ] Reporting implementation uses a reusable report-category model for cache update, periodic send, and stale detection rather than hard-coding a separate lifecycle for each current report.
- [ ] Patrol device run/status/coordinate gRPC report calls no longer directly send ISP messages upstream; they update local memory and return a local acceptance response.
- [ ] A periodic reporter sends cached patrol device run data upstream according to `patroldevice_run_interval` after successful registration, while status/coordinate categories use the default 1 minute interval unless overridden later.
- [ ] Local stale detection marks cached downstream data expired when it is not refreshed within the configured freshness window.
- [ ] Missing interval fields fall back to configured defaults or disable the corresponding periodic report explicitly.
- [ ] Existing task dispatch, task control, model sync, registration, heartbeat, and generic command behavior are not regressed.
- [ ] Tests cover registration interval parsing, cache update behavior, periodic reporting decision logic, and stale marking behavior.
- [ ] Documentation explains protocol interval fields and the distinction between heartbeat, upstream report interval, and downstream freshness timeout.

## Out of Scope

- `CloudHost` root tag and `run_params_interval` handling.
- Real robot/drone hardware control behavior.
- Persistent storage for transient device report cache.
- Changing ISP protocol XML field names beyond the interval fields listed above.

## Notes

- Keep `prd.md` focused on requirements, constraints, and acceptance criteria.
- Lightweight tasks can remain PRD-only.
- For complex tasks, add `design.md` for technical design and `implement.md` for execution planning before `task.py start`.
