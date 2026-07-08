# ISP Agent gRPC 服务

## Goal

基于 gnetx 开发长连接 TCP Client gRPC 服务 `ispagent`，对接 Java 侧区域型变电站远程智能巡视协议服务。ISP 表示 Inspection Substation Protocol，避免与视频/语音领域 SIP 混淆。

gRPC 接收指令 → TCP 长连接发送到 Java 协议服务 → 同步等待响应 → 解析后返回给上游。当前目标是先做稳定可测的协议 client，后续可作为巡视主机服务或机器狗 mock 的协议基础。

## Confirmed Facts

### 协议层
- **传输**: TCP，端口 7100
- **分帧**: 0xEB90 分隔符
- **帧格式** (二进制头 23B + XML 体):
  - StartFlag: 2B 大端 0xEB90
  - TransmitSeq (sendSerialNo): 8B 小端
  - ReceiveSeq (receiveSerialNo): 8B 小端
  - SessionSource: 1B (0x00=客户端, 0x01=服务端)
  - XMLLength: 4B 小端
  - XMLContent: UTF-8 变长
  - EndFlag: 2B 大端 0xEB90
- **消息标识**: messageId = (Type << 16) | Command
- **请求响应关联**: TransmitSeq/ReceiveSeq 配对
- **消息根元素**: 支持按配置切换 `PatrolHost` / `PatrolDevice`，与 Java 侧按上级系统属性切换保持一致
- **注册流程**: 251-1 → 251-4 (含心跳间隔)
- **心跳**: 251-2，周期性

### 技术栈
- TCP: `common/gnetx` (Client, LengthPrefixCodec/Codec, Serializer, Router, ReplyPool)
- gRPC: go-zero zrpc
- 参考: Java `allcore-sip` (Netty) + 用户 gnet demo；协议细节以 Java 实现为准

## Requirements

- **R1** TCP 长连接 + 自动重连 + 自动注册
- **R2** 协议编解码（0xEB90 分帧 + 二进制头 + XML）
- **R3** 注册 (251-1) + 心跳 (251-2) 自动管理
- **R4** gRPC 指令下发，同步等待 Java 协议服务响应并解析
- **R5** 消息路由（messageId 分发）
- **R6** sendCode、rootName 等本端身份/协议信息从配置读取；receiveCode 不固定配置，注册后以服务端响应为准

## gRPC 接口

### 通用
- **ExecuteCommand**: type + command + items → 解析后响应

### 特定业务（测试可靠性）
- **SendPatrolDeviceRunData**: 巡视设备运行数据 (Type=2, Command=0)
- **SendPatrolDeviceStatusData**: 巡视设备状态数据 (Type=1, Command=0)
- **SendPatrolDeviceCoordinates**: 巡视设备坐标 (Type=3, Command=0)

注：SendRegister/SendHeartbeat 为内部生命周期方法，不暴露为 gRPC 接口。连接建立后自动注册，心跳自动维护。

## Acceptance Criteria

- [ ] ExecuteCommand 下发指令到 Java 协议服务并同步获取解析后响应
- [ ] SendPatrolDeviceRunData/StatusData/Coordinates 三种指令可正确发送并获取响应
- [ ] 断线自动重连，重连后自动重新注册
- [ ] 心跳按服务端返回间隔正常发送
- [ ] `go test ./app/ispagent/... ./common/isp/...` 通过
- [ ] `go build` 编译通过
- [ ] 配置文件 `app/ispagent/etc/ispagent.yaml` 驱动 sendCode、rootName 和 Java 协议服务地址；receiveCode 由注册响应确定

## Design Decisions

| 决策 | 选择 | 原因 |
|------|------|------|
| 服务命名 | ispagent | ISP = Inspection Substation Protocol，避免 SIP 命名歧义 |
| 分帧方式 | gnetx LengthPrefixCodec + ISP Serializer | 利用 XMLLength 字段，避免 0xEB90 头尾分隔符导致空帧问题 |
| 响应方式 | 同步等待 | gnetx ReplyPool, gRPC 阻塞等回包 |
| sendCode | 配置文件固定 | 服务身份从 `ispagent.yaml` 读取 |
| receiveCode | 注册响应确定 | 对端标识不固定配置，取决于注册报文后的服务端响应 |
| rootName | 配置切换 `PatrolHost` / `PatrolDevice` | 与 Java 侧按上级系统属性切换保持一致 |
| gRPC 接口 | 通用 ExecuteCommand + 3 个特定业务接口 | 通用覆盖所有指令，特定接口验证可靠性 |
| 响应格式 | 解析后结构化数据 | Items 解析为 key-value, 含 code/type/command |

## Out of Scope

- 多 Java 协议服务连接（当前单连接）
- 设备模型文件同步（Type 11 模型文件）
- 任务管理（Type 41）
- Kafka/Redis 中间件集成
