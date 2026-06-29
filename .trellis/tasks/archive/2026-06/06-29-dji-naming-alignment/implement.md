# 执行计划

## 操作顺序

每步完成后 `go build ./...` 验证可编译。

### Phase 1: common/djisdk/method.go — 重命名常量

| 旧名 | 新名 | 值不变 |
|---|---|---|
| `MethodFlightTaskCancel` | `MethodFlightTaskUndo` | `"flighttask_undo"` |
| `MethodFlightTaskResume` | `MethodFlightTaskRecovery` | `"flighttask_recovery"` |
| `MethodPsdkFloatUp` | `MethodPsdkUIResourceUpload` | `"psdk_ui_resource_upload"` |

验证：`go build ./common/djisdk/...`

### Phase 2: common/djisdk/client.go — 重命名方法

| 旧方法 | 新方法 | 内部常量 |
|---|---|---|
| `CancelFlightTask` | `FlightTaskUndo` | `MethodFlightTaskUndo` |
| `ResumeFlightTask` | `FlightTaskRecovery` | `MethodFlightTaskRecovery` |
| `SendCustomDataToPsdk` | `CustomDataTransmissionToPsdk` | `MethodCustomDataTransmissionToPsdk` |
| `SendCustomDataToEsdk` | `CustomDataTransmissionToEsdk` | `MethodCustomDataTransmissionToEsdk` |
| `SendDrcStickControl` | `StickControl` | `MethodStickControl` |
| `SetProperty` | `PropertySet` | `MethodPropertySet` |

验证：`go build ./common/djisdk/...`

### Phase 3: app/djicloud/djicloud.proto — RPC + message 重命名

```
rpc CancelFlightTask → rpc FlightTaskUndo
  CancelFlightTaskReq → FlightTaskUndoReq

rpc ResumeFlightTask → rpc FlightTaskRecovery
  ResumeFlightTaskReq → FlightTaskRecoveryReq

rpc SendCustomDataToPsdk → rpc CustomDataTransmissionToPsdk
  CustomDataToPsdkReq → CustomDataTransmissionToPsdkReq

rpc SendCustomDataToEsdk → rpc CustomDataTransmissionToEsdk
  CustomDataToEsdkReq → CustomDataTransmissionToEsdkReq

rpc SetProperty → rpc PropertySet
  SetPropertyReq → PropertySetReq

rpc SendDrcStickControl → rpc StickControl
  DrcStickControlReq → StickControlReq
```

验证：执行 `protoc` 或 `make proto` 生成新代码，`go build ./app/djicloud/...`

### Phase 4: app/djicloud/ — 更新 gRPC Server 实现

- `djicloudserver.go` — 方法签名更新
- 所有 logic 文件 — 按新 RPC 名 + 新 client 方法对齐

### Phase 5: common/djisdk/ — 测试文件、protocol 文件

- `protocol_drc_test.go` — `SendDrcStickControl` → `StickControl`
- `register_test.go` — string 常量引用检查

### Phase 6: 最终验证

- `go build ./...`
- `go vet ./...`
- `go test ./...`
