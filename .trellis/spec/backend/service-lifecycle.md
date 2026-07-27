# 服务依赖与生命周期

## 适用范围

修改服务配置、`ServiceContext`、数据库/Redis/MQTT/gRPC client、scheduler、worker、后台循环或进程启停时读取。

## 配置边界

- 配置结构位于目标服务 `internal/config`，沿用 go-zero `service.ServiceConf`、`zrpc.RpcServerConf` 或 `rest.RestConf` 的相邻嵌入方式。
- 必需依赖在启动期验证并失败返回；不要等到首个请求才 nil panic，也不要静默回退到错误环境。
- 配置默认值只放在构造/规范化边界，业务 Logic 不读取环境变量或重新解释零值。
- 文档和测试配置使用占位凭据，不复制真实连接串。

## 装配规则

- `ServiceContext` 统一创建共享依赖，并把具体实现收窄为 Logic 需要的接口或回调。
- 公共组件若只需一个动作，优先注入窄函数或接口，例如人工任务执行注入 `TaskRunFunc`，不要让协议 client 持有整个 scheduler。
- 初始化有网络或存储失败时，沿用目标服务现有构造签名传播错误；不要为了统一风格吞错或跨服务改签名。
- 全局变量只用于不可变常量或框架要求；连接、缓存、会话和 store 不用包级可变单例隐藏所有权。

依据：`app/trigger/internal/svc/servicecontext.go`、`app/ispagent/internal/svc/servicecontext.go`、`app/djicloud/internal/svc/servicecontext.go`。

## 生命周期规则

- 入口负责启动/停止 server、scheduler、consumer 和后台 worker；需要组合时使用 go-zero `proc.ServiceGroup` 或相邻模式。
- 创建后台 goroutine 的组件必须有 `Start/Stop`、`Close` 或 context 取消路径，并保证重复关闭可接受。
- 网络、定时器和统计循环的所有者负责释放；不要让调用方猜测是否需要关闭。
- 请求结束后仍需继续的任务必须显式说明语义，并用 `context.WithoutCancel` 保留 values；仍应设置自己的超时或停止条件。
- 关闭时先阻止新工作，再取消/关闭资源并等待内部 goroutine 退出；回调与慢 I/O 尽量在锁外执行。

依据：`app/trigger/trigger.go`、`common/crontask/crontask.go`、`common/antsx/replypool.go`、`common/wsx/client.go`。

## 反模式

- 在 Logic 每次请求创建长期连接、scheduler 或 timing wheel。
- 后台循环使用 `context.Background()` 且没有 Stop/Close。
- 在持锁状态执行 gRPC、MQTT、数据库或用户回调。
- 初始化部分成功后直接返回，遗留已启动 goroutine 或连接。

## 验证

- 测试成功启动、配置无效、依赖初始化失败、重复关闭和 context 取消。
- 对连接/worker 变更运行目标包测试和 `go test -race`；检查测试结束后无挂起进程。
- 审查入口与 `ServiceContext`，确认每个长期资源都有唯一所有者和关闭路径。
