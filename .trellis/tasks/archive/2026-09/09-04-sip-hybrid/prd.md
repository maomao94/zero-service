# SIP 电话拨号功能

## Goal

在现有 LiveKit 会议系统中增加 SIP 电话拨号能力，支持两种场景：
1. **拨打电话**：发起点对点电话通话（自动创建 S 前缀会议）
2. **电话会议**：在已有会议中邀请电话参会者加入（M 前缀会议）

第一阶段以局域网 FreeSWITCH 为验证目标，接口设计兼容后续对接国内 SIP trunk 供应商。

## 用户场景

### 场景 1：拨打电话

用户 A（浏览器）想和用户 B（电话）通话：
1. 用户 A 在前端点击"拨打电话"，输入电话号码
2. 后端自动创建 Meeting（S 前缀会议号）+ Room，发起 SIP 外呼
3. 用户 B 手机振铃 → 接听 → 自动进入 Room
4. 用户 A 在浏览器看到电话用户已加入，双方开始通话

### 场景 2：会议中邀请电话加入

用户 A 正在开会，需要邀请场外人员：
1. 用户 A 在会议界面点击"邀请电话参会者"
2. 输入电话号码，发起外呼（传当前 M 前缀会议号）
3. 对方接听后以 SIP Participant 身份加入当前 Room
4. Room 内所有用户（Web + 电话）可互相通话

## 功能范围

### In Scope（本次实现）

| 模块 | 功能 | 说明 |
|------|------|------|
| Proto | `DialSip` RPC | 一个 RPC 覆盖两种场景（meeting_no 是否为空） |
| Model | `LiveSipTrunk` | SIP trunk 配置模型 |
| Model | `LiveSipCallLog` | 通话记录模型（含 call_type: phone/conference） |
| Logic | DialSip 逻辑 | 自动创建/选择 trunk → CreateSipParticipant → 写 call_log |
| Logic | 自动 trunk 管理 | dial 时无 trunk 则自动创建默认 FreeSWITCH outbound trunk |
| 网关 | `/sip-calls/dial` | HTTP 端点，JWT 鉴权 |
| 前端 | 拨打电话 UI | 拨号盘组件（大厅 + 会议内） |
| 前端 | 邀请电话参会者 UI | 会议内弹窗（传当前 meeting_no） |
| Webhook | call_log 状态更新 | participant_joined/left 更新通话状态 |

### Out of Scope（后续迭代）

- Trunk CRUD 管理 API（当前自动创建，后续需要手动管理时再加）
- Dispatch Rule CRUD（来电接入，P1）
- 挂断 API（通话自然结束，后续需要手动挂断时再加）
- 通话记录列表查询 API（后续做）
- 来电提醒 UI（P1）
- 供应商适配（P1，当前只对接 FreeSWITCH）

## 接口设计

### DialSip RPC

```protobuf
rpc DialSip(DialSipReq) returns (DialSipRes);

message DialSipReq {
    // 被叫号码（必填），如 "1001" 或 "+8613800138000"
    string callee_number = 1;
    // 会议号（可选），为空则自动创建 S 前缀会议
    // 传当前会议号（M 前缀）= 电话会议场景
    string meeting_no = 2;
    // 指定 trunk ID（可选），不填则自动选择/创建
    string sip_trunk_id = 3;
    // 电话参会者显示名（可选）
    string participant_name = 4;
}

message DialSipRes {
    // 会议信息（新建或已有）
    MeetingInfo meeting = 1;
    // SIP 通话 ID
    string sip_call_id = 2;
    // 通话状态：ringing / active / failed
    string call_status = 3;
}
```

### HTTP 端点

| Method | Path | 说明 | 鉴权 |
|--------|------|------|------|
| POST | `/live/v1/sip-calls/dial` | 外呼拨号 | JWT |

### 前端入口

| 入口 | 位置 | 行为 |
|------|------|------|
| 拨打电话 | 大厅 + 会议内 | 不传 meeting_no → 自动创建 S 会议 → 拨号后跳转进入房间 |
| 邀请电话参会者 | 会议内 | 传当前 meeting_no → 拨号后留在当前房间 |

## 会议号前缀规则

| 前缀 | 场景 | 生成方式 |
|------|------|---------|
| M | 正常创建的会议 | `IdUtil.NextId("M", "live")` |
| S | 电话通话（自动创建） | `IdUtil.NextId("S", "live")` |

## 数据模型

### LiveSipTrunk

```go
type LiveSipTrunk struct {
    gormx.LegacyStringBaseModel
    gormx.VersionMixin
    Name               string         // trunk 名称
    Direction          string         // inbound/outbound/both
    Address            string         // SIP 服务器地址
    Numbers            string         // 关联号码 JSON
    AuthUsername        string         // 认证用户名
    AuthPassword        string         // 认证密码
    Provider           string         // 供应商标识
    SipTrunkID         string         // LiveKit SIP trunk ID
    Status             int32          // 1-启用 2-禁用
    CreateUser         sql.NullString
    UpdateUser         sql.NullString
    DeptCode           sql.NullString
}
```

### LiveSipCallLog

```go
type LiveSipCallLog struct {
    gormx.LegacyStringBaseModel
    CallID         string         // LiveKit SIP call ID
    TrunkID        string         // 关联 trunk ID
    Direction      string         // inbound/outbound
    CallerNumber   string         // 主叫号码
    CalleeNumber   string         // 被叫号码
    MeetingNo      string         // 关联会议号
    CallType       string         // phone(S前缀)/conference(M前缀)
    Status         string         // ringing/active/completed/failed
    StartTime      sql.NullTime
    EndTime        sql.NullTime
    Duration       int32          // 通话时长秒
    DeptCode       sql.NullString
}
```

## 自动 Trunk 管理

dial 时自动检查/创建 trunk：
1. 查询数据库是否有可用的 outbound trunk（status=1）
2. 没有 → 自动创建默认 FreeSWITCH outbound trunk（address=freeswitch:5080）
3. 缓存 trunk_id，后续 dial 复用

## 验收标准

- [ ] Proto 编译通过，生成 Go 代码
- [ ] 拨打电话：前端输入号码 → 后端自动创建 S 会议 → 软电话振铃 → 接听 → 双方通话
- [ ] 电话会议：会议内邀请电话 → 软电话接听 → 加入当前会议 → 全员通话
- [ ] call_log 正确记录（call_type 区分 phone/conference）
- [ ] 自动 trunk 管理：首次 dial 自动创建 trunk

## 技术约束

- LiveKit SDK v2.18.1
- Go 1.26, go-zero v1.10.3
- GORM + MySQL/PostgreSQL
- 局域网 FreeSWITCH 验证环境已部署（deploy/livekit/docker-compose.yaml）
