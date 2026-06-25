# Research: `app/djicloud` Package Analysis

- **Query**: Analyze `app/djicloud/` package for Trellis spec content
- **Scope**: Internal
- **Date**: 2026-06-25

## Findings

### File Organization and Responsibilities

| File | Lines | Responsibility |
|---|---|---|
| `app/djicloud/djicloud.go` | 75 | Main entry point; initializes Nacos, gRPC server, cron job |
| `app/djicloud/djicloud.proto` | ~1353 | gRPC service definition: 70+ RPCs, message types |
| `app/djicloud/djicloud/djicloud.pb.go` | generated | Protobuf message types |
| `app/djicloud/djicloud/djicloud_grpc.pb.go` | generated | gRPC server/client stubs |
| `internal/config/config.go` | — | YAML configuration struct |
| `internal/svc/servicecontext.go` | 156 | Dependency wiring: MQTT client, DB, caches, DRC manager, hook registration |
| `internal/svc/device_online_refresher.go` | 53 | Cron job marking expired devices offline |
| `internal/server/djicloudserver.go` | 612 | gRPC server — generated goctl skeleton; delegates to logic |
| `internal/hooks/register.go` | 59 | Registers all MQTT uplink handlers into `djisdk.Client` |
| `internal/hooks/sys_status_up.go` | 115 | Handles `sys/.../status` (update_topo → device + topo persistence) |
| `internal/hooks/telemetry_up.go` | 164 | Handles `thing/.../osd` and `thing/.../state` (device online + snapshot + socket push) |
| `internal/hooks/event_notify_up.go` | 218 | Handles `thing/.../events` (task progress, HMS, OTA, log progress) |
| `internal/hooks/mqtt_request_up.go` | 41 | Handles `thing/.../requests` (org bind, flight areas get) |
| `internal/hooks/mqtt_drc_up.go` | 75 | Handles `thing/.../drc/up` (DRC heartbeats, acks -> DRC manager + socket push) |
| `internal/hooks/store_helper.go` | 87 | Utility functions: time conversion, JSON marshaling, version extraction, mission state text |
| `internal/hooks/online_cache.go` | — | Online cache helper (`IsOnline`, `OnlineValue`) |
| `internal/drc/manager.go` | — | DRC session lifecycle manager |
| `internal/drc/state.go` | — | DRC state tracking |
| `internal/logic/helper.go` | 174 | gRPC response helpers, model ↔ proto converters |
| `internal/logic/drchelper.go` | 35 | DRC MQTT broker config conversion |
| `internal/logic/listdeviceslogic.go` | 135 | ListDevices: paginated query + parallel sub-queries |
| `internal/logic/getdevicedetaillogic.go` | 71 | GetDeviceDetail: single device + OSD + State + topo |
| `internal/logic/livestartpushlogic.go` | 40 | Example DJI command logic (typical pattern) |
| `internal/logic/*logic.go` | ~90 files | One logic file per gRPC RPC, most are thin wrappers |
| `model/gormmodel/dji_device.go` | 73 | DjiDevice + DjiDeviceTopo GORM models |
| `model/gormmodel/dji_osd_state.go` | 39 | DjiDeviceOsdSnapshot + DjiDeviceStateSnapshot |
| `model/gormmodel/dji_event.go` | 151 | DjiHmsAlert, DjiDockFlightTask, DjiDockDeviceFlightTaskState, DjiFlightTaskReady, DjiRemoteLogEvent, DjiReturnHomeEvent, DjiDrcUpEvent |

### Architecture Overview

```
djicloud.go (main)
    └── svc.NewServiceContext(config)
            ├── djisdk.MustNewClient(mqttConfig, opts...)  // MQTT client
            ├── collection.NewCache(dockOnlineTTL)          // online status cache
            ├── gormx.MustOpenWithConf(dbConfig)            // DB
            ├── socketpush.NewSocketPushClient(...)         // WebSocket push (optional)
            ├── drc.NewManager(djiCli, drcConfig)           // DRC lifecycle manager
            ├── hooks.RegisterDjiClient(djiCli, options)    // register all MQTT handlers
            ├── djiCli.SubscribeAll()                       // start MQTT subscriptions
            └── svc.NewDeviceOnlineRefreshCron(db)          // 15s cron for offline expiry

djicloudserver.go (gRPC server)
    ├── Ping -> PingLogic
    ├── LiveStartPush -> LiveStartPushLogic -> djiClient.LiveStartPush()
    ├── ListDevices -> ListDevicesLogic -> DB query
    └── ... 70+ RPCs
```

### Key Patterns

#### 1. Hook Registration (hooks/register.go)
Central point for all MQTT handler registration. Organized by message type:
```go
func RegisterDjiClient(c *djisdk.Client, o RegisterDjiClientOptions) {
    registerEventHandlers(c, o.DB)        // OnFlightTaskProgress, OnFlightTaskReady, OnReturnHomeInfo, etc.
    registerTelemetryHandlers(c, o.DB, ...) // OnOsd, OnState, OnStatus, OnDrcUp
    registerRequestHandlers(c)             // OnRequest
    registerOnlineChecker(c, o.OnlineCache) // SetOnlineChecker
}
```
**Dependency injection pattern**: `RegisterDjiClientOptions` struct holds all external dependencies (DB, cache, manager, push client). This avoids global state.

#### 2. MQTT Uplink → DB Persistence Pattern (hooks/telemetry_up.go, event_notify_up.go)
Every MQTT handler follows this pattern:
1. Nil check
2. Log receipt
3. Extract gateway_sn from message (or reject if missing)
4. Convert timestamp using `reportTime(ms)` → falls back to `time.Now()` if zero
5. Build GORM model struct
6. `Where(...).Assign(updateData).FirstOrCreate(&record)` — GORM upsert pattern
7. On error, log and return (never block the MQTT callback)

#### 3. GORM Upsert Pattern (used everywhere in hooks)
```go
c := db.WithContext(ctx)
deviceWhere := map[string]any{"device_sn": deviceSn}
if err := c.Where(deviceWhere).Assign(updateData).FirstOrCreate(&device).Error; err != nil {
    logx.WithContext(ctx).Errorf(...)
}
```
- `Where` sets the lookup key (unique index)
- `Assign` sets the values to update on conflict
- `FirstOrCreate` creates if not found, updates if found

#### 4. Status Handler (hooks/sys_status_up.go)
- Only processes `MethodUpdateTopo`; other methods are silently ignored.
- Runs inside a `db.Transact`:
  1. Upsert gateway device record (`DjiDevice`)
  2. Clear stale sub-device topo entries (`DjiDeviceTopo`) not in the current report
  3. Upsert each sub-device topo entry
  4. Upsert sub-device records — **skips GatewaySn update for Domain 0/1** (aircraft/payload in frog-jump scenarios)
  5. Uses `gormx.Restore` for soft-delete recovery

#### 5. OSD/State Handlers (hooks/telemetry_up.go)
- **OSD handler**: Updates device online status in `onlineCache` (in-memory) + DB, upserts `DjiDeviceOsdSnapshot`, pushes to WebSocket room `thing/product/{sn}/osd`
- **State handler**: Extracts firmware/hardware versions from state data, updates `DjiDevice`, upserts `DjiDeviceStateSnapshot`, pushes to WebSocket room `thing/product/{sn}/state`
- WebSocket push runs in `threading.GoSafe` goroutine with `context.WithoutCancel`
- `disableSQLTrace` option removes SQL from logs for high-frequency writes

#### 6. Event Handler Dispatch (hooks/event_notify_up.go)
Each event type gets its own handler function:
- `NewFlightTaskProgressHandler(db)` — upserts two tables: `DjiDockFlightTask` (per-flight) and `DjiDockDeviceFlightTaskState` (per-gateway)
- `NewFlightTaskReadyHandler(db)` — inserts `DjiFlightTaskReady`
- `NewReturnHomeInfoHandler(db)` — inserts `DjiReturnHomeEvent`
- `HandleCustomDataFromPsdkEvent` — logs only (no persistence)
- `NewHmsEventNotifyHandler(db)` — inserts per-item into `DjiHmsAlert`
- `NewRemoteLogFileUploadProgressHandler(db)` — inserts `DjiRemoteLogEvent`
- `HandleOtaProgressEvent` — logs only
- `HandleCustomDataFromEsdkEvent` — logs only

#### 7. DRC Up Handler (hooks/mqtt_drc_up.go)
- Non-high-frequency methods (not heart_beat, osd, HSI, delay, initial_state_subscribe) are persisted to `DjiDrcUpEvent`
- Heartbeat: refreshes `drcMgr.OnDeviceHeartbeat` + pushes to WebSocket room `drc:heartbeat:{sn}`
- Logs short summary via `djisdk.DrcUpPayloadSummary`

#### 8. Request Handler (hooks/mqtt_request_up.go)
Returns static/safe defaults for device uplink requests:
- `airport_organization_get` → empty org
- `airport_bind_status` → status 0
- `flight_areas_get` → empty list
- Unknown methods → success with nil output

#### 9. gRPC Logic Pattern (internal/logic/)
Each RPC follows go-zero convention:
```go
type XxxLogic struct {
    ctx    context.Context
    svcCtx *svc.ServiceContext
    logx.Logger
}

func (l *XxxLogic) Xxx(in *djicloud.XxxReq) (*djicloud.CommonRes, error) {
    // 1. Convert proto request → djisdk data struct
    // 2. Call svcCtx.DjiClient.Method()
    // 3. On err: l.Errorf(...); return errRes(tid, err), nil
    // 4. Return okRes(tid), nil
}
```
Example from `livestartpushlogic.go`:
```go
func (l *LiveStartPushLogic) LiveStartPush(in *djicloud.LiveStartPushReq) (*djicloud.CommonRes, error) {
    data := &djisdk.LiveStartPushData{
        URLType:      int(in.UrlType),
        URL:          in.Url,
        VideoID:      in.VideoId,
        VideoQuality: int(in.VideoQuality),
    }
    tid, err := l.svcCtx.DjiClient.LiveStartPush(l.ctx, in.DeviceSn, data)
    if err != nil {
        l.Errorf("[live] live start push failed: %v", err)
        return errRes(tid, err), nil
    }
    return okRes(tid), nil
}
```

#### 10. Query Pattern for Device Lists (logic/listdeviceslogic.go)
Uses `mr.Finish` (parallel fan-out from go-zero) to query 4 sub-resources concurrently:
1. `DjiDeviceOsdSnapshot` by `device_sn`
2. `DjiDeviceStateSnapshot` by `device_sn`
3. `DjiDeviceTopo` by `gateway_sn OR sub_device_sn`
4. `DjiDockDeviceFlightTaskState` by `gateway_sn`
Errors except `RecordNotFound` are propagated.

#### 11. Device Online Management
**Three layers**:
1. **In-memory cache** `collection.Cache(dockOnlineTTL=60s)` — set on OSD receive, read by `SetOnlineChecker` + `IsDeviceOnline`
2. **DB column** `is_online` on `DjiDevice` — set true on OSD, set false by cron
3. **Cron** `DeviceOnlineRefreshCron` — every 15s, sets `is_online=false` for devices whose `last_online_at < now - 60s`

#### 12. Helper Functions (hooks/store_helper.go)
- `reportTime(ms int64) time.Time` — convert millis to Time, fallback to now
- `toJSONString(v any) string` — safe JSON marshaling with `{}` fallback
- `sqlNullTime(t time.Time) sql.NullTime` — convert Time to NullTime
- `extractDeviceVersions(v any) deviceVersions` — marshal/unmarshal to extract fw/hw version from state payload
- `waylineMissionStateText(state int) string` — numeric state to human-readable label
- `appendVersionUpdateColumns` — build conditional column list for version updates

#### 13. Proto/Model Conversion Helpers (logic/helper.go)
- `toDeviceInfo(m *DjiDevice) *DeviceInfo` — DB model → proto
- `toOsdSnapshot`, `toStateSnapshot` — snapshot DB → proto
- `toTelemetrySnapshotBrief`, `toTelemetrySnapshotBriefFromState` — brief snapshot (no raw JSON in list context)
- `toTopoInfoList` — topo slice → proto
- `toDockFlightTaskStateInfo` — task state → proto
- `errRes(tid, err) *CommonRes`, `okRes(tid) *CommonRes` — response constructors
- `normalizePage(page, pageSize)` — pagination bounds
- `deviceOnlineExpired(m, now)` — TTL check helper

### Database Model Design

All tables use `gormx.LegacyBaseModel` (includes `id`, `create_time`, `update_time`, potentially `delete_at`).

#### `dji_device` (dji_device.go:32-53)
- **PK**: `device_sn` (uniqueIndex)
- **Key fields**: `gateway_sn` (index), `is_online` (index, default false), `firmware_version`, `hardware_version`, `first_online_at`, `last_online_at`
- **Usage**: Master device table; records every device seen via OSD/State/update_topo
- **Frog-jump**: For domain 0/1 (aircraft/payload), `gateway_sn` is updated by OSD/State only (not update_topo); multiple gateway bindings expressed via `DjiDeviceTopo`
- **Online semantics**: DB default false; OSD sets true; cron sets false after 60s TTL

#### `dji_device_topo` (dji_device.go:62-73)
- **PK**: `gateway_sn` + `sub_device_sn` (composite uniqueIndex)
- **Key fields**: `domain`, `sub_device_type`, `sub_device_sub_type`, `sub_device_index`, `thing_version`
- **Allows** multiple `sub_device_sn` across different `gateway_sn` (frog-jump)

#### `dji_device_osd_snapshot` (dji_osd_state.go:15-23)
- **PK**: `device_sn` (uniqueIndex)
- **Key fields**: `raw_json` (jsonb), `reported_at` (index)
- **Strategy**: Upsert-only — one row per device

#### `dji_device_state_snapshot` (dji_osd_state.go:31-38)
- Same pattern as OSD snapshot

#### `dji_hms_alert` (dji_event.go:16-34)
- Insert-only per item; `acked`, `acked_at`, `acked_by` for manual confirmation

#### `dji_dock_flight_task` (dji_event.go:37-56)
- **PK**: `gateway_sn` + `flight_id` (composite uniqueIndex)
- Upsert on each progress update

#### `dji_dock_device_flight_task_state` (dji_event.go:59-80)
- **PK**: `gateway_sn` (uniqueIndex)
- Upsert — one row per gateway, always the latest state

#### `dji_flight_task_ready` (dji_event.go:88-97)
- Insert-only

#### `dji_remote_log_event` (dji_event.go:105-114)
- Insert-only

#### `dji_return_home_event` (dji_event.go:122-134)
- Insert-only

#### `dji_drc_up_event` (dji_event.go:142-151)
- Insert-only (for non-high-frequency DRC methods)

### DRC Manager Integration

- `drc.NewManager(djiCli, drcConfig, opts...)` manages DRC session lifecycle
- Handlers: `WithOnSessionEnabled`, `WithOnSessionDisabled`, `WithOnSessionExpired` — push callbacks to WebSocket
- DRC mode enter/exit calls go through `services` + `services_reply` (not drc/* topic)
- `sendDrcStickControl` goes through `drc/down` (fire-and-forget)
- Heartbeat on `drc/up` refreshes manager state

### External References

- DJI Cloud API docs (same as djisdk — the protocol is the interface)
- `third_party/dji_error_code` — protobuf error code enum used by `djisdk.NewDJIError`
- `common/mqttx` — MQTT client abstraction used by djisdk
- `common/gormx` — GORM wrapper (LegacyBaseModel, Transact, QueryPage, Restore)
- `socketapp/socketpush/socketpush` — WebSocket push service client

### Caveats / Not Found

- LSP errors in `listdeviceslogic.go:40/44` — `in.TopoGatewaySn` does not match generated proto (likely field mismatch with `.pb.go`); the proto field name should be `topo_gateway_sn` -> `TopoGatewaySn` but generated code might use different casing.
- No comprehensive integration tests for the MQTT → DB flow.
- No unit tests for the process of "OSD arrival → online check → refresher cron".
- `extractDeviceVersions` uses marshal/unmarshal round-trip on `data.Data` which is `any` — if the state data contains unexpected nested structures, it silently produces empty versions.
- Some DJI protocol modules (AirSense, PSDK upload) are handled purely for future expansion — current handlers just log.
- The service lacks subscription to `sys/product/+/status_reply` (device confirms cloud's ack) — this is one-directional only.
