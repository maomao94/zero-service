# livegtw 用户身份校验设计

## Boundary

`livegtw` 的 JWT 路由先由 go-zero `rest.WithJwt` 验证，再由 `MeetingAuthMiddleware` 桥接 claims 到 `authctx` 并校验非空 `user_id`。校验失败在 HTTP 网关边界结束请求，不调用 `app/live` gRPC。

票据加入路由不使用 JWT 和 `MeetingAuth`，LiveKit webhook 由独立签名校验处理，二者不受本中间件影响。

## Contract

- 接受标准 authctx claim `user-id`，也接受配置映射的外部 claim `user_id`。
- 缺少、空字符串或仅空白的 user ID 返回统一未认证错误。
- 中间件必须将成功桥接后的 context 传给下游 handler，保持现有 metadata interceptor 的传播路径。
- 不记录 Authorization header 或 token。

## Compatibility

- 不改变 JWT 密钥校验、claim mapping、票据路由和 webhook 路由。
- 不把校验下沉到 `app/live`，因为内部异步通知 RPC 明确允许没有登录上下文。
- 不新增用户服务调用；身份存在性仅以已验证 JWT claim 为准。
