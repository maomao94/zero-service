# Trellis 工作流规范

> 项目级 Trellis 使用规范。OpenCode 通过 `AGENTS.md` 和 `.trellis/spec/**` 获取项目规则；`.trae/**` 不再作为未来开发入口。

## 0. 边界

本文件位于 `.trellis/spec/`，属于项目可维护规范。

不要修改以下工具自带/托管文件，除非用户明确要求：

- `AGENTS.md` 中 `<!-- TRELLIS:START -->` 到 `<!-- TRELLIS:END -->` 的托管块
- `.trellis/workflow.md`
- `.trellis/scripts/**`
- `.trellis/config.yaml`
- `.trellis/worktree.yaml`
- `.agent/workflows/**`
- `.trellis/workspace/**` 和 `.trellis/tasks/**` 的运行期记录，除非任务本身要求维护 Trellis 记录

长期项目规则写入 `.trellis/spec/**`；OpenCode 会话入口规则写入 `AGENTS.md` 的非托管区域。

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

## 2. CP/Sprint 联通

当任务涉及 CP 开发流程、Backlog、Sprint、任务拆解或角色调度时，采用最小上下文：

```text
Trellis context
  -> Backlog.md 单入口
  -> 当前 Sprint / 当前任务
  -> Backend / Frontend / QA / DevOps 按需分工
  -> 开发、验证、最小回填
```

老板需求唯一入口：

```text
CP-开发流程/{项目名}/Backlog.md
```

新项目不创建 `需求输入.md`。旧项目只在用户明确指定或迁移旧需求时读取。

---

## 3. 任务分级

| 级别 | 场景 | 流程 |
| --- | --- | --- |
| Level 0 查询解释 | 解释命令、说明机制、阅读少量文件、回答架构问题 | 只读必要文件，不启动完整开发流程，不改文件 |
| Level 1 小改/Bugfix | 单点修复、小范围重构、补测试、改配置 | Trellis context -> 定位代码 -> 修改 -> 最小验证 -> 交付说明 |
| Level 2 Sprint/Backlog | 需求梳理、任务拆解、跨文件功能开发 | Trellis context -> PM/Dev/QA 分工 -> 开发 -> 验证 -> 回填 |
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
- 不修改 `.trellis/workflow.md`、`.trellis/scripts/**`、`.agent/workflows/**` 等工具托管文件。

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
3. CP scoped：CP 文档默认只读 Backlog 当前条目和任务清单当前 Sprint。
4. Role scoped：角色材料只在职责边界、交接协议或权限冲突不清时按小节读取。
5. Tool boundary：不修改工具托管文件，不把 `.trae/**` 作为未来运行时依赖。

---

## 6. 最佳实践

1. Read before write：先启动 Trellis、读相关 index，再写代码。
2. Backlog 单入口：老板需求只进 `Backlog.md`。
3. 规范即代码：发现稳定规则后沉淀到 `.trellis/spec/**`。
4. 自修复循环：检查发现问题后修复并重新验证。
5. 不跳生成：`.api` / `.proto` 变更必须执行 `gen.sh`。
6. AI 不 commit：除非用户明确要求，否则不提交代码。
