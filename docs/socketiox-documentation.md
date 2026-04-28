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

### 1.2 服务端配置参考

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

### 5.5 StatDown（服务器统计信息）

| 字段名        | 类型                  | 描述        |
|------------|---------------------|-----------|
| `sId`      | `string`            | 会话ID      |
| `rooms`    | `[]string`          | 当前加入的房间列表 |
| `nps`      | `string`            | 命名空间      |
| `metadata` | `map[string]string` | 会话元数据     |
| `roomLoadError` | `string`        | 房间加载错误信息，为空表示加载成功 |

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
  },
  "roomLoadError": ""
}
```

**错误示例**：

```json
{
  "sId": "3fcd4875-cbfb-4d93-85b6-6f00540e9d45",
  "rooms": [],
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
    console.log('收到聊天消息:', data);
});

// 监听订单状态变化
socket.on('order_status_change', (data) => {
    console.log('订单状态变化:', data);
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
socket.on('__down__', (response) => {
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
    console.log('收到统计信息:', data);
    
    // 检查房间加载错误
    if (data.roomLoadError) {
        console.error('房间加载失败:', data.roomLoadError);
        
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
        console.log('房间加载成功，当前房间:', data.rooms);
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
    - `payload`使用JSON对象格式，无需转换为字符串
    - 支持各种数据类型，包括字符串、数字、布尔值、数组、对象等
4. **鉴权方式**：
    - 使用SocketIO原生的`auth`选项传递令牌，不要使用其他方式

## 11. MQTT 桥接指导

### 11.1 概述

SocketIO 消息网关支持通过 MQTT 协议桥接其他系统的消息，例如 IEC 104 协议的工业设备数据。桥接后，这些消息会以统一的格式通过
SocketIO 推送给前端客户端。不同的 MQTT topic 会映射到不同的 SocketIO room，确保消息准确路由到对应的客户端。

### 11.2 桥接消息格式

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
    console.log('收到IEC104设备消息:', data);
});

// 监听告警消息
socket.on('alarm', (data) => {
    console.log('收到告警消息:', data);
});

// 监听设备状态消息
socket.on('deviceStatus', (data) => {
    console.log('收到设备状态消息:', data);
});

// 监听心跳消息
socket.on('heartbeat', (data) => {
    console.log('收到心跳消息:', data);
});

// 监听默认MQTT消息（未匹配到映射规则的消息）
socket.on('mqtt', (data) => {
    console.log('收到其他MQTT消息:', data);
});
```

#### 11.6.3 通配符支持

事件映射配置支持 MQTT 主题通配符：
- `+`：匹配单个主题层级
- `#`：匹配多个主题层级（必须放在主题末尾）

例如：
- `device/+/status` 可以匹配 `device/1/status`、`device/2/status` 等
- `iec104/#` 可以匹配 `iec104/device1`、`iec104/device2/data` 等

## 12. 后端推送 API 参考

后端服务通过 gRPC 调用 socketpush 向前端推送消息，以下为核心接口：

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

## 13. 版本历史

| 版本  | 日期         | 说明                                        |
|-----|------------|-------------------------------------------|
| 2.5 | 2026-03-19 | 添加架构概览、服务端配置参考、GenToken 接口说明、后端推送 API 参考 |
| 2.4 | 2026-03-06 | 添加房间加载错误处理机制，在 `__stat_down__` 事件中包含 `roomLoadError` 字段 |
| 2.3 | 2026-03-03 | 添加MQTT事件映射配置，支持自定义事件名称和通配符匹配 |
| 2.2 | 2026-03-03 | 明确__down__事件为异步响应事件，添加非ack模式处理章节 |
| 2.1 | 2026-02-11 | 添加 MQTT 桥接指导                              |
| 2.0 | 2026-02-11 | 重写版本，支持SocketIO原生auth鉴权，payload使用object类型 |
| 1.0 | 2025-12-30 | 初始版本                                      |