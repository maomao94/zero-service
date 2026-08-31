# LiveKit 视频会议对接规范

## 适用范围

修改 `common/livekitx`、`app/meeting`、LiveKit JWT/Webhook/Twirp、房间/参与者、Egress、Ingress、SIP 或 Agent 调度时读取。API 版本基线见 [对接指南](../../../docs/livekit-integration-guide.md)，本规范不把本地开发工作树当作稳定版本。

## 版本与依赖

- 审计基线为 LiveKit Server `v1.13.6`、`github.com/livekit/server-sdk-go/v2 v2.18.1`。
- Protocol 版本由选定 SDK 的 `go.mod` 解析；不得手工猜测或截断 pseudo-version。
- 本地源码提交只能作为编译/源码证据，必须与稳定 tag 分栏记录。
- 升级后重新执行核心示例的 `go mod tidy`、`go test ./...`，并检查生成字段、枚举、oneof、配置和官方链接。

## 所有权与客户端

`LiveKitAPI` 是管理 API 统一入口。服务启动时通过 `NewLiveKitAPI`、`WithURL`、`WithAPIKey` 构造并复用；通过 `.Room()`、`.Egress()`、`.Ingress()`、`.SIP()`、`.AgentDispatch()` 访问子客户端。SDK 管理调用自动处理认证，但不表示外部执行服务已部署。`ConnectToRoom`/`ConnectToRoomWithToken` 是实时参与者路径，不能与管理 API 权限混淆。

```go
// 服务启动时构造一次；密钥从安全配置注入，不写入日志或响应。
api, err := lksdk.NewLiveKitAPI(
    lksdk.WithURL(cfg.LiveKit.ServerURL),
    lksdk.WithAPIKey(cfg.LiveKit.APIKey, cfg.LiveKit.APISecret),
)
if err != nil {
    return nil, err
}
```

直接使用 Protocol 仅限 Webhook、token 或确有需要的生成 client；不要依据旧版本文档使用未经验证的 `New*JSONClient` 或字段。

## Token 与授权

- 业务服务使用 `auth.NewAccessToken` 签发短期、指定房间、最小权限的 join token；管理 token 不返回客户端。
- `VideoGrant` 的房间、创建、管理、录制、Ingress、Agent、发布/订阅和数据字段必须以锁定 Protocol 生成类型核对。
- SIP 使用 `SIPGrant`，不得写成 `VideoGrant` 字段。
- worker 注册、房间 dispatch 和 Cloud Agents 管理是不同授权边界，不能互相推导。
- API key/secret、Webhook signing key、SIP 密码只能来自安全配置。

```go
// 只允许指定身份加入指定房间；具体有效期由业务契约确定。
token, err := auth.NewAccessToken(apiKey, apiSecret).
    SetIdentity(identity).
    SetName(name).
    SetValidFor(time.Hour).
    SetVideoGrant(&auth.VideoGrant{RoomJoin: true, Room: room}).
    ToJWT()
```

## 状态与 Webhook

LiveKit 拥有房间、参与者、轨道和录制运行态；zero-service 拥有用户、会议单据、角色授权、审计和业务状态。Webhook 只触发同步或对账，不授予权限，也不替代管理 API。

使用 `webhook.ReceiveWebhookEvent` 处理原始 body 和 `Authorization`；前置中间件不能消费或改写 body。事件可能重复、迟到、乱序或丢失，消费者必须以 event ID 去重、持久化后异步处理，并对关键状态主动对账。未知事件安全记录并忽略。HTTP 响应、重试和失败传播必须与幂等实现一致，不能承诺 Exactly Once。

## 功能边界

- 房间/参与者管理使用 SDK 对应方法和最小 grant。
- Egress 的管理调用不等于录制成功；需要独立 Egress 服务、存储和网络。
- Ingress、SIP、Agent dispatch 同样需要对应服务、凭据、worker 或运营商连接。
- Cloud failover、Cloud Agents、托管 TURN/运营商能力不能推导为自托管能力。
- 请求字段、枚举、输出格式和配置默认值必须以选定版本源码和官方配置样例核对，不能复制历史固定数值。

## 错误、生命周期与反模式

- 每个外部调用设置 context 超时，保留 Twirp code/message；不要把全部错误改成“服务不可用”。
- ServiceContext 复用 client，关闭服务时释放由本任务创建的资源；异步 Webhook/对账任务必须可退出、重试和去重。
- 不在请求中重复创建 API client，不硬编码 secret，不把 Webhook 当唯一状态源，不在公共包固化业务 grant。
- 不手工拼接 Twirp 请求或绕过已验证 SDK；确需 Protocol client 时先编译验证构造函数和生成类型。

## 部署与验证

生产使用可信 TLS 的 HTTPS/WSS；WebRTC UDP/TCP、ICE、NAT、公网地址、TURN、WebSocket、Redis、多节点负载均衡和独立 Egress/Ingress/SIP/Agent 进程按官方当前配置逐项验收。开放 API 端口不等于媒体链路可用。

验证至少包括：临时 module 锁定稳定 SDK 后 `go mod tidy` 与 `go test ./...`；Token/Webhook/API 的成功、权限失败、超时、重复和边界测试；`git diff --check` 以及文档链接、版本、旧 API、secret 和个人路径扫描。没有 Server、凭据、浏览器媒体、TURN、Redis 集群或外部运营商时，必须报告未完成，不得声称端到端通过。
