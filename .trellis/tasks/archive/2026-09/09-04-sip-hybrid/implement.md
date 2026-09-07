# SIP 拨号 — gRPC 基础模块实现计划

## 范围

只做 gRPC 侧：Proto + Model + livekitx 辅助 + DialSip Logic + Repo。
HTTP 网关、前端 UI、Webhook 扩展后续再做。

## Step 1: Proto 定义

在 `app/live/live.proto` 新增：

```protobuf
rpc DialSip(DialSipReq) returns (DialSipRes);

message DialSipReq {
    string callee_number = 1;      // 被叫号码（必填）
    string meeting_no = 2;         // 会议号（可选，空=S前缀自动创建）
    string sip_trunk_id = 3;       // trunk ID（可选，空=自动选择/创建）
    string participant_name = 4;   // 显示名（可选）
}

message DialSipRes {
    MeetingInfo meeting = 1;       // 会议信息
    string sip_call_id = 2;       // SIP 通话 ID
    string call_status = 3;       // ringing/active/failed
}
```

同时新增 `SipTrunkInfo` 和 `SipCallInfo` message（供后续 List 接口复用）。

**验证**：`goctl rpc protoc` 编译 + `cd app/live && go build ./...`

---

## Step 2: 数据模型

新增文件：
- `app/live/model/gormmodel/sip_trunk.go` — `LiveSipTrunk`
- `app/live/model/gormmodel/sip_call_log.go` — `LiveSipCallLog`

ServiceContext 注册 auto-migrate。

**验证**：`cd app/live && go build ./...`

---

## Step 3: livekitx SIP 辅助

新增 `common/livekitx/sip.go`：
- `EnsureOutboundTrunk(ctx, API, name, address, numbers)` — 查询/创建 trunk
- `DialParticipant(ctx, API, trunkID, callTo, roomName, opts)` — CreateSIPParticipant 封装

**验证**：`go test -v -run TestSIP ./common/livekitx/`

---

## Step 4: Repo 层

新增 `app/live/internal/repo/sip_repo.go`：
- `CreateSipTrunk` / `GetSipTrunk` / `ListSipTrunks`
- `CreateSipCallLog` / `UpdateSipCallLog` / `GetSipCallLog`

**验证**：`cd app/live && go build ./...`

---

## Step 5: DialSip Logic

新增 `app/live/internal/logic/dialsiplogic.go`：
1. meeting_no 为空 → `IdUtil.NextId("S", "live")` + CreateMeeting
2. meeting_no 不为空 → 校验会议存在且进行中
3. 自动选择/创建 trunk（EnsureOutboundTrunk）
4. DialParticipant → 写 call_log → 返回

**验证**：`cd app/live && go build ./...`

---

## 依赖关系

```
Step 1 (Proto) → Step 2 (Model) → Step 4 (Repo) → Step 5 (Logic)
                                ↗
              Step 3 (livekitx) ↗
```

Step 3 可与 Step 1/2 并行。

## 文件清单

| 类型 | 文件 | 说明 |
|------|------|------|
| 修改 | `app/live/live.proto` | 新增 DialSip RPC + message |
| 新增 | `app/live/model/gormmodel/sip_trunk.go` | trunk 模型 |
| 新增 | `app/live/model/gormmodel/sip_call_log.go` | call_log 模型 |
| 新增 | `common/livekitx/sip.go` | SIP 辅助函数 |
| 新增 | `app/live/internal/repo/sip_repo.go` | trunk + call_log CRUD |
| 新增 | `app/live/internal/logic/dialsiplogic.go` | 拨号逻辑 |
| 修改 | `app/live/internal/svc/servicecontext.go` | auto-migrate + repo 注入 |
