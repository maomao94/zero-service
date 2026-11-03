# AI 协作与技能集

本项目在开发中使用 AI 编码助手，但**不把外部技能集的目录抄进仓库**：技能会随上游频繁更新，抄录必然过期。本文只约定协作契约与来源。

## 协作契约

- 行为约束以仓库根目录 [`AGENTS.md`](../AGENTS.md) 为准：中文沟通、改动前查证源码与契约、只改任务范围、完成前跑验证。
- 工程契约与领域规范存放在 OpenViking 记忆系统，见 [OpenViking 记忆系统](./openviking-memory.md)。
- 项目内的文档与代码规范以 [`CONTRIBUTING.md`](../.github/CONTRIBUTING.md) 与本文档索引为准。

## 技能集来源

技能以 OpenCode skill 机制安装在本机，不纳入版本控制：

| 技能集 | 上游 | 本地位置 |
| --- | --- | --- |
| mattpocock/skills | <https://github.com/mattpocock/skills> | `~/.skills/vendor/mattpocock-skills` |
| superpowers | <https://github.com/obra/superpowers> | 通过 OpenCode 插件安装，未启用时不生效 |

安装、更新与版本核对：

```bash
git -C ~/.skills/vendor/mattpocock-skills describe --tags
git -C ~/.skills/vendor/mattpocock-skills pull --ff-only
```

技能的增减只影响本机开发体验，不修改仓库文件；因此本仓库不记录技能清单，也不为单个技能写说明文档。

## 使用约定

1. **优先级**：项目内规范（`AGENTS.md`、`.github/CONTRIBUTING.md`、本目录文档）> 个人技能 > 外部技能集。
2. **技能是流程建议，不是事实来源**：涉及端口、字段、错误码时，一律以源码、`.proto`/`.api` 与 `etc/*.yaml` 为准。
3. **不批量导入文档**：项目文档以本目录为唯一事实源，技能只做工作流辅助。

## 相关文档

- [Agent 协作约定](../AGENTS.md)
- [OpenViking 记忆系统](./openviking-memory.md)
- [贡献指南](../.github/CONTRIBUTING.md)
