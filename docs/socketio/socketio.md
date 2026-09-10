# Socket.IO 对接文档

本文是 `socketgtw` 的客户端和后端对接契约，覆盖连接鉴权、身份上下文、内置事件、房间操作和服务端推送。

## 约定

- 客户端通过 Socket.IO 连接 `socketgtw` 的 HTTP 端口，默认示例端口为 `11003`。
- 请求优先使用 Socket.IO Ack；没有 Ack 时，服务端通过 `__down__` 返回响应。
- `__up__`、`__join_room_up__` 等以下划线开头的事件是网关保留事件，不要用于业务广播。
- Token 只在连接握手中传入，业务 handler 通过 context 读取解析后的身份，不要重复解析 Token。

## 架构

| 服务 | 目录 | 职责 |
|------|------|------|
| **socketgtw** | `socketapp/socketgtw` | WebSocket 连接管理、房间管理、消息路由、Token 认证 |
| **socketpush** | `socketapp/socketpush` | Token 生成/验证、gRPC 推送接口（后端服务调用入口） |

```
前端客户端 ──WebSocket──> socketgtw ──gRPC──> 业务服务
                                  <──gRPC──
后端服务 ──gRPC──> socketpush ──gRPC──> socketgtw ──WebSocket──> 前端客户端
```

## 连接与鉴权

### 基本连接

```javascript
const socket = io('http://your-server:11003', {
    transports: ['websocket', 'polling'],
    reconnection: true,
    reconnectionDelay: 1000,
    reconnectionDelayMax: 5000,
    reconnectionAttempts: Infinity,
});

socket.on('connect', () => console.log('已连接, sid:', socket.id));
socket.on('disconnect', (reason) => console.log('断开:', reason));
```

### Token 认证

Token 可以由 `socketpush.GenToken` 生成，也可以由业务网关生成后交给客户端。Token 必须使用 `socketgtw.JwtAuth.AccessSecret` 对应的密钥签名。

客户端通过 Socket.IO handshake 的 `auth.token` 传入：

```javascript
const socket = io('http://your-server:11003', {
    transports: ['websocket', 'polling'],
    auth: { token: 'your-token-value' },
    reconnection: true,
    reconnectionDelay: 1000,
    reconnectionDelayMax: 5000,
    reconnectionAttempts: Infinity,
});
```

连接失败时，服务端会拒绝握手；客户端应通过 `connect_error` 获取失败状态：

```javascript
socket.on('connect_error', (err) => {
    console.error('Socket.IO 连接失败:', err.message);
});
```

### 用户与设备鉴权

socketgtw 支持单实例同时服务用户和设备连接，通过 token claims 自动识别身份类型：

| Token 类型 | claims特征 | auth-type |
|-----------|-----------|-----------|
| 用户 Token | 包含 `user-id`/`user_id`/`userId`/`uid` | `user` |
| 设备 Token | 包含 `device-id`/`device_id`/`deviceId` | `device` |

**内置识别逻辑**（`common/socketiox` 连接流程默认执行）：
1. 从 token claims 提取标准身份键到 session metadata（`auth-type` 优先采用 claims 中的值）；
2. claims 缺失 `auth-type` 时按设备身份键兜底推导 `user`/`device`。

> `auth-type` 属于受保护元数据：首次写入后 `SetMetadata` 拒绝覆盖，确保会话身份在生命周期内不可篡改。

**业务层判断**：
```go
authType := authctx.GetAuthType(ctx)
if authType == "device" {
    deviceId := authctx.GetDeviceId(ctx)
    // 设备逻辑
} else {
    userId := authctx.GetUserId(ctx)
    // 用户逻辑
}
```

事件处理 ctx（handler/hook 接收到的）已携带解析后的身份键，可直接用 `authctx.GetUserId(ctx)`、`authctx.GetDeviceId(ctx)` 等读取，无需再解析 Token。

业务服务通过 gRPC 继续调用下游时，`common/grpcx` 会自动传播以下身份 metadata：

| Context 键 | gRPC metadata |
| --- | --- |
| `authorization` | `authorization` |
| `user-id` | `x-user-id` |
| `user-name` | `x-user-name` |
| `dept-code` | `x-dept-code` |
| `auth-type` | `x-auth-type` |
| `device-id` | `x-device-id` |

### 断线重连

客户端配置重连参数后，断开连接会自动重连：

| 参数 | 说明 | 推荐值 |
|------|------|--------|
| `reconnection` | 启用自动重连 | `true` |
| `reconnectionDelay` | 初始延迟（毫秒） | `1000` |
| `reconnectionDelayMax` | 最大延迟（毫秒） | `5000` |
| `reconnectionAttempts` | 最大重试次数 | `Infinity` |

**服务端 ping/pong 配置**：

| 参数 | 默认值 | 说明 |
|------|--------|------|
| `pingInterval` | 25s | 服务端发送 ping 的间隔 |
| `pingTimeout` | 25s | 等待 pong 的超时时间 |

网络断开检测时间：25s + 25s = 50秒

**重连行为**：
- 客户端断开（网络问题等）：最多 50秒后 socket.io 自动重连，指数退避 1s → 2s → 4s → 5s
- 服务端主动断开（房间加载失败等）：5秒后手动重连
- 连接成功后自动重新加入房间

**服务端主动断开重连**：
```javascript
socket.on('disconnect', (reason) => {
  // 服务端主动断开时，手动重连（socket.io 不会自动重连这种情况）
  if (reason === 'io server disconnect') {
    setTimeout(() => socket.connect(), 5000)
  }
})
```

### 服务端配置

```yaml
# socketgtw
Name: socketgtw
ListenOn: 0.0.0.0:25001          # gRPC
http:
  Port: 11003                     # WebSocket (前端连接)
JwtAuth:
  AccessSecret: your-secret
# SocketMetaData: [custom_claim]  # 可选：增补额外 claim 名，按原名存入 session metadata
EnableStreamEventNotify: false    # 是否通知下游 StreamEvent 服务

# socketpush
Name: socketpush.rpc
ListenOn: 0.0.0.0:25002
JwtAuth:
  AccessSecret: your-secret
SocketGtwConf:
  Endpoints: [127.0.0.1:25001]
```

**内置默认身份键提取**（`common/authctx.DefaultClaimAliases`，无需配置）：

| 标准键（metadata 存储键） | 支持的 token claim 名 |
|--------------------------|----------------------|
| `user-id` | `user-id`、`user_id`、`userId`、`uid` |
| `user-name` | `user-name`、`user_name` |
| `dept-code` | `dept-code`、`dept_code` |
| `auth-type` | `auth-type`（受保护，不可覆盖） |
| `device-id` | `device-id`、`device_id`、`deviceId` |

按元数据推送/剔除（`SendToMetaSession` 等）时，Key 支持上述任一别名写法，查询侧自动归一化到标准键。非标准的自定义 claim 使用配置中的原始 key 查询。

`SocketMetaData` 不配置也能完成默认身份提取。只有业务 Token 里还有额外 claim（例如 `tenant_id`、`dept_id`）需要用于 Session 定位时，才需要追加配置：

```yaml
SocketMetaData: [tenant_id, dept_id]
```

## 事件体系

### 客户端发送

| 事件 | 说明 | 数据格式 |
|------|------|----------|
| `__up__` | 浏览器上行消息 | `SocketUpReq` |
| `__join_room_up__` | 加入房间 | `SocketUpRoomReq` |
| `__leave_room_up__` | 离开房间 | `SocketUpRoomReq` |
| `__rooms_page_up__` | 分页查询已加入房间 | `SocketRoomsPageReq` |
| `__room_broadcast_up__` | 房间广播 | `SocketUpReq` |
| `__global_broadcast_up__` | 全局广播 | `SocketUpReq` |

### 服务器推送

| 事件 | 说明 | 数据格式 |
|------|------|----------|
| `__down__` | 非 ack 模式的异步响应 | `SocketResp` |
| `__stat_down__` | 统计信息（含 roomLoadError） | `StatDown` |
| 自定义事件 | 后端主动业务推送 | `SocketDown` |

> `__down__` 是系统保留事件。后端主动推送业务通知时推荐使用自定义事件名（如 `mqtt`、`drc:heart_beat`），数据结构仍为 `SocketDown`。

### Ack 与 `__down__`

客户端带 Ack 时，响应通过 Ack 返回；客户端不带 Ack 时，响应通过 `__down__` 返回。两种模式的响应体一致：

```json
{
  "code": 200,
  "msg": "处理成功",
  "payload": {},
  "reqId": "req-123"
}
```

建议客户端统一封装响应解析，不要同时为同一个请求注册 Ack 和 `__down__` 业务回调。

### 方向说明

| 术语 | SocketIO 场景 | DJI Cloud API 场景 |
|------|-------------|--------------------|
| `up`（上行） | 浏览器 → socketgtw | DJI 设备 → djicloud |
| `down`（下行） | socketgtw → 浏览器 | djicloud → DJI 设备 |

跨协议桥接时方向名因端侧不同而不同：

```
设备状态到前端：DJI drc/up 或 events → djicloud → SocketIO 自定义下行事件
前端控制设备：SocketIO __up__ → djicloud → DJI services 或 drc/down
```

## 数据结构

### SocketUpReq（客户端上行）

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `payload` | object | 是 | 业务数据 |
| `reqId` | string | 是 | 请求唯一标识，建议 UUID |
| `room` | string | 否 | 房间名称（广播） |
| `event` | string | 否 | 自定义事件名称（广播） |

```json
{ "event": "custom_event", "payload": { "key": "value" }, "reqId": "uuid" }
```

### SocketUpRoomReq（房间操作）

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `reqId` | string | 是 | 请求唯一标识 |
| `room` | string | 是 | 房间名称 |

### SocketResp（服务器响应）

| 字段 | 类型 | 说明 |
|------|------|------|
| `code` | int | 状态码，200 成功 |
| `msg` | string | 状态描述 |
| `payload` | object | 业务数据 |
| `reqId` | string | 对应请求的 reqId |

### SocketDown（服务器主动推送）

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `payload` | object | 是 | 业务数据 |
| `reqId` | string | 是 | 消息唯一标识 |
| `event` | string | 否 | 事件名称 |

### StatDown（统计信息）

| 字段 | 类型 | 说明 |
|------|------|------|
| `socketId` | string | 当前会话 ID |
| `roomCount` | int | 当前会话加入的房间数 |
| `rooms` | string[] | 当前会话房间列表，最多包含 50 个房间 |
| `nps` | string | Socket.IO 连接统计值 |
| `metadata` | object | 当前会话的标准身份 metadata |
| `roomLoadError` | string | 初始房间加载失败原因 |

## 错误处理

### 网关响应码

| 码 | 说明 |
|----|------|
| 200 | 成功 |
| 400 | 请求参数错误、payload 解析失败或缺少必填字段 |
| 500 | 业务处理失败、房间操作或下游服务失败 |

Token 无效时连接握手直接失败，不会进入业务事件响应流程。

### 房间加载错误

连接建立后，服务端通过 `__stat_down__` 事件推送当前会话统计信息。如果加载初始房间失败，`roomLoadError` 字段携带失败原因：

```javascript
socket.on('__stat_down__', (data) => {
    const stat = normalizeSocketPayload(data);
    if (stat.roomLoadError) {
        console.warn('房间加载失败:', stat.roomLoadError.failedRooms);
    }
});
```

## 业务场景

### MQTT 桥接

后端通过 socketpush 将 MQTT 消息推送到前端。前端加入对应 MQTT topic 房间即可接收。

```javascript
// 加入房间（房间名即 MQTT topic）
socket.emit('__join_room_up__', { reqId: uuid(), room: 'device/status/#' });

// 监听推送（事件名与 topic 对应）
socket.on('device/status/#', (data) => {
    const msg = normalizeSocketPayload(data);
    console.log('MQTT 消息:', msg.payload);
});
```

### DRC 远程控制

接收 DRC 心跳、OSD 和事件推送。房间名格式为 `drc:{type}:{deviceSn}`。

| 事件 | 房间 | 说明 |
|------|------|------|
| `drc:heart_beat` | `drc:heartbeat:{gatewaySn}` | DRC 心跳推送 |
| `drc:osd` | `thing/product/{deviceSn}/osd` | DRC 模式下 OSD 数据 |
| `drc:event` | `drc:event:{gatewaySn}` | DRC 相关业务事件 |

```javascript
// 加入 DRC 心跳房间
socket.emit('__join_room_up__', {
    reqId: uuid(), room: `drc:heartbeat:${gatewaySn}`
});
socket.on('drc:heart_beat', (data) => {
    const msg = normalizeSocketPayload(data);
    // msg.payload 包含 session_id、gateway_sn
});
```

### 设备遥测

前端加入设备遥测房间接收 OSD 和 State 数据。

| 事件 | 房间 | 说明 |
|------|------|------|
| `telemetry:osd` | `thing/product/{deviceSn}/osd` | OSD 遥测（0.5Hz，每 2 秒） |
| `telemetry:state` | `thing/product/{deviceSn}/state` | State 状态（变化时上报） |

```javascript
socket.emit('__join_room_up__', {
    reqId: uuid(), room: `thing/product/${deviceSn}/osd`
});
socket.on('telemetry:osd', (data) => {
    const msg = normalizeSocketPayload(data);
    updateDeviceOsdDisplay(msg.payload);
});
```

> 进入监控页面时加入房间，退出时离开。Payload 为 DJI 协议原始 JSON。

### Live 会议邀请

会议邀请沿用 DRC 定向事件的命名方式：事件名保持稳定，房间名在事件名后追加接收方身份。

| 事件 | 房间 | 说明 |
|------|------|------|
| `live:meeting-invite` | `live:meeting-invite:{identity}` | 通知指定登录身份加入会议 |

```javascript
const event = 'live:meeting-invite';
const room = `${event}:${identity}`;

socket.emit('__join_room_up__', { reqId: uuid(), room });
socket.on(event, (data) => {
    const msg = normalizeSocketPayload(data);
    // msg.payload 包含 meetingNo、meetingCode、meetingTitle、identity、invitedAt，
    // userId 和 userName 为可选的邀请人信息。
});
```

后端使用 `BroadcastRoom` 向上述房间发送自定义事件。`room` 只负责选择接收连接，`event` 负责客户端事件分发，两者不能互换。

## 后端推送 API

后端通过 gRPC 调用 socketpush 推送消息（集群扇出模型）：

| 方法 | 说明 |
|------|------|
| `GenToken` / `VerifyToken` | 生成 / 验证连接令牌 |
| `JoinRoom` / `LeaveRoom` | 服务端控制房间 |
| `BroadcastRoom` | 向指定房间广播 |
| `BroadcastGlobal` | 全局广播 |
| `SendToSession` / `SendToSessions` | 按 Session ID 推送 |
| `SendToMetaSession` / `SendToMetaSessions` | 按元数据（`user-id`、`device-id` 等）推送 |
| `KickSession` / `KickMetaSession` | 剔除会话 |
| `SocketGtwStat` | 网关统计 |

协议定义：[`socketpush.proto`](../../socketapp/socketpush/socketpush.proto) · [`socketgtw.proto`](../../socketapp/socketgtw/socketgtw.proto)

### 后端推送请求示例

按用户推送：

```json
{
  "reqId": "req-123",
  "key": "userId",
  "value": "user-001",
  "event": "notification",
  "payload": "{\"type\":\"meeting-invite\"}"
}
```

`key` 支持 canonical 键和内置别名，例如 `user-id`、`user_id`、`userId`、`uid`；网关会统一归一化。`payload` 是 JSON 字符串，不是嵌套 protobuf message。

## 最佳实践

- 请求-响应模式使用 ack 回调，单向通知使用自定义事件
- 前端进入页面时 `join_room`，离开时 `leave_room`
- 服务端消息体为 JSON 字符串，前端统一封装解析函数
- OSD 数据 0.5Hz，避免每次收到时重渲染
- 生产环境务必配置 `reconnection` 相关参数，确保断线自动恢复
- 用户和设备使用不同的 Token，服务端通过 claims 自动识别身份类型
- 默认只提取标准身份 claim；只有需要按自定义 claim 定位 Session 时才配置 `SocketMetaData`
