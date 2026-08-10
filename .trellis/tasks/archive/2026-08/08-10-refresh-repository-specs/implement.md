# Implementation Plan: Refresh Repository Trellis Specs

## Steps

1. 清点 `.trellis/spec/` 文件集合、索引链接、触发范围和当前证据路径，建立审计矩阵。
2. 核验基础规范：目录边界、编码、质量、go-zero、契约生成、生命周期、错误和公共包设计。
3. 核验基础设施规范：GORM、并发、消息、crontask，以及对应实现和代表性测试。
4. 核验领域规范：Trigger、ISP、IEC 104、DJI、GIS、实时事件、AI/MCP 和跨层 guides。
5. 按证据做最小更新：修失效路径、删过时/重复规则、补遗漏的稳定契约；必要时调整文件边界。
6. 最后同步 `README.md`、`backend/index.md` 和 `guides/index.md`，确保导航与最终文件集合一致。
7. 执行占位、路径、索引、Markdown diff 和范围检查，完成独立质量复核。

## Validation

- 搜索 `TBD|TODO: fill|To be filled|placeholder|待补充|待填写`。
- 枚举 `.trellis/spec/**/*.md` 并与两个 index 的链接集合比对。
- 校验正文引用的仓库相对路径存在，人工排除 glob、目录和符号引用。
- `git diff --check`。
- `git status --short`，确认产品源码未修改。

## Review Gates

- 实施前评审 PRD、设计和本计划，之后才激活任务。
- 新增 Spec 文件必须说明为何现有文件无法承载该规则。
- 删除或弱化现有规则必须记录其证据失效原因。
- 最终检查由独立 check agent 执行，主会话负责整合和修正。

## Rollback Points

- 完成基础规范审计后检查范围是否过大；若出现多个可独立验收的新领域，再拆子任务。
- 调整索引前确认所有正文已稳定，避免产生临时断链。
