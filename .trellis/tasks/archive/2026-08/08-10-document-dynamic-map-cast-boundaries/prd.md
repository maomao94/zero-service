# 记录动态 Map 转换边界

## Goal

将 HMS 动态 JSON 参数使用 `spf13/cast` 时发现的边界语义固化到项目规范，避免缺失值被当作有效零值，以及为了复用而过早增加领域 getter 或公共包装。

## Background

- `HmsArgs` 的数据来源是 JSON，默认数字类型为 `float64`，调用处已直接复用仓库现有依赖 `github.com/spf13/cast`。
- 调试与定向测试证明：`cast.ToIntE(nil)` 返回 `0, nil`，直接转换缺失的 `component_index` 会错误渲染为索引 `1`；`cast.ToIntE(float64(1.5))` 采用宽松截断语义。
- 当前实现通过转换前检查 map key 存在且值非 `nil`，保持缺失 HMS 参数的占位符不被替换。
- 仓库当前不存在 `src/templates/markdown/spec/` 或其他 spec 模板镜像目录，因此本任务不创建新的模板体系。

## Requirements

- 更新 `.trellis/spec/guides/code-reuse-thinking-guide.md`：复用第三方转换工具前，必须验证缺失值、`nil`、空字符串、小数、布尔值、溢出和错误语义；动态 map 不得仅依赖 `To*E` 判断字段是否存在。
- 更新 `.trellis/spec/backend/dji-guidelines.md`：HMS 参数继续使用开放 `HmsArgs map[string]any` 和 `cast.To*E`，但转换前必须检查 key 存在且值非 `nil`；缺失参数继续保留模板占位符。
- 规范明确区分“字段存在性”和“值转换”：存在性由 map lookup 决定，类型转换由 `cast` 决定。
- 规范明确 `cast` 是宽松转换工具，不把 `E` 后缀解释为严格类型校验；是否允许小数截断由具体领域契约决定。
- 不新增 `HmsArgs` getter、自定义 JSON 解码器、`common/jsonmap` 或第三方依赖。
- 完成文档检查后，将本任务范围内的 spec 和任务工件作为独立提交，不夹带当前并行任务或产品代码变更。

## Acceptance Criteria

- [ ] 代码复用指南包含动态 map 使用转换工具前的边界检查清单。
- [ ] DJI 规范明确 HMS map key/`nil` 检查、`cast` 转换和缺参占位符行为。
- [ ] 两份规范不要求全局禁止 `cast`，也不把 HMS 的领域约束错误升级成所有 map 的统一策略。
- [ ] 确认模板镜像目录不存在，并将模板同步标记为不适用，而不是创建新目录。
- [ ] `git diff --check` 通过，最终提交只包含本任务的 spec 与 Trellis 工件。

## Out of Scope

- 修改 HMS 产品代码、测试或 `HmsArgs` 类型。
- 全仓审计或重构所有 `cast.To*` 调用。
- 创建通用 typed map/jsonmap 包。
- 修改或实现 `persist-topology-device-name-type` 任务。

## Risks

- 规则写得过宽会阻止合理的宽松转换；因此通用指南只要求先验证边界语义，严格/宽松由领域规范决定。
- 当前存在并行任务和提交活动；提交前必须重新检查状态并只暂存本任务文件。

## Notes

- 本任务是轻量文档任务，采用 PRD-only，不创建 `design.md` 或 `implement.md`。
