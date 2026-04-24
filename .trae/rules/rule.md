# 项目开发规则

## 技术栈
go-zero 微服务 + eino AI 框架 + Trellis（`.trellis/`）

## 技能路由

| 场景 | 激活技能 |
|------|---------|
| Sprint/Backlog/任务拆解/角色调度 | `agile-dev-manager`（含角色体系） |
| go-zero 框架 | `zero-skills` |
| eino 框架 | `eino-skills` / `eino-learning` |
| 部署模块 | `module-deploy` |

> `ai-team` 为角色详细参考文档，已内联到 agile-dev-manager，无需单独激活。

## Workflow 自动触发

每次新会话开始时，自动读取并执行 `.agent/workflows/start.md` 中的流程，完成会话初始化（读取开发者身份、git 状态、活跃任务、项目 spec 等）。

当用户要求执行以下关键词时，自动读取并执行对应的 workflow 文件：

| 关键词 | Workflow 文件 |
|--------|--------------|
| `/start` 或 "开始会话" | `.agent/workflows/start.md` |
| `/brainstorm` 或 "头脑风暴" | `.agent/workflows/brainstorm.md` |
| `/before-dev` 或 "开发前检查" | `.agent/workflows/before-dev.md` |
| `/check` 或 "检查代码" | `.agent/workflows/check.md` |
| `/check-cross-layer` 或 "跨层检查" | `.agent/workflows/check-cross-layer.md` |
| `/finish-work` 或 "完成工作" | `.agent/workflows/finish-work.md` |
| `/improve-ut` 或 "改进测试" | `.agent/workflows/improve-ut.md` |
| `/break-loop` 或 "分析 Bug" | `.agent/workflows/break-loop.md` |
| `/update-spec` 或 "更新规范" | `.agent/workflows/update-spec.md` |
| `/record-session` 或 "记录会话" | `.agent/workflows/record-session.md` |
| `/onboard` 或 "入职培训" | `.agent/workflows/onboard.md` |
| `/create-command` 或 "创建命令" | `.agent/workflows/create-command.md` |
| `/integrate-skill` 或 "集成技能" | `.agent/workflows/integrate-skill.md` |

## 编码底线
- 先改 `.api`/`.proto` → `gen.sh` → 写 Logic，禁止跳过
- Go/go-zero/Google 规范，禁止 Java 风格
- 工具类函数须有单测，proto/api 注释完整一致
- Git 提交信息中文

## 规范层级

| 层级 | 位置 | 加载时机 |
|------|------|---------|
| 编码标准 | `.trellis/spec/` | `/start` 时注入 |
| 敏捷流程 | `agile-dev-manager` 技能 | 按需激活 |
| 项目文档 | `CP-开发流程/{项目}/` | Sprint 时读取 |
| 模板 | `CP-开发流程/template/` | 新项目初始化 |
