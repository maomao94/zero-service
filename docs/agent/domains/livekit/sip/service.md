# LiveKit SIP 电话集成

适用：SIP 电话与 LiveKit 会议混合场景，包括供应商管理、外呼拨号。

### 1. SIP 架构总览

```
浏览器(WebRTC) ◄──► LiveKit Server ◄──► LiveKit SIP Server ◄──► SIP 供应商 ◄──► 电话网络
     :7880                :7880                :5070
```

| 组件 | 职责 |
|------|------|
| LiveKit Server | WebRTC 媒体服务器，房间和参与者管理 |
| LiveKit SIP Server | SIP↔WebRTC 协议转换桥接 |
| SIP 供应商 | 外部 SIP 服务器（FreeSWITCH 本地测试 / Telnyx/Twilio/Plivo 生产） |

生产环境 LiveKit SIP Server 直连供应商，不需要 FreeSWITCH。FreeSWITCH 仅用于本地测试。

### 2. SIP 供应商管理（SipProvider CRUD）

供应商配置持久化到数据库（`live_sip_providers`），通过 `provider_code` 关联。

#### Proto 接口

```protobuf
rpc CreateSipProvider(CreateSipProviderReq) returns (CreateSipProviderRes);
rpc UpdateSipProvider(UpdateSipProviderReq) returns (UpdateSipProviderRes);
rpc ListSipProviders(ListSipProvidersReq) returns (ListSipProvidersRes);
rpc DeleteSipProvider(DeleteSipProviderReq) returns (DeleteSipProviderRes);
```

#### 数据模型

```go
type LiveSipProvider struct {
    gormx.LegacyStringBaseModel  // 无 VersionMixin（低并发配置表，不需要乐观锁）
    Code         string          // 供应商编码（唯一索引）
    Name         string          // 供应商名称
    Address      string          // SIP 服务器地址
    Numbers      string          // 主叫号码池 JSON 数组
    AuthUsername  string          // SIP 认证用户名
    AuthPassword  string          // SIP 认证密码
    Status       int32           // 1-启用 2-禁用
}
```

#### 关键约束

- `provider_code` 唯一索引，创建时检查唯一性
- `UpdateSipProvider` 只更新非空字段（partial update）
- `DeleteSipProvider` 硬删除
- 不使用 `VersionMixin`（配置表，低并发）

### 3. SIP 外呼拨号（DialSip）

```protobuf
rpc DialSip(DialSipReq) returns (DialSipRes);

message DialSipReq {
    string callee_number = 1;      // 被叫号码（必填）
    string meeting_no = 2;         // 会议号（可选，空=S前缀自动创建）
    string participant_name = 3;   // 显示名（可选）
    string provider_code = 4;      // 供应商编码（必填）
}

message DialSipRes {
    MeetingInfo meeting = 1;       // 会议信息
    string sip_call_id = 2;       // SIP 通话 ID
}
```

#### DialSipLogic 流程

1. **确定会议**：`meeting_no` 为空 → 自动创建 S 前缀会议；不为空 → 校验会议存在且进行中
2. **查询供应商**：按 `provider_code` 查询 `live_sip_providers`，查不到报错
3. **选择/创建 trunk**：`ListSIPOutboundTrunk` 复用已有 outbound trunk；没有则用供应商配置 `CreateSIPOutboundTrunk`
4. **发起外呼**：`CreateSIPParticipant`（`WaitUntilAnswered=false`）
5. **返回**：meeting info + sip_call_id

#### Trunk 管理约定

- Trunk 不持久化到数据库，由 LiveKit 侧管理
- 一个 SIP 供应商对应一个 trunk，所有外呼复用
- `ListSIPOutboundTrunk` 查询已有 trunk，复用第一个；没有才创建
- 创建 trunk 时使用供应商配置（address、numbers、auth）
- 使用 `ListSIPOutboundTrunk`（非废弃的 `ListSIPTrunk`）

#### 会议号前缀

| 前缀 | 场景 | 生成方式 |
|------|------|---------|
| M | 正常创建的会议 | `IdUtil.NextId("M", "live")` |
| S | 电话通话（自动创建） | `IdUtil.NextId("S", "live")` |

### 4. SIP 补插会议记录（Webhook 处理）

SIP 外呼/API 创建的房间没有 meeting 记录，需要在 `room_started` webhook 事件中补插：

```go
// handleRoomStarted 补插逻辑
// 1. 检查数据库是否有对应会议记录（GetMeeting）
// 2. 如果没有，生成 S 开头的会议号（IdUtil.NextId("S", "live")）
// 3. 生成 9 位用户会议号（RandomDigits(9)），加锁防重复
// 4. 从 webhook 事件获取房间参数（EmptyTimeout, DepartureTimeout, MaxParticipants, Sid, Metadata）
// 5. 创建会议记录，创建人标记为 "SIP"
```

**关键约束**：
- 会议号用 `S` 前缀（区别于正常创建的 `M` 前缀）
- 房间参数完全来自 webhook 事件，不自己塞默认值
- 创建人/更新人标记为 `SIP`（无用户身份）
- 幂等：已有记录则跳过

### 5. livekitx SDK 约定

- `Client.SIP()` 暴露 `livekit.SIP` 接口，业务直接调用 SDK 原生方法
- 不封装 SIP 辅助函数（与 `Room()` 风格一致）
- `ListSIPOutboundTrunk` 替代已废弃的 `ListSIPTrunk`

### 6. 模型风格

- `LiveSipProvider`：`LegacyStringBaseModel`，无 `VersionMixin`（低并发配置表）
- `LiveMeeting` / `LiveMeetingParticipant`：`LegacyStringBaseModel`，无 `VersionMixin`（已有 Redis 悲观锁）
- 只有真正高并发且无悲观锁保护的表才加 `VersionMixin`

