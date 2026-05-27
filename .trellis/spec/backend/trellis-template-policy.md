# Trellis 模板跟随策略

> 每次 `trellis update` 后必须保证 Trellis 托管文件与模板最新版一致，避免本地分叉导致后续升级冲突。

---

## 原则

- Trellis 托管的模板文件**全量跟随最新版**，不做项目级定制。
- 所有用户沉淀、项目规则、长期知识都放在 Trellis 预留的用户数据区。
- 每次 `trellis update` 后必须执行 `trellis update --dry-run`，确认输出为 `Already up to date!`。

---

## Trellis 托管文件（跟随模板，不做定制）

| 路径 | 说明 |
|------|------|
| `.trellis/workflow.md` | 工作流定义、phase、breadcrumb |
| `.trellis/config.yaml` | Trellis 项目配置、hooks、channel |
| `.trellis/scripts/**` | Trellis Python runtime 脚本 |
| `.trellis/.template-hashes.json` | 模板哈希索引，不手工编辑 |
| `.trellis/.version` | Trellis 版本记录 |
| `.opencode/agents/**` | OpenCode sub-agent 定义 |
| `.opencode/plugins/**` | OpenCode 插件（session-start、workflow-state、subagent-context） |
| `.opencode/lib/**` | OpenCode 插件共享库 |
| `.opencode/skills/**` | OpenCode skill 模板（包括 trellis-spec-bootstarp 等） |
| `.opencode/package.json` | OpenCode 插件依赖版本 |
| `.qoder/**` | Qoder 平台 hooks、agents、skills、settings |
| `AGENTS.md` 的 Trellis block | 不修改 Trellis 标记块内容 |

---

## 用户数据区（项目定制放这里）

| 路径 | 说明 |
|------|------|
| `.trellis/spec/**` | 项目代码规范、约定、深层知识 |
| `.trellis/tasks/**` | 任务 PRD、设计文档、实现计划、research |
| `.trellis/workspace/**` | 个人 journal、session 记录 |

> 业务项目自己的规则文件不属于 Trellis 模板，保持不动即可。

---

## 操作流程

### 1. 执行 `trellis update`

```bash
trellis update
```

### 2. 检查输出

- `Unchanged files` + `User data (preserved)` + `Already up to date!` = 正常。
- `Modified by you` = 本地分叉，必须处理。

### 3. 处理本地分叉

如果有 `Modified by you` 文件，直接覆盖回模板：

```bash
trellis update --force
```

### 4. 验证无残余

```bash
trellis update --dry-run
# 预期输出：
# ✓ Already up to date!
```

### 5. 检查备份

`trellis update --force` 会在 `.trellis/.backup-*/` 创建备份。确认不需要后可删除。

---

## 禁止事项

- 不在 Trellis 托管文件里写项目业务规则。
- 不手工编辑 `.trellis/.template-hashes.json`。
- 不让 `trellis update --dry-run` 输出任何 `Modified by you`。
- 不在 `.opencode/skills/`、`.opencode/plugins/` 等目录下放项目私有内容。
