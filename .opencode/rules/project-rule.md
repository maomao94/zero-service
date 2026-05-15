---
apply: 始终
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

## AI 工具边界

- `.aiassistant/rules/**` 是 GoLand AI / JetBrains AI 配置。
- `.opencode/rules/**` 是 OpenCode 规则配置。
- 两个目录下规则文件名和内容保持一致，便于后续覆盖同步。
- 当前项目不再依赖已删除的旧技能包或旧工作流入口；开发上下文以 Trellis SessionStart、当前任务材料和 `.trellis/spec/**` 为准。

## Trellis 工作流

- 新会话或开发前，优先读取 Trellis 注入的上下文；必要时执行 `python3 ./.trellis/scripts/get_context.py`。
- 复杂需求先沉淀为 Trellis 任务材料：`prd.md`、`design.md`、`implement.md`，轻量任务可直接按 spec 和相邻实现处理。
- 开发前按需读取 `.trellis/spec/backend/index.md`、`.trellis/spec/guides/index.md` 和任务相关具体规范，不默认全文加载所有历史资料。
- 完成后按变更范围运行测试、构建或检查，并说明未执行项原因。

## 规范层级

| 层级 | 位置 | 加载时机 |
|------|------|---------|
| 通用 AI 规则 | `.opencode/rules/ai-rule.md` / `.aiassistant/rules/ai-rule.md` | 会话规则加载时 |
| 项目规则 | `.opencode/rules/project-rule.md` / `.aiassistant/rules/project-rule.md` | 会话规则加载时 |
| Trellis 任务上下文 | `.trellis/tasks/` | 有活跃任务时按 `prd.md` / `design.md` / `implement.md` 读取 |
| Trellis 项目规范 | `.trellis/spec/` | 开发前检查、代码审查和规范回填时按需读取 |

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
- 涉及跨层变更时执行对应跨层检查；收尾时按 Trellis 完成流程检查。
