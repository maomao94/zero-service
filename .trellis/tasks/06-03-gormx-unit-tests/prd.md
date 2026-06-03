# 补充 gormx 包单元测试

## Goal

提高 `common/gormx/` 包的单元测试覆盖率，补充缺失的核心功能测试，确保批量操作、租户隔离、审计回调等关键功能的可靠性。

## Confirmed Facts

**当前测试覆盖情况：**
- 已测试：context、driver、pagination、legacy、logger、model_tenant、upsert、user_context、trace、explain（11个测试文件）
- 未测试：batch.go、callbacks.go、tenant_scope.go、model.go、model_audit_mixins.go、model_audit.go、model_legacy.go

**现有测试质量：**
- 使用内存 SQLite 进行隔离测试
- 测试命名遵循 Go 规范
- 使用 `openTestDB` 辅助函数统一初始化
- 缺少 `t.Parallel()` 支持

**关键未测试函数：**
- `batch.go`：BatchInsert、BatchUpdateByIds、BatchDeleteByIds、BatchDeleteByCondition、BatchInsertWithTenant、BatchUpdateByIdsWithTenant、BatchDeleteByIdsWithTenant、BatchDeleteByConditionWithTenant、RestoreWithTenant、UnscopedDeleteWithTenant
- `callbacks.go`：RegisterCallbacks、beforeCreateHook、beforeUpdateHook、beforeDeleteHook
- `tenant_scope.go`：TenantScope、TenantScopeStrict、TenantScopeWithDelete、TenantEq、TenantNotEq、TenantIn

## Requirements

1. 为 `batch.go` 中的批量操作函数添加单元测试
   - 测试正常场景
   - 测试边界条件（空数组、缺少 id 字段等）
   - 测试带租户的版本

2. 为 `tenant_scope.go` 中的租户作用域函数添加单元测试
   - 测试 TenantScope 有/无租户上下文
   - 测试 TenantScopeStrict 无租户时返回 `1 = 0`
   - 测试 TenantScopeWithDelete 的 Unscoped 行为
   - 测试 TenantEq、TenantNotEq、TenantIn

3. 为 `callbacks.go` 中的回调机制添加单元测试
   - 测试 beforeCreateHook 填充审计字段
   - 测试 beforeUpdateHook 填充更新字段和版本递增
   - 测试 beforeDeleteHook 填充删除字段

4. 测试代码规范
   - 遵循现有测试的命名和结构规范
   - 使用 `openTestDB` 辅助函数
   - 与现有测试保持一致的风格（不使用 `t.Parallel()`）

## Acceptance Criteria

- [ ] batch.go 中所有公开函数有对应测试
- [ ] tenant_scope.go 中所有公开函数有对应测试
- [ ] callbacks.go 中回调机制有测试覆盖
- [ ] 所有新增测试通过
- [ ] 测试代码遵循现有规范

## Out of Scope

- model.go、model_audit_mixins.go、model_audit.go、model_legacy.go 的测试（这些主要是结构定义，测试价值较低）
- 性能优化（如测试数据库连接复用）
- Benchmark 测试

## Notes

- 这是一个轻量级任务，PRD-only 即可
- 测试文件命名：batch_test.go、tenant_scope_test.go、callbacks_test.go
