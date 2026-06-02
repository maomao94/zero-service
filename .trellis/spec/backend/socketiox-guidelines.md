# SocketIO 实时通信规范

> `common/socketiox/` 包的使用约定和优化模式。

---

## 包结构

```
common/socketiox/
├── server.go      # SocketIO 服务器核心（Session 管理、事件处理、房间管理）
├── container.go   # 服务发现与 gRPC 客户端容器（Etcd/Nacos/Direct）
└── handler.go     # HTTP 处理器封装
```

---

## 核心 API

### Server 创建

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
```

### Session 操作

```go
session := srv.GetSession(socketId)
sessions, ok := srv.GetSessionByKey("userId", userId)
sessions, ok := srv.GetSessionByDeviceId(deviceId)

session.JoinRoom(room)
session.LeaveRoom(room)
session.EmitDown(event, payload, reqId)
session.Close()
```

### 广播操作

```go
srv.BroadcastRoom(room, event, payload, reqId)
srv.BroadcastGlobal(event, payload, reqId)
```

---

## 事件处理约定

### 自定义事件处理器

实现 `EventHandler` 接口：

```go
type EventHandler interface {
    Handle(ctx context.Context, event string, payload *socketio.EventPayload) (string, error)
}
```

注册方式：

```go
socketiox.WithHandler("my_event", socketiox.EventHandlerFunc(func(ctx context.Context, event string, payload *socketio.EventPayload) (string, error) {
    // 处理逻辑
    return "response", nil
}))
```

### 内置事件

| 事件 | 常量 | 说明 |
|------|------|------|
| 连接 | `EventConnection` | 客户端连接时触发 |
| 断开 | `EventDisconnect` | 客户端断开时触发 |
| 上行 | `EventUp` | 通用上行消息 |
| 加入房间 | `EventJoinRoom` | 客户端请求加入房间 |
| 离开房间 | `EventLeaveRoom` | 客户端请求离开房间 |
| 房间广播 | `EventRoomBroadcast` | 客户端请求向房间广播 |
| 全局广播 | `EventGlobalBroadcast` | 客户端请求全局广播 |

---

## 响应模式

### 统一响应函数

使用 `sendResponse` 统一处理 Ack/Reply：

```go
// 内部函数，已在 server.go 中实现
func sendResponse(session *Session, payload *socketio.EventPayload, code int, msg string, data any, reqId string) {
    if payload.Ack != nil {
        payload.Ack(string(buildRespJson(code, msg, data, reqId)))
    } else {
        _ = session.ReplyEventDown(code, msg, data, reqId)
    }
}
```

### Payload 解析

使用 `parseJsonPayload` 统一处理 JSON 字符串：

```go
// 内部函数，已在 socketgtw/internal/logic/helper.go 中实现
func parseJsonPayload(raw string) any {
    b := []byte(raw)
    var js json.RawMessage
    if jsonx.Unmarshal(b, &js) == nil {
        return json.RawMessage(b)
    }
    return raw
}
```

---

## 客户端容器

### 创建容器

```go
container := socketiox.MustNewPubContainer(zrpc.RpcClientConf{
    Endpoints: []string{"localhost:8080"},  // 直连
    // 或
    Etcd: discov.EtcdConf{Hosts: []string{"..."}, Key: "socketgtw"},
    // 或
    Target: "nacos://user:pass@host/service",
})
```

### 使用容器

```go
cli := container.GetClient(endpoint)
clis := container.GetClients()

// 广播到所有客户端
for _, cli := range clis {
    threading.GoSafe(func() {
        cli.BroadcastRoom(ctx, &socketgtw.BroadcastRoomReq{...})
    })
}
```

---

## 并发安全

### 锁使用约定

1. **Server.sessions**：使用 `sync.RWMutex`，读多写少场景
2. **Session.metadata**：使用 `sync.Mutex`，保护元数据读写
3. **SocketContainer.ClientMap**：使用 `sync.RWMutex`，保护客户端映射

### 锁优化模式

```go
// 先复制快照，释放锁后再遍历
func (srv *Server) GetSessionByKey(key, value string) ([]*Session, bool) {
    srv.lock.RLock()
    snapshot := make([]*Session, 0, len(srv.sessions))
    for _, sess := range srv.sessions {
        snapshot = append(snapshot, sess)
    }
    srv.lock.RUnlock()

    var sessions []*Session
    for _, sess := range snapshot {
        if sess.GetMetadata(key) == value {
            sessions = append(sessions, sess)
        }
    }
    // ...
}
```

---

## 禁止模式

### Don't: 在锁内调用其他锁保护的方法

```go
// ❌ 错误：持有 srv.lock 时调用 sess.GetMetadata
func (srv *Server) GetSessionByKey(key, value string) ([]*Session, bool) {
    srv.lock.RLock()
    defer srv.lock.RUnlock()
    for _, sess := range srv.sessions {
        if sess.GetMetadata(key) == value {  // GetMetadata 也会获取 sess.lock
            // ...
        }
    }
}
```

### Don't: 忽略 EmitString 错误

```go
// ❌ 错误：静默忽略发送失败
sess.EmitString(EventStatDown, string(b))

// ✅ 正确：至少记录日志
if err := sess.EmitString(EventStatDown, string(b)); err != nil {
    logx.Errorf("[socketio] failed to send stat: conn=%s, err=%v", sess.ID(), err)
}
```

---

## 常见错误

### 错误: 日志参数缺失

**症状**: 日志输出 `%!s(MISSING)` 或格式错误

**原因**: Debugf/Errorf 格式字符串参数数量不匹配

**修复**: 确保格式占位符与参数数量一致

```go
// ❌ 错误
logx.WithContext(ctx).Debugf("[socketio] processing request: conn=%s")

// ✅ 正确
logx.WithContext(ctx).Debugf("[socketio] processing request: conn=%s", socket.Id)
```

---

## 设计决策

### 决策: handler.go 保持独立

**背景**: handler.go 仅 40 行，是否合并到 server.go？

**选项**:
1. 合并到 server.go — 减少文件数量
2. 保持独立 — 职责分离

**决策**: 保持独立，原因：
- 职责独立（HTTP 集成 vs SocketIO 核心逻辑）
- server.go 已 800+ 行，不宜再增加
- 符合 Go 标准库按职责拆文件的惯例

### 决策: socketpush logic 文件保持原样

**背景**: socketpush 的 11 个 logic 文件结构相同（遍历客户端转发），是否去重？

**选项**:
1. 提取通用转发函数 — 减少重复
2. 保持 go-zero 生成结构 — 遵循框架约定

**决策**: 保持原样，原因：
- go-zero 框架每个 RPC 方法对应一个 logic 文件
- 文件结构清晰，易于维护和扩展
- 去重收益不大（每个文件仅 40-50 行）
