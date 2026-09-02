# LiveKit 会议票据机制 - 技术设计

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                          livegtw (HTTP 网关)                    │
├─────────────────────────────────────────────────────────────────┤
│  POST /live/v1/meeting/generateTicket  [JWT + MeetingAuth]      │
│  POST /live/v1/meeting/joinByTicket    [无鉴权]                  │
│  POST /live/v1/meeting/join            [JWT + MeetingAuth]      │
└─────────────────────────────────────────────────────────────────┘
                              │
                              │ gRPC
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                          app/live (gRPC 服务)                   │
├─────────────────────────────────────────────────────────────────┤
│  GenerateMeetingTicket  - 生成票据，存入 Redis                   │
│  JoinMeetingByTicket    - 验证票据，生成 LiveKit token           │
│  JoinMeeting            - 根据会议号和用户信息加入会议            │
└─────────────────────────────────────────────────────────────────┘
                              │
                              │ Redis
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                          Redis                                  │
├─────────────────────────────────────────────────────────────────┤
│  live:ticket:{ticket}  - 票据数据（TTL 自动过期）                │
└─────────────────────────────────────────────────────────────────┘
```

## Data Flow

### 1. 生成票据流程

```
用户(已登录) → livegtw → app/live → Redis
     │            │          │         │
     │ POST       │ gRPC     │ SET     │
     │ /generate  │ ────────>│ ───────>│
     │ Ticket     │          │ TTL     │
     │            │<────────│<────────│
     │<───────────│ 票据信息 │         │
     │ joinUrl    │          │         │
```

### 2. 根据票据加入会议流程

```
用户(未登录) → livegtw → app/live → Redis → LiveKit
     │            │          │         │         │
     │ POST       │ gRPC     │ GET     │ 生成    │
     │ /joinBy    │ ────────>│ ───────>│ token   │
     │ Ticket     │          │ DEL     │ ───────>│
     │            │          │ UPSERT  │         │
     │            │          │ 参会人   │         │
     │            │<────────│<────────│<────────│
     │<───────────│ token    │         │         │
     │ wsUrl      │          │         │         │
```

**关键步骤**：
1. 从 Redis 获取票据数据
2. 验证票据是否存在且未过期
3. 验证 identity 是否与票据绑定的 identity 匹配
4. **Upsert 参会人记录**（与标准加入会议一致）
5. 删除已使用的票据
6. 生成 LiveKit join token
7. 返回 token 和会议信息

### 3. 标准加入会议流程

```
用户(已登录) → livegtw → app/live → LiveKit
     │            │          │         │
     │ POST       │ gRPC     │ 生成    │
     │ /join      │ ────────>│ token   │
     │            │          │ ───────>│
     │            │<────────│<────────│
     │<───────────│ token    │         │
     │ wsUrl      │          │         │
```

## Contracts

### gRPC Proto 修改

```protobuf
// 新增消息类型
message GenerateMeetingTicketReq {
  // 会议号
  string meeting_no = 1 [json_name = "meetingNo"];
  // 绑定的参会人身份
  string identity = 2 [json_name = "identity"];
  // 票据有效期秒数（默认 3600）
  uint32 expire_seconds = 3 [json_name = "expireSeconds"];
}

message GenerateMeetingTicketRes {
  // 票据字符串
  string ticket = 1 [json_name = "ticket"];
  // 过期时间，格式：yyyy-MM-dd HH:mm:ss
  string expire_time = 2 [json_name = "expireTime"];
  // 共用的中间地址，携带票据参数
  string join_url = 3 [json_name = "joinUrl"];
}

message JoinMeetingByTicketReq {
  // 票据字符串
  string ticket = 1 [json_name = "ticket"];
  // 参会人身份（必须与票据绑定的 identity 匹配）
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

### API 修改

```api
// 新增类型定义
type GenerateMeetingTicketReq {
    MeetingNo string `json:"meetingNo"`
    Identity string `json:"identity"`
    ExpireSeconds int32 `json:"expireSeconds,optional"`
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

## Redis 设计

### Key 格式

```
live:ticket:{ticket}
```

### Value 结构

```json
{
  "meetingNo": "M20250902001",
  "identity": "device-001",
  "expireTime": "2025-09-02 12:00:00",
  "createUser": "user-123"
}
```

### TTL

- 由 `expire_seconds` 参数决定
- 默认 3600 秒（1小时）
- Redis 自动过期删除

### 票据格式

- 格式: `{yyyyMMdd}-{uuid}`（如 `20250902-a1b2c3d4-e5f6-7890-abcd-ef1234567890`）
- 生成方式: `fmt.Sprintf("%s-%s", carbon.Now().Format("Ymd"), uuid.New().String())`

## 聊天记录设计

### 数据模型

```go
// LiveMeetingMessage 会议聊天消息
type LiveMeetingMessage struct {
    ID          int64          `gorm:"primaryKey;autoIncrement"`
    MeetingNo   string         `gorm:"size:32;not null;index:idx_meeting_message_meeting"`
    MessageID   string         `gorm:"size:64;not null;uniqueIndex"`
    SenderID    string         `gorm:"size:64;not null"`
    SenderName  string         `gorm:"size:64"`
    Content     string         `gorm:"type:text;not null"`
    MessageType string         `gorm:"size:32;default:text"`
    CreateTime  time.Time      `gorm:"autoCreateTime"`
}
```

### 接口设计

#### 上报聊天消息

**接口**: `POST /live/v1/meeting/reportMessage`

**gRPC**: `ReportMeetingMessage(ReportMeetingMessageReq) returns (ReportMeetingMessageRes)`

**请求**:
```protobuf
message ReportMeetingMessageReq {
  string meeting_no = 1;
  string message_id = 2;
  string content = 3;
  string message_type = 4; // text, image, file
}
```

**响应**:
```protobuf
message ReportMeetingMessageRes {}
```

#### 查询聊天记录

**接口**: `GET /live/v1/meeting/messages`

**gRPC**: `ListMeetingMessages(ListMeetingMessagesReq) returns (ListMeetingMessagesRes)`

**请求**:
```protobuf
message ListMeetingMessagesReq {
  string meeting_no = 1;
  int64 page = 2;
  int64 page_size = 3;
}
```

**响应**:
```protobuf
message ListMeetingMessagesRes {
  repeated MeetingMessageInfo messages = 1;
  int64 total = 2;
}

message MeetingMessageInfo {
  string message_id = 1;
  string sender_id = 2;
  string sender_name = 3;
  string content = 4;
  string message_type = 5;
  string create_time = 6;
}
```

## Validation & Error Matrix

| 场景 | 错误码 | 错误信息 |
|------|--------|----------|
| 票据不存在 | 102102 | 票据不存在或已过期 |
| 票据已过期 | 102102 | 票据不存在或已过期 |
| identity 不匹配 | 105102 | 身份不匹配 |
| 会议已结束 | 105102 | 会议已结束 |
| 会议不存在 | 102102 | 会议不存在 |

## Compatibility

### 向后兼容

- 现有的 `JoinMeeting` 接口保持不变
- 新增的 `JoinMeetingByTicket` 接口是独立的
- 现有的 webhook 逻辑不受影响

### 迁移步骤

1. 修改 `live.proto`，新增消息类型和 RPC 方法
2. 重新生成 gRPC 代码
3. 实现 `GenerateMeetingTicket` 和 `JoinMeetingByTicket` 逻辑
4. 修改 `livegtw.api`，新增类型和路由
5. 重新生成网关代码
6. 实现网关逻辑

## Trade-offs

### 选择 Redis 存储票据

**优点**:
- 利用 TTL 自动过期，不需要手动清理
- 读写性能好
- 实现简单

**缺点**:
- Redis 重启会丢失未过期的票据
- 不支持票据查询和统计

### 选择一次性票据

**优点**:
- 安全性高，防止票据被重复使用
- 实现简单，使用后直接删除

**缺点**:
- 用户断开重连需要重新获取票据
- 不适合需要多次加入的场景

## Operational Considerations

### 监控

- 监控 Redis 中票据的数量
- 监控票据生成和使用的频率
- 监控票据验证失败的次数

### 回滚

- 如果出现问题，可以快速回滚到不支持票据的版本
- 回滚步骤：
  1. 移除网关层的票据路由
  2. 移除 gRPC 层的票据方法
  3. 清理 Redis 中的票据数据
