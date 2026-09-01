# 服务依赖与生命周期

> 修改服务配置、`ServiceContext`、数据库/Redis/MQTT/gRPC client、scheduler、worker、后台循环或进程启停时读取。
> 通用生命周期规则见 [core-rules.md](./core-rules.md)。

## 配置边界

| 规则 | 说明 |
|------|------|
| 配置位置 | 目标服务 `internal/config`，沿用 go-zero 嵌入方式 |
| 启动期验证 | 必需依赖启动期验证并失败返回，不等到首个请求才 nil panic |
| 默认值 | 只放在构造/规范化边界，Logic 不读取环境变量或重新解释零值 |
| 占位凭据 | 文档和测试配置使用占位凭据，不复制真实连接串 |

## 装配规则

| 规则 | 说明 |
|------|------|
| 窄注入 | 只需一个动作时优先注入窄函数或接口（如 `TaskRunFunc`） |
| 错误传播 | 初始化失败时沿用现有构造签名传播错误，不吞错或改签名 |
| 全局变量 | 只用于不可变常量或框架要求；连接/缓存/store 不用包级可变单例 |

依据：`app/trigger/internal/svc/servicecontext.go`、`app/ispagent/internal/svc/servicecontext.go`

## 生命周期

| 规则 | 说明 |
|------|------|
| 入口职责 | 启动/停止 server、scheduler、consumer 和后台 worker |
| 资源释放 | 网络、定时器和统计循环的所有者负责释放 |
| 请求后任务 | 用 `context.WithoutCancel` 保留 values，设置自己的超时或停止条件 |

依据：`app/trigger/trigger.go`、`common/crontask/crontask.go`、`common/antsx/replypool.go`

## 反模式

| 错误做法 | 正确做法 |
|---------|---------|
| Logic 每次请求创建长期连接/scheduler | 在 ServiceContext 创建共享实例 |
| 后台循环用 `context.Background()` 且无 Stop/Close | 必须有退出路径 |
| 持锁执行 gRPC/MQTT/数据库/回调 | 锁外执行慢操作 |
| 初始化部分成功后直接返回 | 确保所有资源都初始化成功或回滚 |

## 本地联调冒烟规范

> 端口被旧实例占用会导致新实例**静默启动失败**（无报错），客户端仍连到旧代码。

**必须按顺序操作：**

```bash
# 1. 杀干净旧实例
pkill -f "exe/<svc>" && pkill -f "go run .*<svc>"
pgrep -fl "exe/<svc>"  # 确认无残留

# 2. 启动新实例
(go run ./<svc> -f etc/<svc>.yaml > /tmp/<svc>.log 2>&1 &)

# 3. 校验新实例特征
grep "auto migrate" /tmp/<svc>.log  # 或检查启动日志时间戳

# 4. 端口确认
nc -z 127.0.0.1 <port>  # 确认端口持有进程是新启动的 PID

# 5. 最后才跑冒烟客户端
```

**排查原则：** 先验证运行时版本（进程启动时间、日志时间戳、PID），再做代码层假设。

依据：`app/live` 本地联调（break-loop 复盘：update_user 落库误判案例）

## 验证

- 测试成功启动、配置无效、依赖初始化失败、重复关闭和 context 取消
- 对连接/worker 变更运行目标包测试和 `go test -race`；检查测试结束后无挂起进程
- 审查入口与 `ServiceContext`，确认每个长期资源都有唯一所有者和关闭路径
