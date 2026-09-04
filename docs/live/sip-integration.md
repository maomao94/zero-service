# SIP 电话集成指南

## 概述

LiveKit SIP 集成允许浏览器用户（WebRTC）和电话用户（SIP）在同一个 Room 中通话。

## 架构总览

```
┌──────────────┐     ┌──────────────┐     ┌──────────────┐     ┌──────────────┐
│   浏览器     │     │  LiveKit     │     │  LiveKit     │     │  FreeSWITCH  │
│   (WebRTC)   │◄───►│  Server      │◄───►│  SIP Server  │◄───►│  (电话交换机) │
│              │     │              │     │              │     │              │
│  音视频通话   │     │  房间管理    │     │  协议转换    │     │  分机管理    │
│  Data/RPC    │     │  参与者管理  │     │  SIP↔WebRTC  │     │  电话振铃    │
└──────────────┘     └──────────────┘     └──────────────┘     └──────────────┘
     :7880                :7880                :5070                :5060
   (WebRTC)           (HTTP API)           (SIP 信令)           (SIP 信令)
```

## 各组件职责

| 组件 | 端口 | 职责 |
|------|------|------|
| **LiveKit Server** | 7880 | WebRTC 媒体服务器，管理房间和参与者，提供 HTTP API |
| **LiveKit SIP Server** | 5070 | SIP↔WebRTC 协议转换桥接，把电话转成 LiveKit 参与者 |
| **FreeSWITCH** | 5060 | SIP 电话交换机，管理分机注册、振铃、路由 |
| **Redis** | 36379 | LiveKit 内部通信总线 |

## 为什么需要三个组件

```
LiveKit SIP Server = 翻译官
  ✓ 发起/接收 SIP 呼叫
  ✓ SIP ↔ WebRTC 协议转换
  ✗ 管理 SIP 分机注册

FreeSWITCH = 电话簿 + 接线员
  ✓ 管理分机号（1000-1019）
  ✓ 接收 SIP 注册
  ✓ 路由电话（1001 打给 1002）
  ✓ 振铃、接听、挂断

LiveKit Server = 会议室管理员
  ✓ WebRTC 房间管理
  ✓ 参与者管理
  ✓ API 接口
```

没有 FreeSWITCH，SIP 电话没有地方注册，无法接收来电。
没有 LiveKit SIP Server，浏览器（WebRTC）和电话（SIP）协议不同，无法直接通信。

## FreeSWITCH vs SIP 供应商

```
FreeSWITCH = 公司内部电话总机（自建）
  • 你自己部署和维护
  • 管理内部分机（1000, 1001, 1002...）
  • 内部通话免费（分机之间互打）
  • 不能直接拨打外部电话（手机、固话）

SIP 供应商 = 电信运营商（外包）
  • 阿里云通信、腾讯云、天润融通等
  • 提供外部电话号码
  • 连接公共电话网络（PSTN）
  • 可以拨打/接听手机、固话
  • 按分钟收费
```

**生产环境架构**：

```
浏览器 ◄──► LiveKit ◄──► SIP Server ◄──► FreeSWITCH ◄──► SIP 供应商 ◄──► 手机/固话
                                    │              │              │
                                 翻译官        内部总机        电信运营商
```

FreeSWITCH 管理内部分机，SIP 供应商提供外部电话能力，两者配合使用。

## 通话流程

### 外呼流程（浏览器 → 电话）

```
浏览器用户 A                LiveKit Server            LiveKit SIP Server          FreeSWITCH            软电话
    │                          │                          │                          │                    │
    │ 1. 调用 API 拨号 1001    │                          │                          │                    │
    │ ────────────────────────►│                          │                          │                    │
    │                          │ 2. CreateSIPParticipant  │                          │                    │
    │                          │ ────────────────────────►│                          │                    │
    │                          │                          │ 3. SIP INVITE            │                    │
    │                          │                          │ ────────────────────────►│                    │
    │                          │                          │                          │ 4. 振铃             │
    │                          │                          │                          │ ──────────────────►│
    │                          │                          │                          │                    │
    │                          │                          │                          │ 5. 用户接听         │
    │                          │                          │                          │ ◄──────────────────│
    │                          │                          │                          │                    │
    │                          │ 6. SIP ↔ WebRTC 转换     │                          │                    │
    │                          │ ◄────────────────────────│                          │                    │
    │                          │                          │                          │                    │
    │ 7. 音频流双向传输         │                          │                          │                    │
    │ ◄───────────────────────►│◄────────────────────────►│◄────────────────────────►│◄──────────────────►│
```

### 来电流程（电话 → 浏览器）

```
软电话                    FreeSWITCH            LiveKit SIP Server          LiveKit Server            浏览器
  │                          │                          │                          │                    │
  │ 1. 拨打号码              │                          │                          │                    │
  │ ────────────────────────►│                          │                          │                    │
  │                          │ 2. SIP INVITE            │                          │                    │
  │                          │ ────────────────────────►│                          │                    │
  │                          │                          │ 3. 匹配 Dispatch Rule    │                    │
  │                          │                          │ ────────────────────────►│                    │
  │                          │                          │                          │                    │
  │                          │                          │ 4. 创建 SIP Participant  │                    │
  │                          │                          │ ────────────────────────►│                    │
  │                          │                          │                          │                    │
  │                          │ 5. SIP ↔ WebRTC 转换     │                          │                    │
  │                          │ ◄────────────────────────│                          │                    │
  │                          │                          │                          │                    │
  │ 6. 音频流双向传输         │                          │                          │                    │
  │ ◄───────────────────────►│◄────────────────────────►│◄────────────────────────►│◄──────────────────►│
```

## 部署配置

### Docker Compose 部署

```yaml
services:
  # LiveKit Server - WebRTC 媒体服务器
  livekit-server:
    image: livekit/livekit-server:latest
    ports:
      - "7880:7880"
      - "60000-60100:60000-60100/udp"  # WebRTC RTP 媒体流

  # FreeSWITCH - SIP 电话交换机
  freeswitch:
    image: safarov/freeswitch:latest
    ports:
      - "5060:5060/udp"           # SIP 信令
      - "16384-16484:16384-16484/udp"  # RTP 媒体流

  # LiveKit SIP Server - SIP↔WebRTC 桥接
  livekit-sip:
    image: livekit/sip:latest
    ports:
      - "5070:5060/udp"           # SIP 信令（避免和 FreeSWITCH 冲突）
      - "11000-11100:11000-11100/udp"  # RTP 媒体流
```

### 关键配置项

| 配置 | 说明 |
|------|------|
| FreeSWITCH `domain` | 必须设为 `127.0.0.1`（容器内部 IP 会导致 SIP 注册失败） |
| FreeSWITCH `default_password` | 固定为已知值（不要用随机密码） |
| LiveKit SIP Server `ws_url` | 使用 Docker 内部网络地址（`ws://livekit-server:7880`） |
| Outbound Trunk `address` | 使用 FreeSWITCH 的 external profile（端口 5080，不需要认证） |

## API 调用示例

### 创建 Outbound Trunk

```go
outRes, err := sip.CreateSIPOutboundTrunk(ctx, &livekit.CreateSIPOutboundTrunkRequest{
    Trunk: &livekit.SIPOutboundTrunkInfo{
        Name:         "freeswitch-outbound",
        Address:      "freeswitch:5080",
        Numbers:      []string{"1000"},
        AuthUsername: "username",
        AuthPassword: "password",
    },
})
```

### 创建 Dispatch Rule

```go
ruleRes, err := sip.CreateSIPDispatchRule(ctx, &livekit.CreateSIPDispatchRuleRequest{
    Rule: &livekit.SIPDispatchRule{
        Rule: &livekit.SIPDispatchRule_DispatchRuleDirect{
            DispatchRuleDirect: &livekit.SIPDispatchRuleDirect{
                RoomName: "sip-room",
            },
        },
    },
    TrunkIds: []string{trunkID},
    Name:     "default-rule",
})
```

### 外呼拨号

```go
participant, err := sip.CreateSIPParticipant(ctx, &livekit.CreateSIPParticipantRequest{
    SipTrunkId:          trunkID,
    SipCallTo:           "1001",
    RoomName:            "call-room",
    ParticipantIdentity: "sip-1001",
    ParticipantName:     "Phone User",
    WaitUntilAnswered:   false,
})
```

## Webhook 事件处理

SIP 外呼/API 创建的房间没有 meeting 记录，需要在 `room_started` webhook 事件中补插：

```go
case "room_started":
    // 1. 检查数据库是否有对应会议记录
    // 2. 如果没有，生成 S 开头的会议号（IdUtil.NextId("S", "live")）
    // 3. 生成 9 位用户会议号（RandomDigits(9)）
    // 4. 从 webhook 事件获取房间参数（不自己塞默认值）
    // 5. 创建会议记录，创建人标记为 "SIP"
```

## 数据模型

### live_sip_trunk - SIP 中继线

| 字段 | 类型 | 说明 |
|------|------|------|
| id | string | 主键 |
| name | string | trunk 名称 |
| direction | string | 方向：inbound/outbound/both |
| address | string | SIP 服务器地址 |
| numbers | text | 关联号码列表 JSON |
| auth_username | string | 认证用户名 |
| auth_password | string | 认证密码 |
| provider | string | 供应商标识 |
| destination_country | string | 目标国家代码 |
| sip_trunk_id | string | LiveKit SIP trunk ID |
| status | int | 状态：1-启用 2-禁用 |

### live_sip_dispatch_rule - 路由规则

| 字段 | 类型 | 说明 |
|------|------|------|
| id | string | 主键 |
| name | string | 规则名称 |
| trunk_id | string | 关联 trunk ID |
| rule_type | string | 规则类型：fixed/meeting |
| room_pattern | string | Room 名模板 |
| pin_code | string | pin 码 |
| sip_dispatch_rule_id | string | LiveKit dispatch rule ID |
| status | int | 状态：1-启用 2-禁用 |

### live_sip_call_log - 通话记录

| 字段 | 类型 | 说明 |
|------|------|------|
| id | string | 主键 |
| call_id | string | SIP call ID |
| trunk_id | string | 关联 trunk ID |
| direction | string | 方向：inbound/outbound |
| caller_number | string | 主叫号码 |
| callee_number | string | 被叫号码 |
| meeting_no | string | 关联会议号 |
| room_name | string | LiveKit Room 名 |
| status | string | 通话状态：ringing/active/completed/failed |
| start_time | time | 开始时间 |
| end_time | time | 结束时间 |
| duration | int | 通话时长（秒） |
| hangup_cause | string | SIP 挂断原因码 |

## 常见问题

### SIP 注册失败

**症状**：软电话无法注册到 FreeSWITCH

**原因**：FreeSWITCH 的 `domain` 设置为容器内部 IP（如 `172.19.0.2`），但软电话连接的是 `127.0.0.1`

**解决**：修改 FreeSWITCH 的 `vars.xml`，将 `domain` 设为 `127.0.0.1`

### 拨号 403 Forbidden

**症状**：LiveKit SIP Server 拨号返回 `403 Forbidden`

**原因**：使用了 FreeSWITCH 的 internal profile（端口 5060），需要认证但账号密码不对

**解决**：使用 external profile（端口 5080），不需要认证

### 拨号成功但没有振铃

**症状**：API 调用成功，但软电话没有振铃

**原因**：FreeSWITCH 的 `default_password` 是随机生成的，与 trunk 配置的密码不一致

**解决**：修改 FreeSWITCH 的 `vars.xml`，将 `default_password` 设为固定值（如 `1234`）
