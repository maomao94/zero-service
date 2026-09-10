# Socket.IO Server

修改 `common/socketiox`、Socket.IO Session、房间、广播、Ack、多节点转发或 Nacos 发现时读取。

### Server 架构

- 基于 `github.com/doquangtan/socketio/v4`，封装房间管理、广播、统计上报。
- `Server` 持有 `eventHandlers map[string]EventHandler` 和 `sessions map[string]*Session`（`sync.RWMutex` 保护）。
- 钩子: `tokenValidator`、`connectHook`、`disconnectHook`、`preJoinRoomHook`。
- `TokenValidator` 同时负责校验 token 和返回已验证 claims，签名为 `func(string) (map[string]any, bool)`；认证和 metadata 提取共用一次校验结果。
- 连接时从 JWT claims 提取身份到 session metadata：标准身份键按 `authctx.DefaultClaimAliases`（canonical 键 + 别名）默认提取，`WithContextKeys` 只增补额外 claim 名（按原名存储）；claims 缺失 `auth-type` 时按 `device-id` 有无兜底推导 `user`/`device`。

依据：`common/socketiox/server.go`、`common/authctx/claims.go`。

### 内置事件

| 事件 | 方向 | 用途 |
|------|------|------|
| `__connection__` | 系统 | 连接建立，加入初始房间 |
| `__disconnect__` | 系统 | 断开，清理 session |
| `__up__` | 上行 | 主上行事件，路由到 `EventUp` handler |
| `__join_room_up__` | 上行 | 加入房间（经过 preJoinRoomHook） |
| `__leave_room_up__` | 上行 | 离开房间 |
| `__rooms_page_up__` | 上行 | 分页获取房间列表 |
| `__room_broadcast_up__` | 上行 | 房间广播 |
| `__global_broadcast_up__` | 上行 | 全局广播 |
| `__stat_down__` | 下行 | 每分钟推送统计（Nps、房间数、metadata） |
| `__down__` | 下行 | 下游事件推送 |

### 响应模式

- **Ack 优先 + ReplyDown 兜底**: 客户端带 Ack 回调用 Ack 响应；无 Ack 时通过 `__down__` 事件下行推送。
- 响应码: `200`（成功）、`400`（参数错误）、`500`（业务错误）。
- `__down__` 是保留事件名，不能用于广播。

依据：`common/socketiox/server.go`、`common/socketiox/handler.go`。

### Session 管理

- `Session` 提供房间操作 (`JoinRoom`、`LeaveRoom`)、多种 Emit 方法 (`EmitAny`、`EmitString`、`EmitDown`、`EmitEventDown`、`ReplyEventDown`)。
- Session 元数据: `GetMetadata(key)`、`AllMetadata()`、`SetMetadata(key, val)`；`auth-type` 首次写入后不可覆盖。
- 事件上下文: 一律用 `session.NewCtx(event)` 构造，内含 authorization token、metadata 中全部标准身份键和 log 字段 `socketId`/`event`，禁止在事件回调里手工拼 `logx.WithFields` + `authctx.WithAuthorization`。
- 身份 metadata 存储键为 canonical 标准键（`user-id`、`device-id`…）；`GetSessionByKey` 内部用 `authctx.ResolveClaimKey` 归一化查询键（`userId`/`user_id`/`uid` 均可命中），未知键（配置增补的 claim 名）按原名比对。
- 查询: `GetSession(id)`、`GetSessionByDeviceId()`、`GetSessionByUserId()`、`GetSessionByKey()`。

### 多节点 (SocketContainer)

- `SocketContainer` 管理 gRPC 客户端池 (`map[string]socketgtw.SocketGtwClient`)。
- 支持三种服务发现: Direct endpoint、Etcd discovery、Nacos discovery。
- Nacos: 订阅服务变更 + 60s 轮询兜底。
- Etcd: go-zero `discov.Subscriber` + subset 采样（最多 32 节点）。
- Nacos gRPC 端口从 `metadata["gRPC_port"]` 提取。

依据：`common/socketiox/container.go`。

### Nacos 服务发现 (nacosx / socketiox 共用)

`common/nacosx` 实现 grpc `resolver`，`common/socketiox` 的 `SocketContainer` 复用同样的订阅模式。

**架构与数据流**（两处一致，依据 `common/nacosx/builder.go`、`common/socketiox/container.go`）：

```
Subscribe 回调（服务端推送） ─┐
                             ├→ pipe chan []string → populateEndpoints / populateClientMap → UpdateState / syncClientMap
ticker 每 60s SelectAllInstances ─┘
```

**核心约定**：

- 订阅回调与 60s 轮询读的是 **同一份 SDK 缓存**（`serviceInfoHolder`，`SelectAllInstances` 内部也读它），因此两条通路必须使用**同一个过滤函数**，禁止内联 copy。
- 唯一合法的地址过滤函数是包内 `extractHealthyGRPCInstances(services []model.Instance) []string`：
  - 格式: `ip:metadata["gRPC_port"]`；无 `gRPC_port`、不健康、未启用的实例一律丢弃（日志说明原因）。
  - `healthy`/`enabled` 由 nacos-server JSON 透传（`model.Instance`），不是恒真：节点心跳停止到被删（ephemeral 15s 窗口）、健康检查失败期间会为 false；SDK 自己的 `SelectInstances(HealthyOnly)` 也依赖该字段（`host.Healthy == healthy && host.Enable && host.Weight > 0`，SDK naming_client.go 内）。
- 所有发往 `pipe` 的发送必须用 `select` + `ctx.Done()` 保护，关闭后无消费者时不会永久阻塞 goroutine：

```go
addrs := extractHealthyGRPCInstances(instances)
select {
case pipe <- addrs:
case <-ctx.Done():
    return
}
```

- resolver 关闭：`resolvr.Close()` 用 `sync.Once` 包住 `cancelFunc()` + `client.CloseClient()`，幂等、资源释放更彻底。

依据：`common/nacosx/builder.go`、`common/nacosx/resolver.go`、`common/socketiox/container.go`。

### 反模式 (nacosx / SocketContainer Nacos)

- 订阅回调里内联复制实例过滤逻辑（不复用 `extractHealthyGRPCInstances`）——回调推全量（含不健康、无 gRPC_port 实例 fallback 普通端口），轮询推健康子集，两通路交替 `UpdateState` 导致地址抖动；fallback 的普通端口根本不是 gRPC 端口。
- 删除 `healthy`/`enable` 过滤——节点老化窗口期会把死节点推给 grpc client。
- `pipe <- addrs` 裸阻塞发送——resolver/容器关闭后（若 select 已选中 ticker 分支）goroutine 永久阻塞泄露，必须用 `select` + `ctx.Done()`。
- 对 nacos 实例字段做"反正都传 healthy"的假设——SDK 不转换、不默认填充，字段值完全来自服务端。

### 反模式 (socketiox)

- 使用 `__down__` 作为广播事件名（会被拒绝）。
- 绕过 session 直接操作底层 socket 连接。
- 在 `preJoinRoomHook` 中进行耗时操作（阻塞连接流程）。
- 不检查 Ack 就直接使用 ReplyDown 模式（可能重复推送）。

验证连接、断开、房间、广播、Ack/ReplyDown 和多节点 gRPC 转发。
