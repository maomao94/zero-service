# Research: `common/djisdk` Package Analysis

- **Query**: Analyze `common/djisdk/` package for Trellis spec content
- **Scope**: Internal
- **Date**: 2026-06-25

## Findings

### File Organization and Responsibilities

| File | Lines | Responsibility |
|---|---|---|
| `common/djisdk/doc.go` | 19 | Package-level godoc: topic links, behavior conventions |
| `common/djisdk/client.go` | ~1116 | `Client` struct, option pattern, handler registration, command sending, DRC controls |
| `common/djisdk/protocol.go` | ~1094 | All message structs: request/reply/event payloads |
| `common/djisdk/method.go` | 524 | DJI Cloud API method string constants, organized by feature module |
| `common/djisdk/topic.go` | 174 | MQTT topic construction functions and wildcard pattern functions |
| `common/djisdk/protocol_drc.go` | 372 | DRC-specific protocol: down/up message types, unmarshaling, summary |
| `common/djisdk/error.go` | 45 | `DJIError` struct, creation, type assertion helper |
| `common/djisdk/error_descriptions.go` | 452 | Chinese error code description map |
| `common/djisdk/client_test.go` | — | Client tests |
| `common/djisdk/protocol_drc_test.go` | — | DRC protocol tests |

### Key Types and Their Roles

#### 1. `Client` (client.go:78-99)
- Central orchestrator wrapping `mqttx.Client` (a thin MQTT layer)
- **Handler fields** (private function pointer per event type):
  - `onFlightTaskProgress`, `onFlightTaskReady`, `onReturnHomeInfo`, `onCustomDataFromPsdk`, `onCustomDataFromEsdk`, `onHmsEventNotify`, `onRemoteLogProgress`, `onOtaProgress`, `onTopoUpdate` — typed notification handlers for `events` topic
  - `onOsd`, `onState` — typed telemetry handlers for `osd`/`state` topics
  - `onStatus` — `StatusHandler` for `sys/.../status`
  - `onRequest` — `RequestHandler` for `thing/.../requests`
  - `onDrcUp` — `DrcUpHandler` for `thing/.../drc/up`
- **Config fields**: `pendingTTL`, `replyOptions`, `eventMethodFallbacks` map, `onlineChecker`
- **Construction**: `MustNewClient(config, opts...)` creates own MQTT client; `NewClient(mqttClient, opts...)` reuses existing.

#### 2. Handler Types (client.go)
- `StatusHandler` `func(ctx, gatewaySn string, data *StatusMessage) int` — returns result code
- `RequestHandler` `func(ctx, gatewaySn string, req *RequestMessage) (result int, output any, err error)`
- `DrcUpHandler` `func(ctx, gatewaySn string, msg *DrcUpMessage, parsed any) error` — `parsed` is strongly typed
- `EventMethodFallback` `func(ctx, event *EventMessage) (result int, err error)` — fallback for unregistered event methods
- On* handlers: `func(ctx, gatewaySn string, data *TypedData)` — void, fire-and-forget notification pattern

#### 3. ReplyRouter and Pending TTL (client.go)
- `replyRouters(ttl)` registers `mqttx.WithReplyRouter` for `services_reply` and `property/set_reply` patterns.
- `decodeServiceReply(kind)` decodes JSON into `ServiceReply`, extracting `tid`, `method`, `result` for correlation.
- The router enables `mqttx.RequestReply` blocking pattern used by `SendCommand` and `SetProperty`.

#### 4. Message Structures (protocol.go)
- **`ServiceRequest`**: `tid`, `bid`, `timestamp`, `method`, `gateway`, `data any`
- **`ServiceReply`**: `tid`, `bid`, `timestamp`, `method`, `data` (result + output)
- **`EventMessage`**: additional `need_reply` field
- **`EventReply`**: result-only response
- **`TelemetryMessage`** / `OsdMessage` / `StateMessage`: shared structure
- **`RequestMessage`**: device-to-cloud requests
- **`StatusMessage`**: sys-level status
- **`PropertySetData`**: `map[string]any` — key-value write payload

#### 5. DRC Protocol Structures (protocol_drc.go)
- **`DrcDownMessage`**: cloud-to-device realtime control (stick_control, heart_beat, emergency_stop)
- **`DrcUpMessage`**: device-to-cloud DRC reports (heart_beat, HSI, delay, OSD)
- **`DrcUnknownUpData`**: graceful fallback for unmodeled methods
- **`DrcUnmarshalUpData(method, data)`**: switch-based type dispatch, returns typed struct or `DrcUnknownUpData`
- **Custom `UnmarshalJSON`** on `DrcHsiInfoPushData` to handle `around_distance`/`around_distances` key variation

### Common Patterns

#### 1. Client Option Pattern (client.go:47-66)
```go
type ClientOption func(*clientOptions)

func WithPendingTTL(ttl time.Duration) ClientOption { ... }
func WithReplyOptions(ro ReplyOptions) ClientOption { ... }
```
Used via: `djisdk.MustNewClient(config, djisdk.WithPendingTTL(...))`

#### 2. Handler Registration Pattern
- Registration: `OnEvent(method, handler)`, `OnFlightTaskProgress(handler)`, etc.
- Does **not** use interface-based dispatch — uses function pointer fields directly.
- Event dispatch (client.go:230-343): `tryDispatchEventNotify` with a big `switch` on `Method*` constant.
  - Falls to `eventMethodFallbacks[m][ethod]` if no pre-built On* registered.
  - If neither, the event is silently dropped.
- Registration is idempotent — last-wins (simple field assignment).

#### 3. Command Sending Patterns
**Blocking** (`SendCommand`):
```go
tid := uuid.New().String()
bid := uuid.New().String()
req := NewServiceRequest(tid, bid, method, data)
payload, _ := json.Marshal(req)
topic := ServicesTopic(gatewaySn)
reply, err := mqttx.RequestReply[*ServiceReply](ctx, c.mqttClient, ServicesReplyTopicPattern(), tid, func() error {
    return c.mqttClient.Publish(ctx, topic, payload)
}, c.pendingTTL)
```
**Fire-and-forget** (`SendCommandFireAndForget`): publishes without reply wait.
**DRC down** (`publishDrcDown`): publishes to `drc/down` topic without reply wait.

#### 4. Reply Routing
- `services_reply` and `property/set_reply` use `mqttx.ReplyRouter` registered at client creation.
- Router matches incoming reply by `tid`, decodes, and completes the pending `RequestReply` future.
- `events_reply` is inline: `replyEvent` publishes directly after handler returns.

#### 5. Error Handling (error.go)
- `DJIError` wraps DJI device result codes with name (from protobuf enum) + Chinese description.
- `NewDJIError(code)` — lookup in `dji_error_code.DJIErrorCode_name` map + `djiErrorDescriptions` map.
- `IsDJIError(err)` — typed `errors.As` check.

#### 6. MQTT Topic Construction (topic.go)
- Each topic channel has two functions: `*Topic(gatewaySn)` for specific Publish, `*TopicPattern()` for wildcard Subscribe.
- Topics grouped as: `thing/product/{sn}/{channel}` and `sys/product/{sn}/{channel}`.
- Wildcard always uses `+` for single-level matching.

### Method Constants Organization (method.go)
Methods are grouped by feature module, each with doc header pointing to DJI docs:
- **Properties**: `property_set`
- **Device**: `update_topo`
- **Organization**: `airport_organization_bind/get`, `airport_bind_status`
- **Live**: `live_start_push`, `live_stop_push`, `live_set_quality`, `live_lens_change`, `live_camera_change`
- **Media**: `upload_flighttask_media_prioritize`, `media_fast_upload`, `highest_priority_upload_flighttask_media`
- **Wayline**: `flighttask_prepare/execute/undo/pause/recovery/stop`, `return_home`, `return_home_cancel`, `return_specific_home`
- **HMS**: `hms`
- **Cmd**: `debug_mode_open/close`, `cover_open/close/force_close`, `drone_open/close`, `device_reboot`, `charge_open/close`, `drone/device_format`, `supplement_light_open/close`, `battery_store_mode_switch`, `alarm_state_switch`, `battery_maintenance_switch`, `air_conditioner_mode_switch`
- **Firmware**: `ota_create`, `ota_progress`
- **Log**: `fileupload_list/start/update/cancel/progress`
- **Config**: `config_update`
- **DRC**: `flight_authority_grab`, `payload_authority_grab`, `drc_mode_enter/exit`, `stick_control`, `drone_emergency_stop`, `fly_to_point/stop`, `takeoff_to_point`
- **DRC Channel**: `heart_beat`, `hsi_info_push`, `delay_info_push`, `osd_info_push`
- **Camera**: `camera_mode_switch`, `camera_photo_take/stop`, `camera_recording_start/stop`, `camera_focal_length_set`, `gimbal_reset`, `camera_aim`, etc.
- **Flysafe**: `unlock_license_switch/update/list`
- **Flysafe Remote Control**: `drc_force_landing`, `drc_emergency_landing`, `drc_linkage_zoom_set`, etc.

### API Service Coverage (client.go)
Client provides typed methods for nearly all method constants:
- `LiveStartPush`, `LiveStopPush`, `LiveSetQuality`, `LiveLensChange`, `LiveCameraChange`
- `MediaUploadFlighttaskMediaPrioritize`, `MediaFastUpload`, `MediaHighestPriorityUploadFlighttask`
- `FlightTaskPrepare`, `FlightTaskExecute`, `CancelFlightTask`, `PauseFlightTask`, `ResumeFlightTask`, `StopFlightTask`
- `ReturnHome`, `ReturnHomeCancelAutoReturn`, `ReturnSpecificHome`
- `DebugModeOpen`, `DebugModeClose`, `CoverOpen`, `CoverClose`, `CoverForceClose`
- `DroneOpen`, `DroneClose`, `DeviceReboot`, `ChargeOpen`, `ChargeClose`
- `DroneFormat`, `DeviceFormat`, `SupplementLightOpen`, `SupplementLightClose`
- `BatteryStoreModeSwitch`, `AlarmStateSwitch`, `BatteryMaintenanceSwitch`, `AirConditionerModeSwitch`
- `OtaCreate`, `ConfigUpdate`, `SetProperty`
- `FlightAuthorityGrab`, `PayloadAuthorityGrab`, `DrcModeEnter`, `DrcModeExit`
- `SendDrcStickControl`, `SendDrcHeartBeat`, `DroneEmergencyStop`
- `CameraModeSwitch`, `CameraPhotoTake`, `CameraPhotoStop`, `CameraRecordingStart`, `CameraRecordingStop`, `CameraFocalLengthSet`, `GimbalReset`, `CameraAim`, `CameraPointFocusAction`, `CameraScreenSplit`, `CameraPhotoStorageSet`, `CameraVideoStorageSet`, `CameraLookAt`, `CameraScreenDrag`, `CameraIrMeteringPoint`, `CameraIrMeteringArea`
- `RemoteLogFileList`, `RemoteLogFileUploadStart`, `RemoteLogFileUploadUpdate`, `RemoteLogFileUploadCancel`
- Not yet covered: `flight_areas_update`, `psdk_ui_resource_upload`, `custom_data_transmission_to_*`, `unlock_license_*`, `drc_*_set` DRC remote control methods

### Platform Result Constants (protocol.go:7-11)
```go
PlatformResultOK           = 0  // success
PlatformResultHandlerError = 1  // unregistered handler / internal parse error
PlatformResultTimeout      = 2  // DJI protocol timeout convention; do NOT repurpose
```

### Protocol Direction Conventions (doc.go)
- **events**: device → cloud; strong-type dispatch via `tryDispatchEventNotify`, fallback via `OnEvent`
- **status**: device → cloud; `StatusHandler` returns result; reply controlled by `ReplyOptions`
- **requests**: device → cloud (pull); cloud returns `result + data.output`
- **property/set**: cloud → device (write); `property/set_reply` device → cloud
- **drc/down**: cloud → device (realtime)
- **drc/up**: device → cloud (realtime status/ack)
- **drc_mode_enter/exit**: NOT drc/*, goes through services + services_reply

### Noteworthy Patterns

1. **Graceful unknown method handling**: `DrcUnmarshalUpData` returns `DrcUnknownUpData` for unmodeled methods, allowing hook continuation.
2. **JSON key compatibility**: Custom `UnmarshalJSON` on `DrcHsiInfoPushData` handles both `around_distance` and `around_distances` due to spec variations.
3. **Online checker**: Optional pre-flight check via `SetOnlineChecker`; if set, `SendCommand` rejects offline devices early.
4. **Reply options**: Global toggle for whether to send `events_reply`, `status_reply`, `requests_reply`.
5. **Log helper**: `logFields(kv...)` formats key-value pairs for structured logging.
6. **Event dispatch hierarchy**: Pre-built typed handlers > eventMethodFallbacks map > silent drop.

### Anti-patterns / Pitfalls

1. **Mutable struct fields**: All On* setters directly assign struct fields — no locking, no atomic. Not safe for concurrent registration (though typically done once at init).
2. **Big switch dispatch**: `tryDispatchEventNotify` is a 100+ line switch statement — adding a new event type requires adding both a `case` branch and a new handler field + setter.
3. **Package-level error descriptions**: `djiErrorDescriptions` is a 450-line map literal in a separate file — easy to forget to update when upstream error codes change.
4. **Panic in constructor**: `MustNewClient` panics if MQTT connection fails — caller cannot gracefully degrade.
5. **Inconsistent On* naming**: `OnUpdateTopo` is deprecated in favor of `OnTopoUpdate` (line 430-432), both exist.

## External References

- [DJI Cloud API Topic Definition](https://developer.dji.com/doc/cloud-api-tutorial/cn/api-reference/dock-to-cloud/mqtt/topic-definition.html)
- [DJI Dock/Device Status](https://developer.dji.com/doc/cloud-api-tutorial/cn/api-reference/dock-to-cloud/mqtt/dock/dock3/device.html)
- [DJI Organization Requests](https://developer.dji.com/doc/cloud-api-tutorial/cn/api-reference/dock-to-cloud/mqtt/dock/dock3/organization.html)
- [DJI Properties](https://developer.dji.com/doc/cloud-api-tutorial/cn/api-reference/dock-to-cloud/mqtt/dock/dock3/properties.html)
- [DJI DRC (Direct Remote Control)](https://developer.dji.com/doc/cloud-api-tutorial/cn/api-reference/dock-to-cloud/mqtt/dock/dock3/drc.html)
- [DJI Error Codes](https://developer.dji.com/doc/cloud-api-tutorial/cn/error-code.html)

## Caveats / Not Found

- No interface-based abstraction for MQTT client (tightly coupled to `mqttx.Client`).
- No subscription lifecycle management exposed (internal `SubscribeAll`).
- No rate limiting or back-pressure on `SendDrcStickControl`.
