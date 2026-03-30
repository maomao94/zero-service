---
name: example
description: "示例 Skill - 展示 SKILL.md 的标准格式"
allowed-tools:
  - Read
  - Grep
  - Glob
---

# Example Skill

这是一个示例 Skill，用于展示 SKILL.md 文件的标准格式。

## 格式说明

### Frontmatter

文件顶部的 YAML frontmatter 定义 Skill 元数据：

```yaml
---
name: skill-name           # Skill 唯一标识
description: "描述信息"      # Skill 功能描述
allowed-tools:             # 可选：允许使用的工具列表
  - Read
  - Grep
---
```

### 正文内容

Frontmatter 之后是 Skill 的完整内容，通常包括：

- **使用场景**: 什么时候应该使用这个 Skill
- **核心功能**: 主要提供哪些能力
- **最佳实践**: 如何正确使用
- **示例代码**: 实际使用示例

## MCP 资源格式

当 AI 客户端调用 `resources/read` 时，返回的内容格式为：

```
# {skill_name}

{description}

**Allowed Tools:** {tools}

---

{content}
```

## MCP Prompt 格式

当 AI 客户端调用 `prompts/get` 时，生成的提示词格式为：

```
你是 {name} 领域的专家。

## 领域描述
{description}

## 用户任务
{task}

## 参考知识
{content}
```

## 下一步

1. 复制此文件到你的 skills 目录
2. 修改 frontmatter 的 name 和 description
3. 编写你的 Skill 内容
4. 重启 MCP Server 即可生效
