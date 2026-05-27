# Matt Pocock Skills 使用指南

> 来源：[github.com/mattpocock/skills](https://github.com/mattpocock/skills)
> 维护者：Matt Pocock
> 用途：AI 编程助手的工程化技能，提升开发质量

---

## 全局架构

```text
~/.skills/vendor/mattpocock-skills/     # 技能源（Git 仓库，只读）
  ├── OpenCode:  ~/.config/opencode/skills/* -> ~/.skills/vendor/...
  ├── Qoder:     ~/.qoder/skills/* -> ~/.skills/vendor/...
  └── 其他工具:  软链接到同一个技能源
```

### 为什么用这种结构

- 技能源只存一份，不重复
- 更新一次，所有工具都能拿到最新
- 各工具入口互不影响

### 更新技能

```bash
git -C ~/.skills/vendor/mattpocock-skills pull --ff-only
```

---

## 已安装技能清单（18 个）

### 工程类（Engineering）

| 技能 | 用途 | 触发方式 |
|---|---|---|
| `diagnose` | 系统化排查 bug / 性能问题。复现、缩小范围、假设、打点、修复、回归 | `用 diagnose 排查这个异常` |
| `grill-with-docs` | 开发前追问需求，同时维护领域语言和 ADR | `用 grill-with-docs 先问清楚需求` |
| `improve-codebase-architecture` | 分析代码架构，找耦合和边界问题 | `用 improve-codebase-architecture 看看这个模块` |
| `prototype` | 做一次性原型探索方案 | `用 prototype 快速验证这个思路` |
| `setup-matt-pocock-skills` | 初始化这套技能的项目级配置 | `用 setup-matt-pocock-skills 初始化` |
| `tdd` | 测试驱动开发，红绿重构 | `用 tdd 实现，先写失败测试` |
| `to-issues` | 把 PRD / 计划拆成可独立开发的 issues | `用 to-issues 拆成小任务` |
| `to-prd` | 把讨论整理成 PRD | `用 to-prd 整理成 PRD` |
| `triage` | issue 分诊，判断优先级和类型 | `用 triage 整理这些 issue` |
| `zoom-out` | 从系统视角解释代码 | `用 zoom-out 解释这个类的作用` |

### 效率类（Productivity）

| 技能 | 用途 | 触发方式 |
|---|---|---|
| `caveman` | 极简沟通，减少废话 | `用 caveman 模式回答` |
| `grill-me` | 对想法连续追问，直到需求清楚 | `用 grill-me 追问我这个方案` |
| `handoff` | 生成交接文档，方便下一个会话接手 | `用 handoff 总结当前工作` |
| `write-a-skill` | 帮你写新的 skill | `用 write-a-skill 写一个排查 skill` |

### 杂项（Misc）

| 技能 | 用途 | 触发方式 |
|---|---|---|
| `git-guardrails-claude-code` | Git 安全钩子，防危险命令 | `用 git-guardrails-claude-code 加安全限制` |
| `migrate-to-shoehorn` | TS 测试断言迁移 | `用 migrate-to-shoehorn 迁移断言` |
| `scaffold-exercises` | 生成练习题目录结构 | `用 scaffold-exercises 创建练习` |
| `setup-pre-commit` | 配置 Husky、lint-staged 等 | `用 setup-pre-commit 配置提交前检查` |

---

## 使用方法

### 推荐：自然语言点名

最稳的方式是直接说技能名：

```text
用 tdd 修这个 bug，先写失败测试
```

```text
用 diagnose 排查这个 NPE，先不要直接改代码
```

```text
用 grill-with-docs 帮我梳理这个需求
```

```text
用 zoom-out 解释这个模块的整体设计
```

### 也可尝试：Slash Command

部分工具支持 slash 调用：

```text
/tdd
/diagnose
/grill-with-docs
/zoom-out
```

如果 slash 不识别，重启工具后再试；自然语言点名通常最稳。

### 会自动路由吗

可以，但不保证每次都选中。复杂需求或你想强约束流程时，直接点名。

---

## 常用工作流

### 需求不清楚

```text
用 grill-with-docs 帮我梳理这个需求，先问问题，不要写代码
```

### 修复杂 Bug

```text
用 diagnose 排查这个 bug，先复现和定位根因，不要直接加防御代码
```

### 按测试驱动开发

```text
用 tdd 实现这个接口，先写失败测试再修实现
```

### 看不懂代码

```text
用 zoom-out 解释这个类在整个系统里的职责和上下游关系
```

### 交接上下文

```text
用 handoff 总结当前会话，包括已完成、未完成、验证情况和下一步
```

---

## 给其他工具软链接

从 `~/.skills/vendor/` 软链接即可：

```bash
# Cursor
ln -s ~/.skills/vendor/mattpocock-skills/skills/engineering/tdd ~/.cursor/skills/tdd

# Trae CN
ln -s ~/.skills/vendor/mattpocock-skills/skills/engineering/tdd ~/.trae-cn/skills/tdd

# Qoder
ln -s ~/.skills/vendor/mattpocock-skills/skills/engineering/tdd ~/.qoder/skills/tdd
```

批量软链接（以 Qoder 为例）：

```bash
for category in engineering productivity misc; do
  for skill_dir in ~/.skills/vendor/mattpocock-skills/skills/$category/*/; do
    name=$(basename "$skill_dir")
    if [ -f "$skill_dir/SKILL.md" ] && [ ! -e "$HOME/.qoder/skills/$name" ]; then
      ln -s "$skill_dir" "$HOME/.qoder/skills/$name"
    fi
  done
done
```

---

## 未安装的技能

仓库里还有这几类，我没有默认安装：

- `deprecated`：已废弃，不建议新用
- `in-progress`：开发中，不稳定
- `personal`：作者个人工作流，不一定通用

如需安装，从 `~/.skills/vendor/mattpocock-skills/skills/` 对应目录软链接即可。

---

## 安全提醒

- GitHub Fine-grained Token 只给 `Contents: Read-only`，有效期 7-30 天
- 公开仓库一般不需要 token；如果需要，不要贴到聊天里
- token 贴出后视为泄露，立即 revoke 并重新生成
