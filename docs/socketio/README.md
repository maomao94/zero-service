# Socket.IO 实时通信

连接鉴权、事件、数据结构与后端推送的完整契约见[Socket.IO 对接文档](./socketio.md)；本页仅作服务入口与职责导航。

Socket.IO 网关提供浏览器连接管理、Token 鉴权、房间操作、消息路由和服务端推送。

## 快速入口

| 场景 | 入口 |
| --- | --- |
| 浏览器连接、Token、事件和房间 | [Socket.IO 对接文档](./socketio.md) |
| 后端按 Session、用户或设备推送 | [Socket.IO 对接文档：后端推送 API](./socketio.md#后端推送-api) |

## 服务职责

| 服务 | 目录 | 职责 |
| --- | --- | --- |
| `socketgtw` | `socketapp/socketgtw` | WebSocket/Socket.IO 连接、鉴权、Session、房间和事件处理 |
| `socketpush` | `socketapp/socketpush` | Token 生成/验证，以及向网关集群转发推送请求 |
