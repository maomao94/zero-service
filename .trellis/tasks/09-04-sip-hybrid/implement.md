# SIP 电话混合会议 — 实现计划

## 执行顺序

按垂直切片推进，每步可独立验证。

### Step 1: SDK 验证 — livekitx SIP API 测试

**目标**：验证现有 `livekitx` SIP API 可用，确认 LiveKit SIP Server 已部署。

**任务**：
1. 确认 LiveKit SIP Server 已部署（Docker Compose 或单独容器）
2. 编写 `common/livekitx/sip_example_test.go`，测试以下 API：
   - `CreateSIPInboundTrunk` — 创建 inbound trunk
   - `CreateSIPOutboundTrunk` — 创建 outbound trunk
   - `CreateSIPDispatchRule` — 创建 dispatch rule
   - `ListSIPTrunk` — 列出 trunk
   - `DeleteSIPTrunk` — 删除 trunk
3. 确认 API 调用成功，日志输出 trunk/rule ID

**验证命令**：
```bash
go test -v -run TestSIP ./common/livekitx/ -count=1
```

**产物**：`common/livekitx/sip_example_test.go`

---

### Step 2: Proto 定义 — live.proto 新增 SIP RPC

**目标**：定义 SIP 管理和通话 RPC，编译生成 Go 代码。

**任务**：
1. 在 `app/live/live.proto` 中新增以下 RPC 和 message：
   - `CreateSipTrunk` / `ListSipTrunks` / `DeleteSipTrunk`
   - `CreateSipDispatchRule` / `ListSipDispatchRules` / `DeleteSipDispatchRule`
   - `DialSip` / `HangupSip` / `ListSipCalls`
   - 对应的 Request/Response message 和 `SipTrunkInfo` / `SipDispatchRuleInfo` / `SipCallInfo`
2. 编译 proto：
   ```bash
   goctl rpc protoc app/live/live.proto --go_out=./app/live --go-grpc_out=./app/live \
     --zrpc_out=./app/live --style=go_zero
   ```
3. 确认编译无错误，生成的 Go 代码可编译通过

**验证命令**：
```bash
cd app/live && go build ./...
```

**产物**：`app/live/live.pb.go`, `app/live/live_grpc.pb.go`（goctl 重新生成）

---

### Step 3: 数据模型 — SIP 相关 GORM 模型

**目标**：创建 SIP trunk、dispatch rule、call log 的数据库模型。

**任务**：
1. 创建 `app/live/model/gormmodel/sip_trunk.go`
   - `LiveSipTrunk` 结构体，字段与 proto `SipTrunkInfo` 对齐
   - `TableName()` 返回 `live_sip_trunk`
2. 创建 `app/live/model/gormmodel/sip_dispatch_rule.go`
   - `LiveSipDispatchRule` 结构体
   - `TableName()` 返回 `live_sip_dispatch_rule`
3. 创建 `app/live/model/gormmodel/sip_call_log.go`
   - `LiveSipCallLog` 结构体
   - `TableName()` 返回 `live_sip_call_log`
4. 在 `app/live/internal/svc/servicecontext.go` 中注册 auto-migrate

**验证命令**：
```bash
cd app/live && go build ./...
```

**产物**：3 个 model 文件

---

### Step 4: livekitx SIP 辅助函数

**目标**：封装 SIP 操作为业务友好的辅助函数。

**任务**：
1. 创建 `common/livekitx/sip.go`，提供：
   - `CreateOutboundTrunk(ctx, API, name, address, numbers, opts)` — 封装 trunk 创建
   - `CreateDispatchRule(ctx, API, name, trunkIDs, roomPattern, pin)` — 封装 rule 创建
   - `DialParticipant(ctx, API, trunkID, callTo, roomName, opts)` — 封装外呼
   - `HangupCall(ctx, API, callID)` — 封装挂断（注：LiveKit SIP API 无直接 Hangup，
     需通过 `RemoveParticipant` 实现）
2. 编写单元测试

**验证命令**：
```bash
go test -v -run TestSIP ./common/livekitx/ -count=1
```

**产物**：`common/livekitx/sip.go`, `common/livekitx/sip_test.go`

---

### Step 5: 业务逻辑 — SIP Trunk CRUD

**目标**：实现 trunk 创建、列出、删除的业务逻辑。

**任务**：
1. 创建 `app/live/internal/logic/createsiptrunklogic.go`
   - 调用 livekitx 创建 trunk（根据 direction 调 inbound/outbound/both）
   - 持久化到数据库
2. 创建 `app/live/internal/logic/listsiptrunkslogic.go`
   - 从数据库查询，支持 direction/provider/status 过滤
3. 创建 `app/live/internal/logic/deletesiptrunklogic.go`
   - 调用 livekitx 删除 trunk
   - 数据库软删除

**验证命令**：
```bash
cd app/live && go build ./...
```

**产物**：3 个 logic 文件

---

### Step 6: 业务逻辑 — Dispatch Rule CRUD

**目标**：实现路由规则创建、列出、删除。

**任务**：
1. 创建 `app/live/internal/logic/createsipdispatchrulelogic.go`
   - 调用 livekitx 创建 dispatch rule（根据 rule_type 构造 SIPDispatchRule）
   - 持久化到数据库
2. 创建 `app/live/internal/logic/listsipdispatchruleslogic.go`
3. 创建 `app/live/internal/logic/deletesipdispatchrulelogic.go`

**验证命令**：
```bash
cd app/live && go build ./...
```

**产物**：3 个 logic 文件

---

### Step 7: 业务逻辑 — SIP 通话操作

**目标**：实现外呼拨号、挂断、通话记录查询。

**任务**：
1. 创建 `app/live/internal/logic/dialsiplogic.go`
   - 若 `meeting_no` 为空，调用现有 `CreateMeeting` 逻辑创建新会议
   - 调用 livekitx `CreateSIPParticipant` 发起外呼
   - 写 `live_sip_call_log` 记录
2. 创建 `app/live/internal/logic/hangupsiplogic.go`
   - 调用 livekitx `RemoveParticipant` 挂断
   - 更新 call log 状态
3. 创建 `app/live/internal/logic/listsipcallslogic.go`
   - 从数据库查询通话记录

**验证命令**：
```bash
cd app/live && go build ./...
```

**产物**：3 个 logic 文件

---

### Step 8: HTTP 网关 — SIP API 端点

**目标**：暴露 SIP 管理和通话 API 给前端调用。

**任务**：
1. 在 `app/livegtw/internal/types/types.go` 中新增 SIP 请求/响应类型
2. 创建 SIP HTTP handler：
   - `app/livegtw/internal/handler/live/createsiptrunkhandler.go`
   - `app/livegtw/internal/handler/live/listsiptrunkshandler.go`
   - `app/livegtw/internal/handler/live/deletesiptrunkhandler.go`
   - `app/livegtw/internal/handler/live/createsipdispatchrulehandler.go`
   - `app/livegtw/internal/handler/live/listsipdispatchruleshandler.go`
   - `app/livegtw/internal/handler/live/deletesipdispatchrulehandler.go`
   - `app/livegtw/internal/handler/live/dialsiphandler.go`
   - `app/livegtw/internal/handler/live/hangupsiphandler.go`
   - `app/livegtw/internal/handler/live/listsipcallshandler.go`
3. 在 `app/livegtw/internal/handler/routes.go` 中注册 SIP 路由
4. 网关 logic 层（`app/livegtw/internal/logic/live/`）调用 gRPC client

**验证命令**：
```bash
cd app/livegtw && go build ./...
# 手动 curl 测试 API
curl -X POST http://localhost:11002/live/v1/sip-trunks \
  -H "Authorization: Bearer <token>" \
  -d '{"name":"test","direction":"outbound","address":"192.168.1.100:5080","numbers":["1000"]}'
```

**产物**：9 个 handler 文件，9 个 logic 文件，types 更新

---

### Step 9: Webhook 扩展 — SIP 事件处理

**目标**：处理 LiveKit SIP 相关的 Webhook 事件。

**任务**：
1. 在现有 `app/live/internal/logic/webhooknotifylogic.go` 中扩展处理：
   - `participant_joined` — SIP 参与者加入时更新 call log 状态为 active
   - `participant_left` — SIP 参与者离开时更新 call log 状态为 completed
   - `sip_call_status` — SIP 呼叫状态变化（如果 LiveKit 推送此事件）
2. 更新 `live_sip_call_log` 记录

**验证命令**：
```bash
cd app/live && go build ./...
```

**产物**：修改 webhook 逻辑

---

### Step 10: 部署配置 — LiveKit SIP Server

**目标**：配置 LiveKit SIP Server，局域网可用。

**任务**：
1. 创建 `deploy/livekit/docker-compose.yaml`：
   ```yaml
   version: "3.9"
   services:
     redis:
       image: redis:7-alpine
       ports:
         - "6379:6379"
     livekit-server:
       image: livekit/livekit-server:v2.18.1
       ports:
         - "7880:7880"
         - "50000-50100:50000-50100/udp"
       volumes:
         - ./livekit-server.yaml:/etc/livekit.yaml
       command: --config /etc/livekit.yaml
       depends_on:
         - redis
     livekit-sip:
       image: livekit/sip:latest
       ports:
         - "5060:5060/udp"
         - "10000-10100:10000-10100/udp"
       volumes:
         - ./sip.yaml:/etc/sip.yaml
       command: --config /etc/sip.yaml
       environment:
         - LIVEKIT_URL=ws://livekit-server:7880
         - LIVEKIT_API_KEY=devkey
         - LIVEKIT_API_SECRET=secret
       depends_on:
         - redis
         - livekit-server
   ```
2. 创建 `deploy/livekit/sip.yaml`：
   ```yaml
   api_key: devkey
   api_secret: secret
   ws_url: ws://livekit-server:7880
   redis:
     address: redis:6379
   sip_port: 5060
   rtp_port: 10000-10100
   use_external_ip: false
   logging:
     level: debug
   ```
3. 更新 `deploy/livekit/livekit-server.yaml` 添加 webhook 配置（已有）

**验证命令**：
```bash
cd deploy/livekit && docker compose up -d
docker compose ps  # 确认三个服务都 running
```

**产物**：`deploy/livekit/docker-compose.yaml`, `deploy/livekit/sip.yaml`

---

## 回滚策略

- 每个 Step 独立，失败不影响已完成的 Step
- Step 2（Proto 编译）如果出错，`git checkout app/live/live.pb.go app/live/live_grpc.pb.go`
- Step 10（Docker Compose）如果出错，`docker compose down` 清理

## 依赖关系

```
Step 1 (SDK 验证)
    ↓
Step 2 (Proto 定义)
    ↓
Step 3 (数据模型) ← 需要 Step 2 的生成代码
    ↓
Step 4 (livekitx 辅助函数) ← 需要 Step 1 验证通过
    ↓
Step 5 (Trunk CRUD) ← 需要 Step 3, 4
    ↓
Step 6 (Rule CRUD) ← 需要 Step 3, 4
    ↓
Step 7 (通话操作) ← 需要 Step 3, 4, 5
    ↓
Step 8 (HTTP 网关) ← 需要 Step 5, 6, 7
    ↓
Step 9 (Webhook) ← 需要 Step 7
    ↓
Step 10 (部署) ← 可与 Step 1 并行
```

## 总文件变更

| 类型 | 数量 | 文件列表 |
|------|------|---------|
| 新增 Logic | 9 | createsiptrunk, listsiptrunks, deletesiptrunk, createsipdispatchrule, listsipdispatchrules, deletesipdispatchrule, dialsip, hangupsip, listsipcalls |
| 新增 Model | 3 | sip_trunk, sip_dispatch_rule, sip_call_log |
| 新增 Handler | 9 | 对应 9 个 HTTP 端点 |
| 新增 Gateway Logic | 9 | 对应 9 个 HTTP 端点 |
| 新增 livekitx | 2 | sip.go, sip_test.go |
| 新增 测试 | 2 | sip_example_test.go, sip_test.go |
| 新增 部署 | 2 | docker-compose.yaml, sip.yaml |
| 修改 Proto | 1 | live.proto |
| 修改 Types | 1 | types.go |
| 修改 Routes | 1 | routes.go |
| 修改 ServiceContext | 1 | auto-migrate |
| **总计** | **~40 文件** | |
