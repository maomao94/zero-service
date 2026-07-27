# IEC 104 控制与追踪规范

## 适用范围

修改 `common/iec104`、`app/ieccaller`、IEC 控制 RPC、ASDU 发送、MQTT 集群路由或 trace 传播时读取。

## 控制命令契约

- 对外控制方法和字段以 `app/ieccaller/ieccaller.proto` 为源，生成代码不手改。
- type ID 决定单点/双点、归一化/标度化/短浮点和带时标命令类型；值转换与范围校验使用 `common/iec104` 现有 helper。
- IOA 显示/解析沿用 `IoaHexAddress` 等项目格式，不在网关、Logic 和 MQTT 层各定义一种十六进制表示。
- `withTime`、select/execute、COT、COA、IOA 和 value 是协议字段，不能用默认值猜测或跨命令复用不兼容 payload。

依据：`app/ieccaller/ieccaller.proto`、`common/iec104/client/interface.go`、`common/iec104` 测试。

## 命令应答关联

- 每条 IEC 连接拥有自己的 `CommandReplyPool`，关联键包含 `coa:typeID:ioa`；不同连接不能共享 pending map。
- 发送前注册 pending，应答到达后按连接与关联键 resolve/reject；超时、断连和关闭必须拒绝等待者并清理。
- activation confirmation/termination 等 COT 按当前解析决定 accepted、rejected 或协议错误，不能把“收到 ASDU”一律当成功。
- 相同关联键的并发命令要遵循当前 duplicate/pending 约束，不通过覆盖旧 Promise 支持并发。

依据：`common/iec104/client/command_reply.go`、`common/iec104/client/clientmanager_test.go`、`app/ieccaller/internal/iec`。

## 本地与集群路由

- ieccaller 本地存在目标 client 时走本地 typed 调用；仅本地缺失时通过 MQTT 广播请求，避免同一命令本地和集群重复下发。
- MQTT 请求/响应使用 typed payload 和 TID，先注册 ReplyRouter 再 publish；topic 与 payload 由 IEC 领域层拥有。
- publish/收到响应/设备确认是三个阶段，RPC 成功语义以当前控制流程的最终确认点为准，不能从 client API 推断 Exactly Once。
- 集群请求可能重复、迟到或来自非目标节点，handler 必须按 TID 和设备连接状态处理。

## Trace 传播

- 在 ASDU 进入多通道转发前注入项目 trace header；同一 ASDU 复制到 MQTT/StreamEvent 等路径时保留同一 trace 语义。
- 下游收到 ASDU 时从 header 恢复 context，再调用业务/消息边界；不要创建无关联的新 trace。
- trace 扩展不能改变 IEC payload、COT 或设备可见协议字段；无法提取时按现有无 trace 路径处理。

依据：`common/iec104/trace.go`、`app/ieccaller/internal` 中 ASDU 分发与测试。

## 反模式

- 全局共享所有连接的 CommandReplyPool。
- 本地已连接仍广播，导致设备执行两次。
- MQTT publish 成功就返回设备控制成功。
- 用通用 map/JSON 手工表达已有 proto 控制请求。
- 转发每个分支重新生成 trace ID。

## 验证

- 覆盖每种命令类型、边界值、带时标、select/execute 和错误 COT。
- 覆盖快速确认、重复关联键、超时、断线、不同连接同键、本地命中和本地缺失广播。
- trace 测试断言多通道携带同一上下文且不改变 ASDU；并发路径运行 race test。
