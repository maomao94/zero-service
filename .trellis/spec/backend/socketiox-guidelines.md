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

### Scenario: `__stat_down__` 房间统计下行契约

#### 1. Scope / Trigger

- Trigger: `common/socketiox` 向浏览器周期性发送统计事件，属于 SocketIO 服务端 → 前端的跨层响应契约。
- Scope: 仅描述通用统计 payload；业务房间命名、机构层级和订阅策略不写入通用契约。

#### 2. Signatures


```go
type StatDown struct {
    SocketId      string            `json:"socketId"`
    RoomCount     int               `json:"roomCount"`
    Rooms         []string          `json:"rooms,omitempty"`
    Nps           string            `json:"nps"`
    MetaData      map[string]string `json:"metadata,omitempty"`
    RoomLoadError string            `json:"roomLoadError,omitempty"`
}
```

#### 3. Contracts

- Event: `__stat_down__`
- `socketId`: 当前 SocketIO session id。
- `roomCount`: 当前 session 实际加入的房间总数，必须使用全量 room 列表长度。
- `rooms`: 调试样本，最多返回 `50` 个房间；不得全量返回大量 room。
- `nps`: SocketIO namespace。
- `metadata`: 从 token claims 中按 `SocketMetaData` 配置提取的元数据。
- `roomLoadError`: 初始房间加载失败或 JoinRoom 失败时写入错误信息；空值表示无加载错误。

#### 4. Validation & Error Matrix

| Condition | Behavior |
|-----------|----------|
| session 加入 0 个 room | `roomCount=0`，`rooms` 为空或省略 |
| session 加入 1-50 个 room | `roomCount=len(allRooms)`，`rooms` 返回这些 room |
| session 加入超过 50 个 room | `roomCount=len(allRooms)`，`rooms` 只返回前 50 个样本 |
| connectHook 返回错误 | 不断开连接，`roomLoadError` 写入错误信息 |
| 单个初始 JoinRoom 失败 | 继续处理后续 room，`roomLoadError` 写入失败信息 |

#### 5. Good/Base/Bad Cases

- Good: `roomCount=10000` 且 `len(rooms)<=50`，前端可知道真实订阅规模并抽样检查。
- Base: `roomCount=4` 且 `rooms` 返回全部 4 个房间，便于本地调试。
- Bad: `rooms` 返回全部 10000 个房间，导致周期性统计 payload 过大。

#### 6. Tests Required

- Unit or integration test should assert `RoomCount` equals full room count.
- Test with more than 50 rooms should assert serialized `rooms` length is 50.
- Test with zero rooms should assert payload remains valid and `roomCount=0`.
- Test connectHook error path should assert `roomLoadError` appears in `__stat_down__`.

#### 7. Wrong vs Correct

##### Wrong

```go
stat := StatDown{
    Rooms: sess.socket.Rooms(), // can emit thousands of room names every stat tick
}
```

##### Correct

```go
rooms := sess.socket.Rooms()
statRooms := rooms
if len(statRooms) > maxStatRooms {
    statRooms = statRooms[:maxStatRooms]
}

stat := StatDown{
    RoomCount: len(rooms),
    Rooms:     statRooms,
}
```

### Scenario: `__rooms_page_up__` 当前会话房间分页查询

#### 1. Scope / Trigger

- Trigger: 前端需要按需查询当前 session 已加入的房间列表，用于排查订阅状态。属于 SocketIO 客户端 → 服务端的查询事件。
- Scope: 仅查询当前连接自己的房间，不支持查其他 socketId。会过滤 SocketIO 自动加入的 socketId 内部房间。

#### 2. Signatures

```go
const EventRoomsPage = "__rooms_page_up__"

type SocketRoomsPageReq struct {
    ReqId    string `json:"reqId"`
    Page     int    `json:"page,omitempty"`
    PageSize int    `json:"pageSize,omitempty"`
}

type SocketRoomsPageRes struct {
    Total      int      `json:"total"`
    Page       int      `json:"page"`
    PageSize   int      `json:"pageSize"`
    TotalPages int      `json:"totalPages"`
    Rooms      []string `json:"rooms"`
}
```

#### 3. Contracts

- Event: `__rooms_page_up__`
- `reqId`: 必填，请求唯一标识。
- `page`: 可选，默认 1。
- `pageSize`: 可选，默认 50，最大 200。
- 响应通过 ack 或 `__down__` 返回，遵循现有 `sendResponse` 模式。
- 返回的 rooms 已按名称排序，且过滤掉 socketId 内部房间。

#### 4. Validation & Error Matrix

| Condition | Behavior |
|-----------|----------|
| `reqId` 为空 | 返回 400 "reqId为必填项" |
| `page <= 0` | 归一化为 1 |
| `pageSize <= 0` | 归一化为 50 |
| `pageSize > 200` | 截断为 200 |
| `page` 超出总页数 | 返回空 rooms 列表 |

#### 5. Good/Base/Bad Cases

- Good: `page=1, pageSize=200`，前端一次拿到大量房间用于排查。
- Base: `page=1, pageSize=50`（默认），前端分页翻页。
- Bad: 不过滤 socketId 房间，前端看到内部自房间。

#### 6. Tests Required

- `TestBuildRoomsPageResSortsAndPaginates` — 排序 + 分页正确
- `TestBuildRoomsPageResNormalizesDefaults` — 默认值归一化
- `TestBuildRoomsPageResCapsPageSize` — 上限截断
- `TestBuildRoomsPageResOutOfRangePage` — 越界页返回空
- `TestVisibleSessionRoomsFiltersSocketIdRoom` — 过滤 socketId 房间

#### 7. Wrong vs Correct

##### Wrong

```go
rooms := session.socket.Rooms()
res := buildRoomsPageRes(rooms, req.Page, req.PageSize)
```

##### Correct

```go
rooms := visibleSessionRooms(session.socket.Rooms(), session.ID())
res := buildRoomsPageRes(rooms, req.Page, req.PageSize)
```

---

### Scenario: `EnableStreamEventNotify` UpSocketMessage 通知开关

#### 1. Scope / Trigger

- Trigger: socketgtw 连接/断开/加入房间时会调用下游 StreamEvent 服务的 `UpSocketMessage` RPC。某些场景（本地调试、StreamEvent 不可用）需要关闭这些通知。
- Scope: 仅控制 socketgtw → StreamEvent 的生命周期通知，不影响 `__up__` 业务消息转发。

#### 2. Signatures

```go
// config.go
type Config struct {
    EnableStreamEventNotify bool `json:",default=true"`
}
```

#### 3. Contracts

- 默认 `true`，保持现有行为。
- 设为 `false` 时，以下 hook 跳过 RPC 调用：
  - `connectHook` (`__connection__`)：返回空 rooms，不调用 UpSocketMessage
  - `disconnectHook` (`__disconnect__`)：直接返回 nil
  - `preJoinRoomHook` (`__join_room_up__`)：直接返回 nil
- `sockethandler` 的 `EventUp` (`__up__`) 不受此开关影响。

#### 4. Validation & Error Matrix

| Condition | Behavior |
|-----------|----------|
| `EnableStreamEventNotify = true` | 正常调用 UpSocketMessage |
| `EnableStreamEventNotify = false` | 跳过调用，connectHook 返回空 rooms |
| StreamEvent 服务不可用 + 开关为 true | 连接失败，`roomLoadError` 写入错误 |

#### 5. Good/Base/Bad Cases

- Good: 本地开发时设为 `false`，避免 StreamEvent 依赖。
- Base: 生产环境保持 `true`。
- Bad: 关闭后忘记开启，导致前端收不到初始房间列表。

#### 6. Tests Required

- 集成测试：关闭开关后连接不调用 UpSocketMessage。
- 集成测试：开启开关后连接正常加载房间。

#### 7. Wrong vs Correct

##### Wrong

```go
// 全部硬编码调用，无法关闭
socketiox.WithConnectHook(func(ctx context.Context, session *socketiox.Session) ([]string, error) {
    res, err := svcCtx.StreamEventCli.UpSocketMessage(ctx, ...)
    ...
})
```

##### Correct

```go
socketiox.WithConnectHook(func(ctx context.Context, session *socketiox.Session) ([]string, error) {
    if !c.EnableStreamEventNotify {
        return nil, nil
    }
    res, err := svcCtx.StreamEventCli.UpSocketMessage(ctx, ...)
    ...
})
```

---

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

### 决策: 场站级房间订阅策略

**背景**: 机构有父子层级关系，需求是"儿子不能收到父亲消息，父亲可以收到儿子消息"。

**选项**:
1. 父级连接时订阅所有子场站 room（连接时 fan-out）
2. 子级发送时广播给父链 inbox room（发送时 fan-out）
3. 业务路由层统一计算目标 rooms（路由层 fan-out）

**决策**: 选择方案 1（连接时 fan-out），原因：
- 发送端最简单，只发数据所属场站的精确 room
- Socket.IO room 本身轻量，大量 room 不是性能瓶颈
- 避免发送端需要知道父链结构

**规则**:
- room 只绑定场站，不绑定父级机构
- 父级用户连接时 join 所有下级场站 room
- 场站用户连接时只 join 自己场站 room
- `__stat_down__` 不全量上报 rooms，只返回 roomCount + 最多 50 个样本
- `__rooms_page_up__` 按需分页查询完整房间列表
