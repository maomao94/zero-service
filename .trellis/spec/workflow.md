# Trellis 工作流规范

> 项目级 Trellis 使用规范。只描述本项目如何借助 Trellis 启动、读取规范和验证代码；Sprint 敏捷流程由 `agile-dev-manager` 调度，角色细节由 `ai-team` 按需补充。

## 0. 边界

本文件位于 `.trellis/spec/`，属于项目规范，可按项目需要维护。

不要修改以下工具自带/托管文件，除非用户明确要求：

- `.trellis/workflow.md`
- `.trellis/scripts/**`
- `.trellis/config.yaml`
- `.trellis/worktree.yaml`
- `.agent/workflows/**`

---

## 1. 会话启动

开发相关任务开始前，先执行 Trellis 启动：

```bash
python3 ./.trellis/scripts/get_context.py
```

需要规范包索引时再执行：

```bash
python3 ./.trellis/scripts/get_context.py --mode packages
```

启动阶段只读取当前任务、git 状态和必要索引，不默认全文读取全部 spec、workspace journal、CP 文档或 `ai-team`。

---

## 2. CP/Sprint 联通

当任务涉及 CP 开发流程、Backlog、Sprint、任务拆解或角色调度时：

```text
trellis:start
  → agile-dev-manager
  → AI Team 按阶段调度
  → Backlog.md 单入口
  → before-dev / check / finish-work 门禁
```

老板需求唯一入口：

```text
CP-开发流程/{项目名}/Backlog.md
```

新项目不创建 `需求输入.md`。旧项目只在用户明确指定或迁移旧需求时读取。

---

## 3. 编码三段式

### 3.1 编码前

必须先完成最小上下文加载：

```bash
python3 ./.trellis/scripts/get_context.py --mode packages
cat .trellis/spec/backend/index.md
cat .trellis/spec/guides/index.md
```

然后根据 index 的 Pre-Development Checklist，只读取当前任务相关的具体规范。后端 go-zero 任务通常需要：

```bash
cat .trellis/spec/coding-standards.md
cat .trellis/spec/go-zero-conventions.md
```

如果任务不涉及后端、go-zero、跨层或复用问题，不要为了保险读取所有规范文件。

### 3.2 编码中

- 遵循当前任务相关 spec。
- 需要 go-zero 框架知识时激活 `zero-skills`。
- 需要 Eino / Agent / A2UI 知识时激活 `eino-skills` 或 `eino-learning`。
- 先检索现有代码和 `common/` 复用点，再新增工具函数或模块。
- 修改 `.api` / `.proto` 时必须：定义 → 执行 `gen.sh` → 写 Logic。

### 3.3 编码后

根据变更范围执行项目实际可用验证命令。Go 后端优先考虑：

```bash
go build ./...
go test ./...
go vet ./...
git diff --name-only HEAD
```

然后对照相关 spec index 的 Quality Check 小节逐项验证。未执行项必须说明原因。

---

## 4. Trellis 命令与项目流程关系

| 命令 / Workflow | 用途 | CP 阶段 |
| --- | --- | --- |
| `/start` | 轻量加载 Trellis 上下文并路由任务 | Phase 0 |
| `/brainstorm` | 需求探索，仅在需求不清晰时使用 | Planning |
| `/before-dev` | 编码前最小规范注入 | Planning → Execute |
| `/check` | 代码完成后的规范和验证检查 | Execute / Review |
| `/finish-work` | 交付前完整性检查 | Retro |
| `/record-session` | 记录会话摘要 | 会话结束 |
| `/update-spec` | 沉淀稳定规范和踩坑经验 | 按需 |

这些命令和 workflow 文件由工具提供时，只按现有能力使用，不把 `.agent/workflows/**` 当作本项目自定义提示词修改对象。

---

## 5. 规范读取原则

1. Index first：先读 index，再读具体规范。
2. Task scoped：只读当前任务相关文件。
3. CP scoped：CP 文档默认只读 Backlog 当前条目和任务清单当前 Sprint。
4. Role scoped：`ai-team` 只在角色边界不清时按小节读取。
5. Tool boundary：不修改 `.trellis/workflow.md`、`.agent/workflows/**` 等工具托管文件。

---

## 6. 最佳实践

1. Read before write：先启动 Trellis、读相关 index，再写代码。
2. Backlog 单入口：老板需求只进 `Backlog.md`。
3. 规范即代码：发现稳定规则后沉淀到 `.trellis/spec/**`。
4. 自修复循环：`/check` 发现问题后修复并重新验证。
5. 不跳生成：`.api` / `.proto` 变更必须执行 `gen.sh`。
6. AI 不 commit：除非用户明确要求，否则不提交代码。
