# 统一 ISP 基础通信与 Handler 返回设计

## Current Evidence

- `common/isp.Message` already implements gnetx routing and request-response interfaces.
- `common/isp.NewResponse` already assembles a 251-3 / 251-4 style response from a request, but callers still need to choose `SessionSource`, command, items, and `SendSeq`.
- ISP client/server 的 TCP 通信骨架原来分散在 `app/ispagent/internal/isp` 和 `app/ispserver/internal/isp`，业务包仍暴露 gnetx client/server、router、fallback 等通信细节。
- 目标是先把基础通信下沉到 `common/isp`，再在下沉后的 `Client` / `Server` 上统一 handler 返回和 response 包装。

## Target Contract

- `common/isp` owns `Client` and `Server`基础通信对象：codec、root 校验、router、gnetx client/server、注册、心跳、请求应答、SendSeq/RecvSeq。
- 业务系统只注册协议指令 handler：`func(ctx, conn, req) (*isp.Message, error)`。
- wrapper 基于下沉后的 `Client` / `Server` 简化，不再暴露 `WrapOptions` / `ClientContext` 这类中间配置模型。
- wrapper 统一处理：类型断言、入站日志、nil→251-3、error→251-3、非 nil response 补 SendSeq。

## Server vs Client Direction

- `common/isp.Server` 使用 `ServerConfig` 和 `ServerHandler` 构造，业务服务只提供 handler 注册函数。
- `common/isp.Client` 使用 `ClientConfig` 和 `ClientHandler` 构造，业务服务只提供 handler 注册函数和注册成功后的业务回调。
- `app/ispagent/internal/isp.Client` 变为薄业务封装：内嵌 `*isp.Client`，保留任务、模型、上报缓存等业务依赖。
- `app/ispserver` 不再保留私有 TCP server 类型，只在 `internal/isp` 提供协议 handler 注册。

## Proposed API Shape

- `type ClientConfig` / `type ServerConfig`
- `func NewClient(cfg ClientConfig, opts ...ClientOption) *Client`
- `func NewServer(cfg ServerConfig, register ServerHandler) (*Server, error)`
- `type ClientHandler func(*ClientRouter)` / `type ServerHandler func(*ServerRouter)`
- `ClientRouter.Handle` / `HandlePairs`; `ServerRouter.Handle` / `Fallback`
- `Client.NewItemsResponse(req, items)` for client-side 251-4 with current RootName/SendCode/ReceiveCode.

## Migration Plan

- Move transport lifecycle from app internal packages into `common/isp`.
- Convert app configs to alias/embed common ISP config types to avoid conversion helpers.
- Keep app handler comments and protocol direction comments because they document the ISP command map.
- After migration, continue optimizing response construction and handler registration against the common client/server API.

## Compatibility

- No XML schema or wire protocol changes.
- No XML schema or wire protocol changes.
- No app package dependency from `common/isp`.
- go-zero ServiceContext still owns dependency injection; it constructs common ISP client/server using service config and business handler registration.
