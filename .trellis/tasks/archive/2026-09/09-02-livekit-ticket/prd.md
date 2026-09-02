# LiveKit 会议票据机制

## Goal

为 LiveKit 会议系统添加票据机制，支持未登录用户通过临时票据加入会议。主要用于临时设备（如安全帽）加入会议的场景。

## Background

当前系统只有两种加入会议的方式：
1. 已登录用户通过 JWT 鉴权后调用 `joinMeeting` 接口加入会议
2. webhook 接收 LiveKit 事件

用户需要支持第三种场景：临时设备（如安全帽）不需要登录，通过票据机制加入会议。

## Requirements

### 1. 获取邀请会议票据（需要鉴权）

**接口**: `POST /live/v1/meeting/generateTicket`

**功能**:
- 已登录用户调用，为指定会议生成邀请票据
- 生成临时票据字符串和过期时间
- 不存储参会人数据
- 返回票据信息和 joinUrl

**请求参数**:
- `meetingNo`: 会议号（必填）
- `identity`: 绑定的参会人身份（必填）
- `expireSeconds`: 票据有效期秒数（可选，默认 3600）

**响应**:
- `ticket`: 票据字符串
- `expireTime`: 过期时间（yyyy-MM-dd HH:mm:ss）
- `joinUrl`: 共用的中间地址，携带票据参数（如 `/live/v1/meeting/joinByTicket?ticket=xxx`）
  - 可以跳转到测试页面：`/test/meeting?ticket=xxx`
  - 可以跳转到业务侧系统：`https://business.example.com/meeting?ticket=xxx`
  - 前端根据场景选择跳转目标

### 2. 根据票据加入会议（不需要鉴权）

**接口**: `POST /live/v1/meeting/joinByTicket`

**功能**:
- 未登录用户调用，通过票据加入会议
- 验证票据有效性和过期时间
- 验证 identity 是否与票据绑定的 identity 匹配
- 生成 LiveKit join token
- 返回 token 和会议信息

**请求参数**:
- `ticket`: 票据字符串（必填）
- `identity`: 参会人身份（必填，必须与票据绑定的 identity 匹配）
- `name`: 展示名（可选）

**响应**:
- `token`: LiveKit join token
- `wsUrl`: LiveKit WebSocket 地址
- `meeting`: 会议信息

### 3. 修改现有加入会议接口

**接口**: `POST /live/v1/meeting/join`

**修改**:
- 确保从 authctx 获取用户信息（userId、userName）
- 传递给 gRPC 的 identity 和 name 字段使用 authctx 中的用户信息

### 4. 上报聊天消息（需要鉴权）

**接口**: `POST /live/v1/meeting/reportMessage`

**功能**:
- 客户端发送聊天消息后，同时 HTTP 上报给服务端
- 服务端存储消息到数据库
- 用于审计和历史查询

**请求参数**:
- `meetingNo`: 会议号（必填）
- `messageId`: 消息 ID（必填，客户端生成）
- `content`: 消息内容（必填）
- `messageType`: 消息类型（可选，默认 text）

**响应**:
- 无（成功返回即可）

### 5. 查询聊天记录（需要鉴权）

**接口**: `GET /live/v1/meeting/messages`

**功能**:
- 查询指定会议的聊天记录
- 支持分页

**请求参数**:
- `meetingNo`: 会议号（必填）
- `page`: 页码（可选，默认 1）
- `pageSize`: 每页数量（可选，默认 50）

**响应**:
- `messages`: 消息列表
- `total`: 总数量

## Technical Design

### gRPC Proto 修改

```protobuf
// 新增消息类型
message GenerateMeetingTicketReq {
  // 会议号
  string meeting_no = 1 [json_name = "meetingNo"];
  // 票据有效期秒数（默认 3600）
  uint32 expire_seconds = 2 [json_name = "expireSeconds"];
  // 回调 URL（可选）
  string callback_url = 3 [json_name = "callbackUrl"];
}

message GenerateMeetingTicketRes {
  // 票据字符串
  string ticket = 1 [json_name = "ticket"];
  // 过期时间，格式：yyyy-MM-dd HH:mm:ss
  string expire_time = 2 [json_name = "expireTime"];
  // 直接加入会议的 URL
  string join_url = 3 [json_name = "joinUrl"];
}

message JoinMeetingByTicketReq {
  // 票据字符串
  string ticket = 1 [json_name = "ticket"];
  // 参会人身份（房间内唯一）
  string identity = 2 [json_name = "identity"];
  // 展示名（可空）
  string name = 3 [json_name = "name"];
}

message JoinMeetingByTicketRes {
  // join token（浏览器直连 LiveKit 使用）
  string token = 1 [json_name = "token"];
  // LiveKit WebSocket 地址
  string ws_url = 2 [json_name = "wsUrl"];
  // 会议信息
  MeetingInfo meeting = 3 [json_name = "meeting"];
}

// 修改现有消息类型
message JoinMeetingReq {
  // 会议号
  string meeting_no = 1 [json_name = "meetingNo"];
  // 参会人身份（房间内唯一）- 从 authctx 获取
  string identity = 2 [json_name = "identity"];
  // 展示名（可空）- 从 authctx 获取
  string name = 3 [json_name = "name"];
}

// 新增 RPC 方法
service LiveRpc {
  // ... 现有方法 ...
  
  // 生成会议邀请票据（需要鉴权）
  rpc GenerateMeetingTicket(GenerateMeetingTicketReq) returns (GenerateMeetingTicketRes);
  
  // 根据票据加入会议（不需要鉴权）
  rpc JoinMeetingByTicket(JoinMeetingByTicketReq) returns (JoinMeetingByTicketRes);
}
```

### 票据设计

- 票据格式: `{yyyyMMdd}-{uuid}`（如 `20250902-a1b2c3d4-e5f6-7890-abcd-ef1234567890`）
- 存储位置: Redis，key 格式 `live:ticket:{ticket}`
- 过期时间: Redis TTL 自动过期
- 票据内容: meetingNo、identity、expireTime、createUser
- 使用次数: 限制一次，使用后立即删除 Redis key

### API 修改

```api
// 新增类型定义
type GenerateMeetingTicketReq {
    MeetingNo string `json:"meetingNo"`
    ExpireSeconds int32 `json:"expireSeconds,optional"`
    CallbackUrl string `json:"callbackUrl,optional"`
}

type GenerateMeetingTicketRes {
    Ticket string `json:"ticket"`
    ExpireTime string `json:"expireTime"`
    JoinUrl string `json:"joinUrl"`
}

type JoinMeetingByTicketReq {
    Ticket string `json:"ticket"`
    Identity string `json:"identity"`
    Name string `json:"name,optional"`
}

type JoinMeetingByTicketRes {
    Token string `json:"token"`
    WsUrl string `json:"wsUrl"`
    Meeting MeetingInfo `json:"meeting"`
}

// 修改现有 JoinMeetingReq - 移除 Identity 和 Name（从 authctx 获取）
type JoinMeetingReq {
    MeetingNo string `json:"meetingNo"`
}

// 新增路由组（不需要鉴权）
@server (
    prefix: live/v1/meeting
    group:  ticket
)
service livegtw {
    @doc "根据票据加入会议（无需JWT）"
    @handler joinMeetingByTicket
    post /joinByTicket (JoinMeetingByTicketReq) returns (JoinMeetingByTicketRes)
}

// 修改现有路由组（需要鉴权）
@server (
    prefix:     live/v1/meeting
    group:      meeting
    jwt:        JwtAuth
    middleware: MeetingAuth
)
service livegtw {
    // ... 现有接口 ...
    
    @doc "生成会议邀请票据"
    @handler generateMeetingTicket
    post /generateTicket (GenerateMeetingTicketReq) returns (GenerateMeetingTicketRes)
}
```

## Acceptance Criteria

1. 已登录用户可以调用 `generateTicket` 接口为指定会议生成票据
2. 票据包含过期时间，过期后无法使用
3. 未登录用户可以通过 `joinMeetingByTicket` 接口使用票据加入会议
4. 票据验证失败（无效或过期）返回明确错误信息
5. 现有的 `joinMeeting` 接口继续工作，从 authctx 获取用户信息
6. 临时设备（如安全帽）可以通过票据机制加入会议

## Out of Scope

- 票据的持久化存储（使用 Redis TTL 自动过期）
- 票据的撤销机制
- 票据的使用次数统计
- 聊天室功能（客户端直接通过 LiveKit WebSocket 实现）
- 参会人记录的存储（临时用户不记录）

## Open Questions

1. ~~票据回调 URL 的具体用途是什么？~~ → 已确认：前端跳转，生成的 `joinUrl` 包含票据
2. ~~票据是否需要绑定特定的 identity？~~ → 已确认：绑定，票据生成时指定 identity，使用时必须匹配
3. ~~是否需要限制每个票据的使用次数？~~ → 已确认：限制一次，使用后立即失效

## Artifacts

- PRD: `.trellis/tasks/09-02-livekit-ticket/prd.md`
- Design: `.trellis/tasks/09-02-livekit-ticket/design.md` (待创建)
- Implement: `.trellis/tasks/09-02-livekit-ticket/implement.md` (待创建)
