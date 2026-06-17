# SocketIO 消息网关客户端对接文档

## 1. 概述

本文档用于指导前端浏览器客户端对接 SocketIO 消息网关服务，实现实时双向通信，包括默认事件和自定义事件，支持 ack 回调和事件返回机制。

### 1.1 架构概览

SocketIO 消息网关由两个服务组成：

| 服务 | 目录 | 职责 |
|------|------|------|
| **socketgtw** | `socketapp/socketgtw` | 网关服务 -- WebSocket 连接管理、房间管理、消息路由、Token 认证 |
| **socketpush** | `socketapp/socketpush` | 推送服务 -- Token 生成/验证、gRPC 推送接口（后端服务调用入口） |

**工作流程**：

```
前端客户端 ──WebSocket──> socketgtw ──gRPC──> StreamEvent（业务处理）
                                  <──gRPC──
后端服务 ──gRPC──> socketpush ──gRPC──> socketgtw ──WebSocket──> 前端客户端
```

- 前端通过 WebSocket 连接 socketgtw，发送/接收消息
- 后端通过 gRPC 调用 socketpush，向前端推送消息
- socketgtw 连接时通过 StreamEvent 服务加载用户初始房间列表

### 1.2 方向术语

本文档中的 `up` / `down` 均以云平台或服务端为方向锚点：

| 术语 | 通用含义 | SocketIO 场景 | DJI Cloud API 场景 |
|------|----------|---------------|--------------------|
| `up` / 上行 | 端侧 -> 云平台/服务端 | 浏览器客户端 -> socketgtw | DJI 设备 -> djicloud |
| `down` / 下行 | 云平台/服务端 -> 端侧 | socketgtw/socketpush -> 浏览器客户端 | djicloud -> DJI 设备 |

因此跨协议桥接时，同一条业务链路会在不同连接段使用不同方向名：

```text
设备状态到前端：DJI drc/up 或 events/osd/state -> djicloud -> SocketIO 自定义下行事件
前端控制设备：SocketIO __up__ 或业务接口 -> djicloud -> DJI services 或 drc/down
```

这不是方向定义冲突，而是端侧对象不同：SocketIO 的端侧是浏览器，DJI Cloud API 的端侧是 DJI 设备。

### 1.3 服务端配置参考

**socketgtw 配置** (socketgtw.yaml)：

```yaml
Name: socketgtw
ListenOn: 0.0.0.0:25007           # gRPC 监听地址
http:                              # WebSocket 服务
  Name: socketgtw-wss
  Host: 0.0.0.0
  Port: 11003                      # 前端连接端口
JwtAuth:
  AccessSecret: your-secret
  AccessExpire: 31536000           # Token 过期时间（秒）
SocketMetaData:                    # 从 Token 声明中提取的元数据字段
  - userId
  - deviceId
StreamEventConf:                   # 业务处理服务
  Endpoints:
    - 127.0.0.1:21009
```

**socketpush 配置** (socketpush.yaml)：

```yaml
Name: socketpush.rpc
ListenOn: 0.0.0.0:25008           # gRPC 监听地址
JwtAuth:
  AccessSecret: your-secret
  PrevAccessSecret: ""             # 前一个密钥（密钥轮换时使用）
  AccessExpire: 31536000
SocketGtwConf:                     # socketgtw 连接配置
  Endpoints:
    - 127.0.0.1:25007
```

> `SocketMetaData` 配置决定了哪些 Token 声明字段会被提取为会话元数据，可用于 `SendToMetaSession` / `KickMetaSession` 等按元数据寻址的操作。

## 2. 快速开始

### 2.1 客户端库要求

- **推荐版本**：`socket.io-client@4.x`
- **官网文档**：[https://socket.io/zh-CN/docs/v4/](https://socket.io/zh-CN/docs/v4/)

### 2.2 基本连接示例

```javascript
function normalizeSocketPayload(data) {
    if (typeof data === 'string') {
        try {
            return JSON.parse(data);
        } catch (e) {
            return data;
        }
    }
    return data;
}

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

// 监听非 ack 模式下的请求响应
socket.on('__down__', (data) => {
    console.log('收到服务器响应:', normalizeSocketPayload(data));
});
```

## 3. 鉴权配置

### 3.1 鉴权概述

当SocketIO网关开启鉴权后，客户端必须携带有效的令牌才能建立连接。服务器只支持SocketIO原生的`auth`方式传递令牌。

### 3.2 获取令牌

客户端需要通过后端服务获取有效的令牌。后端调用 socketpush 的 `GenToken` gRPC 接口生成令牌：

```protobuf
// socketpush.proto
rpc GenToken(GenTokenReq) returns (GenTokenRes);

message GenTokenReq {
  string uid = 1;                        // 用户标识
  map<string, string> payload = 2;       // 自定义声明（会成为会话元数据）
}

message GenTokenRes {
  string accessToken = 1;
  int64 accessExpire = 2;
  int64 refreshAfter = 3;
}
```

获取方式：
- **后端传递**（推荐）：后端服务调用 `GenToken` 生成令牌，通过 HTTP 接口返回给前端
- **前端直接获取**：通过 BFF 网关封装的认证 API 获取

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
| `__up__`                  | 浏览器上行消息（浏览器 -> socketgtw） | `SocketUpReq`     |
| `__join_room_up__`        | 浏览器请求加入房间 | `SocketUpRoomReq` |
| `__leave_room_up__`       | 浏览器请求离开房间 | `SocketUpRoomReq` |
| `__rooms_page_up__`       | 浏览器分页查询当前会话已加入房间 | `SocketRoomsPageReq` |
| `__room_broadcast_up__`   | 浏览器请求房间广播 | `SocketUpReq`     |
| `__global_broadcast_up__` | 浏览器请求全局广播 | `SocketUpReq`     |

### 4.2 服务器推送事件

| 事件名称            | 描述      | 数据格式         |
|-----------------|---------|--------------|
| `__down__`      | 非 ack 模式下的服务器异步响应（socketgtw -> 浏览器） | `SocketResp` |
| `__stat_down__` | 统计信息下行推送 | `StatDown`   |
| 自定义事件           | 服务器主动业务推送 | `SocketDown` |

`__down__` 是系统保留事件，主要用于客户端请求的异步响应。后端主动推送业务通知时，推荐使用自定义事件名（如 `mqtt`、`alarm`、`drc:heart_beat`），数据结构仍使用 `SocketDown`。

> 当前服务端实现会以 JSON 字符串形式发送 `__down__`、`__stat_down__` 和自定义事件的消息体。前端可按需使用 `JSON.parse` 或统一封装解析函数将其转换为对象。

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

### 5.2 SocketUpRoomReq（客户端房间操作请求）

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

### 5.4 SocketDown（服务器自定义事件推送）

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

### 5.5 SocketRoomsPageReq（当前会话房间分页查询）

客户端通过 `__rooms_page_up__` 查询当前 socket session 已加入的房间。该事件只返回当前连接自己的业务房间，不支持传入其他 `socketId`，且会过滤 SocketIO 自动加入的 socketId 内部房间。

| 字段名       | 类型       | 必填 | 描述              |
|-----------|----------|----|-----------------|
| `reqId`   | `string` | ✅  | 请求唯一标识         |
| `page`    | `number` | ❌  | 页码，从 1 开始，默认 1 |
| `pageSize` | `number` | ❌ | 每页数量，默认 50，最大 200 |

**请求示例**：

```json
{
  "reqId": "uuid-rooms-page",
  "page": 1,
  "pageSize": 50
}
```

**响应 payload 示例**：

```json
{
  "total": 128,
  "page": 1,
  "pageSize": 50,
  "totalPages": 3,
  "rooms": [
    "alarm:dept:001",
    "device:dept:001"
  ]
}
```

服务端会先按房间名排序再分页，保证翻页结果稳定。

**前端示例**：

```javascript
socket.emit('__rooms_page_up__', {
    reqId: crypto.randomUUID(),
    page: 1,
    pageSize: 50
}, (ack) => {
    const resp = normalizeSocketPayload(ack);
    console.log('当前会话房间分页:', resp.payload);
});
```

### 5.6 StatDown（服务器统计信息）

| 字段名        | 类型                  | 描述        |
|------------|---------------------|-----------|
| `socketId` | `string`            | 会话ID      |
| `roomCount` | `number`           | 当前加入的房间数量 |
| `rooms`    | `[]string`          | 当前加入的房间列表样本，最多返回 50 个 |
| `nps`      | `string`            | 命名空间      |
| `metadata` | `map[string]string` | 会话元数据     |
| `roomLoadError` | `string`        | 房间加载错误信息，为空表示加载成功 |

**示例**：

```json
{
  "socketId": "socket-123",
  "roomCount": 128,
  "rooms": [
    "room1",
    "room2"
  ],
  "nps": "/",
  "metadata": {
    "userId": "user123",
    "deviceId": "device456"
  },
  "roomLoadError": ""
}
```

**错误示例**：

```json
{
  "socketId": "3fcd4875-cbfb-4d93-85b6-6f00540e9d45",
  "roomCount": 0,
  "nps": "/",
  "roomLoadError": "rpc error: code = Unavailable desc = last connection error: connection error: desc = \"transport: Error while dialing: dial tcp 127.0.0.1:21009: connect: connection refused\""
}
```

## 6. 事件返回机制

服务器支持两种消息推送方式：

### 6.1 响应式返回（`__down__`事件）

- **触发条件**：客户端发送请求后，服务器异步响应
- **数据格式**：`SocketResp`
- **特点**：包含状态码、消息和业务数据，通过事件方式返回
- **用途**：用于请求-响应模式的交互，特别是在客户端非ack模式下
- **异步特性**：服务器会在后台异步处理请求，处理完成后通过`__down__`事件推送响应结果

### 6.2 事件式返回（自定义事件）

- **触发条件**：服务器主动推送或异步响应
- **数据格式**：`SocketDown`
- **特点**：包含事件名称和业务数据
- **用途**：用于实时通知、推送和事件驱动场景

**示例**：

```javascript
// 监听自定义事件
socket.on('chat_message', (data) => {
    const message = normalizeSocketPayload(data);
    console.log('收到聊天消息:', message);
});

// 监听订单状态变化
socket.on('order_status_change', (data) => {
    const event = normalizeSocketPayload(data);
    console.log('订单状态变化:', event);
});
```

## 7. 非ack模式处理

### 7.1 模式说明

当客户端使用非ack模式发送请求时，服务器会采用以下处理流程：

1. **接收请求**：服务器接收到客户端发送的事件（如`__up__`、`__join_room_up__`等）
2. **异步处理**：服务器在后台异步处理请求，不阻塞主事件循环
3. **事件响应**：处理完成后，服务器通过`__down__`事件推送响应结果给客户端
4. **客户端监听**：客户端需要监听`__down__`事件来接收服务器的响应

### 7.2 客户端实现示例

```javascript
// 非ack模式发送请求
socket.emit('__up__', {
    event: 'custom_event',
    payload: {
        // 业务数据
    },
    reqId: 'unique-request-id' // 唯一请求标识
});

// 监听__down__事件获取响应
socket.on('__down__', (data) => {
    const response = normalizeSocketPayload(data);
    console.log('收到服务器响应:', response);
    
    // 根据reqId匹配请求和响应
    if (response.reqId === 'unique-request-id') {
        if (response.code === 200) {
            // 处理成功响应
            console.log('请求处理成功:', response.payload);
        } else {
            // 处理错误响应
            console.error('请求处理失败:', response.msg);
        }
    }
});
```

### 7.3 注意事项

- **请求标识**：每次请求必须生成唯一的`reqId`，以便在`__down__`事件中匹配响应
- **事件监听**：客户端必须持续监听`__down__`事件，才能接收到服务器的异步响应
- **响应格式**：`__down__`事件的响应格式为`SocketResp`，包含状态码、消息和业务数据
- **错误处理**：客户端需要根据响应中的`code`字段判断请求是否成功

## 8. 房间加载错误处理

### 8.1 错误检测

客户端可以通过监听`__stat_down__`事件来检测房间加载错误：

```javascript
// 监听 __stat_down__ 事件
socket.on('__stat_down__', (data) => {
    const stat = normalizeSocketPayload(data);
    console.log('收到统计信息:', stat);
    
    // 检查房间加载错误
    if (stat.roomLoadError) {
        console.error('房间加载失败:', stat.roomLoadError);
        
        // 根据业务需求决定处理方式
        // 方式1：断联重连
        socket.disconnect();
        setTimeout(() => {
            socket.connect();
        }, 1000);
        
        // 方式2：显示错误提示
        // showErrorMessage('房间加载失败，请刷新页面重试');
    } else {
        // 房间加载成功，处理其他逻辑
        console.log('房间加载成功，当前房间:', stat.rooms);
    }
});
```

### 8.2 错误处理建议

1. **断联重连**：如果房间加载失败，客户端可以选择断联后重新连接，重新尝试加载房间
2. **错误提示**：显示错误提示给用户，告知房间加载失败的原因

## 9. 错误码体系

| 错误码 | 描述   | 处理建议                  |
|-----|------|-----------------------|
| 200 | 成功   | 正常处理响应数据              |
| 400 | 参数错误 | 检查请求参数格式和必填字段         |
| 500 | 业务错误 | 查看msg字段获取详细错误信息，必要时重试 |

## 10. 最佳实践

1. **请求ID管理**：每次请求必须生成唯一的`reqId`，建议使用UUID
2. **事件命名规范**：
    - 系统事件使用`__`前缀，如`__up__`、`__down__`
    - 自定义事件使用驼峰命名，如`chatMessage`、`orderStatusChange`
    - 避免使用特殊字符和空格
3. **数据格式**：
    - 客户端发送 `payload` 时使用 JSON 对象格式，无需转换为字符串
    - 服务端当前下行消息体以 JSON 字符串发送，客户端监听后建议统一解析为对象再处理
    - 支持各种数据类型，包括字符串、数字、布尔值、数组、对象等
4. **鉴权方式**：
    - 使用SocketIO原生的`auth`选项传递令牌，不要使用其他方式

## 11. MQTT 桥接指导

### 11.1 概述

SocketIO 消息网关支持通过 MQTT 协议桥接其他系统的消息，例如 IEC 104 协议的工业设备数据。桥接后，这些消息会以统一的格式通过
SocketIO 推送给前端客户端。不同的 MQTT topic 会映射到不同的 SocketIO room，确保消息准确路由到对应的客户端。

### 11.2 桥接消息格式

桥接的 MQTT 消息会转换为统一的格式，遵循通用的 `event`、`payload`、`reqId` 结构。以下示例基于 IEC 104 协议桥接，详细协议定义请参考 [`iec104-message.md`](iec104-message.md) 文件：

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

### 11.3 字段说明

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

### 11.4 前端处理示例

前端可以通过监听 "mqtt" 事件来处理桥接的消息。以下是处理 IEC 104 协议数据的示例（基于 [`iec104-message.md`](iec104-message.md) 定义）：

```javascript
// 监听 MQTT 桥接消息
socket.on('mqtt', (data) => {
    const message = normalizeSocketPayload(data);
    console.log('收到 MQTT 桥接消息:', message);

    // 处理 IEC 104 协议数据（示例，基于 iec104-message.md 定义）
    const payload = message.payload;
    if (payload.asdu === 'M_SP_NA_1') {
        // 处理单点信息
        const body = payload.body;
        console.log(`设备 ${payload.host}:${payload.port} 状态变化: 点号=${body.ioa}, 值=${body.value}, 时间=${payload.time}`);
    }
});
```

### 11.5 注意事项

1. **通用格式**：无论原始协议是什么，桥接后的消息都会遵循通用的 `event`、`payload`、`reqId` 格式，便于前端统一处理。
2. **Topic 与 Room 映射**：不同的 MQTT topic 会映射到不同的 SocketIO room，确保消息准确路由。具体来说，MQTT 的 topic 模板会直接作为 SocketIO 的 room 名称。
3. **Payload 结构**：不同协议的 `payload` 结构会有所不同，前端需要根据具体协议类型进行处理。
4. **事件名称**：默认情况下，桥接消息的事件名称固定为 "mqtt"，但可以通过配置文件自定义事件映射。
5. **数据类型**：前端需要注意 payload 中不同字段的数据类型，例如 port 是数字类型，而 host 是字符串类型。
6. **时间格式**：时间戳字段的格式可能因原始协议不同而有所差异，前端需要进行适当的处理。
7. **前端区分不同 room**：前端可以通过以下方式区分不同的 MQTT topic：
   - 首先，前端需要加入对应的 room（topic）
   - 然后，在监听相应的事件时，通过消息中的原始 topic 信息来区分不同的 MQTT 主题
   - 或者，根据业务需求，为不同的 room 设置不同的处理逻辑

### 11.6 事件映射配置

#### 11.6.1 配置示例

在 bridgemqtt 配置文件中，可以通过顶层 `EventMapping` 配置项来定义 MQTT 主题模板到 SocketIO 事件的映射规则。`EventMapping` 和 `DefaultEvent` 属于 bridgemqtt 的 SocketIO 推送配置，不属于通用 `MqttConfig`：

```yaml
MqttConfig:
  Broker:
    - "tcp://localhost:1883"
  SubscribeTopics:
    - "iec104/#"
    - "alarm/#"
    - "device/+/status"
    - "heartbeat/#"

EventMapping:
  - topicTemplate: "iec104/#"
    event: "iec104"
  - topicTemplate: "alarm/#"
    event: "alarm"
  - topicTemplate: "device/+/status"
    event: "deviceStatus"
  - topicTemplate: "heartbeat/#"
    event: "heartbeat"
DefaultEvent: "mqtt"
```

#### 11.6.2 前端处理示例

前端可以根据配置的事件名称来监听不同类型的消息：

```javascript
// 监听IEC104设备消息
socket.on('iec104', (data) => {
    console.log('收到IEC104设备消息:', normalizeSocketPayload(data));
});

// 监听告警消息
socket.on('alarm', (data) => {
    console.log('收到告警消息:', normalizeSocketPayload(data));
});

// 监听设备状态消息
socket.on('deviceStatus', (data) => {
    console.log('收到设备状态消息:', normalizeSocketPayload(data));
});

// 监听心跳消息
socket.on('heartbeat', (data) => {
    console.log('收到心跳消息:', normalizeSocketPayload(data));
});

// 监听默认MQTT消息（未匹配到映射规则的消息）
socket.on('mqtt', (data) => {
    console.log('收到其他MQTT消息:', normalizeSocketPayload(data));
});
```

#### 11.6.3 通配符支持

事件映射配置支持 MQTT 主题通配符：
- `+`：匹配单个主题层级
- `#`：匹配多个主题层级（必须放在主题末尾）

例如：
- `device/+/status` 可以匹配 `device/1/status`、`device/2/status` 等
- `iec104/#` 可以匹配 `iec104/device1`、`iec104/device2/data` 等

## 12. DRC 远程控制对接指导

### 12.1 概述

大疆云端服务通过 DRC（Direct Remote Control）通道实现对设备的远程操控。DRC 会话的生命周期事件通过 SocketIO 推送到前端，前端可根据这些事件实时感知 DRC 模式的开启、关闭和异常过期状态，同步 UI 状态（如 DRC 操控按钮的启停）。

本章节同时涉及 DJI Cloud API 和 SocketIO 两套连接方向：DJI 的 `drc/up` 表示 DJI 设备上行到云平台，SocketIO 的 DRC 自定义事件表示云平台下行推送到浏览器。因此设备心跳到达前端的完整链路是 `DJI drc/up -> djicloud -> SocketIO drc:heart_beat`。

### 12.2 房间规则

所有 DRC 相关事件共用一个房间，房间命名规则：

| 房间名格式 | 说明 |
|-----------|------|
| `drc:heartbeat:{gatewaySn}` | 每个网关设备独立房间，`gatewaySn` 为设备序列号 |

前端在进入 DRC 操控页面时，通过 `__join_room_up__` 事件加入对应设备的房间；退出时通过 `__leave_room_up__` 离开。

### 12.3 事件列表

| 事件名 | 触发时机 | 方向 | 说明 |
|--------|---------|------|------|
| `drc:heart_beat` | 设备心跳上行 | DJI 设备 -> 云 -> 浏览器 | DRC 通道存活心跳，设备经 `drc/up` 周期上报，云平台再通过 SocketIO 下推给前端 |
| `drc:session_enabled` | DRC 模式启用 | 云 -> 浏览器 | gRPC 调用 `DrcModeEnter` 成功后推送，前端可开启 DRC 操控 UI |
| `drc:session_disabled` | DRC 模式停用 | 云 -> 浏览器 | gRPC 调用 `DrcModeExit` 成功后推送，前端应关闭 DRC 操控按钮 |
| `drc:session_expired` | DRC 会话自动过期 | 云 -> 浏览器 | 会话因 MaxDeadline 到期、设备心跳超时、cleanLoop 孤儿清理等非主动原因被清除时推送，前端应关闭 DRC 操控按钮 |

### 12.4 数据结构

#### 12.4.1 drc:heart_beat

设备心跳上行数据，Payload 为 DJI DRC 协议的 `heart_beat` 上行原始 JSON。

| 字段名 | 类型 | 描述 |
|--------|------|------|
| `gateway_sn` | `string` | 网关设备序列号（从房间名推断） |
| 原始字段 | - | DJI DRC 协议心跳上行数据 |

**示例**：
```json
{
  "event": "drc:heart_beat",
  "payload": {
    "result": 0,
    "timestamp": 1715234567890,
    "gateway_sn": "4TADLBC00100XXXX"
  },
  "reqId": "a1b2c3d4e5f67890abcdef1234567890"
}
```

#### 12.4.2 drc:session_enabled

DRC 模式启用通知。

| 字段名 | 类型 | 描述 |
|--------|------|------|
| `gateway_sn` | `string` | 网关设备序列号 |
| `session_id` | `string` | DRC 会话唯一标识（UUID） |

**示例**：
```json
{
  "event": "drc:session_enabled",
  "payload": {
    "gateway_sn": "4TADLBC00100XXXX",
    "session_id": "61a33d62eb214810850da55e9dde980f"
  },
  "reqId": "b2c3d4e5f6a78901bcdef12345678901"
}
```

#### 12.4.3 drc:session_disabled

DRC 模式停用通知（主动退出）。

| 字段名 | 类型 | 描述 |
|--------|------|------|
| `gateway_sn` | `string` | 网关设备序列号 |
| `session_id` | `string` | DRC 会话唯一标识 |

**示例**：
```json
{
  "event": "drc:session_disabled",
  "payload": {
    "gateway_sn": "4TADLBC00100XXXX",
    "session_id": "61a33d62eb214810850da55e9dde980f"
  },
  "reqId": "c3d4e5f6a7b89012cdef123456789012"
}
```

#### 12.4.4 drc:session_expired

DRC 会话自动过期通知（非主动退出）。

| 字段名 | 类型 | 描述 |
|--------|------|------|
| `gateway_sn` | `string` | 网关设备序列号 |
| `session_id` | `string` | DRC 会话唯一标识（孤儿清理时可能为空） |
| `reason` | `string` | 过期原因：`max_deadline_exceeded`（MaxDeadline 到期）或 `heartbeat_timeout`（心跳超时） |

**示例**：
```json
{
  "event": "drc:session_expired",
  "payload": {
    "gateway_sn": "4TADLBC00100XXXX",
    "session_id": "61a33d62eb214810850da55e9dde980f",
    "reason": "heartbeat_timeout"
  },
  "reqId": "d4e5f6a7b8c90123defa234567890123"
}
```

### 12.5 前端对接示例

```javascript
// 加入 DRC 房间（进入操控页面时）
function joinDrcRoom(gatewaySn) {
  socket.emit('__join_room_up__', {
    reqId: crypto.randomUUID(),
    room: `drc:heartbeat:${gatewaySn}`
  });
}

// 离开 DRC 房间（退出操控页面时）
function leaveDrcRoom(gatewaySn) {
  socket.emit('__leave_room_up__', {
    reqId: crypto.randomUUID(),
    room: `drc:heartbeat:${gatewaySn}`
  });
}

// 监听 DRC 模式启用 — 开启操控 UI
socket.on('drc:session_enabled', (data) => {
  const message = normalizeSocketPayload(data);
  console.log('DRC 模式已启用:', message.payload.gateway_sn, message.payload.session_id);
  // 开启 DRC 操控按钮/面板
  enableDrcControls(message.payload.gateway_sn);
});

// 监听 DRC 模式停用 — 关闭操控 UI（主动退出）
socket.on('drc:session_disabled', (data) => {
  const message = normalizeSocketPayload(data);
  console.log('DRC 模式已停用:', message.payload.gateway_sn, message.payload.session_id);
  // 关闭 DRC 操控按钮/面板
  disableDrcControls(message.payload.gateway_sn);
});

// 监听 DRC 会话过期 — 关闭操控 UI（异常过期）
socket.on('drc:session_expired', (data) => {
  const message = normalizeSocketPayload(data);
  console.warn('DRC 会话已过期:', message.payload.gateway_sn, message.payload.reason);
  // 关闭 DRC 操控按钮/面板，提示用户会话已过期
  disableDrcControls(message.payload.gateway_sn);
  showDrcExpiredWarning(message.payload.reason);
});

// 监听设备心跳 — 更新存活状态
socket.on('drc:heart_beat', (data) => {
  const message = normalizeSocketPayload(data);
  // 心跳持续到达表示 DRC 通道正常
  updateDrcAliveStatus(message.payload.gateway_sn);
});
```

### 12.6 注意事项

1. **房间生命周期**：前端应在进入 DRC 操控页面时加入房间，退出时离开。未加入房间则无法收到任何 DRC 事件。
2. **session_disabled 与 session_expired 的区别**：`session_disabled` 是用户主动退出 DRC 模式（通过 gRPC 调用 `DrcModeExit`）；`session_expired` 是会话因超时等非主动原因被自动清除。前端对两者的典型处理都是关闭 DRC 按钮，但 expired 场景可能需要额外提示用户。
3. **reason 字段**：`session_expired` 的 `reason` 取值：
   - `max_deadline_exceeded`：DRC 会话达到最大允许时长（由后端 WithMaxTimeout 配置），强制清除
   - `heartbeat_timeout`：设备心跳超时（设备与云端 DRC 通道断开），缓存 TTL 驱逐
4. **心跳事件频率**：`drc:heart_beat` 事件频率较高（默认 2 秒间隔），前端监听时应避免在每次收到时执行重渲染，建议仅用于存活状态更新。
5. **事件顺序**：`drc:session_enabled` 一定在 `drc:heart_beat` 之前到达；`drc:session_disabled` 或 `drc:session_expired` 到达后不会再有 `drc:heart_beat`。
6. **方向转换**：前端控制设备时通常是 `SocketIO __up__ 或业务接口 -> djicloud -> DJI services/drc/down`；设备状态展示到前端时通常是 `DJI events/osd/state/drc/up -> djicloud -> SocketIO 自定义下行事件`。

## 13. 设备遥测数据推送

### 13.1 概述

设备遥测数据（OSD 和 State）在写入数据库的同时，会通过 SocketIO 实时推送到前端。前端可以通过加入对应的房间来订阅特定设备的遥测数据流。

### 13.2 房间规则

| 房间名格式 | 说明 |
|-----------|------|
| `thing/product/{deviceSn}/osd` | 设备 OSD 遥测数据，`deviceSn` 为设备序列号 |
| `thing/product/{deviceSn}/state` | 设备 State 状态数据，`deviceSn` 为设备序列号 |

房间命名与 MQTT topic 格式保持一致。前端通过 `__join_room_up__` 事件加入对应设备的房间，退出时通过 `__leave_room_up__` 离开。

### 13.3 事件列表

| 事件名 | 触发时机 | 方向 | 说明 |
|--------|---------|------|------|
| `telemetry:osd` | OSD 数据上报（0.5HZ） | 设备 → 云 → 浏览器 | 设备遥测数据（位置、电量、速度等） |
| `telemetry:state` | State 数据上报（状态变化时） | 设备 → 云 → 浏览器 | 设备状态变更（固件版本、硬件版本等） |

### 13.4 数据结构

#### 13.4.1 telemetry:osd

设备 OSD 遥测数据，Payload 为 DJI OSD 协议的原始 JSON。

| 字段名 | 类型 | 描述 |
|--------|------|------|
| 原始字段 | - | DJI OSD 协议数据（位置、电量、速度、高度等） |

**示例**：
```json
{
  "event": "telemetry:osd",
  "payload": {
    "latitude": 31.2304,
    "longitude": 121.4737,
    "altitude": 100.5,
    "battery": 85,
    "speed": 5.2,
    "heading": 180
  },
  "reqId": "a1b2c3d4e5f67890abcdef1234567890"
}
```

#### 13.4.2 telemetry:state

设备 State 状态数据，Payload 为 DJI State 协议的原始 JSON。

| 字段名 | 类型 | 描述 |
|--------|------|------|
| 原始字段 | - | DJI State 协议数据（固件版本、硬件版本等） |

**示例**：
```json
{
  "event": "telemetry:state",
  "payload": {
    "firmware_version": "v01.02.0300",
    "hardware_version": "v01.00.0000",
    "device_sn": "4TADLBC00100XXXX"
  },
  "reqId": "b2c3d4e5f6a78901bcdef12345678901"
}
```

### 13.5 前端对接示例

```javascript
// 加入设备 OSD 房间（进入设备监控页面时）
function joinOsdRoom(deviceSn) {
  socket.emit('__join_room_up__', {
    reqId: crypto.randomUUID(),
    room: `thing/product/${deviceSn}/osd`
  });
}

// 离开设备 OSD 房间（退出设备监控页面时）
function leaveOsdRoom(deviceSn) {
  socket.emit('__leave_room_up__', {
    reqId: crypto.randomUUID(),
    room: `thing/product/${deviceSn}/osd`
  });
}

// 加入设备 State 房间
function joinStateRoom(deviceSn) {
  socket.emit('__join_room_up__', {
    reqId: crypto.randomUUID(),
    room: `thing/product/${deviceSn}/state`
  });
}

// 监听设备 OSD 数据
socket.on('telemetry:osd', (data) => {
  const message = normalizeSocketPayload(data);
  console.log('收到设备 OSD 数据:', message.payload);
  // 更新设备位置、电量、速度等 UI
  updateDeviceOsdDisplay(message.payload);
});

// 监听设备 State 数据
socket.on('telemetry:state', (data) => {
  const message = normalizeSocketPayload(data);
  console.log('收到设备 State 数据:', message.payload);
  // 更新设备固件版本、硬件版本等 UI
  updateDeviceStateDisplay(message.payload);
});
```

### 13.6 注意事项

1. **房间生命周期**：前端应在进入设备监控页面时加入房间，退出时离开。未加入房间则无法收到任何遥测数据。
2. **数据频率**：OSD 数据频率为 0.5HZ（每 2 秒一次），State 数据在状态变化时上报，前端监听时应避免在每次收到时执行重渲染，建议仅用于数据展示更新。
3. **房间命名**：房间名格式与 MQTT topic 一致：`thing/product/{deviceSn}/osd` 和 `thing/product/{deviceSn}/state`。
4. **数据格式**：Payload 为 DJI 协议的原始 JSON，前端需要根据具体字段进行解析和展示。
5. **方向说明**：遥测数据推送属于下行方向（设备 → 云 → 浏览器），与 DRC 心跳推送模式一致。

## 14. 后端推送 API 参考

后端服务通过 gRPC 调用 socketpush 向前端推送消息，以下为核心接口：

当前 socketpush 到 socketgtw 的推送为集群扇出模型：socketpush 会把广播、单播、按元数据推送等请求转发到已发现的 socketgtw 节点。RPC 返回成功表示推送请求已被后端服务接受并发起转发，不等同于浏览器客户端已经收到消息；业务若需要端到端送达确认，应在业务协议中额外设计客户端 ack。

| 方法 | 说明 |
|------|------|
| `GenToken` | 生成连接令牌 |
| `VerifyToken` | 验证令牌有效性 |
| `JoinRoom` / `LeaveRoom` | 服务端控制房间加入/离开 |
| `BroadcastRoom` | 向指定房间广播消息 |
| `BroadcastGlobal` | 全局广播消息 |
| `SendToSession` / `SendToSessions` | 按 Session ID 单播/批量推送 |
| `SendToMetaSession` / `SendToMetaSessions` | 按元数据（如 userId）寻址推送 |
| `KickSession` / `KickMetaSession` | 剔除会话 |
| `SocketGtwStat` | 获取网关统计信息 |

协议定义：[`socketpush.proto`](../socketapp/socketpush/socketpush.proto) | [`socketgtw.proto`](../socketapp/socketgtw/socketgtw.proto)

## 15. 版本历史

| 版本  | 日期         | 说明                                        |
|-----|------------|-------------------------------------------|
| 2.8 | 2026-06-17 | 添加设备遥测数据推送章节（房间规则、事件列表、数据结构、前端示例） |
| 2.7 | 2026-05-25 | 补充 up/down 方向锚点、DJI 与 SocketIO 桥接方向、下行解析和推送语义说明；修正 `StatDown.socketId` 字段名 |
| 2.6 | 2026-05-09 | 添加 DRC 远程控制对接指导（房间规则、事件列表、数据结构、前端示例） |
| 2.5 | 2026-03-19 | 添加架构概览、服务端配置参考、GenToken 接口说明、后端推送 API 参考 |
| 2.4 | 2026-03-06 | 添加房间加载错误处理机制，在 `__stat_down__` 事件中包含 `roomLoadError` 字段 |
| 2.3 | 2026-03-03 | 添加MQTT事件映射配置，支持自定义事件名称和通配符匹配 |
| 2.2 | 2026-03-03 | 明确__down__事件为异步响应事件，添加非ack模式处理章节 |
| 2.1 | 2026-02-11 | 添加 MQTT 桥接指导                              |
| 2.0 | 2026-02-11 | 重写版本，支持SocketIO原生auth鉴权，payload使用object类型 |
| 1.0 | 2025-12-30 | 初始版本                                      |
