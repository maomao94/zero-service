# 模板说明

本目录提供精简版 CP 开发流程模板，目标是降低上下文成本，让老板只在一个地方写需求，AI 使用 Trellis 和 AI 团队按需推进开发。

## 启动方式

开启开发前必须先执行 Trellis 启动流程：

```text
trellis:start
```

在 Trae 环境中，如果无法直接执行命令形式的 `trellis:start`，等价执行项目启动工作流或读取 Trellis 上下文：

```bash
python3 ./.trellis/scripts/get_context.py
```

启动后由 `agile-dev-manager` 作为总指挥，自动调度 PM / Backend / Frontend / QA / DevOps，不需要老板手动指定角色。

## 使用方法

```bash
cp -r CP-开发流程/template/ CP-开发流程/{项目名称}/
```

复制后，将 `{项目名称}`、`{YYYY-MM-DD}` 等占位符替换为实际内容。

## 文件清单

| 文件 | 用途 | 读取频率 |
| --- | --- | --- |
| `Backlog.md` | 唯一需求入口 + 路线图 + Sprint 摘要 | Planning 必读 |
| `任务清单.md` | 当前 Sprint 的执行任务 | Execute / Review 必读 |
| `开发计划.md` | 慢变架构设计 | 仅架构、接口、边界、数据模型相关时读取 |
| `变更记录.md` | Sprint 级交付摘要 | Retro 或追溯最近交付时读取 |

## 需求输入模式

老板只需要编辑 `Backlog.md` 的「老板输入区」。支持以下写法：

```markdown
- [ ] 一句话需求
- [ ] 大段需求说明
- [ ] 接口草稿 / 配置片段 / 日志现象
- [ ] 外部链接 + 希望 AI 关注的点
```

AI 处理后会将原条目标记为 `[已梳理 → B-XXX]`，并把精简后的 Story、优先级、验收标准写入产品待办。

## 最小上下文原则

1. 会话启动先走 `trellis:start`，不直接进入开发。
2. Sprint Planning 默认只读 `Backlog.md` 未处理输入和待开发条目。
3. Execute 默认只读当前 Task、目标代码和必要规范。
4. 只有涉及架构、接口契约、数据模型、系统边界时才读 `开发计划.md`。
5. 只有交付总结或追溯历史时才读 `变更记录.md`。
6. `ai-team`、`zero-skills`、`eino-skills` 等能力按需载入，不预加载全文。

## 开发流程

```text
trellis:start
  → AI 团队接管
  → 老板在 Backlog.md 写需求
  → PM 梳理为 Backlog 条目和当前 Sprint
  → Backend / Frontend 执行任务
  → QA 验证
  → PM 追加 Sprint 级变更记录
  → 输出交付报告
```

## 兼容说明

旧项目如果已有 `需求输入.md`，AI 只在迁移或用户明确指定时读取；新项目不再创建该文件。
