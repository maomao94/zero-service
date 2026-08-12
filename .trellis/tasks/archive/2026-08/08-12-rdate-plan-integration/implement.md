# Implementation Plan

1. 将 Calc 请求中的两个列表传给共享 Plan 编译入口。
2. CreatePlanTask 将两个列表透传给 Calc，不复制 Set 逻辑。
3. 补充 Calc 和 Create 测试，验证日期、描述、Set、Batch/ExecItem 展开及现有限制。
4. 运行目标 Logic 测试、Trigger 相关测试和 `git diff --check`。

## Rollback

- 仅回滚 Plan Logic 透传和测试；公共 Proto/编译能力保留给其他子任务。
