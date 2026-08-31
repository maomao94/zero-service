# Research: LiveKit 学习/开发指南审计

- **Query**: 审阅 `docs/livekit-integration-guide.md` 与 `.trellis/spec/backend/livekit-guidelines.md`，以本地 `server-sdk-go` 和 `livekit` 源码为权威依据，找出妨碍文档成为可靠中文学习/开发指南的问题。
- **Scope**: mixed（本地文档、`server-sdk-go`、`livekit`、protocol module cache）
- **Date**: 2026-08-31

## Findings

### 严重性：阻断示例编译/运行

1. **Egress StopEgress 示例使用不存在的 `RoomName` 字段。**
   - 文档：[docs/livekit-integration-guide.md:145-147](zero-service/docs/livekit-integration-guide.md:145) 同时传入 `EgressId` 和 `RoomName`。
   - 证据：protocol 生成类型 [`livekit_egress.pb.go:2715-2720`](protocol module cache/livekit/livekit_egress.pb.go:2715) 的 `StopEgressRequest` 只有 `EgressId`；SDK [`egressclient.go:119-124`](server-sdk-go/egressclient.go:119) 也只调用该字段。Protocol 方式的同一错误还出现在 [docs/livekit-integration-guide.md:698-701](zero-service/docs/livekit-integration-guide.md:698)。
   - 修正建议：删除 `RoomName`，并明确停止请求只需要 Egress ID。

2. **SIP Dispatch Rule 示例使用了当前生成代码中不存在的 oneof 类型和字段。**
   - 文档：[docs/livekit-integration-guide.md:742-751](zero-service/docs/livekit-integration-guide.md:742) 使用 `SIPDispatchRuleInfo.TrunkId`、`SIPDispatchRule_Rule`、`SIPDispatchRule_RuleDispatchToIndividual`、`RoomPrefix`。
   - 证据：当前 `SIPDispatchRuleInfo` 的字段是 `TrunkIds []string`，见 [`livekit_sip.pb.go:3354-3371`](protocol module cache/livekit/livekit_sip.pb.go:3354)。oneof 包装类型是 `SIPDispatchRule_DispatchRuleIndividual`，其字段是 `DispatchRuleIndividual`，见 [`livekit_sip.pb.go:3007-3016`](protocol module cache/livekit/livekit_sip.pb.go:3007) 和 [`livekit_sip.pb.go:3087-3108`](protocol module cache/livekit/livekit_sip.pb.go:3087)。当前 individual 规则结构及 `RoomPrefix` 见 [`livekit_sip.pb.go:2884-2925`](protocol module cache/livekit/livekit_sip.pb.go:2884)。
   - 修正建议：按当前 generated protobuf oneof 写示例，或使用新式 `SIPDispatchRuleInfo.Rule` 结构；同时把 `TrunkId` 改为 `TrunkIds`。

3. **“完整集成示例”没有成为可编译示例。**
   - 文档：[docs/livekit-integration-guide.md:930-945](zero-service/docs/livekit-integration-guide.md:930) 的 `roomClient livekit.RoomService` 与 JSON client 是可对应的 protocol API，但 [docs/livekit-integration-guide.md:1041-1045](zero-service/docs/livekit-integration-guide.md:1041) 忽略 `twirp.WithHTTPRequestHeaders` 的错误；更关键的是示例依赖未展示的 `CreateMeetingLogic` 之外，错误处理小节将 `twirp.InternalError` 作为可直接构造的错误码示例，且先创建未使用的 `twirpErr`（[docs/livekit-integration-guide.md:1056-1059](zero-service/docs/livekit-integration-guide.md:1056)），会触发 Go 编译器的 unused variable。
   - 证据：SDK 自己的认证路径使用 `twirp.WithHTTPRequestHeaders` 并处理上下文合并，见 [`auth.go:80-105`](server-sdk-go/auth.go:80)。
   - 修正建议：将代码改成可独立编译的最小 package，处理或返回 header 错误；删除未使用的 `twirpErr`，并使用真实可用的 Twirp 错误判定方式。

### 严重性：认证/授权语义会导致错误实现

4. **文档把 SDK 自动认证和自动重试描述成普遍能力，实际 failover 只对 LiveKit Cloud 生效。**
   - 文档：[docs/livekit-integration-guide.md:12](zero-service/docs/livekit-integration-guide.md:12)、[docs/livekit-integration-guide.md:29](zero-service/docs/livekit-integration-guide.md:29)、[docs/livekit-integration-guide.md:367](zero-service/docs/livekit-integration-guide.md:367)；spec：[livekit-guidelines.md:33-39](zero-service/.trellis/spec/backend/livekit-guidelines.md:33)。
   - 证据：SDK 每个管理方法确实在调用前按 grant 签发 token，见 [`roomclient.go:49-55`](server-sdk-go/roomclient.go:49) 和 [`auth.go:62-78`](server-sdk-go/auth.go:62)。但 failover 的 host 判断明确只接受 Cloud 域名，见 [`failover.go:71-79`](server-sdk-go/failover.go:71)；失败类型是传输错误/5xx，4xx 立即返回，见 [`failover.go:177-181`](server-sdk-go/failover.go:177)。
   - 修正建议：分别说明 API key/secret 自动签发、可选 `WithToken` 的固定 token 模式，以及 Cloud 专属区域 failover；自托管不能据此推断跨区域重试。

5. **Agent token 说明混淆了参与者 Agent grant 与 Cloud Agent 管理 grant。**
   - 文档：[docs/livekit-integration-guide.md:321-328](zero-service/docs/livekit-integration-guide.md:321)、spec：[livekit-guidelines.md:186-189](zero-service/.trellis/spec/backend/livekit-guidelines.md:186)。
   - 证据：`VideoGrant.Agent` 的注释是允许注册 Agent framework worker，见 [`grants.go:268-276`](protocol module cache/auth/grants.go:268)。独立的 `AgentGrant` 具有 `Admin`、`SimulationAdmin`、`DatabaseAdmin`，见 [`grants.go:579-586`](protocol module cache/auth/grants.go:579)。当前 SDK Agent Dispatch 客户端本身使用的是 `VideoGrant{RoomAdmin:true, Room:req.Room}`，见 [`agent_dispatch_client.go:48-70`](server-sdk-go/agent_dispatch_client.go:48)。
   - 修正建议：按“Agent worker 连接”“房间 Agent dispatch”“Cloud Agents API”分开解释，分别给出 `VideoGrant.Agent`、房间管理权限和 `AgentGrant` 的适用边界。

6. **Spec 的 `common/livekitx` webhook API 仍提前承诺未实现的公共接口。**
   - spec：[livekit-guidelines.md:41-43](zero-service/.trellis/spec/backend/livekit-guidelines.md:41) 声明不是现有 API，但 [livekit-guidelines.md:64-87](zero-service/.trellis/spec/backend/livekit-guidelines.md:64) 仍给出 `WebhookHandler`、`HandleHTTP`、`OnEvent` 的固定导出签名；[livekit-guidelines.md:234-260](zero-service/.trellis/spec/backend/livekit-guidelines.md:234) 又直接调用不存在的 `livekitx.NewLiveKitAPI(livekitx.Config{...})`。
   - 证据：当前 SDK 构造入口是 [`livekitapi.go:65-114`](server-sdk-go/livekitapi.go:65)，不存在 `livekitx` 公共包的证据；SDK/protocol webhook 只提供 [`verifier.go:69-83`](protocol module cache/webhook/verifier.go:69) 的 `ReceiveWebhookEvent`。
   - 修正建议：spec 只保留边界/验收条件，删除看似已批准的导出签名和构造函数；实现任务中再以实际调用方确定 API。

### 严重性：Webhook 交付语义不完整或不准确

7. **Webhook 验签实现说明缺少请求体、算法和响应/失败约束。**
   - 文档：[docs/livekit-integration-guide.md:842-865](zero-service/docs/livekit-integration-guide.md:842)、spec：[livekit-guidelines.md:217-220](zero-service/.trellis/spec/backend/livekit-guidelines.md:217)。
   - 证据：`Receive` 会读取并关闭 body，读取 `Authorization`，解析 JWT，按 API key 找 secret，并校验 body SHA-256 的 base64 checksum，见 [`verifier.go:29-66`](protocol module cache/webhook/verifier.go:29)。`ReceiveWebhookEvent` 使用 protojson 且 `DiscardUnknown:true`、`AllowPartial:true`，见 [`verifier.go:69-83`](protocol module cache/webhook/verifier.go:69)。
   - 修正建议：学习指南应说明不能先消费 body 再验签、验签失败应返回 4xx、成功应快速返回 2xx 并异步处理；不要把 `application/webhook+json`、JWT Authorization 和 checksum 验证省略为“验签”。

8. **“记录 event ID、按事件 ID 幂等”是正确方向，但文档没有说明事件 ID 的字段/唯一性边界，也没有说明 LiveKit 队列会丢弃事件。**
   - 文档：[docs/livekit-integration-guide.md:1131-1137](zero-service/docs/livekit-integration-guide.md:1131)、spec：[livekit-guidelines.md:220](zero-service/.trellis/spec/backend/livekit-guidelines.md:220)。示例只按 `event.Event` switch，不读取 `event.Id`（[docs/livekit-integration-guide.md:851-864](zero-service/docs/livekit-integration-guide.md:851)）。
   - 证据：发送日志和处理信息使用 `event.Id` 作为 event ID，见 [`notifier.go:227-232`](protocol module cache/webhook/notifier.go:227)；但资源队列按资源串行、失败重试，并在事件过旧或队列过深时 drop，见 [`resource_url_notifier.go:87-91`](protocol module cache/webhook/resource_url_notifier.go:87) 和默认 `MaxAge=30s`、`MaxDepth=200`，见 [`resource_url_notifier.go:48-56`](protocol module cache/webhook/resource_url_notifier.go:48)。
   - 修正建议：明确消费端应持久化 `event.Id`，用唯一约束/去重表实现幂等；同时说明至少一次并非 exactly-once，事件可能重复、迟到或因发送队列积压而丢失，关键状态需 API 对账。

9. **Webhook 事件表不完整，且学习指南没有覆盖 Ingress、Agent 和连接中止事件。**
   - 文档：[docs/livekit-integration-guide.md:899-911](zero-service/docs/livekit-integration-guide.md:899)、spec：[livekit-guidelines.md:203-215](zero-service/.trellis/spec/backend/livekit-guidelines.md:203)。
   - 证据：当前常量还包括 `participant_connection_aborted`、`ingress_started`、`ingress_ended`、`agent_job_started`、`agent_job_ended`，见 [`consts.go:37-51`](protocol module cache/webhook/consts.go:37)。
   - 修正建议：把事件表标为非穷举或补齐当前 protocol 版本事件，并说明未知事件必须可安全忽略/记录。

### 严重性：Cloud、自托管与独立服务边界不清

10. **自托管配置示例把 Egress/Ingress/SIP/Agent 的“独立服务”概念写成注释，缺少可验证的前置条件和失败表现。**
    - 文档：[docs/livekit-integration-guide.md:1080-1129](zero-service/docs/livekit-integration-guide.md:1080)、spec：[livekit-guidelines.md:127-175](zero-service/.trellis/spec/backend/livekit-guidelines.md:127)。
    - 证据：Server 在创建带 Egress 配置的房间时，若 launcher 未连接直接返回 `ErrEgressNotConnected`，见 [`roomservice.go:73-81`](livekit/pkg/service/roomservice.go:73)；Egress 启动器 client 为空同样返回该错误，见 [`egress.go:237-240`](livekit/pkg/service/egress.go:237)。LiveKit 的 Agent dispatch 服务会校验 deployment 并通过内部 agent dispatch client 执行，见 [`agent_dispatch_service.go:56-95`](livekit/pkg/service/agent_dispatch_service.go:56)。
    - 修正建议：分别列出核心 LiveKit Server、Redis、多节点依赖和 Egress/Ingress/SIP/Agent worker 的部署/连接要求；区分 Cloud-only 字段，例如 generated request 中 `RestartPolicy` 注释为 cloud only，见 [`livekit_agent_dispatch.pb.go:85-92`](protocol module cache/livekit/livekit_agent_dispatch.pb.go:85)。不要让 `egress:` YAML 注释看起来像完成了 Egress 部署。

11. **SIP 权限与调用超时说明过于固定。**
    - 文档：[docs/livekit-integration-guide.md:311-319](zero-service/docs/livekit-integration-guide.md:311)、[docs/livekit-integration-guide.md:755-767](zero-service/docs/livekit-integration-guide.md:755)；spec：[livekit-guidelines.md:145-160](zero-service/.trellis/spec/backend/livekit-guidelines.md:145)。
    - 证据：当前 SDK 创建外呼使用 `SIPGrant{Call:true}`，转接才组合 `Call:true` 与 `VideoGrant{RoomAdmin:true, Room:...}`，见 [`sipclient.go:311-353`](server-sdk-go/sipclient.go:311)。SDK 会把未指定的 `RingingTimeout` 固定为内部默认值，并据此构造超时，见 [`sipclient.go:56-64`](server-sdk-go/sipclient.go:56) 和 [`sipclient.go:321-325`](server-sdk-go/sipclient.go:321)，不是文档所写的固定“默认 30s、等待 80s”的通用契约。
    - 修正建议：以当前 SDK/protocol 的 `RingingTimeout` 为准说明可配置超时，避免把某一服务版本的默认值写成稳定 API 契约。

### 严重性：版本、凭据和环境可复现性风险

12. **版本声明互相不一致且没有锁定“文档示例验证所用版本”。**
    - 文档分别写 `server-sdk-go/v2 v2.18.2`（[docs/livekit-integration-guide.md:24-29](zero-service/docs/livekit-integration-guide.md:24)）和 `protocol v1.50.5`（[docs/livekit-integration-guide.md:245-248](zero-service/docs/livekit-integration-guide.md:245)）；spec 使用 SDK `v2.18.2`（[livekit-guidelines.md:226-232](zero-service/.trellis/spec/backend/livekit-guidelines.md:226）。
    - 证据：本地 SDK `version.go` 是 `2.18.2`，见 [`version.go:15-20`](server-sdk-go/version.go:15)，但其 `go.mod` 实际依赖 protocol pseudo-version `v1.50.5-0.20260829123501-a469dd43727b`，见 [`go.mod:1-14`](server-sdk-go/go.mod:1)。本地 SDK 还声明 `go 1.26.3`，见 [`go.mod:1-4`](server-sdk-go/go.mod:1)。LiveKit Server 当前版本常量是 `1.13.6`，见 [`version.go:15-18`](livekit/version/version.go:15)。
    - 修正建议：声明示例验证矩阵（Go、SDK、protocol、Server），说明 SDK 的 protocol 由 MVS/依赖图解析；不要暗示单独 `go get protocol@v1.50.5` 就等同于 SDK 实际依赖。

13. **示例含有可被误复制的凭据和敏感 SIP 配置，虽是占位符但缺少安全边界。**
    - 文档：[docs/livekit-integration-guide.md:35-41](zero-service/docs/livekit-integration-guide.md:35)、[docs/livekit-integration-guide.md:713-720](zero-service/docs/livekit-integration-guide.md:713)、[docs/livekit-integration-guide.md:727-735](zero-service/docs/livekit-integration-guide.md:727)、[docs/livekit-integration-guide.md:1093-1100](zero-service/docs/livekit-integration-guide.md:1093)。
    - 证据：SDK 支持 `LIVEKIT_URL`、`LIVEKIT_TOKEN`、`LIVEKIT_API_KEY`、`LIVEKIT_API_SECRET` 环境变量回退，见 [`livekitapi.go:65-93`](server-sdk-go/livekitapi.go:65)。LiveKit Server 要求配置 keys 才能启用安全安装，见 [`wire.go:156-160`](livekit/pkg/service/wire.go:156)。
    - 修正建议：所有凭据示例使用明确不可用的变量引用或 secret manager 注入；说明不要把 SIP password、API secret 放进源码、响应、日志或提交记录。当前审阅未发现真实凭据，但占位字符串仍不应被当作可运行生产值。

14. **个人绝对路径风险目前未出现在两份目标文档，但文档索引路径不可直接从业务仓库解析。**
    - 文档：[docs/livekit-integration-guide.md:1139-1146](zero-service/docs/livekit-integration-guide.md:1139) 使用 `server-sdk-go/...`、`livekit/pkg/...` 相对外部仓库路径；spec 也引用 `github.com/livekit/protocol/auth` 和本地 server 路径语义，但没有仓库 URL/版本锚点。
    - 修正建议：保留 repo-relative 的源码证据时同时提供仓库、commit/tag 和可点击 URL；避免把 `/Users/...` 等个人路径写进文档。当前两份目标文件未检测到 `/Users/` 绝对路径或真实 secret。

### 严重性：学习路径与错误排查缺口

15. **学习指南把后端管理 API、前端 token、实时 Bot/Agent 连接和 Webhook 放在连续示例中，但缺少最小依赖闭环。**
    - 文档：[docs/livekit-integration-guide.md:16-227](zero-service/docs/livekit-integration-guide.md:16) 与 [docs/livekit-integration-guide.md:231-430](zero-service/docs/livekit-integration-guide.md:231)。
    - 证据：管理 API 的 SDK 是 Protobuf client 并共享 HTTP client，见 [`livekitapi.go:97-107`](server-sdk-go/livekitapi.go:97)；实时连接则是 WebSocket/RTC `ConnectToRoom`，见 [`room.go:371-382`](server-sdk-go/room.go:371)。两者认证对象、网络端口和故障面不同。
    - 修正建议：明确学习顺序：Server 健康检查 -> 服务端签发 join token -> 浏览器/Go participant 加入并验证媒体 -> 管理 API 查询/变更 -> Webhook 对账 -> 最后再启用 Egress/Ingress/SIP/Agent；每一步给出预期结果与失败定位。

16. **错误排查章节没有覆盖最常见的网络和服务可用性分层。**
    - 文档只给 Twirp 错误码示例（[docs/livekit-integration-guide.md:1051-1074](zero-service/docs/livekit-integration-guide.md:1051)）和简单 curl/参与者闭环（[docs/livekit-integration-guide.md:1131-1137](zero-service/docs/livekit-integration-guide.md:1131)。
    - 证据：服务端将 Egress 未连接区分为 `ErrEgressNotConnected`，见 [`roomservice.go:79-81`](livekit/pkg/service/roomservice.go:79)；Webhook 还可能因 API key 缺失而启动失败，见 [`wire.go:163-171`](livekit/pkg/service/wire.go:163)。SDK Cloud failover 只处理 transport/5xx，不处理 4xx，见 [`failover.go:177-181`](server-sdk-go/failover.go:177)。
    - 修正建议：增加按层排查表：URL/HTTP/Twirp、JWT grant/房间范围、WebSocket/ICE/UDP/TURN、Redis/多节点、独立 Egress/Ingress/SIP/Agent、Webhook Authorization/checksum/队列丢弃；同时说明 401/403、404、InvalidArgument、503 和媒体连接失败不是同一类问题。

## Caveats / Not Found

- 本报告只写入当前任务 `research/`；未修改 docs、spec、源码或其他任务目录。
- 当前本地 `server-sdk-go` 与 protocol module cache 处于 2026-08-29/31 的开发依赖状态；结论以这些源码为准，正式文档仍需把版本/commit 固定到可复现矩阵。
- 未发现两份目标文件中存在 `/Users/...` 个人绝对路径或真实 API/SIP 凭据；发现的是相对外部仓库索引和看似可复制的占位凭据。
- 未审计官方网页内容或未提供的业务 `common/livekitx` 实现；本地 zero-service 中若存在其他分支/未检出的实现，本报告无法证明其状态。
