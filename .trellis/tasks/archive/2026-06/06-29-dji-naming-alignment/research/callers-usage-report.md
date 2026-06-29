# Research: DJI Naming Alignment — Callers & Usages

- **Query**: Search codebase (outside `common/djisdk/` and generated proto code) for all callers/usages of specific Go identifiers and string patterns related to DJI SDK method renaming.
- **Scope**: mixed (internal codebase search)
- **Date**: 2026-06-29

## Findings

### 1. Go Function Identifiers — Non-generated caller files (outside `common/djisdk/`)

#### 1.1 `CancelFlightTask`

| File | Line | Context |
|---|---|---|
| `app/djicloud/internal/logic/cancelflighttasklogic.go` | 28 | `tid, err := l.svcCtx.DjiClient.CancelFlightTask(l.ctx, in.DeviceSn, in.FlightIds)` |
| `app/djicloud/internal/server/djicloudserver.go` | 99–101 | `func (s *DjiCloudServer) CancelFlightTask(...)` — gRPC server handler, calls `l.CancelFlightTask(in)` |

**Total non-generated callers: 2** (logic + server handler)

#### 1.2 `ResumeFlightTask`

| File | Line | Context |
|---|---|---|
| `app/djicloud/internal/logic/resumeflighttasklogic.go` | 29 | `tid, err := l.svcCtx.DjiClient.ResumeFlightTask(l.ctx, in.GetDeviceSn(), &djisdk.FlightTaskResumeData{...})` |
| `app/djicloud/internal/server/djicloudserver.go` | 111–113 | `func (s *DjiCloudServer) ResumeFlightTask(...)` — gRPC server handler, calls `l.ResumeFlightTask(in)` |

**Total non-generated callers: 2** (logic + server handler)

#### 1.3 `PsdkUIResourceUpload`

| File | Line | Context |
|---|---|---|
| `app/djicloud/internal/logic/psdkuiresourceuploadlogic.go` | 34 | `tid, err := l.svcCtx.DjiClient.PsdkUIResourceUpload(l.ctx, in.GetDeviceSn(), data)` |
| `app/djicloud/internal/server/djicloudserver.go` | 417–419 | `func (s *DjiCloudServer) PsdkUIResourceUpload(...)` — gRPC server handler, calls `l.PsdkUIResourceUpload(in)` |

**Total non-generated callers: 2** (logic + server handler)

#### 1.4 `SendCustomDataToPsdk`

| File | Line | Context |
|---|---|---|
| `app/djicloud/internal/logic/sendcustomdatatopsdklogic.go` | 36 | `tid, err := l.svcCtx.DjiClient.SendCustomDataToPsdk(l.ctx, in.DeviceSn, in.Value)` |
| `app/djicloud/internal/server/djicloudserver.go` | 423–425 | `func (s *DjiCloudServer) SendCustomDataToPsdk(...)` — gRPC server handler, calls `l.SendCustomDataToPsdk(in)` |

**Total non-generated callers: 2** (logic + server handler)

#### 1.5 `SendCustomDataToEsdk`

| File | Line | Context |
|---|---|---|
| `app/djicloud/internal/logic/sendcustomdatatoesdklogic.go` | 28 | `tid, err := l.svcCtx.DjiClient.SendCustomDataToEsdk(l.ctx, in.GetDeviceSn(), in.GetValue())` |
| `app/djicloud/internal/server/djicloudserver.go` | 429–431 | `func (s *DjiCloudServer) SendCustomDataToEsdk(...)` — gRPC server handler, calls `l.SendCustomDataToEsdk(in)` |

**Total non-generated callers: 2** (logic + server handler)

#### 1.6 `SendDrcStickControl`

| File | Line | Context |
|---|---|---|
| `app/djicloud/internal/logic/senddrcstickcontrollogic.go` | 34 | `if _, err := l.svcCtx.DjiClient.SendDrcStickControl(l.ctx, deviceSn, int(seq), data); err != nil {` |
| `app/djicloud/internal/server/djicloudserver.go` | 471–473 | `func (s *DjiCloudServer) SendDrcStickControl(...)` — gRPC server handler, calls `l.SendDrcStickControl(in)` |
| `app/djicloud/internal/hooks/doc.go` | 16 | Comment reference: "gRPC **SendDrcStickControl** 经 djisdk 发 drc/down" |
| `common/djisdk/protocol_drc_test.go` | 483 | `tid, err := client.SendDrcStickControl(context.Background(), "gateway-1", 7, &DrcStickControlData{})` (test file) |

**Total non-generated callers: 4** (logic + server handler + doc comment + test)

#### 1.7 `SetProperty` (function call, NOT type references)

| File | Line | Context |
|---|---|---|
| `app/djicloud/internal/logic/setpropertylogic.go` | 35 | `tid, err := l.svcCtx.DjiClient.SetProperty(l.ctx, in.DeviceSn, properties)` |
| `app/djicloud/internal/server/djicloudserver.go` | 33–35 | `func (s *DjiCloudServer) SetProperty(...)` — gRPC server handler, calls `l.SetProperty(in)` |
| `app/djicloud/internal/hooks/doc.go` | 14 | Comment reference: "由 djisdk SetProperty 经 pending 收 set_reply" |
| `common/djisdk/topic.go` | 113 | Comment reference: "由 SetProperty 使用 ServiceRequest+MethodPropertySet 发布" |
| `common/djisdk/protocol.go` | 497 | Comment reference: "由云平台/本服务 SetProperty 随 MethodPropertySet 一同下发" |

**Total non-generated callers: 5** (Hits: only the calls in `djicloudserver.go` and `setpropertylogic.go` are actual function calls; doc/topic/protocol comments are references.)

---

### 2. String Literal Patterns

#### 2.1 `"flighttask_undo"`

| File | Line | Context |
|---|---|---|
| `common/djisdk/method.go` | 100 | `MethodFlightTaskCancel = "flighttask_undo"` — constant definition |

**No external string literal usage found outside `common/djisdk/method.go`**.

#### 2.2 `"flighttask_recovery"`

| File | Line | Context |
|---|---|---|
| `common/djisdk/method.go` | 108 | `MethodFlightTaskResume = "flighttask_recovery"` — constant definition |

**No external string literal usage found outside `common/djisdk/method.go`**.

#### 2.3 `"psdk_ui_resource_upload"`

| File | Line | Context |
|---|---|---|
| `common/djisdk/method.go` | 439 | `MethodPsdkFloatUp = "psdk_ui_resource_upload"` — constant definition |

**No external string literal usage found outside `common/djisdk/method.go`**.

#### 2.4 `"custom_data_transmission_to_psdk"`

| File | Line | Context |
|---|---|---|
| `common/djisdk/method.go` | 443 | `MethodCustomDataTransmissionToPsdk = "custom_data_transmission_to_psdk"` — constant definition |

**No external string literal usage found outside `common/djisdk/method.go`**.

#### 2.5 `"custom_data_transmission_to_esdk"`

| File | Line | Context |
|---|---|---|
| `common/djisdk/method.go` | 456 | `MethodCustomDataTransmissionToEsdk = "custom_data_transmission_to_esdk"` — constant definition |

**No external string literal usage found outside `common/djisdk/method.go`**.

#### 2.6 `"stick_control"`

| File | Line | Context |
|---|---|---|
| `common/djisdk/method.go` | 314 | `MethodStickControl = "stick_control"` — constant definition |
| `app/djicloud/internal/hooks/register_test.go` | 98 | Test payload string: `"method":"stick_control"` |
| `common/djisdk/protocol_drc_test.go` | 307, 355 | Test references via `MethodStickControl` (not literal) |
| `common/djisdk/protocol_drc_test.go` | 388 | Test payload string contains `"method":"stick_control"` |

**Note**: The test files use the literal `"stick_control"` in JSON payloads, not the Go constant.

#### 2.7 `"property_set"`

| File | Line | Context |
|---|---|---|
| `common/djisdk/method.go` | 9 | `MethodPropertySet = "property_set"` — constant definition |

**No external string literal usage found outside `common/djisdk/method.go`**.

---

### 3. Pattern Matches in `app/djicloud/` Directory

All callers found within `app/djicloud/` are detailed in Section 1 above. Summary:

| Identifier | `app/djicloud/internal/logic/` | `app/djicloud/internal/server/djicloudserver.go` | `app/djicloud/internal/hooks/` |
|---|---|---|---|
| `CancelFlightTask` | `cancelflighttasklogic.go:28` | `djicloudserver.go:99-101` | — |
| `ResumeFlightTask` | `resumeflighttasklogic.go:29` | `djicloudserver.go:111-113` | — |
| `PsdkUIResourceUpload` | `psdkuiresourceuploadlogic.go:34` | `djicloudserver.go:417-419` | — |
| `SendCustomDataToPsdk` | `sendcustomdatatopsdklogic.go:36` | `djicloudserver.go:423-425` | — |
| `SendCustomDataToEsdk` | `sendcustomdatatoesdklogic.go:28` | `djicloudserver.go:429-431` | — |
| `SendDrcStickControl` | `senddrcstickcontrollogic.go:34` | `djicloudserver.go:471-473` | `doc.go:16` (comment) |
| `SetProperty` | `setpropertylogic.go:35` | `djicloudserver.go:33-35` | `doc.go:14` (comment) |
| `MethodStickControl` | — | — | `register_test.go:98,1029,1037` |

### 4. Generated Proto Type Names (for reference)

#### 4.1 `CancelFlightTask` pb types (from `djicloud.proto` and generated `djicloud.pb.go`)

- `CancelFlightTaskReq` (message type) — in `djicloud.pb.go:1078`
- `DjiCloud_CancelFlightTask_FullMethodName` — in `djicloud_grpc.pb.go:37`
- gRPC method: `/djicloud.DjiCloud/CancelFlightTask`
- Client interface: `CancelFlightTask(ctx context.Context, in *CancelFlightTaskReq, opts ...grpc.CallOption) (*CommonRes, error)` — in `djicloud_grpc.pb.go:185`
- Server interface: `CancelFlightTask(context.Context, *CancelFlightTaskReq) (*CommonRes, error)` — in `djicloud_grpc.pb.go:1566`

#### 4.2 `ResumeFlightTask` pb types (from `djicloud.proto` and generated `djicloud.pb.go`)

- `ResumeFlightTaskReq` (message type) — in `djicloud.pb.go:1199`
- `DjiCloud_ResumeFlightTask_FullMethodName` — in `djicloud_grpc.pb.go:39`
- gRPC method: `/djicloud.DjiCloud/ResumeFlightTask`
- Client interface: `ResumeFlightTask(ctx context.Context, in *ResumeFlightTaskReq, opts ...grpc.CallOption) (*CommonRes, error)` — in `djicloud_grpc.pb.go:193`
- Server interface: `ResumeFlightTask(context.Context, *ResumeFlightTaskReq) (*CommonRes, error)` — in `djicloud_grpc.pb.go:1574`

### 5. gRPC Method Constant Pattern (`CustomDataTransmissionTo`)

| File | Line | Context |
|---|---|---|
| `common/djisdk/method.go` | 443 | `MethodCustomDataTransmissionToPsdk = "custom_data_transmission_to_psdk"` |
| `common/djisdk/method.go` | 456 | `MethodCustomDataTransmissionToEsdk = "custom_data_transmission_to_esdk"` |

No `CustomDataTransmissionTo` references exist outside of `common/djisdk/method.go` (other than usage via the constant names in `client.go` calls, which are already covered above).

---

### 6. Method Constants Used in `common/djisdk/client.go` (Mappings)

For completeness, here is how each `common/djisdk/client.go` function maps to its method constant:

| Client function | File:Line | Method constant used | String value |
|---|---|---|---|
| `CancelFlightTask` | `client.go:289` | `MethodFlightTaskCancel` | `"flighttask_undo"` |
| `ResumeFlightTask` | `client.go:309` | `MethodFlightTaskResume` | `"flighttask_recovery"` |
| `PsdkUIResourceUpload` | `client.go:874` | `MethodPsdkFloatUp` | `"psdk_ui_resource_upload"` |
| `SendCustomDataToPsdk` | `client.go:885` | `MethodCustomDataTransmissionToPsdk` | `"custom_data_transmission_to_psdk"` |
| `SendCustomDataToEsdk` | `client.go:893` | `MethodCustomDataTransmissionToEsdk` | `"custom_data_transmission_to_esdk"` |
| `SendDrcStickControl` | `client.go:669` | `MethodStickControl` | `"stick_control"` |
| `SetProperty` | `client.go:145,152` | `MethodPropertySet` | `"property_set"` |

---

### 7. Caveats / Not Found

1. **No callers** of any of these identifiers were found outside `app/djicloud/` and `common/djisdk/`. No external services, other apps, or scripts reference them.
2. All string literals (`"flighttask_undo"`, `"flighttask_recovery"`, etc.) are defined only in `common/djisdk/method.go` as Go constants and referenced via those constants elsewhere — no raw string duplication.
3. The `"stick_control"` literal appears raw in test JSON payloads in `common/djisdk/protocol_drc_test.go:388` and `app/djicloud/internal/hooks/register_test.go:98`.
4. The `.proto` file (`app/djicloud/djicloud.proto`) contains the gRPC service definitions for all 7 RPCs and their request/response message types — this is the source of truth for the generated `djicloud.pb.go` and `djicloud_grpc.pb.go`.
