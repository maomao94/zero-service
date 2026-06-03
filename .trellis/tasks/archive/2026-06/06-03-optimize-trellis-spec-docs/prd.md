# 优化 Trellis spec 文档

## Goal

清理并优化 `.trellis/spec/**`，降低 AI 实现前的读取成本，同时提升规范命中效率和判断准确性：索引清晰、文件分层合理、长文档可按主题加载、重复内容减少，且不破坏 Trellis 现有 spec 发现机制。

## Confirmed Facts

- 用户已同意创建 Trellis task；当前任务目录为 `.trellis/tasks/06-03-optimize-trellis-spec-docs`。
- `.trellis/spec/` 当前只包含 `backend/` 和 `guides/` 两个子目录。
- `backend/` 是代码实现规范层，`guides/` 是思考清单层；该边界由 `/trellis-update-spec` skill 明确要求。
- 直接统计显示 spec 总计约 1995 行，明显偏长的文件包括：
  - `.trellis/spec/backend/socketiox-guidelines.md`：578 行
  - `.trellis/spec/backend/error-handling.md`：261 行
  - `.trellis/spec/backend/antsx-invoke-guidelines.md`：227 行
  - `.trellis/spec/backend/coding-standards.md`：153 行
- `.trellis/scripts/common/packages_context.py` 会扫描 `.trellis/spec/` 下除 `guides` 外的目录作为 layer，并展示对应 `index.md`；因此本轮应保持 `backend/index.md` 和 `guides/index.md` 可用。
- 仓库内仅发现 AGENTS/Trellis 上下文对 `.trellis/spec/` 的通用引用，未发现大量硬编码具体规范文件路径。
- 背景审计确认 `.trellis/spec/backend/index.md` 和 `.trellis/spec/backend/error-handling.md` 当前引用的 `../../../code.md` 不存在；错误码说明文档是 `docs/error-codes.md`，项目枚举定义是 `third_party/extproto.proto`。
- 外部文档实践建议：index 只做导航，spec 承载可执行契约，guides 只做短清单；重复规则应建立单一权威来源并通过链接引用。
- 用户已明确目标：采用最佳实践降低 AI 读取成本，但必须同时提升 AI 执行效率和规范判断准确性；清理不能只追求删减行数。

## Requirements

- 保持 `.trellis/spec/` 顶层仅承载分类目录，不恢复散落的根级 Markdown 文件。
- 清理重复：同一规则只保留一个主定义位置，其他文件通过简短引用或索引指向主文件，降低 AI 在多个相似规则之间误判的概率。
- 压缩过长文件：优先处理 `socketiox-guidelines.md`、`error-handling.md`、`antsx-invoke-guidelines.md`、`coding-standards.md` 中的重复、模板化、过细或可下沉内容；允许按最佳实践少量拆分，以减少单次上下文注入成本。
- 保持 code-spec 与 guide 分工：
  - `backend/*.md` 写“如何安全实现”的具体规范、接口、契约、错误行为、验证点。
  - `guides/*.md` 写“实现前要想什么”的短清单，不重复 backend 细节。
- 保留可执行契约：涉及 API、命令、数据库、infra、跨层协议的规范不得只剩原则性描述，必须保留可验证的签名、字段、错误行为和测试点。
- 更新所有受影响索引和相对链接，避免断链、旧路径或错误码规范断链。
- 不改业务代码、不改 Trellis 脚本，除非发现 spec 位置变更必须配套调整且用户另行确认。

## Acceptance Criteria

- [ ] `.trellis/spec/backend/index.md` 准确列出后端规范文件，并说明读取优先级、适用场景和 canonical source，帮助 AI 先读最小必要文件。
- [ ] `.trellis/spec/guides/index.md` 准确列出思考指南，并保持指南只作为短清单入口，不复制 backend 细节。
- [ ] `.trellis/spec/backend/socketiox-guidelines.md` 至少完成一次实质清理：减少重复、提炼场景，按最佳实践可拆出 `socketiox-contracts.md` 等同层文件。
- [ ] `.trellis/spec/backend/error-handling.md` 和 `.trellis/spec/backend/antsx-invoke-guidelines.md` 完成去重/压缩或明确保留理由。
- [ ] `.trellis/spec/backend/coding-standards.md` 删除或迁移与其他 backend 专题规范重复的内容，只保留全局协作、安全、命名、Git 等通用规则。
- [ ] 修复 `../../../code.md` 断链，错误码链接同时指向 `docs/error-codes.md` 和 `third_party/extproto.proto` 的合适相对路径。
- [ ] `rg` 检查无旧路径、断裂相对路径、`code.md` 残留或明显重复索引条目；长文件拆分后索引必须能说明 AI 应读哪个文件。
- [ ] `git diff --check` 通过。
- [ ] 若 Markdown LSP 仍未配置，最终说明未执行 LSP 的原因；若可用，则对改动 Markdown 文件执行 `lsp_diagnostics`。

## Out of Scope

- 不新增业务功能。
- 不重写 Trellis 工作流或脚本。
- 不配置 Markdown LSP，除非用户把它作为本任务目标。
- 不删除有明确契约价值的场景规范，只做合并、压缩、迁移或索引优化。

## Decisions

- 按最佳实践允许少量拆分长规范文件；拆分目标是降低 AI 读取成本并提升命中准确性，不是为了机械增加文件数。
- 长文件拆分后，主文件必须保留入口、适用范围和跳转指引；被拆出的文件必须是同层 code-spec，并加入 `backend/index.md`。

## Open Questions

- 无阻塞问题；等待用户审核 planning 并明确是否进入实现。
