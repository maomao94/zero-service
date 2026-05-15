<!-- TRELLIS:START -->
# Trellis Instructions

These instructions are for AI assistants working in this project.

Use the `/trellis:start` command when starting a new session to:
- Initialize your developer identity
- Understand current project context
- Read relevant guidelines

Use `@/.trellis/` to learn:
- Development workflow (`workflow.md`)
- Project structure guidelines (`spec/`)
- Developer workspace (`workspace/`)

If you're using Codex, project-scoped helpers may also live in:
- `.agents/skills/` for reusable Trellis skills
- `.codex/agents/` for optional custom subagents

Keep this managed block so 'trellis update' can refresh the instructions.

<!-- TRELLIS:END -->

# OpenCode Project Guidance

本节是 OpenCode 使用的项目级规则。不要修改上方 `TRELLIS` 托管块；Trellis 更新时需要保持该块可刷新。

## 沟通与协作

- 默认使用中文沟通、提问、列任务和总结；用户明确要求其他语言时再切换。
- 开发前先阅读相邻实现和相关 `.trellis/spec/**`，不要凭经验套用其他项目习惯。
- 能通过搜索代码、读取配置、参考相邻实现推断的低风险事项，直接自主推进。
- 只有在需求多解且影响大、超出项目边界、存在不可逆/高风险操作、缺少关键参数且无法推断、规则冲突时才提问。
- 多步骤任务使用任务清单跟踪；最终说明变更内容、影响范围、验证结果和未验证原因。

## 变更纪律

- 按最小影响范围修改，避免无关 diff、大范围格式化和顺手重构。
- 不主动创建额外文档文件，除非用户明确要求或规则文件本身就是任务目标。
- 不主动添加注释；`.api`、`.proto`、导出公共能力和复杂协议字段按项目规范补齐必要说明。
- 不新增或打印明文密码、Token、密钥、认证头、证书、个人身份信息、手机号、数据库连接、对象存储配置、内网地址或本地绝对路径。
- 对外总结和规则文档中不要记录个人本地路径；必要时改写为“项目内目录”“用户级配置”等通用表述。

## Git 纪律

- 不主动执行 `git commit`；只有用户明确要求提交时才提交。
- 提交前必须查看实际 diff，提交说明结合真实变更，不写泛泛模板。
- 提交标题优先中文，可使用 conventional commit 前缀，例如 `feat: 新增火情任务状态同步`。
- 提交信息不得包含敏感信息、内网地址、账号、Token、密钥或完整异常堆栈。

## 项目边界

- 本项目是 Go 微服务项目，核心技术栈为 go-zero、gRPC、Protocol Buffers、Eino、Trellis。
- 工程规范优先级：用户当前要求 > 当前 Trellis 任务文档 > `.trellis/spec/**` > README/相邻实现 > 通用 Go/go-zero/Eino 最佳实践。
- go-zero 服务遵循 Handler/Server -> Logic -> Model/SDK/Client 分层；业务编排放在 `internal/logic/`。
- 涉及 `.api` 或 `.proto` 的契约变更时，必须先改契约，再执行对应 `gen.sh`，最后实现 Logic 并检查生成代码 diff。
- 非必要不手改 go-zero、gRPC、API 生成代码；需要调整时优先修改 `.api`、`.proto` 或生成模板。
- 新增依赖、工具函数或公共封装前，先检查 `go.mod`、相邻模块、`common/` 和已有 client/cache/config/model 封装。

## Trellis 使用

- 开发任务开始前先运行或参考 `python3 ./.trellis/scripts/get_context.py`；需要包索引时再运行 `--mode packages`。
- 只读取当前任务相关的 spec 和上下文，避免默认全文读取所有 `.trellis/spec/**`、workspace journal、历史 Sprint 或 CP 文档。
- 可维护的长期规则放入 `.trellis/spec/**`；不要修改 `.trellis/workflow.md`、`.trellis/scripts/**`、`.trellis/config.yaml`、`.trellis/worktree.yaml` 或 `.agent/workflows/**`，除非用户明确要求。
