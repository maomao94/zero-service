# SocketIO 消息网关客户端对接文档

## 1. 概述

本文档用于指导前端浏览器客户端对接SocketIO消息网关服务，实现实时双向通信，包括默认事件和自定义事件，支持ack回调和事件返回机制。

## 2. 快速开始

### 2.1 客户端库要求

- **推荐版本**：`socket.io-client@4.x`
- **官网文档**：[https://socket.io/zh-CN/docs/v4/](https://socket.io/zh-CN/docs/v4/)

### 2.2 基本连接示例

```javascript
// 建立连接
const socket = io('http://your-server-url:port', {
  transports: ['websocket', 'polling'],
  reconnection: true
});

// 监听连接事件
socket.on('connect', () => {
  console.log('连接成功，socket ID:', socket.id);
});

// 监听断开事件
socket.on('disconnect', (reason) => {
  console.log('连接断开:', reason);
});

// 监听服务器推送消息
socket.on('__down__', (data) => {
  console.log('收到服务器消息:', data);
});
```

## 3. 鉴权配置

### 3.1 鉴权概述

当SocketIO网关开启鉴权后，客户端必须携带有效的令牌才能建立连接。服务器只支持SocketIO原生的`auth`方式传递令牌。

### 3.2 获取令牌

客户端需要通过认证服务获取有效的令牌，获取方式取决于具体的认证体系：

- **前端直接获取**：通过调用认证API获取令牌
- **后端传递**：由后端服务通过接口返回令牌

### 3.3 使用令牌连接

使用SocketIO的`auth`选项传递令牌对象：

```javascript
// 携带auth建立连接
const socket = io('http://your-server-url:port', {
  transports: ['websocket', 'polling'],
  reconnection: true,
  // 使用auth选项传递令牌对象
  auth: {
    token: 'your-token-value' // 替换为实际的令牌
  }
});
```

## 4. 核心事件体系

### 4.1 客户端发送事件

| 事件名称 | 描述 | 数据格式 |
|---------|------|---------|
| `__up__` | 客户端上行消息 | `SocketUpReq` |
| `__join_room_up__` | 加入房间 | `SocketUpRoomReq` |
| `__leave_room_up__` | 离开房间 | `SocketUpRoomReq` |
| `__room_broadcast_up__` | 房间广播 | `SocketUpReq` |
| `__global_broadcast_up__` | 全局广播 | `SocketUpReq` |

### 4.2 服务器推送事件

| 事件名称 | 描述 | 数据格式 |
|---------|------|---------|
| `__down__` | 服务器响应消息 | `SocketResp` |
| `__stat_down__` | 统计信息推送 | `StatDown` |
| 自定义事件 | 业务事件推送 | `SocketDown` |

## 5. 数据结构定义

### 5.1 SocketUpReq（客户端上行请求）

| 字段名 | 类型 | 必填 | 描述 |
|-------|------|------|------|
| `payload` | `object` | ✅ | 业务数据 |
| `reqId` | `string` | ✅ | 请求唯一标识，建议使用UUID |
| `room` | `string` | ❌ | 房间名称（用于广播） |
| `event` | `string` | ❌ | 自定义事件名称（用于广播） |

**示例**：
```json
{
  "event": "custom_event",
  "payload": [
    {
      "bool": true,
      "float": 1.1,
      "int": 1,
      "list": [
        "1",
        "2",
        "3"
      ],
      "map": {
        "1": "1",
        "2": "2",
        "3": "3"
      },
      "nil": null,
      "string": "string",
      "struct": {
        "Name": "test",
        "Age": 1
      }
    },
    {
      "bool": true,
      "float": 1.1,
      "int": 1,
      "list": [
        "1",
        "2",
        "3"
      ],
      "map": {
        "1": "1",
        "2": "2",
        "3": "3"
      },
      "nil": null,
      "string": "string",
      "struct": {
        "Name": "test",
        "Age": 1
      }
    }
  ],
  "reqId": "6fb17fd2-d2a7-447c-a679-8aa07b44665a"
}
```

### 5.2 SocketUpRoomReq（房间操作请求）

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

### 5.3 SocketResp（服务器响应消息）

| 字段名 | 类型 | 描述 |
|-------|------|------|
| `code` | `int` | 状态码，200为成功 |
| `msg` | `string` | 状态描述 |
| `payload` | `object` | 业务数据 |
| `reqId` | `string` | 对应请求的reqId |

**示例**：
```json
{
  "code": 200,
  "msg": "处理成功",
  "payload": {
    "result": "ok",
    "data": {
      "name": "test",
      "age": 18
    }
  },
  "reqId": "uuid-12345"
}
```

### 5.4 SocketDown（自定义事件推送）

| 字段名 | 类型 | 描述 |
|-------|------|------|
| `event` | `string` | 事件名称 |
| `payload` | `object` | 业务数据 |
| `reqId` | `string` | 对应请求的reqId |

**示例**：
```json
{
  "event": "chat_message",
  "payload": {
    "content": "Hello from server",
    "sender": "server",
    "timestamp": 1633046400
  },
  "reqId": "uuid-12345"
}
```

### 5.5 StatDown（统计信息）

| 字段名 | 类型 | 描述 |
|-------|------|------|
| `sId` | `string` | 会话ID |
| `rooms` | `[]string` | 当前加入的房间列表 |
| `nps` | `string` | 网络性能分数 |
| `metadata` | `map[string]string` | 会话元数据 |

**示例**：
```json
{
  "sId": "socket-123",
  "rooms": ["room1", "room2"],
  "nps": "85",
  "metadata": {
    "userId": "user123",
    "deviceId": "device456"
  }
}
```

## 6. 事件返回机制

服务器支持两种消息推送方式：

### 6.1 响应式返回（`__down__`事件）

- **触发条件**：客户端发送请求后，服务器直接响应
- **数据格式**：`SocketResp`
- **特点**：包含状态码、消息和业务数据
- **用途**：用于请求-响应模式的交互

### 6.2 事件式返回（自定义事件）

- **触发条件**：服务器主动推送或异步响应
- **数据格式**：`SocketDown`
- **特点**：包含事件名称和业务数据
- **用途**：用于实时通知、推送和事件驱动场景

**示例**：
```javascript
// 监听自定义事件
socket.on('chat_message', (data) => {
  console.log('收到聊天消息:', data);
});

// 监听订单状态变化
socket.on('order_status_change', (data) => {
  console.log('订单状态变化:', data);
});
```

## 7. 错误码体系

| 错误码 | 描述 | 处理建议 |
|-------|------|---------|
| 200 | 成功 | 正常处理响应数据 |
| 400 | 参数错误 | 检查请求参数格式和必填字段 |
| 500 | 业务错误 | 查看msg字段获取详细错误信息，必要时重试 |

## 8. 最佳实践

1. **请求ID管理**：每次请求必须生成唯一的`reqId`，建议使用UUID
2. **事件命名规范**：
   - 系统事件使用`__`前缀，如`__up__`、`__down__`
   - 自定义事件使用驼峰命名，如`chatMessage`、`orderStatusChange`
   - 避免使用特殊字符和空格
3. **数据格式**：
   - `payload`使用JSON对象格式，无需转换为字符串
   - 支持各种数据类型，包括字符串、数字、布尔值、数组、对象等
4. **鉴权方式**：
   - 使用SocketIO原生的`auth`选项传递令牌，不要使用其他方式

## 9. 版本历史

| 版本 | 日期 | 说明 |
|------|------|------|
| 2.0 | 2026-02-11 | 重写版本，支持SocketIO原生auth鉴权，payload使用object类型 |
| 1.0 | 2025-12-30 | 初始版本 |