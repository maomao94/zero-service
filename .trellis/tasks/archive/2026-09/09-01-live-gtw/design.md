# livegtw 网关 — 技术设计

## 目录结构（参考 gtw/，精简）

```
app/livegtw/
├── livegtw.go              # main
├── livegtw.api             # goctl api 定义（业务路由）
├── gen.sh
├── etc/livegtw.yaml
├── internal/
│   ├── config/config.go    # rest.RestConf + JwtAuth(可选) + LiveRpcConf + WebhookKey
│   ├── handler/routes.go   # 路由注册
│   ├── handler/meeting/*.go  # 业务 handler（调 logic）
│   ├── handler/webhook/webhookhandler.go
│   ├── handler/testpage/testpagehandler.go  # embed HTML
│   ├── logic/meeting/*.go  # zrpc client 转发逻辑
│   ├── logic/webhook/webhooklogic.go
│   └── svc/servicecontext.go  # LiveRpc zrpc client、webhook KeyProvider
```

## API 契约（HTTP ↔ gRPC 映射）

| HTTP | RPC | 说明 |
|---|---|---|
| POST /live/v1/meeting/create | CreateMeeting | body: title, creatorIdentity |
| POST /live/v1/meeting/join | JoinMeeting | body: meetingNo, identity, name → {token, wsUrl} |
| GET /live/v1/meeting/detail | GetMeeting | query: meetingNo |
| GET /live/v1/meeting/list | ListMeetings | query: page, pageSize, status |
| POST /live/v1/meeting/end | EndMeeting | body: meetingNo |
| POST /live/v1/meeting/kick | KickParticipant | body: meetingNo, identity |
| POST /live/v1/meeting/mute | MuteParticipant | body: meetingNo, identity, muted, kind(audio/video) |
| GET /live/v1/meeting/participants | ListParticipants | query: meetingNo |
| POST /live/v1/meeting/sendData | SendMeetingData | body: meetingNo, topic, payload(base64), destinations[] |
| POST /live/v1/meeting/rpc | PerformMeetingRpc | body: meetingNo, identity, method, payload, responseTimeoutMs |

统一响应：`{code, msg, data}`（参考 gtw 的 JsonBaseResponseCtx 风格——检查 gtw 现有响应封装并复用模式）。

## Webhook 接收

```go
// handler/webhook
func WebhookHandler(svcCtx) http.HandlerFunc {
  return func(w, r) {
    event, err := webhook.ReceiveWebhookEvent(r, livekitx.NewWebhookKeyProvider(svcCtx.Config.WebhookKey))
    if err != nil { w.WriteHeader(401); return }   // 验签失败，不执行任何业务
    // 转发 app/live WebhookNotify（扁平字段），失败记录日志
    // 始终返回 200（LiveKit 重试语义：非 2xx 会重推）
  }
}
```

- 转发成功/失败策略：转发失败仍返回 200 会丢事件 → 返回 500 让 LiveKit 重推；幂等在 app/live 侧
- LiveKit 要求 webhook 快速响应（异步处理避免超时重推风暴）：logic 里用 go func 异步转发 + 立即 200？——MVP 同步转发（本地低延迟），超时设置 3s

## 测试页路由

- `//go:embed static/index.html` 内嵌，handler 直接写回
- 路由 `GET /test/meeting`（非 api 前缀，避免 goctl api 框架的请求解析干扰——用 rest 原生路由注册，参考 gtw 的 mfs downloadFile 原生 handler 模式）

## 配置（etc/livegtw.yaml）

```yaml
Name: livegtw
Host: 0.0.0.0
Port: 11002
Timeout: 5000
Mode: dev
Log: ...
#JwtAuth:                     # 参考 socketgtw，默认关闭
#  AccessSecret: ...
LiveRpcConf:
  Endpoints: [127.0.0.1:21017]
  NonBlock: true
  Timeout: 3000
LiveKit:
  WebhookKey: secret
```

## 测试策略

- webhook handler 单测：构造带签名请求（auth.NewAccessToken 签发含 body sha256 的 token）+ 伪造签名请求，断言 200/401
- 转发逻辑单测：mock zrpc client（接口注入）