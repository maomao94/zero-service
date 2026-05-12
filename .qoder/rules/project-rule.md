---
trigger: always_on
alwaysApply: true
---

# zero-service 项目规则

## 项目画像

- Go 微服务项目。
- 核心技术栈：go-zero、Eino AI 框架、Trellis（`.trellis/`）。
- 协作方式：以 Trellis 任务、项目 spec、workflow 和技能路由驱动需求拆解、开发前检查、编码、测试和收尾。
- 开发时优先遵循 Go、go-zero、Eino、Trellis 约定，不套用 Java 分层和命名习惯。

## 项目约束

- 涉及接口或 RPC 契约变更时，先改 `.api` / `.proto`，再执行 `gen.sh`，最后编写 Logic，三步顺序不得跳过。
- go-zero、gRPC、API 生成代码非必要不手改；需要调整时优先修改源 `.api` / `.proto` 或生成模板。
- 工具函数必须配单测；优先覆盖手写 Logic、工具函数和关键业务分支；生成代码非必要不写单测。
- `.api` / `.proto` 注释必须完整，并与实现行为保持一致。
- 先遵循 `.trellis/spec/` 和当前任务文档，再遵循通用 Go/go-zero/Eino 最佳实践。

## 技能路由

| 场景 | 激活技能 |
|------|---------|
| Sprint、Backlog、任务拆解、角色调度 | `agile-dev-manager`（含角色体系） |
| go-zero 框架开发、排错、生成代码 | `zero-skills` |
| Eino 框架技术参考、组件开发 | `eino-skills` |
| Eino 学习路径、入门解释、A2UI 协议 | `eino-learning` |
| 部署模块 | `module-deploy` |

- `ai-team` 为角色详细参考文档，已内联到 `agile-dev-manager`，无需单独激活。
- 涉及多个场景时，优先激活与当前任务最直接相关的技能，避免无关技能增加上下文。

## Workflow 触发

- 新会话开始时，自动按 `.agent/workflows/start.md` 完成初始化，读取开发者身份、git 状态、活跃任务、项目 spec 等。
- 用户输入下列关键词时，自动读取并执行对应 workflow 文件：

| 关键词 | Workflow 文件 |
|--------|--------------|
| `/start` 或“开始会话” | `.agent/workflows/start.md` |
| `/brainstorm` 或“头脑风暴” | `.agent/workflows/brainstorm.md` |
| `/before-dev` 或“开发前检查” | `.agent/workflows/before-dev.md` |
| `/check` 或“检查代码” | `.agent/workflows/check.md` |
| `/check-cross-layer` 或“跨层检查” | `.agent/workflows/check-cross-layer.md` |
| `/finish-work` 或“完成工作” | `.agent/workflows/finish-work.md` |
| `/improve-ut` 或“改进测试” | `.agent/workflows/improve-ut.md` |
| `/break-loop` 或“分析 Bug” | `.agent/workflows/break-loop.md` |
| `/update-spec` 或“更新规范” | `.agent/workflows/update-spec.md` |
| `/record-session` 或“记录会话” | `.agent/workflows/record-session.md` |
| `/onboard` 或“入职培训” | `.agent/workflows/onboard.md` |
| `/create-command` 或“创建命令” | `.agent/workflows/create-command.md` |
| `/integrate-skill` 或“集成技能” | `.agent/workflows/integrate-skill.md` |

## 规范层级

| 层级 | 位置 | 加载时机 |
|------|------|---------|
| 编码标准 | `.trellis/spec/` | `/start` 或开发前检查时注入 |
| 敏捷流程 | `agile-dev-manager` 技能 | Sprint、Backlog、任务拆解时按需激活 |
| 项目文档 | `CP-开发流程/{项目}/` | Sprint 时按需读取，优先 Backlog + 当前任务清单 |
| 模板 | `CP-开发流程/template/` | 新项目初始化或规范更新时参考 |

## 编码规则

- 遵循 Go / go-zero / Google 规范，禁止 Java 风格命名、分层和异常处理习惯。
- Handler 负责参数接收、校验、调用 Logic 和返回结果；业务编排放 Logic，公共能力沉淀到 service/internal 工具层。
- 新增依赖前先检查 `go.mod`、相邻模块和现有工具封装，不重复引入功能相近的库。
- 先阅读相邻 Handler、Logic、svc、model、config、types，再按既有目录和命名扩展。
- 涉及数据库、Redis、消息队列或第三方服务时，优先复用已有 model、client、cache、配置和封装。
- 新需求涉及表结构、初始化数据、修复数据等独立 SQL 时，优先放到项目 `sql` 目录；文件名按 `yyyyMMdd-{需求号或Trellis任务号}-{简短说明}.sql` 组织，便于和 Trellis 需求/任务追踪关联。
- 不主动添加注释；但 `.api` / `.proto`、导出公共能力、复杂协议字段按项目规范补齐必要说明。

## 构建与验证

- 修改 `.api` / `.proto` 后必须执行对应 `gen.sh` 或项目既有生成脚本，并检查生成代码 diff。
- 手写 Logic、工具函数、关键业务分支变更后，优先运行相关包或模块测试，不盲目全量测试。
- 构建、测试、lint 命令以 `.trellis/spec/`、workflow、README 或项目脚本为准；不确定时先查脚本再执行。
- 涉及跨层变更时执行 `/check-cross-layer` 对应检查；收尾时优先执行 `/finish-work` 对应流程。
