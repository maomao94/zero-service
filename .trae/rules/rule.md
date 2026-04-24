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
