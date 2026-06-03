# SocketIO 实时通信规范

> `common/socketiox/` 包 API、事件处理、Session、房间、容器和并发规则。协议 payload、事件契约和测试矩阵见 [`socketiox-contracts.md`](./socketiox-contracts.md)。

## When to read

- 改 `common/socketiox/` 的 Server、Session、handler、container、房间管理或事件处理实现。
- 改 `socketapp/` 中调用 `socketiox` 的推送、广播或 hook 逻辑。
- 排查 SocketIO 锁、Session metadata、Emit 错误、房间查询或容器客户端并发问题。
- 若任务涉及事件名、payload、`UpSocketMessage`、`__stat_down__` 或 `__rooms_page_up__`，继续读 [`socketiox-contracts.md`](./socketiox-contracts.md)。

## 包结构

```text
common/socketiox/
├── server.go      # SocketIO 服务器核心，Session、事件、房间
├── container.go   # 服务发现与 gRPC 客户端容器
└── handler.go     # HTTP 处理器封装
```

Decision: `handler.go` 保持独立。它只承载 HTTP 集成，`server.go` 已承载核心逻辑，合并会降低职责清晰度。

## 核心 API

```go
srv := socketiox.MustServer(
    socketiox.WithContextKeys([]string{"userId", "deviceId"}),
    socketiox.WithTokenValidator(func(token string) bool { ... }),
    socketiox.WithTokenValidatorWithClaims(func(token string) (map[string]any, bool) { ... }),
    socketiox.WithConnectHook(func(ctx context.Context, session *socketiox.Session) ([]string, error) { ... }),
    socketiox.WithDisconnectHook(func(ctx context.Context, session *socketiox.Session, reason string) error { ... }),
    socketiox.WithPreJoinRoomHook(func(ctx context.Context, session *socketiox.Session, reqId, room string) error { ... }),
    socketiox.WithHandler(socketiox.EventUp, myHandler),
)

session := srv.GetSession(socketId)
sessions, ok := srv.GetSessionByKey("userId", userId)
sessions, ok := srv.GetSessionByDeviceId(deviceId)

session.JoinRoom(room)
session.LeaveRoom(room)
session.EmitDown(event, payload, reqId)
session.Close()

srv.BroadcastRoom(room, event, payload, reqId)
srv.BroadcastGlobal(event, payload, reqId)
```

## 事件处理

```go
type EventHandler interface {
    Handle(ctx context.Context, event string, payload *socketio.EventPayload) (string, error)
}

socketiox.WithHandler("my_event", socketiox.EventHandlerFunc(func(ctx context.Context, event string, payload *socketio.EventPayload) (string, error) {
    return "response", nil
}))
```

| 事件 | 常量 | 用途 |
| --- | --- | --- |
| 连接 | `EventConnection` | 客户端连接时触发 |
| 断开 | `EventDisconnect` | 客户端断开时触发 |
| 上行 | `EventUp` | 通用上行消息 |
| 加入房间 | `EventJoinRoom` | 客户端请求加入房间 |
| 离开房间 | `EventLeaveRoom` | 客户端请求离开房间 |
| 房间广播 | `EventRoomBroadcast` | 客户端请求向房间广播 |
| 全局广播 | `EventGlobalBroadcast` | 客户端请求全局广播 |

## 响应和解析工具

统一响应使用 `sendResponse`：

```go
func sendResponse(session *Session, payload *socketio.EventPayload, code int, msg string, data any, reqId string) {
    if payload.Ack != nil {
        payload.Ack(string(buildRespJson(code, msg, data, reqId)))
    } else {
        _ = session.ReplyEventDown(code, msg, data, reqId)
    }
}
```

JSON 字符串 payload 用 `parseJsonPayload` 归一化：

```go
func parseJsonPayload(raw string) any {
    b := []byte(raw)
    var js json.RawMessage
    if jsonx.Unmarshal(b, &js) == nil {
        return json.RawMessage(b)
    }
    return raw
}
```

## 客户端容器

```go
container := socketiox.MustNewPubContainer(zrpc.RpcClientConf{
    Endpoints: []string{"localhost:8080"},
    // or Etcd / Nacos Target
})

for _, cli := range container.GetClients() {
    threading.GoSafe(func() {
        cli.BroadcastRoom(ctx, &socketgtw.BroadcastRoomReq{...})
    })
}
```

- `GetClient(endpoint)` 获取指定客户端。
- `GetClients()` 返回快照后再遍历，不在容器锁内执行 RPC。
- 广播是 best-effort fan-out，调用方负责日志和失败统计。

## 并发安全

| 资源 | 锁 | 规则 |
| --- | --- | --- |
| `Server.sessions` | `sync.RWMutex` | 读多写少，复制快照后再跨 Session 操作 |
| `Session.metadata` | `sync.Mutex` | metadata 读写必须走 Session 方法 |
| `SocketContainer.ClientMap` | `sync.RWMutex` | 容器客户端 map 只在锁内复制，不在锁内 RPC |

Wrong, 持有 `srv.lock` 时调用 `sess.GetMetadata`：

```go
srv.lock.RLock()
defer srv.lock.RUnlock()
for _, sess := range srv.sessions {
    if sess.GetMetadata(key) == value {
        // nested lock risk
    }
}
```

Correct, 先复制快照，释放锁后再访问 Session：

```go
srv.lock.RLock()
snapshot := make([]*Session, 0, len(srv.sessions))
for _, sess := range srv.sessions {
    snapshot = append(snapshot, sess)
}
srv.lock.RUnlock()

for _, sess := range snapshot {
    if sess.GetMetadata(key) == value {
        // safe
    }
}
```

## 禁止模式

- 不在锁内调用可能再次拿锁的方法或外部 RPC。
- 不忽略 `EmitString`、`EmitDown`、`ReplyEventDown` 等发送错误。
- 不在高频统计事件中全量发送大量 room 名。
- 不把业务房间命名、机构层级、鉴权语义写进通用 `facade` proto 注释。
- 不为了去重合并 go-zero 生成的 socketpush logic 文件；每个 RPC 方法对应一个 logic 文件，保持生成结构清晰。

Wrong:

```go
sess.EmitString(EventStatDown, string(b))
logx.WithContext(ctx).Debugf("[socketio] processing request: conn=%s")
```

Correct:

```go
if err := sess.EmitString(EventStatDown, string(b)); err != nil {
    logx.Errorf("[socketio] failed to send stat: conn=%s, err=%v", sess.ID(), err)
}

logx.WithContext(ctx).Debugf("[socketio] processing request: conn=%s", socket.Id)
```

## Tests Required

- Session 查询：多连接、多 metadata key、并发读写不死锁。
- 房间操作：Join、Leave、BroadcastRoom、BroadcastGlobal 的成功和失败路径。
- 容器广播：客户端快照、RPC 失败记录、并发 fan-out 不持锁。
- 发送错误：`EmitString` / reply 错误至少记录日志。
- 协议变更另按 [`socketiox-contracts.md`](./socketiox-contracts.md) 的测试矩阵补齐。
