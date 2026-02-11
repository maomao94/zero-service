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

| 事件名称                      | 描述      | 数据格式              |
|---------------------------|---------|-------------------|
| `__up__`                  | 客户端上行消息 | `SocketUpReq`     |
| `__join_room_up__`        | 加入房间    | `SocketUpRoomReq` |
| `__leave_room_up__`       | 离开房间    | `SocketUpRoomReq` |
| `__room_broadcast_up__`   | 房间广播    | `SocketUpReq`     |
| `__global_broadcast_up__` | 全局广播    | `SocketUpReq`     |

### 4.2 服务器推送事件

| 事件名称            | 描述      | 数据格式         |
|-----------------|---------|--------------|
| `__down__`      | 服务器响应消息 | `SocketResp` |
| `__stat_down__` | 统计信息推送  | `StatDown`   |
| 自定义事件           | 业务事件推送  | `SocketDown` |

## 5. 数据结构定义

### 5.1 SocketUpReq（客户端上行请求）

| 字段名       | 类型       | 必填 | 描述              |
|-----------|----------|----|-----------------|
| `payload` | `object` | ✅  | 业务数据            |
| `reqId`   | `string` | ✅  | 请求唯一标识，建议使用UUID |
| `room`    | `string` | ❌  | 房间名称（用于广播）      |
| `event`   | `string` | ❌  | 自定义事件名称（用于广播）   |

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

| 字段名     | 类型       | 必填 | 描述     |
|---------|----------|----|--------|
| `reqId` | `string` | ✅  | 请求唯一标识 |
| `room`  | `string` | ✅  | 房间名称   |

**示例**：

```json
{
  "reqId": "uuid-12345",
  "room": "room1"
}
```

### 5.3 SocketResp（服务器响应消息）

| 字段名       | 类型       | 描述         |
|-----------|----------|------------|
| `code`    | `int`    | 状态码，200为成功 |
| `msg`     | `string` | 状态描述       |
| `payload` | `object` | 业务数据       |
| `reqId`   | `string` | 对应请求的reqId |

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

| 字段名       | 类型       | 描述         |
|-----------|----------|------------|
| `event`   | `string` | 事件名称       |
| `payload` | `object` | 业务数据       |
| `reqId`   | `string` | 对应请求的reqId |

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

| 字段名        | 类型                  | 描述        |
|------------|---------------------|-----------|
| `sId`      | `string`            | 会话ID      |
| `rooms`    | `[]string`          | 当前加入的房间列表 |
| `nps`      | `string`            | 命名空间      |
| `metadata` | `map[string]string` | 会话元数据     |

**示例**：

```json
{
  "sId": "socket-123",
  "rooms": [
    "room1",
    "room2"
  ],
  "nps": "/",
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

| 错误码 | 描述   | 处理建议                  |
|-----|------|-----------------------|
| 200 | 成功   | 正常处理响应数据              |
| 400 | 参数错误 | 检查请求参数格式和必填字段         |
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

## 9. MQTT 桥接指导

### 9.1 概述

SocketIO 消息网关支持通过 MQTT 协议桥接其他系统的消息，例如 IEC 104 协议的工业设备数据。桥接后，这些消息会以统一的格式通过
SocketIO 推送给前端客户端。不同的 MQTT topic 会映射到不同的 SocketIO room，确保消息准确路由到对应的客户端。

### 9.2 桥接消息格式

桥接的 MQTT 消息会转换为统一的格式，遵循通用的 `event`、`payload`、`reqId` 结构。以下示例基于 IEC 104 协议桥接，详细协议定义请参考 [`iec104-protocol.md`](iec104-protocol.md) 文件：

```json
{
  "event": "mqtt",
  "payload": {
    "msgId": "f5871b411ded48c39c0438633e4e33c9",
    "host": "127.0.0.1",
    "port": 2404,
    "asdu": "M_SP_NA_1",
    "typeId": 1,
    "dataType": 0,
    "coa": 1,
    "body": {
      "ioa": 5,
      "value": false,
      "qds": 0,
      "qdsDesc": "QDS(00000000)[QDSGood]",
      "ov": false,
      "bl": false,
      "sb": false,
      "nt": false,
      "iv": false,
      "time": ""
    },
    "time": "2026-02-11 13:15:52.13621",
    "metaData": {
      "arrayId": [
        1,
        2,
        3
      ],
      "stationId": "330KV"
    }
  },
  "reqId": "d30749eef0b14de4b26d8e3e6e69ead7"
}
```

### 9.3 字段说明

| 字段名                | 类型       | 描述                                      |
|--------------------|----------|-----------------------------------------|
| `event`            | `string` | 固定为 "mqtt"，标识这是一个桥接的 MQTT 消息            |
| `payload`          | `object` | 桥接的消息内容，包含原始协议的数据。以下是基于 IEC 104 协议的示例结构 |
| `payload.msgId`    | `string` | 消息唯一标识                                  |
| `payload.host`     | `string` | 设备主机地址                                  |
| `payload.port`     | `number` | 设备端口号                                   |
| `payload.asdu`     | `string` | 应用服务数据单元类型（如 IEC 104 协议中的类型）            |
| `payload.typeId`   | `number` | 类型 ID                                   |
| `payload.dataType` | `number` | 数据类型                                    |
| `payload.coa`      | `number` | 公共地址                                    |
| `payload.body`     | `object` | 消息主体，包含具体的设备数据                          |
| `payload.time`     | `string` | 消息时间戳                                   |
| `payload.metaData` | `object` | 消息元数据，包含额外的信息                           |
| `reqId`            | `string` | 请求唯一标识                                  |

### 9.4 前端处理示例

前端可以通过监听 "mqtt" 事件来处理桥接的消息。以下是处理 IEC 104 协议数据的示例（基于 [`iec104-protocol.md`](iec104-protocol.md) 定义）：

```javascript
// 监听 MQTT 桥接消息
socket.on('mqtt', (data) => {
    console.log('收到 MQTT 桥接消息:', data);

    // 处理 IEC 104 协议数据（示例，基于 iec104-protocol.md 定义）
    const payload = data.payload;
    if (payload.asdu === 'M_SP_NA_1') {
        // 处理单点信息
        const body = payload.body;
        console.log(`设备 ${payload.host}:${payload.port} 状态变化: 点号=${body.ioa}, 值=${body.value}, 时间=${payload.time}`);
    }
});
```

### 9.5 注意事项

1. **通用格式**：无论原始协议是什么，桥接后的消息都会遵循通用的 `event`、`payload`、`reqId` 格式，便于前端统一处理。
2. **Topic 与 Room 映射**：不同的 MQTT topic 会映射到不同的 SocketIO room，确保消息准确路由。
3. **Payload 结构**：不同协议的 `payload` 结构会有所不同，前端需要根据具体协议类型进行处理。
4. **事件名称**：桥接消息的事件名称固定为 "mqtt"，前端需要通过这个事件名来监听。
5. **数据类型**：前端需要注意 payload 中不同字段的数据类型，例如 port 是数字类型，而 host 是字符串类型。
6. **时间格式**：时间戳字段的格式可能因原始协议不同而有所差异，前端需要进行适当的处理。

## 10. 版本历史

| 版本  | 日期         | 说明                                        |
|-----|------------|-------------------------------------------|
| 2.1 | 2026-02-11 | 添加 MQTT 桥接指导                              |
| 2.0 | 2026-02-11 | 重写版本，支持SocketIO原生auth鉴权，payload使用object类型 |
| 1.0 | 2025-12-30 | 初始版本                                      |