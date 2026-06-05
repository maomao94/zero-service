# 全部 Send 指令 ACK replyPool 实现清单

## 1. clienthandler.go - 通用 ACK 解析

- [ ] 新增 `parseCommandAckIOA(packet) (uint, any, bool)` — 按 packet.Type 分支返回 IOA 和 value
- [ ] 新增 `resolveCommandAck(ctx, packet)` — 统一 ACK pool 匹配逻辑，替代 `resolveSetpointFloatAck`
- [ ] `onCommandAck` 调用 `resolveCommandAck(ctx, packet)`，删除 SetpointFloat 专属分支
- [ ] 删除 `resolveSetpointFloatAck` 函数

## 2. 各 Logic 文件改造

每个 Logic 文件遵循统一模式：register → send → await → return。按以下清单逐个改造：

- [ ] `sendsinglecommandlogic.go`
  - typeID: C_SC_NA_1(45) / C_SC_TA_1(58)
  - expectedValue: bool → `cast.ToString(in.Value)`
  - ACK 解析: `GetSingleCmd()` → Ioa, Value(bool)

- [ ] `senddoublecommandlogic.go`
  - typeID: C_DC_NA_1(46) / C_DC_TA_1(59)
  - expectedValue: DoubleCommandValue → `cast.ToString(in.Value)`
  - ACK 解析: `GetDoubleCmd()` → Ioa, Value(DoubleCommand)

- [ ] `sendstepcommandlogic.go`
  - typeID: C_RC_NA_1(47) / C_RC_TA_1(60)
  - expectedValue: int32 → `cast.ToString(in.Value)`
  - ACK 解析: `GetStepCmd()` → Ioa, Value(StepCommand)

- [ ] `sendsetpointnormalizedlogic.go`
  - typeID: C_SE_NA_1(48) / C_SE_TA_1(61)
  - expectedValue: int32 → `cast.ToString(in.Value)`
  - ACK 解析: `GetSetpointNormalCmd()` → Ioa, Value(int16)

- [ ] `sendsetpointscaledlogic.go`
  - typeID: C_SE_NB_1(49) / C_SE_TB_1(62)
  - expectedValue: int32 → `cast.ToString(in.Value)`
  - ACK 解析: `GetSetpointCmdScaled()` → Ioa, Value(int16)

- [ ] `sendsetpointfloatlogic.go`
  - 已实现，确认无回归

- [ ] `sendbitstringcommandlogic.go`
  - typeID: C_BO_NA_1(51) / C_BO_TA_1(64)
  - expectedValue: uint64 → `cast.ToString(in.Value)`
  - ACK 解析: `GetBitsString32Cmd()` → Ioa, Value(uint32)

## 3. sendcommandlogic.go（通用 SendCommand）

- [ ] 保持不变：不接入 replyPool，继续发送即返回

## 4. 验证

- [ ] `go build ./app/ieccaller/...` 零错误
- [ ] `go test ./app/ieccaller/internal/iec ./common/iec104/client` 通过
- [ ] `go vet ./app/ieccaller/...` 零 warning
