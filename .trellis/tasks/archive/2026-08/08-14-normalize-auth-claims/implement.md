# Claim 规范化实施计划

## 前置

- [x] 用户审查并确认本 PRD / design / implement 后 `task.py start`（开发前人工门禁）。
- [x] 确认前置任务已归档（audit、typed-auth-context-keys）。

## 执行清单

### 阶段 1 - authctx 核心转换
- [ ] `common/authctx/claims.go`：新增 `normalizeClaimString(v any) string`（类型白名单 + 精度规则，见 design）。
- [ ] `ClaimString` 改调 `normalizeClaimString`。
- [ ] `common/authctx/context.go` `toStringClaim` 改调 `normalizeClaimString`（或直接复用）。

### 阶段 2 - mcpx 行为随动
- [ ] `common/mcpx/context_meta.go` `ExtractFromMeta`：确认行为随 `ClaimString` 变化，无需代码改动。
- [ ] `common/mcpx/auth.go` `NewDualTokenVerifier`：确认 `UserID` 精确、`Extra` 保留原始值。

### 阶段 3 - 测试更新
- [ ] `common/authctx/claims_test.go`：bool→`""`、分数→`""`；float64 整数/int64/string/json.Number 保留精确断言。
- [ ] `common/authctx/context_test.go`：bool/数组→`""`。
- [ ] `common/mcpx/context_meta_test.go`、`auth_test.go`：整数值保留、分数/超大跳过。
- [ ] 新增 `normalizeClaimString` 全矩阵单元测试（string/int/int64/uint/float 整数/float 分数/float>2^53/json.Number 整数/json.Number 非整数/bool/数组/对象/nil/缺失）。

### 阶段 4 - 兼容验证
- [ ] 确认 zerorpc int64、socketpush string 签发链路在新转换下继续工作（JSON number 经 go-zero json.Number、MCP/socket float64 小值）。

## 验证命令

```bash
gofmt -w <changed-go-files>
go test -count=1 ./common/authctx ./common/grpcx ./common/mcpx ./common/tool ./common/socketiox
go test -count=1 ./aiapp/aigtw/... ./gtw/... ./socketapp/socketgtw/...
go vet ./common/authctx ./common/grpcx ./common/mcpx
go build ./...
git diff --check
```

## 人工门禁

- 开发前：用户确认本 PRD / design / implement。（已通过）
- 开发后：完成 + check 通过后暂停，提交用户确认 diff / 行为 / 测试结果；确认后才提交、归档，并进入 `enforce-authorization-policy` 规划。

## Rollback Points

- `normalizeClaimString` 独立函数可整体回退为 `convertor.ToString`。
- 不改 wire key、`b64:`、Authorization 传播、metadata 冲突行为。
