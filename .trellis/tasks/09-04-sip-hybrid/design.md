# SIP 电话混合会议 — 技术设计

## 一、整体架构

```
┌─────────────────────────────────────────────────────────────┐
│                    浏览器 / 前端                              │
│              WebRTC 直连 LiveKit Server                      │
└──────────────┬──────────────────────────────┬───────────────┘
               │ HTTP                         │ WebSocket
┌──────────────▼──────────────────────────────▼───────────────┐
│                livegtw (HTTP 网关 :11002)                    │
│           /live/v1/sip-trunks  /live/v1/sip-calls           │
└──────────────┬──────────────────────────────────────────────┘
               │ gRPC
┌──────────────▼──────────────────────────────────────────────┐
│                   live (gRPC 服务 :21017)                    │
│  ┌────────────┐ ┌────────────┐ ┌────────────┐              │
│  │ Trunk 管理 │ │ Rule 管理  │ │ Call 操作  │              │
│  └─────┬──────┘ └─────┬──────┘ └─────┬──────┘              │
│        │              │              │                      │
│  ┌─────▼──────────────▼──────────────▼─────┐               │
│  │        common/livekitx (SDK wrapper)     │               │
│  │   SIP API · Room API · Token · Webhook   │               │
│  └───────────────────┬─────────────────────┘               │
└──────────────────────┼──────────────────────────────────────┘
                       │ Twirp/HTTP
              ┌────────▼────────┐
              │  LiveKit Server │
              │    :7880        │
              └────────┬────────┘
                       │ Redis
              ┌────────▼────────┐
              │  LiveKit SIP    │
              │  Server :5060   │
              └────────┬────────┘
                       │ SIP/RTP
              ┌────────▼────────┐
              │  SIP Provider   │
              │ (FreeSWITCH/    │
              │  国内运营商)     │
              └─────────────────┘
```

## 二、数据模型

### 2.1 live_sip_trunk — SIP 中继线

```go
// LiveSipTrunk SIP 中继线配置，存储 trunk 信息及其供应商元数据。
// 与 LiveKit SIP API 对齐，同时支持后续对接国内运营商。
type LiveSipTrunk struct {
    gormx.LegacyStringBaseModel
    gormx.VersionMixin

    // 名称，如 "FreeSWITCH-局域网" / "阿里云-北京"
    Name string `gorm:"column:name;size:128;not null;comment:trunk名称"`
    // 方向：inbound / outbound / both
    Direction string `gorm:"column:direction;size:16;not null;comment:方向 inbound|outbound|both"`
    // SIP 服务器地址，如 "192.168.1.100:5080" 或 "sip.telnyx.com"
    Address string `gorm:"column:address;size:256;comment:SIP服务器地址"`
    // 关联号码列表 JSON，如 ["+8613800138000", "1000"]
    Numbers string `gorm:"column:numbers;type:text;comment:关联号码JSON"`
    // SIP 认证用户名
    AuthUsername string `gorm:"column:auth_username;size:128;comment:认证用户名"`
    // SIP 认证密码（加密存储）
    AuthPassword string `gorm:"column:auth_password;size:256;comment:认证密码"`
    // 供应商标识：freeswitch / aliyun / tencent / custom
    Provider string `gorm:"column:provider;size:32;not null;default:custom;comment:供应商标识"`
    // 目标国家/地区代码，如 "CN"
    DestinationCountry string `gorm:"column:destination_country;size:8;default:CN;comment:目标国家代码"`
    // LiveKit SIP trunk ID（创建后回填）
    SipTrunkID string `gorm:"column:sip_trunk_id;size:64;comment:LiveKit SIP trunk ID"`
    // 最大并发数
    MaxConcurrent int32 `gorm:"column:max_concurrent;default:10;comment:最大并发数"`
    // 状态：1-启用 2-禁用
    Status int32 `gorm:"column:status;default:1;not null;comment:状态 1启用 2禁用"`
    // 供应商扩展配置 JSON
    Metadata string `gorm:"column:metadata;type:text;comment:扩展配置JSON"`

    CreateUser sql.NullString `gorm:"column:create_user;size:64;comment:创建人"`
    UpdateUser sql.NullString `gorm:"column:update_user;size:64;comment:更新人"`
    DeptCode   sql.NullString `gorm:"column:dept_code;size:64;comment:机构code"`
}
```

### 2.2 live_sip_dispatch_rule — 路由规则

```go
// LiveSipDispatchRule 来电路由规则，定义 SIP 来电如何路由到 Room。
type LiveSipDispatchRule struct {
    gormx.LegacyStringBaseModel
    gormx.VersionMixin

    // 规则名称
    Name string `gorm:"column:name;size:128;not null;comment:规则名称"`
    // 关联 trunk ID
    TrunkID string `gorm:"column:trunk_id;size:64;comment:关联trunk ID"`
    // 规则类型：fixed（固定Room）/ meeting（按会议号）/ pin（需pin码）
    RuleType string `gorm:"column:rule_type;size:32;not null;comment:规则类型 fixed|meeting|pin"`
    // Room 名模板，支持占位符：{number} 来电号码, {meeting_no} 会议号
    RoomPattern string `gorm:"column:room_pattern;size:256;comment:Room名模板"`
    // pin 码（可选）
    PinCode string `gorm:"column:pin_code;size:32;comment:pin码"`
    // LiveKit dispatch rule ID
    DispatchRuleID string `gorm:"column:dispatch_rule_id;size:64;comment:LiveKit dispatch rule ID"`
    // 状态：1-启用 2-禁用
    Status int32 `gorm:"column:status;default:1;not null;comment:状态 1启用 2禁用"`
    // 扩展配置 JSON
    Metadata string `gorm:"column:metadata;type:text;comment:扩展配置JSON"`

    CreateUser sql.NullString `gorm:"column:create_user;size:64;comment:创建人"`
    UpdateUser sql.NullString `gorm:"column:update_user;size:64;comment:更新人"`
    DeptCode   sql.NullString `gorm:"column:dept_code;size:64;comment:机构code"`
}
```

### 2.3 live_sip_call_log — 通话记录

```go
// LiveSipCallLog SIP 通话记录，记录每次电话呼叫的状态和时长。
type LiveSipCallLog struct {
    gormx.LegacyStringBaseModel

    // LiveKit SIP call ID
    CallID string `gorm:"column:call_id;size:128;not null;comment:LiveKit SIP call ID"`
    // 关联 trunk ID
    TrunkID string `gorm:"column:trunk_id;size:64;comment:关联trunk ID"`
    // 方向：inbound（来电）/ outbound（外呼）
    Direction string `gorm:"column:direction;size:16;not null;comment:方向 inbound|outbound"`
    // 主叫号码
    CallerNumber string `gorm:"column:caller_number;size:32;comment:主叫号码"`
    // 被叫号码
    CalleeNumber string `gorm:"column:callee_number;size:32;comment:被叫号码"`
    // 关联会议号
    MeetingNo string `gorm:"column:meeting_no;size:64;comment:关联会议号"`
    // LiveKit Room 名
    RoomName string `gorm:"column:room_name;size:128;comment:LiveKit Room名"`
    // 通话状态：ringing / active / completed / failed / cancelled
    Status string `gorm:"column:status;size:32;not null;default:ringing;comment:通话状态"`
    // 开始时间
    StartTime sql.NullTime `gorm:"column:start_time;comment:开始时间"`
    // 结束时间
    EndTime sql.NullTime `gorm:"column:end_time;comment:结束时间"`
    // 通话时长（秒）
    Duration int32 `gorm:"column:duration;default:0;comment:通话时长秒"`
    // SIP 挂断原因码
    HangupCause string `gorm:"column:hangup_cause;size:64;comment:SIP挂断原因码"`
    // 供应商扩展数据 JSON
    Metadata string `gorm:"column:metadata;type:text;comment:扩展数据JSON"`

    DeptCode sql.NullString `gorm:"column:dept_code;size:64;comment:机构code"`
}
```

## 三、Proto 接口设计

> 字段命名原则：与 LiveKit SDK (`livekit.CreateSIPParticipantRequest` 等) 保持一致，
> 使用 snake_case（proto 规范），JSON 输出自动转 camelCase。
> 字段含义和用法参考 LiveKit Protocol v1.49.0。

### 3.1 SIP Trunk 管理

```protobuf
// ===== SIP Trunk 管理 =====

// CreateSipTrunk 创建 SIP 中继线。
// 同时在 LiveKit 侧创建对应的 inbound/outbound trunk，
// 数据库持久化 trunk 配置，LiveKit 侧通过 sip_trunk_id 关联。
rpc CreateSipTrunk(CreateSipTrunkReq) returns (CreateSipTrunkRes);

message CreateSipTrunkReq {
    // trunk 名称，如 "FreeSWITCH-局域网" / "阿里云-北京"
    string name = 1;
    // 方向：inbound（来电）/ outbound（外呼）/ both（双向）
    string direction = 2;
    // SIP 服务器地址，如 "192.168.1.100:5080" 或 "sip.telnyx.com"
    // 对应 LiveKit SIPOutboundTrunkInfo.address
    string address = 3;
    // 关联号码列表，如 ["+8613800138000", "1000"]
    // 对应 LiveKit SIPInboundTrunkInfo.numbers / SIPOutboundTrunkInfo.numbers
    repeated string numbers = 4;
    // SIP 认证用户名（可选，为空则无认证）
    // 对应 LiveKit SIPInboundTrunkInfo.auth_username
    string auth_username = 5;
    // SIP 认证密码（可选）
    // 对应 LiveKit SIPInboundTrunkInfo.auth_password
    string auth_password = 6;
    // 供应商标识：freeswitch / aliyun / tencent / custom
    // 用于后续供应商适配，不影响 LiveKit 侧配置
    string provider = 7;
    // 目标国家/地区代码，ISO 3166-1 alpha-2，如 "CN"
    // 对应 LiveKit SIPOutboundTrunkInfo.destination_country
    // LiveKit 根据此代码路由通话到最近的数据中心
    string destination_country = 8;
    // 最大并发数（业务层限制，非 LiveKit 侧配置）
    int32 max_concurrent = 9;
    // 扩展配置 JSON（供应商特定参数，如 IP 白名单、前缀等）
    string metadata = 10;
}

message CreateSipTrunkRes {
    // 创建的 trunk 信息
    SipTrunkInfo trunk = 1;
}

// ListSipTrunks 列出 SIP 中继线。
rpc ListSipTrunks(ListSipTrunksReq) returns (ListSipTrunksRes);

message ListSipTrunksReq {
    // 过滤方向，空字符串表示全部
    string direction = 1;
    // 过滤供应商，空字符串表示全部
    string provider = 2;
    // 过滤状态，0 表示全部，1-启用，2-禁用
    int32 status = 3;
}

message ListSipTrunksRes {
    // trunk 列表
    repeated SipTrunkInfo trunks = 1;
}

// DeleteSipTrunk 删除 SIP 中继线。
// 同时删除 LiveKit 侧对应的 trunk（通过 sip_trunk_id）。
rpc DeleteSipTrunk(DeleteSipTrunkReq) returns (DeleteSipTrunkRes);

message DeleteSipTrunkReq {
    // 要删除的 trunk ID（数据库主键）
    string trunk_id = 1;
}

message DeleteSipTrunkRes {}

// SipTrunkInfo SIP 中继线信息。
// 业务层聚合结构，合并数据库字段和 LiveKit 侧状态。
message SipTrunkInfo {
    // 数据库主键 ID
    string id = 1;
    // trunk 名称
    string name = 2;
    // 方向：inbound / outbound / both
    string direction = 3;
    // SIP 服务器地址
    string address = 4;
    // 关联号码列表
    repeated string numbers = 5;
    // 供应商标识
    string provider = 6;
    // 目标国家/地区代码
    string destination_country = 7;
    // LiveKit SIP trunk ID（创建时由 LiveKit 返回，用于后续 API 调用）
    string sip_trunk_id = 8;
    // 最大并发数
    int32 max_concurrent = 9;
    // 状态：1-启用 2-禁用
    int32 status = 10;
    // 创建时间，格式：yyyy-MM-dd HH:mm:ss
    string create_time = 11;
}
```

### 3.2 Dispatch Rule 管理

```protobuf
// ===== Dispatch Rule 管理 =====

// CreateSipDispatchRule 创建来电路由规则。
// 定义 SIP 来电如何路由到 LiveKit Room。
// 同时在 LiveKit 侧创建对应的 dispatch rule。
rpc CreateSipDispatchRule(CreateSipDispatchRuleReq) returns (CreateSipDispatchRuleRes);

message CreateSipDispatchRuleReq {
    // 规则名称，如 "默认来电路由" / "客服热线路由"
    string name = 1;
    // 关联 trunk ID 列表（可选，为空则匹配所有 trunk）
    // 对应 LiveKit SIPDispatchRuleInfo.trunk_ids
    repeated string trunk_ids = 2;
    // 规则类型：
    //   "fixed"   - 固定 Room，所有来电进入同一个 Room（room_pattern 必填）
    //   "meeting" - 按会议号路由，来电根据号码匹配到对应会议
    string rule_type = 3;
    // Room 名模板，支持占位符：
    //   {number}     - 来电号码
    //   {meeting_no} - 会议号
    // 示例："phone-{number}" 或 "meeting-{meeting_no}"
    // 当 rule_type=fixed 时，直接填固定 Room 名
    string room_pattern = 4;
    // pin 码（可选，来电需输入此码才能加入 Room）
    // 对应 LiveKit SIPDispatchRuleInfo.rule.dispatch_rule_direct.pin
    string pin_code = 5;
    // 扩展配置 JSON
    string metadata = 6;
}

message CreateSipDispatchRuleRes {
    // 创建的路由规则信息
    SipDispatchRuleInfo rule = 1;
}

// ListSipDispatchRules 列出来电路由规则。
rpc ListSipDispatchRules(ListSipDispatchRulesReq) returns (ListSipDispatchRulesRes);

message ListSipDispatchRulesReq {}

message ListSipDispatchRulesRes {
    // 路由规则列表
    repeated SipDispatchRuleInfo rules = 1;
}

// DeleteSipDispatchRule 删除来电路由规则。
// 同时删除 LiveKit 侧对应的 dispatch rule。
rpc DeleteSipDispatchRule(DeleteSipDispatchRuleReq) returns (DeleteSipDispatchRuleRes);

message DeleteSipDispatchRuleReq {
    // 要删除的规则 ID（数据库主键）
    string rule_id = 1;
}

message DeleteSipDispatchRuleRes {}

// SipDispatchRuleInfo 来电路由规则信息。
message SipDispatchRuleInfo {
    // 数据库主键 ID
    string id = 1;
    // 规则名称
    string name = 2;
    // 关联 trunk ID 列表
    repeated string trunk_ids = 3;
    // 规则类型：fixed / meeting
    string rule_type = 4;
    // Room 名模板
    string room_pattern = 5;
    // LiveKit dispatch rule ID（创建时由 LiveKit 返回）
    string sip_dispatch_rule_id = 6;
    // 状态：1-启用 2-禁用
    int32 status = 7;
    // 创建时间，格式：yyyy-MM-dd HH:mm:ss
    string create_time = 8;
}
```

### 3.3 SIP 通话操作

```protobuf
// ===== SIP 通话操作 =====

// DialSip 发起 SIP 外呼。
// 根据 sip_call_to 号码发起外呼，将对方接入 LiveKit Room。
// 支持两种场景：
//   1. 点对点拨号：不指定 meeting_no，自动创建新会议
//   2. 群内呼号：指定 meeting_no，将电话参会者加入已有会议
// 字段命名与 LiveKit CreateSIPParticipantRequest 对齐。
rpc DialSip(DialSipReq) returns (DialSipRes);

message DialSipReq {
    // 被叫号码（必填），如 "1001"（局域网分机）或 "+8613800138000"（手机号）
    // 对应 LiveKit CreateSIPParticipantRequest.sip_call_to
    string sip_call_to = 1;
    // 主叫号码（可选），不填则使用 trunk 默认号码
    // 对应 LiveKit CreateSIPParticipantRequest.sip_number
    string sip_number = 2;
    // 会议号（可选），为空则自动创建新会议（标题："电话通话-{sip_call_to}"）
    // 对应 LiveKit CreateSIPParticipantRequest.room_name（会议号 = Room 名）
    string meeting_no = 3;
    // 指定 trunk ID（可选），不填则自动选择可用的 outbound trunk
    // 对应 LiveKit CreateSIPParticipantRequest.sip_trunk_id
    string sip_trunk_id = 4;
    // 是否等待对方接听后才返回（默认 true）
    // 对应 LiveKit CreateSIPParticipantRequest.wait_until_answered
    bool wait_until_answered = 5;
    // DTMF 按键（可选），用于 IVR 导航，字符 'w' 可加 0.5 秒延迟
    // 对应 LiveKit CreateSIPParticipantRequest.dtmf
    string dtmf = 6;
    // 电话参会者在 Room 中的显示名（可选）
    // 对应 LiveKit CreateSIPParticipantRequest.participant_name
    string participant_name = 7;
    // 电话参会者在 Room 中的唯一身份标识（可选）
    // 对应 LiveKit CreateSIPParticipantRequest.participant_identity
    // 不填则默认使用电话号码作为 identity
    string participant_identity = 8;
}

message DialSipRes {
    // 会议信息（新建或已有）
    MeetingInfo meeting = 1;
    // SIP 通话 ID（LiveKit 侧的 call ID）
    string sip_call_id = 2;
    // 通话状态：ringing（振铃中）/ active（通话中）/ failed（失败）
    string call_status = 3;
}

// HangupSip 挂断进行中的 SIP 通话。
rpc HangupSip(HangupSipReq) returns (HangupSipRes);

message HangupSipReq {
    // 要挂断的 SIP 通话 ID
    string sip_call_id = 1;
}

message HangupSipRes {}

// ListSipCalls 列出通话记录。
rpc ListSipCalls(ListSipCallsReq) returns (ListSipCallsRes);

message ListSipCallsReq {
    // 过滤关联的会议号，空字符串表示全部
    string meeting_no = 1;
    // 过滤方向：inbound / outbound，空字符串表示全部
    string direction = 2;
    // 页码，从 1 开始
    int64 page = 3;
    // 每页数量
    int64 page_size = 4;
}

message ListSipCallsRes {
    // 通话记录列表
    repeated SipCallInfo calls = 1;
    // 总数量
    int64 total = 2;
}

// SipCallInfo 通话记录信息。
message SipCallInfo {
    // 数据库主键 ID
    string id = 1;
    // SIP 通话 ID
    string sip_call_id = 2;
    // 方向：inbound（来电）/ outbound（外呼）
    string direction = 3;
    // 主叫号码
    string caller_number = 4;
    // 被叫号码
    string callee_number = 5;
    // 关联会议号
    string meeting_no = 6;
    // 通话状态：ringing / active / completed / failed / cancelled
    string call_status = 7;
    // 开始时间，格式：yyyy-MM-dd HH:mm:ss
    string start_time = 8;
    // 结束时间，格式：yyyy-MM-dd HH:mm:ss（未结束为空字符串）
    string end_time = 9;
    // 通话时长（秒）
    int32 duration = 10;
}
```

## 四、HTTP 网关端点

| Method | Path | 说明 | 鉴权 |
|--------|------|------|------|
| POST | `/live/v1/sip-trunks` | 创建 trunk | JWT |
| GET | `/live/v1/sip-trunks` | 列出 trunk | JWT |
| DELETE | `/live/v1/sip-trunks/:id` | 删除 trunk | JWT |
| POST | `/live/v1/sip-dispatch-rules` | 创建路由规则 | JWT |
| GET | `/live/v1/sip-dispatch-rules` | 列出路由规则 | JWT |
| DELETE | `/live/v1/sip-dispatch-rules/:id` | 删除路由规则 | JWT |
| POST | `/live/v1/sip-calls/dial` | 外呼拨号 | JWT |
| POST | `/live/v1/sip-calls/hangup` | 挂断通话 | JWT |
| GET | `/live/v1/sip-calls` | 通话记录 | JWT |

## 五、文件变更清单

| 文件 | 变更类型 | 说明 |
|------|---------|------|
| `app/live/live.proto` | 修改 | 新增 SIP RPC 和 message 定义 |
| `app/live/internal/logic/createsiptrunklogic.go` | 新增 | 创建 trunk 逻辑 |
| `app/live/internal/logic/listsiptrunkslogic.go` | 新增 | 列出 trunk 逻辑 |
| `app/live/internal/logic/deletesiptrunklogic.go` | 新增 | 删除 trunk 逻辑 |
| `app/live/internal/logic/createsipdispatchrulelogic.go` | 新增 | 创建路由规则逻辑 |
| `app/live/internal/logic/listsipdispatchruleslogic.go` | 新增 | 列出路由规则逻辑 |
| `app/live/internal/logic/deletesipdispatchrulelogic.go` | 新增 | 删除路由规则逻辑 |
| `app/live/internal/logic/dialsiplogic.go` | 新增 | 外呼拨号逻辑 |
| `app/live/internal/logic/hangupsiplogic.go` | 新增 | 挂断通话逻辑 |
| `app/live/internal/logic/listsipcallslogic.go` | 新增 | 通话记录逻辑 |
| `app/live/model/gormmodel/sip_trunk.go` | 新增 | SIP trunk 数据模型 |
| `app/live/model/gormmodel/sip_dispatch_rule.go` | 新增 | 路由规则数据模型 |
| `app/live/model/gormmodel/sip_call_log.go` | 新增 | 通话记录数据模型 |
| `app/livegtw/internal/handler/live/sip*.go` | 新增 | SIP HTTP handler |
| `app/livegtw/internal/types/types.go` | 修改 | 新增 SIP 请求/响应类型 |
| `app/livegtw/internal/handler/routes.go` | 修改 | 新增 SIP 路由 |
| `deploy/livekit/docker-compose.yaml` | 修改 | 新增 LiveKit SIP Server |
| `common/livekitx/sip.go` | 新增 | SIP 操作辅助函数 |

## 六、关键设计决策

### 6.1 Trunk 是否持久化

**决策：持久化到数据库。**

理由：
- Trunk 配置需要在服务重启后保留
- 需要记录供应商信息（provider）用于后续适配
- LiveKit SIP API 是无状态的，每次 ListSIPTrunk 都要调 LiveKit
- 数据库作为 single source of truth，LiveKit 侧同步

### 6.2 通话状态同步

**决策：Webhook 驱动 + 轮询兜底。**

- LiveKit Webhook 推送 SIP 事件（participant_joined, participant_left）
- 现有 Webhook 流程扩展处理 SIP 事件
- 前端通过轮询或 WebSocket 获取通话状态更新

### 6.3 外呼与 Meeting 的关系

**决策：外呼自动创建或加入 Meeting。**

- 点对点拨号：自动创建新 Meeting（标题："电话通话-{号码}"）
- 群内呼号：加入已有 Meeting
- Meeting 与通话记录通过 meeting_no 关联
