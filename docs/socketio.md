# SocketIO 消息网关

前端浏览器客户端对接文档，实现实时双向通信。

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
    reconnection: true
});

socket.on('connect', () => console.log('已连接, sid:', socket.id));
socket.on('disconnect', (reason) => console.log('断开:', reason));
```

### Token 认证

后端通过 `GenToken` gRPC 接口生成令牌，前端携带连接：

```javascript
const socket = io('http://your-server:11003', {
    transports: ['websocket', 'polling'],
    auth: { token: 'your-token-value' }
});
```

### 服务端配置

```yaml
# socketgtw
Name: socketgtw
ListenOn: 0.0.0.0:25007          # gRPC
http:
  Port: 11003                     # WebSocket (前端连接)
JwtAuth:
  AccessSecret: your-secret
  AccessExpire: 31536000
SocketMetaData: [userId, deviceId]  # Token 声明中提取的元数据字段

# socketpush
Name: socketpush.rpc
ListenOn: 0.0.0.0:25008
JwtAuth:
  AccessSecret: your-secret
SocketGtwConf:
  Endpoints: [127.0.0.1:25007]
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
| `sessionCount` | int | 在线会话数 |
| `roomCount` | int | 房间总数 |
| `sessionRooms` | object | 当前会话房间列表 |
| `roomLoadError` | object | 房间加载错误（含 `failedRooms` 和 `error`） |

## 错误处理

### 错误码

| 码 | 说明 |
|----|------|
| 200 | 成功 |
| 1000 | 连接已存在（新连接会踢掉旧连接） |
| 2000 | 请求超时 |
| 2001 | 消息队列满（高负载） |
| 3000 | 解析消息失败 |
| 4000 | Token 不存在 |
| 4001 | Token 无效或已过期 |
| 4002 | 无权限操作该房间 |
| 5000 | 房间不存在 |
| 5001 | 不在房间中 |
| 5002 | 已在房间中 |
| 9000 | 内部错误 |

### 房间加载错误

连接建立后，服务端通过 `__stat_down__` 事件推送统计信息。如果加载初始房间失败，`roomLoadError` 字段携带失败原因：

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

## 后端推送 API

后端通过 gRPC 调用 socketpush 推送消息（集群扇出模型）：

| 方法 | 说明 |
|------|------|
| `GenToken` / `VerifyToken` | 生成 / 验证连接令牌 |
| `JoinRoom` / `LeaveRoom` | 服务端控制房间 |
| `BroadcastRoom` | 向指定房间广播 |
| `BroadcastGlobal` | 全局广播 |
| `SendToSession` / `SendToSessions` | 按 Session ID 推送 |
| `SendToMetaSession` / `SendToMetaSessions` | 按元数据（userId 等）推送 |
| `KickSession` / `KickMetaSession` | 剔除会话 |
| `SocketGtwStat` | 网关统计 |

协议定义：[`socketpush.proto`](../socketapp/socketpush/socketpush.proto) · [`socketgtw.proto`](../socketapp/socketgtw/socketgtw.proto)

## 最佳实践

- 请求-响应模式使用 ack 回调，单向通知使用自定义事件
- 前端进入页面时 `join_room`，离开时 `leave_room`
- 服务端消息体为 JSON 字符串，前端统一封装解析函数
- OSD 数据 0.5Hz，避免每次收到时重渲染
