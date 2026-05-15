# Trellis 工作流规范

> 项目级 Trellis 使用规范。OpenCode 与 GoLand AI 使用同名规则文件，规则内容放在 `.opencode/rules/**` 和 `.aiassistant/rules/**`，再按需读取 `.trellis/spec/**`。

## 0. 边界

本文件位于 `.trellis/spec/`，属于项目可维护规范。

不要修改以下工具自带/托管文件，除非用户明确要求：

- `AGENTS.md` 中 `<!-- TRELLIS:START -->` 到 `<!-- TRELLIS:END -->` 的托管块
- `.trellis/workflow.md`
- `.trellis/scripts/**`
- `.trellis/config.yaml`
- `.trellis/worktree.yaml`
- `.opencode/agents/**`
- `.opencode/skills/**`
- `.opencode/commands/**`
- `.opencode/plugins/**`
- `.opencode/lib/**`
- `.trellis/workspace/**` 和 `.trellis/tasks/**` 的运行期记录，除非任务本身要求维护 Trellis 记录

长期项目规则写入 `.trellis/spec/**`；AI 工具规则写入 `.opencode/rules/**` 和 `.aiassistant/rules/**`，两边保持同名同内容，便于后续覆盖同步。`AGENTS.md` 保持 Trellis 托管块即可。

---

## 1. 会话启动

开发相关任务开始前，先执行或参考 Trellis 上下文：

```bash
python3 ./.trellis/scripts/get_context.py
```

需要规范包索引时再执行：

```bash
python3 ./.trellis/scripts/get_context.py --mode packages
```

启动阶段只读取当前任务、git 状态和必要索引，不默认全文读取全部 spec、workspace journal、历史 Sprint、完整 CP 文档或角色材料。

---

## 2. 任务上下文联通

当前项目不再依赖已删除的敏捷技能包、角色技能包或旧工作流入口。OpenCode + Trellis 的有效上下文来自：

```text
用户当前消息
  -> Trellis SessionStart / workflow-state
  -> 当前 Trellis 任务材料（如存在）
  -> .trellis/spec/** 相关索引和规范
  -> 相邻代码、README、项目脚本
```

读取任务材料时按以下顺序：`prd.md` -> `design.md` if present -> `implement.md` if present -> `implement.jsonl` / `check.jsonl` 引用文件。

没有活跃 Trellis 任务时，不要为了简单问答或小改动强制创建任务；跨模块、需求不清或长期工作再进入 Trellis 规划。

---

## 3. 任务分级

| 级别 | 场景 | 流程 |
| --- | --- | --- |
| Level 0 查询解释 | 解释命令、说明机制、阅读少量文件、回答架构问题 | 只读必要文件，不启动完整开发流程，不改文件 |
| Level 1 小改/Bugfix | 单点修复、小范围重构、补测试、改配置 | Trellis context -> 定位代码 -> 修改 -> 最小验证 -> 交付说明 |
| Level 2 Trellis 任务 | 需求梳理、任务拆解、跨文件功能开发 | Trellis context -> PRD/设计/实现计划 -> 开发 -> 验证 -> 必要规范回填 |
| Level 3 跨模块/架构/部署 | 跨服务契约、数据模型、基础设施、发布部署 | Level 2 + 相关 spec + 专项方案；部署必须有用户明确授权 |

升级规则：

- 查询过程中发现需要修改代码，升级到 Level 1。
- 单点修改影响 API、DB、proto、跨层契约，升级到 Level 2 或 Level 3。
- 涉及上线、环境、密钥、镜像、容器，升级到 DevOps 流程并先确认目标环境和权限。

---

## 4. 编码三段式

### 4.1 编码前

必须先完成最小上下文加载：

```bash
python3 ./.trellis/scripts/get_context.py --mode packages
```

然后根据任务读取相关索引。后端 go-zero 任务通常需要：

```bash
cat .trellis/spec/backend/index.md
cat .trellis/spec/coding-standards.md
cat .trellis/spec/go-zero-conventions.md
```

跨层或复用问题再读取：

```bash
cat .trellis/spec/guides/index.md
```

如果任务不涉及后端、go-zero、跨层或复用问题，不要为了保险读取所有规范文件。

### 4.2 编码中

- 遵循当前任务相关 spec。
- 先检索现有代码、相邻实现和 `common/` 复用点，再新增工具函数或模块。
- 修改 `.api` / `.proto` 时必须：定义 -> 执行 `gen.sh` -> 写 Logic -> 检查生成代码 diff。
- 需要 go-zero、Eino、部署或架构细节时，优先读取相关 spec 和相邻实现；外部资料只作为补充。
- 不修改 `.trellis/workflow.md`、`.trellis/scripts/**`、`.opencode/agents/**`、`.opencode/skills/**`、`.opencode/plugins/**` 等工具或生成适配文件。

### 4.3 编码后

根据变更范围执行项目实际可用验证命令。Go 后端优先考虑：

```bash
go build ./...
go test ./...
go vet ./...
git diff --name-only
```

然后对照相关 spec 的质量检查逐项验证。未执行项必须说明原因。

---

## 5. 规范读取原则

1. Index first：先读 index，再读具体规范。
2. Task scoped：只读当前任务相关文件。
3. Task artifact scoped：Trellis 任务材料按 `prd.md` -> `design.md` -> `implement.md` -> JSONL 引用顺序读取。
4. Tool boundary：不修改 Trellis 工具配置和 OpenCode 生成适配文件。
5. Deleted skill boundary：不引用已删除的旧敏捷/角色/部署技能包作为运行时依赖。

---

## 6. 最佳实践

1. Read before write：先启动 Trellis、读相关 index，再写代码。
2. AI 规则入口清晰：`.opencode/rules/**` 与 `.aiassistant/rules/**` 文件名和内容保持一致。
3. 规范即代码：发现稳定规则后沉淀到 `.trellis/spec/**`。
4. 自修复循环：检查发现问题后修复并重新验证。
5. 不跳生成：`.api` / `.proto` 变更必须执行 `gen.sh`。
6. AI 不 commit：除非用户明确要求，否则不提交代码。
