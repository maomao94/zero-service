# SendSetpointFloat ACK replyPool 试验

## Goal

为 `SendSetpointFloat` 做一个最小 ACK replyPool 试验版，让业务系统下发 IEC104 浮点设点命令后，能够同步知道从站是否返回协议级 ACK：接受、拒绝、超时或本地发送失败。

本任务只解决“设备是否接受命令”，不解决“设备实际值是否已到位”。实际到位仍由后续 M_* 监视方向反馈点判断。

## Requirements

- 仅覆盖 `SendSetpointFloat`：`C_SE_NC_1` / `C_SE_TC_1`。
- 发送前为目标命令注册 pending ACK，匹配键以单个控制目标为粒度：`host + port + coa + typeId + ioa`。
- 同一个匹配键同一时间只能有一个未完成设点命令；重复下发必须返回明确失败，不做静默排队。
- `onCommandAck` 收到 `C_SE_NC_1` / `C_SE_TC_1` 时解析 ACK 信息体：`ioa`、`value`、`qos`、可选 `time`。
- ACK 匹配成功后根据 COT 和 `IsNegative` 产生结果：
  - `ActivationCon + IsNegative=false`：设备接受命令。
  - `ActivationCon + IsNegative=true`：设备拒绝命令。
  - `UnknownTypeID/UnknownCOT/UnknownCA/UnknownIOA + IsNegative=true`：设备拒绝命令。
  - 超时：未收到匹配 ACK。
- 默认等待 `ActivationCon`，不默认等待 `ActivationTerm`。
- 保持监视方向 M_* 数据推送逻辑不变；ACK 不进入 Kafka/MQTT/stream event 遥测流。
- 优先复用 `common/antsx.ReplyPool`，不要手写无超时清理的 map。
- 需要保留现有集群广播发送路径；如果当前实例只是广播给其他实例且本地没有 client，本试验版不等待 ACK。

## Acceptance Criteria

- [ ] `SendSetpointFloat` 本地直连发送时，会注册 pending ACK 并等待从站 ACK 或超时。
- [ ] 收到匹配的 `C_SE_NC_1` / `C_SE_TC_1` ACK 且 `ActivationCon + IsNegative=false` 时，RPC 返回成功。
- [ ] 收到匹配 ACK 且 `IsNegative=true` 时，RPC 返回明确错误。
- [ ] 未收到匹配 ACK 时，RPC 返回超时错误。
- [ ] 同一 `host/port/coa/typeId/ioa` 有未完成命令时，新请求返回重复/忙碌错误。
- [ ] 浮点 ACK value 与 expected value 校验使用 float32 语义或小容差，避免 double→float32 转换误判。
- [ ] `go test ./app/ieccaller/internal/iec ./app/ieccaller/internal/logic` 通过。
- [ ] 现有 `go test ./app/ieccaller/internal/iec` 中 `asduLogContext` 测试保持通过。

## Notes

- Confirmed facts:
  - `SendCommandRes` 当前为空消息。
  - `SendSetpointFloatLogic` 当前只调用 `cli.SendSetpointFloatCmd(...)`，发送成功后立即返回空响应。
  - `ClientCall.onCommandAck` 当前只记录日志，不关联调用方。
  - `common/antsx.ReplyPool` 已提供 `Register` / `Resolve` / `Reject` / TTL 超时 / `ErrDuplicateID` / `ErrReplyExpired`。
  - `C_SE_NC_1` / `C_SE_TC_1` ACK 可通过 `packet.GetSetpointFloatCmd()` 解析 `ioa/value/qos/time`。
- Decision: ACK 成功保持现有空 `SendCommandRes`；ACK 失败通过 `tool.NewErrorByPbCode` 表达，reason 即为 `extproto.Code` 数值，调用方可用 `tool.IsErrorByPbCode` 区分。
- Error mapping:
  - ACK accepted: `return &ieccaller.SendCommandRes{}, nil`
  - ACK rejected/cot-error: `extproto.Code__1_06_THIRD_PARTY`
  - ACK timeout: `extproto.Code__1_00_TIMEOUT`
  - duplicate command key: `extproto.Code__1_05_BIZ_REPEAT`
  - value mismatch: `extproto.Code__1_06_THIRD_PARTY`
- Proto `SendCommandRes` 不变更，不影响其他控制命令接口。
