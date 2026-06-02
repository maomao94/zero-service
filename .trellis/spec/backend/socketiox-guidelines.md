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

## UpSocketMessage Proto 契约

### 概念

`UpSocketMessage` 是 socket 事件通知回调。socketgtw 在连接建立、断开、加入房间、业务上行等时机调用该 RPC，把事件通知给下游业务服务，由业务服务决定如何处理。

### payload 结构（socketgtw → 业务服务）

socketgtw 按 event 类型构造 payload，业务服务按对应结构解析：

| event | payload 结构 | 说明 |
|-------|-------------|------|
| `__connection__` | `{"metadata": {...}}` | metadata 来自 token claims，具体 key 由网关配置决定 |
| `__disconnect__` | `{"metadata": {...}}` | 同上 |
| `__join_room_up__` | `{"metadata": {...}, "room": "房间名"}` | metadata + 浏览器请求的房间名 |
| `__up__` | 浏览器原始请求 JSON | 包含 reqId、payload、room(可选)、event(可选) |
| 自定义事件 | 由 socketgtw handler 决定 | 业务服务自行解析 |

### 返回值语义（业务服务 → socketgtw）

| event | res.Payload | 说明 |
|-------|-------------|------|
| `__connection__` | 房间名数组 JSON，如 `["room1","room2"]` | socketgtw 自动 JoinRoom |
| `__disconnect__` | 通常为空 | |
| `__join_room_up__` | 通过时正常返回；拒绝时返回 gRPC 错误 | socketgtw 拒绝加入 |
| `__up__` | 业务数据 | socketgtw 包装成 SocketResp 返回给浏览器 |

### 规则：proto 注释描述 payload 结构，不描述业务逻辑

proto 需要说明 socketgtw 发送的 payload 结构，让业务服务知道怎么解析。但不描述业务层的房间命名、metadata 语义和鉴权规则。

**禁止**：在 proto 注释中写业务属性
```protobuf
// ❌ 错误：把业务属性写进通用回调协议
// metadata.user_id: 用户ID
// metadata.dept_code: 机构编码
// 返回 payload 示例: ["alarm:dept:001","fire_alarm:dept:001"]
```

**正确**：描述 payload 结构，不写业务语义
```protobuf
// ✅ 正确：描述 socketgtw 发送的 payload 结构
// __connection__ payload: {"metadata": {...}}
// metadata 来自 token claims, 具体 key 由 socketgtw 网关配置决定。
// 返回: 房间名数组 JSON, socketgtw 会自动 JoinRoom。
```

**规则**：
1. proto 注释描述 socketgtw 发送的 payload 结构，让业务服务知道怎么解析
2. metadata key 由网关配置决定，不写死具体 key
3. 房间命名、鉴权逻辑由业务服务自行定义，不写进 facade proto

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

### 决策: UpSocketMessage proto 注释描述 payload 结构，不描述业务逻辑

**背景**: `UpSocketMessage` 是 socket 事件通知回调，socketgtw 把事件通知给下游业务服务，业务服务自己决定怎么处理。某次实现中把 Java 业务层的 `dept_code`、`alarm:dept`、`FIRE_ALARM_ROOM_PREFIX` 等写进了 proto 注释。

**决策**: proto 注释需要描述 socketgtw 发送的 payload 结构（让业务服务知道怎么解析），但不描述业务层的房间命名、metadata 语义和鉴权规则。

**原因**：
- payload 结构是 socketgtw → 业务服务的契约，业务服务需要知道怎么解析
- metadata key 由网关配置决定，不写死具体 key
- 房间命名和鉴权逻辑是业务服务自己定义的，不同服务可以有不同的规则

### 决策: socketpush logic 文件保持原样

**背景**: socketpush 的 11 个 logic 文件结构相同（遍历客户端转发），是否去重？

**选项**:
1. 提取通用转发函数 — 减少重复
2. 保持 go-zero 生成结构 — 遵循框架约定

**决策**: 保持原样，原因：
- go-zero 框架每个 RPC 方法对应一个 logic 文件
- 文件结构清晰，易于维护和扩展
- 去重收益不大（每个文件仅 40-50 行）
