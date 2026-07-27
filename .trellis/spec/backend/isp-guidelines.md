# ISP 协议与巡检规范

## 适用范围

修改 `common/isp`、`app/ispagent`、ISP 帧、注册/连接管理、命令处理、周期巡检或任务回执时读取。

## 帧与序列

- 帧结构由 `common/isp/message.go` 和 `common/isp/serializer.go` 定义：头尾标记、发送/接收序号、SessionSource、XML 长度和 XML payload 的端序/宽度不得由调用方重写。
- 编码前校验 payload 长度，解码先校验最小帧、头尾、声明长度和 XML；截断/畸形输入返回 error，不允许越界 panic。
- send/recv sequence 是连接协议状态，不是业务 task/client ID。响应的确认序号遵循当前 message/serializer 测试。
- XML 中的 command、result 和数据模型由 `common/isp` 类型拥有，Handler 负责业务分发，不手工拼 XML。

依据：`common/isp/message.go`、`common/isp/serializer.go`、`common/isp/*_test.go`。

## 身份与注册发布

- `SessionID` 标识连接会话；`ClientID` 是注册后解析出的业务客户端身份。两者查找命名空间独立，不能用 alias 或互相回退。
- 注册期间先完成网络交互和校验，再在同一 client 锁临界区发布注册状态与 `ClientID` 绑定，避免其他 goroutine 看到部分状态。
- 断线/重连必须按当前 Session 清理，旧 session 的 reader、response 或注册结果不得覆盖新连接。
- 业务按 ClientID 路由时必须确认客户端已注册；按 SessionID 处理连接控制时不得误用业务 ID。

依据：`common/isp/client.go`、`common/isp/client_test.go`、`app/ispagent/internal/handler`。

## 错误边界

- `IspError` 表示对端协议返回的业务错误；断连、超时、关闭、序列/解析失败等是传输/运行错误。
- `common/isp/errors.go` 保持传输中立，不返回 gRPC `status`；gRPC/HTTP 边界再映射。
- 请求应答关联必须先注册再发送，并在发送失败、超时、关闭和未知序号时完成清理。
- 错误日志保留 command、非敏感 client/session/task 标识，不记录完整 XML 中的敏感内容。

## 巡检与人工执行

- 周期任务由 `crontask.Scheduler` 驱动，ISP 只实现 handler/store 适配和业务命令。
- 命令 `41-1` 的人工触发通过窄 `TaskRunFunc` 调用 `Scheduler.RunNow(ctx, taskCode)`；不要向 `IspClient` 注入整个 Scheduler，也不要复制 cron 执行逻辑。
- 一次人工/周期执行生成一个 `task_patrolled_id`，通过 context metadata 贯穿发送、异步响应、持久化和上报；不得在中间层重新生成。
- `RunNow` 异步返回，RPC 成功仅表示已接受执行。测试通过有上限轮询等待真实持久化/回执。
- Enable/Disable、终止调度时间、lease/CAS 和 `LastRun` 语义遵循 [crontask-guidelines.md](./crontask-guidelines.md)。

依据：`app/ispagent/internal/handler`、`app/ispagent/internal/crontask`、`app/ispagent/internal/svc/servicecontext.go` 及测试。

## 反模式

- 用 SessionID 作为 ClientID，或注册前发布业务路由。
- 把所有对端拒绝映射成断线/内部错误。
- 先发送后注册响应 Promise，留下快速响应丢失窗口。
- 人工执行直接调用 handler 并更新周期 `NextRun`。
- response、持久化和回报各生成不同 patrol ID。

## 验证

- `common/isp` 测试覆盖帧边界、序列、注册原子性、身份查询、快速响应、超时、关闭和协议错误。
- `app/ispagent` 测试覆盖 41-1 接受语义、同一 patrol ID、成功/失败回执、人工执行不改周期状态。
- 涉及并发运行 `go test -race ./common/isp ./app/ispagent/internal/handler ./app/ispagent/internal/crontask`。
