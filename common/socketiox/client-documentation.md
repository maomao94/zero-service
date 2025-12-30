# SocketIO 消息网关客户端对接文档

## 1. 概述

本文档用于指导前端浏览器客户端对接SocketIO消息网关服务，实现实时双向通信，包括默认事件和自定义事件,支持 ack 回调。

## 2. 连接配置

### 2.1 客户端库要求

- **版本**：`socket.io-client@4.x`
- **官网** `https://socket.io/zh-CN/docs/v4/`

## 3. 核心事件列表

### 3.1 客户端发送默认事件

| 事件名称 | 描述 | 数据格式 |
|---------|------|---------|
| `__up__` | 客户端上行消息 | `SocketUpReq` |
| `__join_room_up__` | 加入房间 | `SocketUpRoomReq` |
| `__leave_room_up__` | 离开房间 | `SocketUpRoomReq` |
| `__room_broadcast_up__` | 房间广播 | `SocketUpReq` |
| `__global_broadcast_up__` | 全局广播 | `SocketUpReq` |

### 3.2 服务器推送默认事件

| 事件名称 | 描述 | 数据格式 |
|---------|------|---------|
| `__down__` | 服务器下行消息 | `SocketResp` |
| `__stat_down__` | 统计信息推送 | `StatDown` |

## 4. 数据结构定义

### 4.1 SocketUpReq（客户端上行消息）

| 字段名 | 类型 | 必填 | 描述 |
|-------|------|------|------|
| `payload` | `any` | ✅ | 业务数据，任意JSON格式 |
| `reqId` | `string` | ✅ | 请求唯一标识，建议使用UUID |
| `room` | `string` | ❌ | 房间名称（用于广播） |
| `event` | `string` | ❌ | 事件名称（用于广播） |

**示例**：
```json
{
  "payload": {"type": "chat", "content": "Hello"},
  "reqId": "uuid-12345",
  "room": "room1",
  "event": "chat_message"
}
```

### 4.2 SocketUpRoomReq（房间操作请求）

| 字段名 | 类型 | 必填 | 描述 |
|-------|------|------|------|
| `reqId` | `string` | ✅ | 请求唯一标识 |
| `room` | `string` | ✅ | 房间名称 |

**示例**：
```json
{
  "reqId": "uuid-12345",
  "room": "room1"
}
```

### 4.3 SocketResp（服务器响应）

| 字段名 | 类型 | 描述 |
|-------|------|------|
| `code` | `int` | 状态码，200为成功 |
| `msg` | `string` | 状态描述 |
| `payload` | `any` | 返回数据，任意JSON格式 |
| `seqId` | `int64` | 消息序列号 |
| `reqId` | `string` | 对应请求的reqId |

**示例**：
```json
{
  "code": 200,
  "msg": "处理成功",
  "payload": {"result": "ok"},
  "seqId": 123456,
  "reqId": "uuid-12345"
}
```

### 4.4 StatDown（统计信息）

| 字段名 | 类型 | 描述 |
|-------|------|------|
| `sId` | `string` | 会话ID |
| `rooms` | `[]string` | 当前加入的房间列表 |
| `nps` | `string` | 网络性能分数 |

**示例**：
```json
{
  "sId": "socket-123",
  "rooms": ["room1", "room2"],
  "nps": "85"
}
```

## 5. 错误码说明

| 错误码 | 描述 |
|-------|------|
| 200 | 成功 |
| 400 | 参数错误（缺少必填字段或格式错误） |
| 500 | 业务处理失败 |

## 6. 简化使用流程

### 6.1 连接与事件监听

```javascript
// 建立连接
const socket = io('http://your-server-url:port');

// 监听服务器消息
socket.on('__down__', (data) => {
  // 处理服务器下行消息
});

// 监听统计信息
socket.on('__stat_down__', (data) => {
  // 处理统计信息
});
```

## 7. 最佳实践

1. **请求ID**：每次请求必须生成唯一的`reqId`
2. **必填字段**：严格按照数据结构要求传递必填字段
3. **事件命名**：自定义事件名避免使用`__`前缀,事件文本格式为前后端约定
4. **错误处理**：根据`code`字段处理响应结果
5. **连接管理**：监听连接状态变化
6. **房间操作**：正确使用`__join_room_up__`和`__leave_room_up__`事件

## 8. 版本信息

| 版本 | 日期 | 说明 |
|------|------|------|
| 1.0 | 2025-12-30 | 初始版本 |
