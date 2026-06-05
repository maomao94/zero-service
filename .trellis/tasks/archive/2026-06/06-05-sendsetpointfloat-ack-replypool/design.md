# SendSetpointFloat ACK replyPool 设计

## 1. 架构边界

### 涉及组件

| 层 | 文件 | 变更 |
|---|------|------|
| ServiceContext | `app/ieccaller/internal/svc/servicecontext.go` | 新增 `SetpointFloatReplyPool *antsx.ReplyPool[SetpointFloatAck]` |
| ClientCall | `app/ieccaller/internal/iec/clienthandler.go` | `onCommandAck` 增加 ACK 匹配逻辑；增加 `SetpointFloatReplyPool` 引用 |
| Logic | `app/ieccaller/internal/logic/sendsetpointfloatlogic.go` | 发送前注册 pending，发送后等待 Promise；ACK 成功返回 nil error，失败返回 pbCode error |
| Proto | 不变更 | `SendCommandRes` 保持空 |
| Client core | `common/iec104/client/core.go` | 不变更 |
| Kafka broadcast | `app/ieccaller/kafka/broadcast.go` | 不变更（广播路径不等待 ACK） |

### 不涉及

- 其他控制命令（SingleCommand / DoubleCommand / StepCommand / Bitstring / SetpointNormalized / SetpointScaled）
- M_* 监视方向数据推送
- 读命令（C_RD_NA_1）
- 总召唤 / 时钟同步 / 测试 / 复位

## 2. 数据流

```
SendSetpointFloat RPC
  |
  v
SendSetpointFloatLogic
  |
  |-- 组装 pendingKey = host + ":" + port + ":" + coa + ":" + typeId + ":" + ioa
  |-- replyPool.Register(pendingKey, ttl)
  |     |-- ErrDuplicateID -> 返回 Code__1_05_BIZ_REPEAT
  |
  |-- cli.SendSetpointFloatCmd(...)
  |     |-- 发送失败 -> replyPool.Reject + 返回 Code__1_06_THIRD_PARTY
  |
  |-- promise.Await(ctx)
        |-- resolved: ACK accepted -> 返回 SendCommandRes{}, nil
        |-- rejected: ACK rejected -> 返回 error（含 reason 详情）
        |-- ctx timeout -> 返回 Code__1_00_TIMEOUT
        |-- replyPool expiry -> 返回 Code__1_00_TIMEOUT
```

ACK 到达方向：

```
go-iecp5 cs104 client
  |
  v
ASDUHandler -> ClientCall.OnASDU
  |
  v
case SetSetpointFloat -> onCommandAck
  |
  |-- packet.GetSetpointFloatCmd() -> ioa, value, qos, time
  |-- 组装 pendingKey
  |-- replyPool.Resolve(pendingKey, ackResult) 或 replyPool.Reject(pendingKey, err)
```

## 3. 核心类型

```go
// SetpointFloatAck ACK 匹配结果
type SetpointFloatAck struct {
    Accepted   bool
    Status     string // "accepted" / "rejected" / "cot_error"
    TypeID     int
    Coa        uint
    Ioa        uint
    Value      float32
    Qos        asdu.QualifierOfSetpointCmd
    Cot        string
    CotCause   int
    IsNegative bool
}
```

## 4. pendingKey 约定

```go
func pendingKey(host string, port int, coa uint, typeId int, ioa uint) string {
    return fmt.Sprintf("%s:%d:%d:%d:%d", host, port, coa, typeId, ioa)
}
```

同一 key 同时只允许一个命令。`ReplyPool.Register` 返回 `ErrDuplicateID` 时直接拒绝。

## 5. ACK 匹配逻辑

`onCommandAck` 增加分支：

```go
case client.SetSetpointFloat:
    cmd := packet.GetSetpointFloatCmd()
    key := pendingKey(c.config.Host, c.config.Port, uint(packet.CommonAddr), int(packet.Type), uint(cmd.Ioa))

    ack := &SetpointFloatAck{
        TypeID:     int(packet.Type),
        Coa:        uint(packet.CommonAddr),
        Ioa:        uint(cmd.Ioa),
        Value:      cmd.Value,
        Qos:        cmd.Qos,
        Cot:        genCOTName(packet.Coa.Cause),
        CotCause:   int(packet.Coa.Cause),
        IsNegative: packet.Coa.IsNegative,
    }

    if packet.Coa.IsNegative {
        ack.Accepted = false
        ack.Status = "rejected"
        c.setpointFloatReplyPool.Reject(key, fmt.Errorf("ACK rejected: cot=%s isNegative=true", ack.Cot))
        return
    }

    switch packet.Coa.Cause {
    case asdu.ActivationCon:
        ack.Accepted = true
        ack.Status = "accepted"
        c.setpointFloatReplyPool.Resolve(key, ack)
    // ActivationTerm 或其他 COT 仍记录日志但不匹配
    }
```

不默认匹配 `ActivationTerm`。`ActivationTerm` 到达时仅日志。

## 6. float value 校验

gRPC 使用 `double`，IEC104 设点是 `float32`。校验使用容差：

```go
const floatMatchEpsilon = 1e-5

func floatValuesMatch(expected, actual float32) bool {
    diff := expected - actual
    if diff < 0 {
        diff = -diff
    }
    return diff < floatMatchEpsilon
}
```

不匹配时 `Reject` 并返回错误；写入日志。

## 7. TTL 和 goroutine 安全

- 默认 TTL：`3s`。IEC104 控制命令 ActivationCon 典型响应在 200ms-3s 内；超过 3s 大概率设备无响应。
- `ReplyPool` 时间轮自动清理过期 entry。
- `promise.Await(ctx)` 的 ctx 来自 Logic 的 `l.ctx`（go-zero 标准 ctx）。
- `Await` 超时后不主动 `Reject`；时间轮稍后自动清理。
- `ReplyPool` 生命周期绑定 `ServiceContext`，不绑定单次 RPC。
- `Close()` 在 `ServiceContext` 退出时调用。

## 8. 兼容性

- 广播路径（集群模式）不等待 ACK，行为不变。
- 其他控制命令继续走 `onCommandAck(ctx)`，不受影响。
- `SendCommandRes` 不变更，proto 不重新生成。
- 现有 `asduLogContext` 测试保持通过。
