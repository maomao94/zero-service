# Typed Context Key 迁移实施计划（无兼容回退 + 桥接中间件）

## 前置

- [x] 用户审查并确认本 PRD、design、implement 后才 `task.py start`（开发前人工门禁，已通过）。
- [x] 确认 `audit-authorization-propagation` 已归档，本任务为其后续（父任务顺序第 2 步）。

## 执行清单

### 阶段 0 - authctx：去掉回退，只读 typed + 新增桥接
- [ ] `common/authctx/context.go`：现有 5 个 getter **移除 string-key 回退**，只读 typed key；`GetAuthType` 同样只读 typed。
- [ ] 保留 5 个公开 string 常量与 `ContextKeys` 顺序（wire 契约）。
- [ ] 新增 `BridgeJWTClaims(ctx, mapping)`：先遍历 `ContextKeys` 直拷 string key（短横线），再调 `ApplyClaimMappingToCtx`（下划线映射）。仅 string 值桥接；非 string（如 int64）跳过。
- [ ] 契约测试更新：getter 只读 typed；string-key 写入不可再被 getter 读到（锁定无回退）；`BridgeJWTClaims` 覆盖短横线直拷、下划线映射、非 string 跳过、无 JWT 空操作。

### 阶段 1 - 仓库内写入方迁移到 typed setter
- [ ] `aiapp/aigtw/aigtw.go:83-84`、`gtw/gtw.go:60`、`socketapp/socketgtw/socketgtw.go:68` 改用 typed setter。（已由上一版实施完成，复核）
- [ ] `common/socketiox/server.go` 9 处 token 写入改 `WithAuthorization`。（已实施，复核）
- [ ] `common/authctx/claims.go`：`ExtractFromClaims` / `ApplyClaimMappingToCtx` 内部改 typed setter。（已实施，复核）

### 阶段 2 - 网关桥接中间件落地
- [ ] `aiapp/aigtw/aigtw.go`：现有 `server.Use` 中 auth-type/authorization 用 typed setter（已实施），将 ClaimMapping 中间件扩展为 `authctx.BridgeJWTClaims(ctx, c.JwtAuth.ClaimMapping)`。
- [ ] `gtw/gtw.go`：新增 `server.Use` 调 `authctx.BridgeJWTClaims(ctx, nil)`（承接 JWT 短横线 claim）。
- [ ] `socketgtw`：无需桥接（HTTP 路由 `WithJwt` 注释状态；Socket.IO 路径已直写 typed key）。复核确认。

### 阶段 3 - 传输边界读取（已实施，复核）
- [ ] `common/grpcx/metadata.go` `InjectToGrpcMD` 经 `authctx.GetByKey` 读；`ExtractFromGrpcMD` 经 `authctx.WithKey` 写。（已实施）
- [ ] `common/mcpx/context_meta.go` `CollectFromCtx` 经 `GetByKey` 读；`ExtractFromMeta` 经 `WithKey` 写。（已实施）
- [ ] `common/mcpx/auth.go`：保持 wire `Extra` 键名，无进程 context 写入。（复核）

### 阶段 4 - 测试收尾
- [ ] `common/grpcx/metadata_test.go`、`client_interceptor_test.go`、`server_interceptor_test.go`：写入方式用 typed setter，锁定 wire key 名不变。（已实施，复核）
- [ ] `common/mcpx/context_meta_test.go`、`auth_test.go`：写入方式用 typed setter。（已实施，复核）
- [ ] aigtw 三个 validation 测试文件改用 `WithUserID`。（已实施，复核）
- [ ] 移除任何依赖 string-key 回退的测试断言；确认 `BridgeJWTClaims` 有专门契约测试。

## 验证命令

```bash
gofmt -w <changed-go-files>
go test ./common/authctx ./common/grpcx ./common/mcpx ./common/tool
go test ./common/socketiox
go test ./aiapp/aigtw/... ./gtw/... ./socketapp/socketgtw/...
go vet ./common/authctx ./common/grpcx ./common/mcpx
go test ./...
git diff --check
```

## 人工门禁

- 开发前：用户确认本 PRD、design、implement。（已通过）
- 开发后：完成 + check 通过后暂停，提交用户确认 diff / 行为 / 测试结果；确认后才提交、归档，并进入 `normalize-auth-claims` 规划。

## Rollback Points

- 每阶段独立可回滚；若迁移后出现身份丢失，回滚为桥接基础上临时恢复 getter string-key 回退，排查后清理。
- 不改 wire key、`b64:`、claim 类型规则、raw-token 传播、metadata 冲突行为。
