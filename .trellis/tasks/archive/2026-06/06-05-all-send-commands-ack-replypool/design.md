# 全部 Send 指令 ACK replyPool 设计

## 范围

扩展 `onCommandAck` 使其对全部 7 种控制命令（SetSingle/SetDouble/SetStep/SetpointNormalized/SetpointScaled/SetpointFloat/SetBitstring）都做 ACK pool 匹配。

不涉及：SendCommand（通用接口）、集群广播、Proto 再改、M_* 监视数据处理。

## ACK 解析函数对照

| 命令类型 | TypeID | go-iecp5 解析函数 | 解析出 Info struct | 关键字段 |
|---------|--------|------------------|-------------------|---------|
| SingleCommand | C_SC_NA_1(45) / C_SC_TA_1(58) | `GetSingleCmd()` | `SingleCommandInfo` | `Ioa`, `Value(bool)`, `Qoc` |
| DoubleCommand | C_DC_NA_1(46) / C_DC_TA_1(59) | `GetDoubleCmd()` | `DoubleCommandInfo` | `Ioa`, `Value(DoubleCommand)`, `Qoc` |
| StepCommand | C_RC_NA_1(47) / C_RC_TA_1(60) | `GetStepCmd()` | `StepCommandInfo` | `Ioa`, `Value(StepCommand)`, `Qoc` |
| SetpointNormalized | C_SE_NA_1(48) / C_SE_TA_1(61) | `GetSetpointNormalCmd()` | `SetpointCommandNormalInfo` | `Ioa`, `Value(int16)`, `Qos` |
| SetpointScaled | C_SE_NB_1(49) / C_SE_TB_1(62) | `GetSetpointCmdScaled()` | `SetpointCommandScaledInfo` | `Ioa`, `Value(int16)`, `Qos` |
| SetpointFloat | C_SE_NC_1(50) / C_SE_TC_1(63) | `GetSetpointFloatCmd()` | `SetpointCommandFloatInfo` | `Ioa`, `Value(float32)`, `Qos` |
| BitstringCommand | C_BO_NA_1(51) / C_BO_TA_1(64) | `GetBitsString32Cmd()` | `BitsString32CommandInfo` | `Ioa`, `Value(uint32)` |

## onCommandAck 统一模式

```go
func (c *ClientCall) onCommandAck(ctx context.Context, packet *asdu.ASDU) {
    logx.WithContext(ctx).Info("Command ACK received")

    // 查找 pool
    cli, err := c.svcCtx.ClientManager.GetClient(c.config.Host, c.config.Port)
    if err != nil { return }
    pool := cli.CommandReplyPool()

    // 按类型解析 IOA 和 Value
    ioa, value, keyOk := parseCommandAckIOA(packet)
    if !keyOk { return }

    key := client.CommandKey(uint(packet.CommonAddr), int(packet.Type), ioa)
    if !pool.Has(key) { return }

    ack := &client.CommandAck{...}

    if packet.Coa.IsNegative {
        ack.Status = client.AckRejected
        pool.Reject(key, ...)
        return
    }

    switch packet.Coa.Cause {
    case asdu.ActivationCon:
        ack.Status = client.AckAccepted
        pool.Resolve(key, ack)
    }
}
```

`parseCommandAckIOA` 和 `parseCommandAckValue` 抽取为两个辅助函数，按 `packet.Type` 分支。

## 各 Logic 改造模式

与 SetpointFloat 一致：

```go
key := client.CommandKey(uint(in.Coa), typeID, uint(in.Ioa))
pool := cli.CommandReplyPool()
promise, _ := pool.Register(key)
// send command
ack, _ := promise.Await(l.ctx)
return &ieccaller.SendCommandRes{AckValue: cast.ToString(ack.Value)}, nil
```

每个 Logic 只需确认 typeID 和 expectedValue 的转换。

## 不涉及

- Proto `SendCommandRes` 已有 `ack_value` 字段，不额外改动。
- `SendCommandReq`（通用 typeId+value 接口）不接入 replyPool。
- 集群广播路径不变。
- `CommandReplyPool` 和 `CommandAck` 类型不变。
