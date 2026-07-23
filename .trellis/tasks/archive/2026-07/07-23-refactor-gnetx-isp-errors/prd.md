# 重构 gnetx 客户端身份与 ISP 错误边界

## Goal

明确 `gnetx` 连接实例与客户端业务身份的边界，并解除 `common/isp`
对 gRPC 状态错误的直接依赖，使公共包错误可由不同传输入口统一处理。

## Requirements

- `gnetx` 必须区分随机生成的连接会话 ID 与客户端注册后的唯一身份。
- 客户端身份的公开命名必须符合 Go 习惯，不使用 `alias` 或 Java 风格 getter。
- 客户端身份重复绑定、冲突替换、连接关闭和并发读取必须保持索引一致且无数据竞争。
- SessionManager 必须提供无歧义的按会话 ID、按客户端 ID 查询能力。
- `common/isp` 不得直接构造或返回 gRPC `status.Error`。
- ISP 协议响应错误与本地客户端运行错误必须在包内明确定义，并支持 `errors.Is` / `errors.As`。
- ISP 请求错误必须保留底层错误链，包括 context 取消/超时和 gnetx 会话错误。
- `app/ispagent` 必须原样返回 ISP error，由现有日志拦截器记录，并交由 grpc-go 统一序列化。
- 不修改现有 proto、RPC 方法签名或 ISP 报文状态码。

## Acceptance Criteria

- [x] `common/gnetx` 不再暴露 `Alias`/`Register(alias)`/`byAlias` 命名。
- [x] 同一连接重新绑定客户端 ID 后旧 ID 不再可查询。
- [x] 相同客户端 ID 被新连接绑定时旧连接关闭，新连接可查询。
- [x] 会话 ID 与客户端 ID 的查询命名空间不再混用。
- [x] `common/isp` 不再导入 `google.golang.org/grpc/codes` 或 `status`。
- [x] ISP 包内错误有独立 `errors.go`，错误包装保留 cause。
- [x] `ExecuteCommand` 不增加自定义 gRPC 错误映射，ISP error 可由日志拦截器完整记录。
- [x] `go test -race -count=1 ./common/gnetx/ ./common/isp/` 通过。
- [x] 相关 `ispagent` 包测试和 `go vet` 通过。

## Notes

- 用户已确认按上述方向实施；本任务按 PRD-only 轻量任务执行。
