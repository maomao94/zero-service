# ISP interval-driven device reporting design

## Scope

This task changes `ispagent` reporting semantics for the currently exposed proto report RPCs:

- `SendPatrolDeviceRunData`
- `SendPatrolDeviceStatusData`
- `SendPatrolDeviceCoordinates`

These RPCs will accept downstream input into local memory. Upstream ISP reporting will be done by the ISP client on a schedule after successful registration.

## Architecture

Add a reusable report category layer inside `app/ispagent/internal/isp` or a small sibling package used by the client.

Core concepts:

| Concept | Responsibility |
|---------|----------------|
| Report category | Stable key for one report stream, such as patrol device run data, status data, or coordinates |
| Report cache | Stores latest `code`, `items`, update time, expired flag, and optional last-sent time per category |
| Interval state | Stores parsed registration intervals by protocol field/category |
| Reporter loop | Runs independently from heartbeat and sends fresh cached data when a category is due |
| Freshness loop | Marks cache entries expired when downstream updates stop arriving |

The model must be data-driven enough that future report types can be added by registering a new category mapping instead of writing a new lifecycle.

## Data Flow

### Registration

1. TCP session connects.
2. `doRegister` sends `251-1` registration.
3. `251-4` response items are parsed into interval state:
   - `heart_beat_interval` updates heartbeat only.
   - `patroldevice_run_interval` updates patrol-device report mode.
   - `nest_run_interval` and `weather_interval` are reserved in state for future reporters.
4. Reporter due times reset after registration to avoid reporting before a successful session exists.

### Downstream gRPC Report

1. Logic converts proto message to `[]isp.Item` using existing converter helpers.
2. Logic calls the client cache API instead of `Execute`.
3. Cache stores the latest payload for the corresponding category and clears expired state.
4. RPC returns local acceptance, not upstream ISP response.

### Periodic Upstream Report

1. Client tick checks heartbeat separately.
2. Reporter loop checks report category due times.
3. If registered and category has a positive interval, fresh data, and non-empty items, send ISP report with mapped Type/Command.
4. If category is expired or empty, skip send and log at a low-noise level.
5. Send failures should log but not clear cached data.

### Freshness Detection

Freshness is not the same as upstream reporting interval. The cache should have a configurable or derived freshness window. For this task, derive freshness from the report interval unless a clearer config already exists:

```text
freshness timeout = max(report interval * 2, report interval + 10s)
```

If no interval exists, use a safe default freshness timeout from config or code default. Expired cache entries are not sent upstream.

## Contracts

### Registration Interval Fields

Known fields are parsed as seconds:

- `heart_beat_interval`
- `patroldevice_run_interval`
- `nest_run_interval`
- `env_interval`
- `weather_interval`

Invalid values are ignored and the previous/default value remains.

### Current Report Categories

| Category | ISP Type | Command | Interval source |
|----------|----------|---------|-----------------|
| Patrol device run data | `TypePatrolDeviceRunData` | `CommandReport` | `patroldevice_run_interval` |
| Patrol device status data | `TypePatrolDeviceStatusData` | `CommandReport` | default 1 minute interval |
| Patrol device coordinates | `TypePatrolDeviceCoordinates` | `CommandReport` | default 1 minute interval; special coordinate/geography protocol |

Status and coordinates are independent ISP report categories. Because the current registration fields do not define separate status/coordinate intervals, they must not reuse heartbeat or `patroldevice_run_interval`; periodic sends use the default 1 minute interval unless a future explicit interval source overrides it.

### RPC Response

The existing `CommandRes` can still be returned. For local acceptance:

- `success=true`
- `code="200"`
- `items` may be empty
- `rawXml` empty

This is a behavior change from upstream synchronous response and must be documented.

## Compatibility

- `ExecuteCommand` remains synchronous and still sends upstream.
- Registration, heartbeat, task dispatch/control, model sync, and fallback response behavior remain unchanged.
- Existing proto method names and request types remain unchanged.
- Generated protobuf files must be regenerated only if proto messages change. This design does not require proto schema changes.

## Operational Notes

- Logging should distinguish heartbeat failures, report send failures, cache updates, and stale-data skips.
- Avoid a goroutine per category if a single existing `tick()` can handle heartbeat, registration, reporting, and freshness without becoming hard to read.
- Closing the client must stop all loops via the existing client context.

## Rollback

Rollback can restore direct sends from the three logic files and remove the cache/reporting layer. No database migration is involved.
