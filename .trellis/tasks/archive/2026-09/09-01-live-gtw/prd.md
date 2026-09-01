# livegtw 网关

## Goal

构建 `livegtw` HTTP 网关（参考 `gtw/` 结构）：会议业务 API 转发到 app/live（zrpc client）、LiveKit webhook 接收（livekitx 验签后转发 app/live）、HTML 测试页静态路由。

## Requirements

1. 目录 `zero-service/app/livegtw/`，参考 gtw/（api 文件 + internal/handler + internal/logic + internal/svc + etc）。
2. 路由：
   - `POST /live/v1/meeting/create`、`/join`、`/end`、`/kick`、`/mute`、`/participants`、`/sendData`、`/rpc`：业务 API，转发 app/live（zrpc LiveRpcConf）
   - `GET /live/v1/meeting/list`、`/detail`：查询
   - `POST /webhook/livekit`：`livekitx.NewWebhookKeyProvider(cfg.WebhookKey)` + `webhook.ReceiveWebhookEvent` 验签，成功转发 `LiveRpc/WebhookNotify`；验签失败 401，不执行任何业务
   - `GET /test/meeting`：HTML 测试页静态路由（返回内嵌 HTML 或静态文件服务）
3. 认证：JwtAuth 结构参考 socketgtw（AccessSecret/PrevAccessSecret 可选字段），默认不启用；测试页与会议 API MVP 免认证。免认证身份经请求头 `X-User-Id`/`X-User-Name`/`X-Dept-Code` 桥接到 authctx，再经 grpcx metadata 透传给 app/live。
4. 配置：`etc/livegtw.yaml`：Host 0.0.0.0 Port 11002、LiveRpcConf(127.0.0.1:21017)、LiveKit webhook key。

## Acceptance Criteria

- [x] `go build ./livegtw/...` + `go vet ./livegtw/...` 通过
- [x] 业务 API 转发到 app/live 成功（mock zrpc 单测 + 真实联调：create/end/participants 全部 code:0，身份透传 + carbon 时间格式验证）
- [x] webhook 验签：正确签名 → 200 且转发（真实签名链路构造验证）；错误签名 → 401 且不转发（单测 + 真实请求验证）
- [x] `/test/meeting` 返回测试页 HTML（200，go:embed 内嵌）
- [x] 配置结构完整，可启动（11002 端口 + LiveRpcConf 21017 联通）
- [x] 单测：webhook 验签（含 base64 sha256 签名构造、Authorization 裸 token 格式）、转发参数映射（fake LiveRpcClient mock create/join/sendData/rpc）+ `go test -race` 全绿
- [x] 响应统一 `xhttp.JsonBaseResponseCtx`（{code,msg,data}）+ `gtwx.SetGrpcErrorHandler`（gRPC 错误码 → HTTP 转换）

## Notes

- 验签逻辑直接复用 gtw 第三方回调模式（paidnotifyhandler）
- 静态页面通过 `//go:embed` 内嵌 index.html（当前为占位，正文由 live-test-page 任务提供）
- 端口 11002 已确认空闲
- LiveKit webhook 的 `Authorization` 头格式为**裸 JWT（无 Bearer 前缀）**，且 body sha256 用 **base64** 编码（非 hex）——与 livekit protocol webhook 校验逻辑一致
- 身份链路：HTTP header `X-User-Id` → livegtw authctx → grpcx.UnaryMetadataInterceptor → grpc metadata → app/live grpcx.LoggerInterceptor → authctx.GetUserId