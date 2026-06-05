# ieccaller 控制命令专项接口

## Goal

提供 7 个带类型的 gRPC 控制命令接口，取代需要手动查阅 typeId 的通用 `SendCommand`。调用方只需传 host/port/coa/ioa/typed_value，无需理解 IEC 104 typeId 编号、CP56Time2a 时标、QOC/QOS 限定词。

## Background

现有 `SendCommand` 是通用接口，要求调用方自行指定 `typeId`（45-51）并将 `value` 传为 string。对接时容易出错：
- 需要查表才知道 typeId=50 是浮点设点、46 是双点命令
- 不知道 value 的合法类型和范围
- 不了解 CP56Time2a 时标变体与 _NA_1 的区别

本任务为 7 种控制方向命令各提供一个专用 RPC，默认行为覆盖 95%+ 工程场景（_NA_1 不带时标 + 直接执行）。

## Requirements

- **R1**: 7 个新 RPC，每种控制命令一个，消息中 value 使用原生类型而非 string
- **R2**: 所有接口复用现有 `SendCommandRes` 作为响应
- **R3**: 默认使用 _NA_1（不带时标）+ 直接执行（InSelect=false）
- **R4**: 保留 `SendCommand` 通用接口用于高级场景（_TA_1 带时标、选控等）
- **R5**: 兼容现有 cluster 模式 Kafka 广播机制
- **R6**: 需输出一份 gRPC 对接指导文档

### 新增接口清单

| RPC | 对应 typeId | value 类型 | 说明 |
|-----|-----------|-----------|------|
| `SendSingleCommand` | 45 (C_SC_NA_1) | `bool` | 单点命令，true=合/false=分 |
| `SendDoubleCommand` | 46 (C_DC_NA_1) | `DoubleCommandValue` enum | 双点命令 |
| `SendStepCommand` | 47 (C_RC_NA_1) | `StepCommandValue` enum | 档位调节 |
| `SendSetpointNormalized` | 48 (C_SE_NA_1) | `int32` | 归一化设点 [-32768,32767] |
| `SendSetpointScaled` | 49 (C_SE_NB_1) | `int32` | 标度化设点 [-32768,32767] |
| `SendSetpointFloat` | 50 (C_SE_NC_1) | `double` | 浮点设点 |
| `SendBitstringCommand` | 51 (C_BO_NA_1) | `uint64` | 32位位串 |

## Acceptance Criteria

- [x] 7 个 RPC 可通过 gRPC 调用并正确编码为 IEC 104 APDU 发出
- [x] proto enum 值映射与 `go-iecp5` 库 `DoubleCommand`/`StepCommand` 常量一致
- [x] cluster 模式下无本地连接时能正确通过 Kafka 广播
- [x] `gen.sh` 编译通过零错误
- [x] lsp_diagnostics 通过所有变更文件
- [x] 输出对接指导文档，包含每种命令的 JSON 示例

## Out of Scope

- ~~带时标变体（_TA_1）：继续走 `SendCommand` 传显式 typeId~~ → 已通过 `withTime` 字段纳入 typed 接口
- 选控（select-before-operate）：继续走 `SendCommand` 传显式 typeId
- 监视方向（M_* 类型）上行数据处理：已有 `OnASDU` 回调，无需新增

## Notes

- `DoubleCommandValue` enum: DCO_NOT_ALLOWED=0, DCO_ON=1(合), DCO_OFF=2(分) — 对齐 go-iecp5 `DCONotAllow0/DCOOn/DCOOff`
- `StepCommandValue` enum: SCO_NOT_ALLOWED=0, SCO_DOWN=1(降), SCO_UP=2(升) — 对齐 go-iecp5 `SCONotAllow0/SCOStepDown/SCOStepUP`
