# 日志脱敏实施计划（L1-L3）

## 前置

- [x] 用户审查并确认本 PRD / design / implement 后 `task.py start`（开发前人工门禁，已通过）。
- [x] 确认前三个子任务已归档（audit、typed-auth-context-keys、normalize-auth-claims）。

## 执行清单

### 阶段 1 - 日志脱敏 L1-L3
- [x] L1 `facade/streamevent/internal/logic/upsocketmessagelogic.go:30-31`：`token: %s` → `auth_present=%t user_id=%s`。
- [x] L2 `aiapp/mcpserver/internal/tools/echo.go:25-28`：去 token 值，保留 `username`。
- [x] L3 `common/mcpx/auth.go:65`：`extra=%v` → `extraKeys=%v`（仅键名，新增 mapKeys helper）；line 61 `extra[authorization]=token` 保留（meta 契约）。

### 阶段 2 - 验证
- [x] `common/mcpx` 测试通过（`auth_test.go:46-48` Extra 契约保持）。
- [x] `facade/streamevent`/`aiapp/mcpserver` 编译通过（无测试文件）。
- [ ] 全仓构建与 diff 检查。

> 设计修订（用户确认）：接收端 report-only 观测为过度设计，已移除，不做观测。

## 验证命令

```bash
gofmt -w <changed-go-files>
go test -count=1 ./common/mcpx ./common/grpcx ./common/authctx
go build ./...
go vet ./common/mcpx ./common/grpcx
git diff --check
```

## 人工门禁

- 开发前：用户确认本 PRD / design / implement。（已通过）
- 开发后：完成 + check 通过后暂停，提交用户确认 diff / 行为 / 测试结果；确认后才提交、归档并完成父任务验收。

## Rollback Points

- 三处日志脱敏独立可回滚，互不影响；不影响传播行为。
- 不改发送端传播、MCP `_meta` 内容、提取语义、wire key、`b64:`、claim 转换。
