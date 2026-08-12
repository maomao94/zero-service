# Implementation Plan

1. 确认 CronJob 集成子任务已检查、提交和归档，重新核对 Proto、编译器、模型与测试中的最终字段和边界。
2. 重命名 CronJob 指南为统一 RRULE 指南，保留仍正确的周期示例并修正过时限制。
3. 增加集合语义、字段表、Plan/CronJob 请求、更新清空、回显和预览示例。
4. 更新 `docs/trigger.md`、`docs/README.md` 及所有旧链接。
5. 更新 Trigger、crontask、契约生成和 GORM Specs，避免跨文档重复实现细节。
6. 用 JSON 解析器验证所有 JSON fenced blocks；搜索旧标题、旧路径、冲突语句和模板占位。
7. 检查 Markdown 相对链接并运行 `git diff --check`。

## Validation

- 搜索旧指南路径和标题，预期无遗留引用。
- 搜索 `specified_times`、`excluded_times`、`RDATE`、`EXDATE`，确认用户文档和四份 Spec 均覆盖对应边界。
- 解析变更文档内所有 `json` 代码块。
- 校验变更 Markdown 的相对链接目标存在。
- 运行 `git diff --check`。

## Risky Files

- `docs/trigger-rrule-api-guide.md`：内容较长，重命名和修正必须避免删除仍有效的 CronJob 行为。
- `.trellis/spec/backend/trigger-guidelines.md`：只追加稳定契约，不重写已有状态机和并发警告。

## Rollback

- 文档和 Spec 可整体回滚，不影响已实现 API。
- 若统一指南过长，优先保留一个权威文件并通过目录导航，不复制成 Plan/CronJob 两份漂移文档。
