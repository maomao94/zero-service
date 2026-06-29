# Research: DJI Identifier References — Full Codebase Scan

- **Query**: Search entire codebase for ALL references to specified Go identifiers, proto RPCs, and messages
- **Scope**: Mixed (internal code search + proto analysis)
- **Date**: 2026-06-29

## Findings

### 1. Go Constants (method.go)

All constants live in `common/djisdk/method.go`.

#### `MethodFlightTaskCancel`

| File | Line | Content |
|---|---|---|
| `common/djisdk/method.go` | 98 | `// MethodFlightTaskCancel 航线任务取消（Flighttask Undo）` |
| `common/djisdk/method.go` | 100 | `MethodFlightTaskCancel = "flighttask_undo"` |
| `common/djisdk/client.go` | 289 | `return c.SendCommand(ctx, gatewaySn, MethodFlightTaskCancel, &FlightTaskCancelData{FlightIDs: flightIDs})` |

#### `MethodFlightTaskResume`

| File | Line | Content |
|---|---|---|
| `common/djisdk/method.go` | 106 | `// MethodFlightTaskResume 航线任务恢复（Flighttask Recovery）` |
| `common/djisdk/method.go` | 108 | `MethodFlightTaskResume = "flighttask_recovery"` |
| `common/djisdk/client.go` | 309 | `return c.SendCommand(ctx, gatewaySn, MethodFlightTaskResume, data)` |

#### `MethodPsdkFloatUp`

| File | Line | Content |
|---|---|---|
| `common/djisdk/method.go` | 437 | `// MethodPsdkFloatUp PSDK UI 资源上传（PSDK UI Resource Upload）` |
| `common/djisdk/method.go` | 439 | `MethodPsdkFloatUp = "psdk_ui_resource_upload"` |
| `common/djisdk/client.go` | 874 | `return c.SendCommand(ctx, gatewaySn, MethodPsdkFloatUp, data)` |

#### `MethodStickControl`

| File | Line | Content |
|---|---|---|
| `common/djisdk/method.go` | 313 | `// MethodStickControl DRC 杆量控制，使用 drc/down 即发即忘下发，设备可经 drc/up 回执。` |
| `common/djisdk/method.go` | 314 | `MethodStickControl = "stick_control"` |
| `common/djisdk/client.go` | 669 | `msg := NewDrcDownMessage(uuid.New().String(), uuid.New().String(), MethodStickControl, data, &seq)` |
| `common/djisdk/protocol_drc.go` | 276 | `case MethodStickControl:` |
| `common/djisdk/protocol_drc_test.go` | 287 | `parsed, err := DrcUnmarshalUpData(MethodStickControl, ...)` |
| `common/djisdk/protocol_drc_test.go` | 307 | `{name: "stick_control", method: MethodStickControl, data: ...}` |
| `common/djisdk/protocol_drc_test.go` | 355 | `{name: "stick_control", method: MethodStickControl, parsed: ...}` |
| `common/djisdk/protocol_drc_test.go` | 383 | `stick := NewDrcDownMessage("tid", "bid", MethodStickControl, ...)` |
| `common/djisdk/protocol_drc_test.go` | 401 | `msg := NewDrcDownMessage("tid", "bid", MethodStickControl, ...)` |
| `common/djisdk/protocol_drc_test.go` | 417 | `if got["method"] != MethodStickControl {` |
| `common/djisdk/protocol_drc_test.go` | 418 | `t.Fatalf("method = %v, want %s", got["method"], MethodStickControl)` |
| `common/djisdk/protocol_drc_test.go` | 490 | `if !strings.Contains(err.Error(), tid) \|\| !strings.Contains(err.Error(), MethodStickControl) {` |
| `app/djicloud/internal/hooks/register_test.go` | 105 | `... djisdk.MethodStickControl ...` |
| `app/djicloud/internal/hooks/register_test.go` | 1029 | `... djisdk.MethodStickControl ...` |
| `app/djicloud/internal/hooks/register_test.go` | 1037 | `... djisdk.MethodStickControl ...` |

---

### 2. Client Methods (client.go in common/djisdk)

#### `CancelFlightTask`

| File | Line | Content |
|---|---|---|
| `common/djisdk/client.go` | 282 | `// CancelFlightTask 取消指定的飞行任务。` |
| `common/djisdk/client.go` | 288 | `func (c *Client) CancelFlightTask(ctx context.Context, gatewaySn string, flightIDs []string) (string, error) {` |
| `common/djisdk/client.go` | 289 | `return c.SendCommand(ctx, gatewaySn, MethodFlightTaskCancel, &FlightTaskCancelData{FlightIDs: flightIDs})` |

**NOTE**: This is the **djisdk Client** method. Not to be confused with:
- Proto RPC: `rpc CancelFlightTask` in `app/djicloud/djicloud.proto:100`
- gRPC server impl: `DjiCloudServer.CancelFlightTask` in `app/djicloud/internal/server/djicloudserver.go:99`
- gRPC logic impl: `CancelFlightTaskLogic.CancelFlightTask` in `app/djicloud/internal/logic/cancelflighttasklogic.go:27`
- Proto generated types: `CancelFlightTaskReq` struct in `app/djicloud/djicloud/djicloud.pb.go:1078`
- All gRPC stub/handler refs in `app/djicloud/djicloud/djicloud_grpc.pb.go`

#### `ResumeFlightTask`

| File | Line | Content |
|---|---|---|
| `common/djisdk/client.go` | 302 | `// ResumeFlightTask 恢复已暂停的飞行任务。` |
| `common/djisdk/client.go` | 308 | `func (c *Client) ResumeFlightTask(ctx context.Context, gatewaySn string, data *FlightTaskResumeData) (string, error) {` |
| `common/djisdk/client.go` | 309 | `return c.SendCommand(ctx, gatewaySn, MethodFlightTaskResume, data)` |

Also all proto/gRPC generated and server references (see `ResumeFlightTask` in the full results above).

#### `SendCustomDataToPsdk`

| File | Line | Content |
|---|---|---|
| `common/djisdk/client.go` | 879 | `// SendCustomDataToPsdk 自定义数据透传至 PSDK 负载设备。` |
| `common/djisdk/client.go` | 881 | `func (c *Client) SendCustomDataToPsdk(ctx context.Context, gatewaySn, value string) (string, error) {` |
| `common/djisdk/client.go` | 885 | `return c.SendCommand(ctx, gatewaySn, MethodCustomDataTransmissionToPsdk, data)` |
| `app/djicloud/internal/logic/sendcustomdatatopsdklogic.go` | 36 | `tid, err := l.svcCtx.DjiClient.SendCustomDataToPsdk(l.ctx, in.DeviceSn, in.Value)` |

Plus all proto/gRPC/SendCustomDataToPsdk references in the generated code and server impl.

#### `SendCustomDataToEsdk`

| File | Line | Content |
|---|---|---|
| `common/djisdk/client.go` | 890 | `// SendCustomDataToEsdk 自定义数据透传至 ESDK 设备。` |
| `common/djisdk/client.go` | 891 | `func (c *Client) SendCustomDataToEsdk(ctx context.Context, gatewaySn, value string) (string, error) {` |
| `common/djisdk/client.go` | 893 | `return c.SendCommand(ctx, gatewaySn, MethodCustomDataTransmissionToEsdk, data)` |
| `app/djicloud/internal/logic/sendcustomdatatoesdklogic.go` | 28 | `tid, err := l.svcCtx.DjiClient.SendCustomDataToEsdk(l.ctx, in.GetDeviceSn(), in.GetValue())` |

Plus all proto/gRPC/SendCustomDataToEsdk references in the generated code and server impl.

#### `SendDrcStickControl`

| File | Line | Content |
|---|---|---|
| `common/djisdk/client.go` | 662 | `// SendDrcStickControl 经 drc/down 即发即忘地下发 stick_control 杆量。` |
| `common/djisdk/client.go` | 668 | `func (c *Client) SendDrcStickControl(ctx context.Context, gatewaySn string, seq int, data *DrcStickControlData) (string, error) {` |
| `common/djisdk/client.go` | 669 | `msg := NewDrcDownMessage(uuid.New().String(), uuid.New().String(), MethodStickControl, data, &seq)` |
| `common/djisdk/protocol_drc_test.go` | 483 | `tid, err := client.SendDrcStickControl(context.Background(), "gateway-1", 7, &DrcStickControlData{})` |
| `app/djicloud/internal/logic/senddrcstickcontrollogic.go` | 34 | `if _, err := l.svcCtx.DjiClient.SendDrcStickControl(l.ctx, deviceSn, int(seq), data); err != nil {` |

Plus all proto/gRPC/SendDrcStickControl references in the generated code and server impl.

#### `SetProperty`

| File | Line | Content |
|---|---|---|
| `common/djisdk/client.go` | 139 | `// SetProperty 设置设备属性。` |
| `common/djisdk/client.go` | 141 | `func (c *Client) SetProperty(ctx context.Context, gatewaySn string, properties PropertySetData) (string, error) {` |
| `common/djisdk/client.go` | 145 | `req := NewServiceRequest(tid, bid, MethodPropertySet, properties)` |
| `app/djicloud/internal/logic/setpropertylogic.go` | 35 | `tid, err := l.svcCtx.DjiClient.SetProperty(l.ctx, in.DeviceSn, properties)` |

Plus all proto/gRPC/SetProperty references in the generated code and server impl.

#### `PropertySet` (as Go type)

| File | Line | Content |
|---|---|---|
| `common/djisdk/method.go` | 9 | `MethodPropertySet = "property_set"` (constant) |
| `common/djisdk/protocol.go` | 497 | `// PropertySetData **仅用于云 → 设备** 的 `property/set` 载荷` |
| `common/djisdk/protocol.go` | 499 | `type PropertySetData map[string]any` |
| `app/djicloud/internal/logic/setpropertylogic.go` | 29 | `var properties djisdk.PropertySetData` |
| `common/djisdk/topic.go` | 114 | `func PropertySetTopic(gatewaySn string) string {` |
| `common/djisdk/topic.go` | 120 | `func PropertySetReplyTopicPattern() string {` |

---

### 3. DJI Cloud API Method String Patterns

#### `CustomDataTransmissionToPsdk` (as Go constant)

| File | Line | Content |
|---|---|---|
| `common/djisdk/method.go` | 441 | `// MethodCustomDataTransmissionToPsdk 自定义数据透传至 PSDK` |
| `common/djisdk/method.go` | 443 | `MethodCustomDataTransmissionToPsdk = "custom_data_transmission_to_psdk"` |
| `common/djisdk/client.go` | 885 | `return c.SendCommand(ctx, gatewaySn, MethodCustomDataTransmissionToPsdk, data)` |

#### `CustomDataTransmissionToEsdk` (as Go constant)

| File | Line | Content |
|---|---|---|
| `common/djisdk/method.go` | 455 | `// MethodCustomDataTransmissionToEsdk 自定义数据透传至 ESDK。` |
| `common/djisdk/method.go` | 456 | `MethodCustomDataTransmissionToEsdk = "custom_data_transmission_to_esdk"` |
| `common/djisdk/client.go` | 893 | `return c.SendCommand(ctx, gatewaySn, MethodCustomDataTransmissionToEsdk, data)` |

#### `FlightTaskUndo` / `FlightTaskRecovery`

**No Go identifiers named `FlightTaskUndo` or `FlightTaskRecovery` exist anywhere in the codebase.**

The string literals `flighttask_undo` and `flighttask_recovery` appear in comments/constants:

- `common/djisdk/method.go:100` — `MethodFlightTaskCancel = "flighttask_undo"`
- `common/djisdk/method.go:108` — `MethodFlightTaskResume = "flighttask_recovery"`

Both only as string values assigned to semantically-named constants.

---

### 4. Proto RPC Declarations

All referenced RPCs exist in `app/djicloud/djicloud.proto`:

| RPC | Proto Line | Request Message | Proto Line | Response |
|---|---|---|---|---|
| `rpc CancelFlightTask` | 100 | `CancelFlightTaskReq` | 760 | `CommonRes` |
| `rpc ResumeFlightTask` | 110 | `ResumeFlightTaskReq` | 780 | `CommonRes` |
| `rpc SendCustomDataToPsdk` | 420 | `CustomDataToPsdkReq` | 1236 | `CommonRes` |
| `rpc SendCustomDataToEsdk` | 427 | `CustomDataToEsdkReq` | 1247 | `CommonRes` |
| `rpc SendDrcStickControl` | 480 | `DrcStickControlReq` | 1318 | `DrcStickControlRes` |
| `rpc SetProperty` | 25 | `SetPropertyReq` | 602 | `CommonRes` |

All proto messages listed in the query exist:

| Proto Message | Proto Line | Go Generated Type |
|---|---|---|
| `message CancelFlightTaskReq` | 760 | `djicloud.pb.go:1078` |
| `message ResumeFlightTaskReq` | 780 | `djicloud.pb.go:1199` |
| `message CustomDataToPsdkReq` | 1236 | `djicloud.pb.go:4353` |
| `message CustomDataToEsdkReq` | 1247 | `djicloud.pb.go:4409` |
| `message DrcStickControlReq` | 1318 | `djicloud.pb.go:4861` |
| `message SetPropertyReq` | 602 | `djicloud.pb.go:193` |

---

### 5. Generated Proto Go Code

The generated Go code exists at:
- `app/djicloud/djicloud/djicloud.pb.go` — message types, marshaling
- `app/djicloud/djicloud/djicloud_grpc.pb.go` — gRPC client/server stubs, handlers

---

### 6. Test File References (`*_test.go`)

#### `common/djisdk/protocol_drc_test.go`
- `MethodStickControl` appears on lines 287, 307, 355, 383, 401, 417, 418, 490
- `SendDrcStickControl` on line 483
- `PropertySetTopic`/`PropertySetReplyTopicPattern` on line 838
- `PropertySetReplyTopicPattern` on lines 864, 865

#### `app/djicloud/internal/hooks/register_test.go`
- `djisdk.MethodStickControl` on lines 105, 1029, 1037

No test files reference the other identifiers (CancelFlightTask, ResumeFlightTask, etc. have no dedicated test coverage).

---

### 7. Architecture Summary

The naming alignment follows this pattern:

| DJI Cloud API Method | djisdk Constant | djisdk Client Method | Proto RPC | gRPC Server | Logic |
|---|---|---|---|---|---|
| `flighttask_undo` | `MethodFlightTaskCancel` | `Client.CancelFlightTask` | `CancelFlightTask` | `DjiCloudServer.CancelFlightTask` | `CancelFlightTaskLogic` |
| `flighttask_recovery` | `MethodFlightTaskResume` | `Client.ResumeFlightTask` | `ResumeFlightTask` | `DjiCloudServer.ResumeFlightTask` | `ResumeFlightTaskLogic` |
| `custom_data_transmission_to_psdk` | `MethodCustomDataTransmissionToPsdk` | `Client.SendCustomDataToPsdk` | `SendCustomDataToPsdk` | `DjiCloudServer.SendCustomDataToPsdk` | `SendCustomDataToPsdkLogic` |
| `custom_data_transmission_to_esdk` | `MethodCustomDataTransmissionToEsdk` | `Client.SendCustomDataToEsdk` | `SendCustomDataToEsdk` | `DjiCloudServer.SendCustomDataToEsdk` | `SendCustomDataToEsdkLogic` |
| `stick_control` | `MethodStickControl` | `Client.SendDrcStickControl` | `SendDrcStickControl` | `DjiCloudServer.SendDrcStickControl` | `SendDrcStickControlLogic` |
| `property_set` | `MethodPropertySet` | `Client.SetProperty` | `SetProperty` | `DjiCloudServer.SetProperty` | `SetPropertyLogic` |

The gRPC server methods (`CancelFlightTask`, `ResumeFlightTask`, etc.) delegate to `l.svcCtx.DjiClient.<ClientMethod>(...)` — the `DjiClient` is the `common/djisdk.Client` instance.

## Caveats / Not Found

- **`FlightTaskUndo`** and **`FlightTaskRecovery`**: No Go identifiers with these names exist. The DJI Cloud API methods `flighttask_undo` and `flighttask_recovery` are mapped to `MethodFlightTaskCancel` and `MethodFlightTaskResume` respectively.
- **`MethodPsdkFloatUp`**: Already exists (constant in `common/djisdk/method.go:439`), but it maps to `psdk_ui_resource_upload`, not to `custom_data_transmission_to_psdk`. This is a separate concept.
- No test coverage found for `CancelFlightTask`, `ResumeFlightTask`, `SendCustomDataToPsdk`, `SendCustomDataToEsdk`, or `SetProperty` in test files.
