# DJI API 命名对齐：proto ↔ SDK const ↔ SDK method

## Goal

将 DJI Cloud API 的 method 值、SDK 常量名、SDK 方法名、proto RPC 名四层对齐，做到给定任意一层名称可沿链直接定位到其他层。

对齐规则：**以 DJI method 值为锚点**，四层命名字面一致（CamelCase ↔ snake_case 映射）。

## 变更映射

| DJI method 值 | SDK 常量 | SDK Client 方法 | Proto RPC | Proto Message |
|---|---|---|---|---|
| `flighttask_undo` | `MethodFlightTaskUndo` | `FlightTaskUndo` | `FlightTaskUndo` | `FlightTaskUndoReq` |
| `flighttask_recovery` | `MethodFlightTaskRecovery` | `FlightTaskRecovery` | `FlightTaskRecovery` | `FlightTaskRecoveryReq` |
| `psdk_ui_resource_upload` | `MethodPsdkUIResourceUpload` | 不变 | 不变 | 不变 |
| `custom_data_transmission_to_psdk` | 不变 | `CustomDataTransmissionToPsdk` | `CustomDataTransmissionToPsdk` | `CustomDataTransmissionToPsdkReq` |
| `custom_data_transmission_to_esdk` | 不变 | `CustomDataTransmissionToEsdk` | `CustomDataTransmissionToEsdk` | `CustomDataTransmissionToEsdkReq` |
| `property_set` | 不变 | `PropertySet` | `PropertySet` | `PropertySetReq` |
| `stick_control` | 不变（或 `MethodDrcStickControl`） | `StickControl` | `StickControl` | `StickControlReq` |

## 约束

- proto 文件变更后需要 `protoc` 重新生成 Go 代码（`make proto` 或等价命令）
- 所有 proto message name 随 RPC 名同步重命名
- 逻辑层文件按 Go 惯例：文件名 = snake_case 化后的逻辑名（如 `flighttaskundologic.go`）

## Acceptance Criteria

- [ ] `go build ./...` 通过，0 个编译错误
- [ ] `go vet ./...` 通过
- [ ] SDK 常量名与其 string value 一对一无歧义
- [ ] proto RPC 名与 SDK Client 方法名一致
- [ ] SDK Client 方法名与它调用的 method 常量名一致（或通过 `SendCommand` 的中介映射透明）
