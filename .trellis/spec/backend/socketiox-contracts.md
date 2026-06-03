# SocketIO 协议契约

> SocketIO 事件名、上下行 payload、`UpSocketMessage`、统计和房间分页的 canonical source。普通 API 和并发实现先读 [`socketiox-guidelines.md`](./socketiox-guidelines.md)。

## When to read

- 改 SocketIO 事件名、JSON 字段、ack / reply 响应、`UpSocketMessage` RPC 或 `facade` proto 注释。
- 改 `__stat_down__`、`__rooms_page_up__`、`EnableStreamEventNotify` 或初始房间加载行为。
- 调整前端、socketgtw、StreamEvent 或业务服务之间的跨层协议。

## UpSocketMessage Proto 契约

`UpSocketMessage` 是 socket 事件通知回调。socketgtw 在连接、断开、加入房间、业务上行等时机调用下游业务服务，由业务服务决定处理方式。

### socketgtw -> 业务服务 payload

| event | payload 结构 | 说明 |
| --- | --- | --- |
| `__connection__` | `{"metadata": {...}}` | metadata 来自 token claims，具体 key 由网关配置决定 |
| `__disconnect__` | `{"metadata": {...}}` | 同上 |
| `__join_room_up__` | `{"metadata": {...}, "room": "房间名"}` | metadata 加浏览器请求的房间名 |
| `__up__` | 浏览器原始请求 JSON | 包含 reqId、payload、room 可选、event 可选 |
| 自定义事件 | 由 socketgtw handler 决定 | 业务服务自行解析 |

### 业务服务 -> socketgtw 返回语义

| event | `res.Payload` | 行为 |
| --- | --- | --- |
| `__connection__` | 房间名数组 JSON，如 `["room1","room2"]` | socketgtw 自动 JoinRoom |
| `__disconnect__` | 通常为空 | socketgtw 不做额外动作 |
| `__join_room_up__` | 正常返回或 gRPC 错误 | 错误时拒绝加入 |
| `__up__` | 业务数据 | socketgtw 包装成 SocketResp 返回浏览器 |

### Proto 注释规则

Proto 注释描述 socketgtw 发送的 payload 结构，不描述业务层房间命名、metadata 语义和鉴权规则。

Wrong:

```protobuf
// metadata.user_id: 用户ID
// metadata.dept_code: 机构编码
// 返回 payload 示例: ["alarm:dept:001","fire_alarm:dept:001"]
```

Correct:

```protobuf
// __connection__ payload: {"metadata": {...}}
// metadata 来自 token claims, 具体 key 由 socketgtw 网关配置决定。
// 返回: 房间名数组 JSON, socketgtw 会自动 JoinRoom。
```

## Scenario: `__stat_down__` 房间统计下行契约

### 1. Scope / Trigger

- Trigger: `common/socketiox` 向浏览器周期性发送统计事件，属于 SocketIO 服务端 -> 前端的跨层响应契约。
- Scope: 仅描述通用统计 payload；业务房间命名、机构层级和订阅策略不写入通用契约。

### 2. Signatures

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

### 3. Contracts

- Event: `__stat_down__`。
- `socketId`: 当前 SocketIO session id。
- `roomCount`: 当前 session 实际加入的房间总数，必须使用全量 room 列表长度。
- `rooms`: 调试样本，最多返回 `50` 个房间，不得全量返回大量 room。
- `nps`: SocketIO namespace。
- `metadata`: 从 token claims 中按 `SocketMetaData` 配置提取的元数据。
- `roomLoadError`: 初始房间加载失败或 JoinRoom 失败时写入错误信息；空值表示无加载错误。

### 4. Validation & Error Matrix

| Condition | Behavior |
| --- | --- |
| session 加入 0 个 room | `roomCount=0`，`rooms` 为空或省略 |
| session 加入 1 到 50 个 room | `roomCount=len(allRooms)`，`rooms` 返回这些 room |
| session 加入超过 50 个 room | `roomCount=len(allRooms)`，`rooms` 只返回前 50 个样本 |
| connectHook 返回错误 | 不断开连接，`roomLoadError` 写入错误信息 |
| 单个初始 JoinRoom 失败 | 继续处理后续 room，`roomLoadError` 写入失败信息 |

### 5. Good/Base/Bad Cases

- Good: `roomCount=10000` 且 `len(rooms)<=50`，前端知道真实订阅规模并抽样检查。
- Base: `roomCount=4` 且 `rooms` 返回全部 4 个房间，便于本地调试。
- Bad: `rooms` 返回全部 10000 个房间，导致周期性统计 payload 过大。

### 6. Tests Required

- Unit or integration test asserts `RoomCount` equals full room count.
- More than 50 rooms asserts serialized `rooms` length is 50.
- Zero rooms asserts payload remains valid and `roomCount=0`.
- connectHook error path asserts `roomLoadError` appears in `__stat_down__`.

### 7. Wrong vs Correct

Wrong:

```go
stat := StatDown{
    Rooms: sess.socket.Rooms(),
}
```

Correct:

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

## Scenario: `__rooms_page_up__` 当前会话房间分页查询

### 1. Scope / Trigger

- Trigger: 前端需要按需查询当前 session 已加入的房间列表，用于排查订阅状态。
- Scope: 仅查询当前连接自己的房间，不支持查其他 socketId。会过滤 SocketIO 自动加入的 socketId 内部房间。

### 2. Signatures

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

### 3. Contracts

- Event: `__rooms_page_up__`。
- `reqId`: 必填，请求唯一标识。
- `page`: 可选，默认 1。
- `pageSize`: 可选，默认 50，最大 200。
- 响应通过 ack 或 `__down__` 返回，遵循 `sendResponse` 模式。
- 返回的 rooms 已按名称排序，且过滤掉 socketId 内部房间。

### 4. Validation & Error Matrix

| Condition | Behavior |
| --- | --- |
| `reqId` 为空 | 返回 400 `reqId为必填项` |
| `page <= 0` | 归一化为 1 |
| `pageSize <= 0` | 归一化为 50 |
| `pageSize > 200` | 截断为 200 |
| `page` 超出总页数 | 返回空 rooms 列表 |

### 5. Good/Base/Bad Cases

- Good: `page=1, pageSize=200`，前端一次拿到大量房间用于排查。
- Base: `page=1, pageSize=50`，前端分页翻页。
- Bad: 不过滤 socketId 房间，前端看到内部自房间。

### 6. Tests Required

- `TestBuildRoomsPageResSortsAndPaginates` asserts sorting and pagination.
- `TestBuildRoomsPageResNormalizesDefaults` asserts default normalization.
- `TestBuildRoomsPageResCapsPageSize` asserts max cap.
- `TestBuildRoomsPageResOutOfRangePage` asserts empty rooms.
- `TestVisibleSessionRoomsFiltersSocketIdRoom` asserts socketId room is filtered.

### 7. Wrong vs Correct

Wrong:

```go
rooms := session.socket.Rooms()
res := buildRoomsPageRes(rooms, req.Page, req.PageSize)
```

Correct:

```go
rooms := visibleSessionRooms(session.socket.Rooms(), session.ID())
res := buildRoomsPageRes(rooms, req.Page, req.PageSize)
```

## Scenario: `EnableStreamEventNotify` UpSocketMessage 通知开关

### 1. Scope / Trigger

- Trigger: socketgtw 连接、断开、加入房间时会调用下游 StreamEvent 服务的 `UpSocketMessage` RPC。某些场景需要关闭这些通知。
- Scope: 仅控制 socketgtw -> StreamEvent 的生命周期通知，不影响 `__up__` 业务消息转发。

### 2. Signatures

```go
type Config struct {
    EnableStreamEventNotify bool `json:",default=true"`
}
```

### 3. Contracts

- 默认 `true`，保持现有行为。
- 设为 `false` 时，`connectHook` 跳过 `__connection__` RPC 并返回空 rooms。
- 设为 `false` 时，`disconnectHook` 跳过 `__disconnect__` RPC 并返回 nil。
- 设为 `false` 时，`preJoinRoomHook` 跳过 `__join_room_up__` RPC 并返回 nil。
- `sockethandler` 的 `EventUp` / `__up__` 不受此开关影响。

### 4. Validation & Error Matrix

| Condition | Behavior |
| --- | --- |
| `EnableStreamEventNotify = true` | 正常调用 UpSocketMessage |
| `EnableStreamEventNotify = false` | 跳过调用，connectHook 返回空 rooms |
| StreamEvent 服务不可用且开关为 true | 连接不断开，`roomLoadError` 写入错误 |

### 5. Good/Base/Bad Cases

- Good: 本地开发时设为 `false`，避免 StreamEvent 依赖。
- Base: 生产环境保持 `true`。
- Bad: 关闭后忘记开启，导致前端收不到初始房间列表。

### 6. Tests Required

- 集成测试：关闭开关后连接不调用 UpSocketMessage。
- 集成测试：开启开关后连接正常加载房间。
- 回归测试：关闭开关不影响 `__up__` 业务消息转发。

### 7. Wrong vs Correct

Wrong:

```go
socketiox.WithConnectHook(func(ctx context.Context, session *socketiox.Session) ([]string, error) {
    res, err := svcCtx.StreamEventCli.UpSocketMessage(ctx, ...)
    // always calls downstream
})
```

Correct:

```go
socketiox.WithConnectHook(func(ctx context.Context, session *socketiox.Session) ([]string, error) {
    if !c.EnableStreamEventNotify {
        return nil, nil
    }
    res, err := svcCtx.StreamEventCli.UpSocketMessage(ctx, ...)
    // normal path
})
```

## Business room strategy note

场站级房间订阅策略曾选择“连接时 fan-out”：父级用户连接时 join 下级场站 room，场站用户只 join 自己场站 room，发送端只发精确 room。该决策属于业务服务策略，不写入通用 proto；通用契约只保留 `roomCount` 和分页查询，避免统计事件全量下发大量 room。
