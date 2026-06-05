# SendSetpointFloat ACK replyPool 实现清单

## 1. ServiceContext 集成

**文件：`app/ieccaller/internal/svc/servicecontext.go`**

- [ ] 新增 `SetpointFloatReplyPool *antsx.ReplyPool[*iec.SetpointFloatAck]`
- [ ] 在 `NewServiceContext` 中初始化：`antsx.NewReplyPool[*iec.SetpointFloatAck](antsx.WithName("setpoint-float-ack"), antsx.WithDefaultTTL(5*time.Second))`
- [ ] 在 `ServiceContext` 中添加 `Close()` 方法或确保 pool 被正确关闭
- [ ] import `zero-service/common/antsx`

## 2. ClientCall 集成

**文件：`app/ieccaller/internal/iec/clienthandler.go`**

- [ ] `ClientCall` 结构体新增字段 `setpointFloatReplyPool *antsx.ReplyPool[*iec.SetpointFloatAck]`
- [ ] `NewClientCall` 新增参数接收 pool
- [ ] `onCommandAck` 函数签名改回为 `func (c *ClientCall) onCommandAck(ctx context.Context, packet *asdu.ASDU)`
- [ ] `onCommandAck` 中 `client.SetSetpointFloat` 分支：
  - 先记录 ACK 日志（现有行为不变）
  - 解析 `packet.GetSetpointFloatCmd()`
  - 组装 `pendingKey`
  - 判断 COT + IsNegative，调用 `Resolve` 或 `Reject`
  - 日志记录匹配结果
- [ ] 新增辅助函数 `pendingKey(host, port, coa, typeId, ioa) string`
- [ ] import `fmt`, `zero-service/common/antsx`

## 3. ACK 结果类型

**文件：新建 `app/ieccaller/internal/iec/ack.go`**

- [ ] 定义 `SetpointFloatAck` 结构体
- [ ] `type SetpointFloatAck struct { ... }`

## 4. Logic 改造

**文件：`app/ieccaller/internal/logic/sendsetpointfloatlogic.go`**

- [ ] 在 `cli != nil` 分支：
  - 构造 `pendingKey`
  - 调用 `l.svcCtx.SetpointFloatReplyPool.Register(pendingKey, ttl)`
  - `ErrDuplicateID` → 返回 `tool.NewErrorByPbCode(extproto.Code__1_05_BIZ_REPEAT, ...)`
  - 注册成功 → 执行 `cli.SendSetpointFloatCmd(...)`
  - 发送失败 → `pool.Reject(pendingKey, err)` + 返回错误
  - 发送成功 → `promise.Await(l.ctx)`
  - 根据 `SetpointFloatAck` 结果：
    - `Accepted=true` → `return &ieccaller.SendCommandRes{}, nil`
    - `Accepted=false` → 返回 `Code__1_06_THIRD_PARTY` 带原因
  - `Await` 返回 error：
    - ctx 超时 → `Code__1_00_TIMEOUT`
    - `ErrReplyExpired` → `Code__1_00_TIMEOUT`
- [ ] 广播路径（`cli == nil && l.svcCtx.IsBroadcast()`）行为不变：不等待 ACK
- [ ] import `time`, `zero-service/common/antsx`

## 5. Float value 校验

**文件：`app/ieccaller/internal/logic/sendsetpointfloatlogic.go` 或 `app/ieccaller/internal/iec/ack.go`**

- [ ] 新增常量 `floatMatchEpsilon = 1e-5` 或函数 `floatValuesMatch`
- [ ] Promise 回调中校验 `ack.Value` 与 `expectedValue`
- [ ] 不匹配时 `Reject(pendingKey, ...)` 并返回 `Code__1_06_THIRD_PARTY`

## 6. 调用链变更

**文件：`app/ieccaller/internal/iec/clienthandler.go` + `app/ieccaller/internal/svc/servicecontext.go`**

- [ ] `ClientManager` / `ClientCall` 创建时需传入 `SetpointFloatReplyPool`
- [ ] 确认 `onCommandAck` 签名改回后 `OnASDU` 的 switch 调用也改回 `c.onCommandAck(ctx, packet)`

## 7. 验证

- [ ] `go build ./app/ieccaller/...` 零错误
- [ ] `go test ./app/ieccaller/internal/iec` 通过
- [ ] `go test ./app/ieccaller/internal/logic` 通过（如有）
- [ ] 手动验证：通过 iecagent 模拟从站，下发 setpoint float，验证 ACK 返回 accepted/rejected/timeout
