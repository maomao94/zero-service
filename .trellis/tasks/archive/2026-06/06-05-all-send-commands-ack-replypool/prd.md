# 全部Send指令ACK replyPool

## Goal

在已有 SetpointFloat replyPool 基础上，将全部 7 种 IEC104 控制命令统一接入 ACK replyPool，使业务系统下发任意控制命令后都能同步知道设备接受/拒绝/超时。

## Requirements

- 覆盖全部控制命令 TypeID：
  - C_SC_NA_1/TA_1 → SendSingleCommand
  - C_DC_NA_1/TA_1 → SendDoubleCommand
  - C_RC_NA_1/TA_1 → SendStepCommand
  - C_SE_NA_1/TA_1 → SendSetpointNormalized
  - C_SE_NB_1/TB_1 → SendSetpointScaled
  - C_SE_NC_1/TC_1 → SendSetpointFloat（已实现）
  - C_BO_NA_1/TA_1 → SendBitstringCommand
- onCommandAck 按 packet.Type 分支解析各自 ACK 信息体，统一走 pool.Resolve/Reject。
- 各 Logic 统一注册 pending、等待 ACK、按错误码返回。
- 公共逻辑（key 构造、pool 查找、Status 判断）抽到 helper 函数，避免 7 个 resolve* 函数重复。
- SendCommandReq（通用 typeId+value 接口）暂不接入，仍保持现有发送即返回。

## Acceptance Criteria

- [ ] 全部 7 种控制命令本地直连时，注册 pending 并等待 ACK 或超时。
- [ ] 收到匹配 ACK + ActivationCon + IsNegative=false 时，RPC 返回成功（AckValue 带回执值）。
- [ ] 收到 ACK + IsNegative=true 时，RPC 返回明确错误。
- [ ] 超时返回 Code__1_00_TIMEOUT。
- [ ] 同一 key 重复下发返回 Code__1_05_BIZ_REPEAT。
- [ ] 不涉及集群广播路径（广播仍返回空），不涉及 proto 再改。
- [ ] `go build ./app/ieccaller/...` 零错误。
- [ ] `go test ./app/ieccaller/internal/iec ./common/iec104/client` 通过。
