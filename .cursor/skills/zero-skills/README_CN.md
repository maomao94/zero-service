# go-zero Skills - AI 助手的知识库

[English](README.md) | 简体中文

这是一个 [Agent Skill（智能体技能）](https://anthropic.com/engineering/equipping-agents-for-the-real-world-with-agent-skills)，包含为 AI 编程助手优化的 go-zero 框架知识和模式，帮助开发者更高效地构建微服务应用。

## 什么是 Skill？

Skills 是包含指令、脚本和资源的文件夹，AI 智能体可以动态发现和加载，以更好地完成特定任务。这个 skill 教会 AI 智能体如何生成生产级的 go-zero 微服务代码。

## 目标

本 skill 使 AI 助手（Claude、GitHub Copilot、Cursor 等）能够：
- 生成符合 go-zero 规范的准确代码
- 理解三层架构（Handler → Logic → Model）
- 应用微服务开发最佳实践
- 高效排查常见问题
- 构建生产就绪的应用

## 快速安装

只需告诉你的 AI 助手：

```
Install zero-skills from https://github.com/zeromicro/zero-skills
```

或者手动安装：

```bash
# 项目级别（推荐）
git clone https://github.com/zeromicro/zero-skills.git .claude/skills/zero-skills

# 个人级别（所有项目可用）
git clone https://github.com/zeromicro/zero-skills.git ~/.claude/skills/zero-skills
```

## Agent Skill 结构

遵循 [Agent Skills 规范](https://github.com/anthropics/skills/blob/main/spec/agent-skills-spec.md) 和 [Claude Code skills 文档](https://code.claude.com/docs/en/skills)：

```
zero-skills/
├── SKILL.md                    # 入口文件，包含 YAML 元数据
├── getting-started/            # 快速开始指南
│   ├── README.md               # 工具对比概览
│   ├── claude-code-guide.md    # Claude Code（推荐）
│   ├── cursor-guide.md         # Cursor IDE
│   ├── copilot-guide.md        # GitHub Copilot
│   └── windsurf-guide.md       # Windsurf IDE
├── references/                 # 详细模式文档
│   ├── rest-api-patterns.md    # REST API 开发模式
│   ├── rpc-patterns.md         # gRPC 服务模式
│   ├── database-patterns.md    # 数据库操作
│   └── resilience-patterns.md  # 弹性和容错
├── best-practices/             # 生产级建议
├── troubleshooting/            # 常见问题和解决方案
├── skill-patterns/             # 高级技能示例（模板）
│   ├── analyze-project.md      # Explore 代理示例
│   ├── generate-service.md     # 参数传递示例
│   └── plan-architecture.md    # Plan 代理示例
└── examples/                   # 演示项目和验证脚本
```

## 使用这个 Skill

### 在 Claude Code 中使用（推荐）

Claude Code 原生支持 [Agent Skills 规范](https://agentskills.io/)。本 skill 针对 Claude Code 进行了优化，支持高级功能：

#### 项目级安装（Git Submodule）
将 zero-skills 添加到项目中以自动发现：

```bash
# 添加为 git submodule
git submodule add https://github.com/zeromicro/zero-skills.git .claude/skills/zero-skills

# 或直接克隆
git clone https://github.com/zeromicro/zero-skills.git .claude/skills/zero-skills
```

Claude Code 会自动发现 `.claude/skills/` 目录中的 skills。

#### 个人级安装
跨所有项目使用，安装到个人 skills 目录：

```bash
# 克隆到个人 skills 目录
git clone https://github.com/zeromicro/zero-skills.git ~/.claude/skills/zero-skills
```

#### 在 Claude Code 中的使用方式
- **自动加载**：处理 go-zero 文件（`.api`、`.proto`、包含 go-zero 的 `go.mod`）时自动加载
- **手动调用**：输入 `/zero-skills` 直接调用获取 go-zero 指导
- **带参数调用**：`/zero-skills 创建用户管理 API` 用于特定任务
- **检查可用性**：询问 "What skills are available?" 查看是否已加载

#### 高级功能
- **动态上下文**：Skills 可以执行 shell 命令获取实时项目数据
- **子代理**：使用 `context: fork` 进行隔离的分析或规划任务
- **工具限制**：`allowed-tools` 确保安全的只读操作
- 参见 [skill-patterns/](skill-patterns/) 获取高级模式和模板

### 在 Claude Desktop 中使用

添加到 `claude_desktop_config.json`：
```json
{
  "mcpServers": {
    "zero-skills": {
      "command": "node",
      "args": ["/path/to/skill-server.js", "/path/to/zero-skills"]
    }
  }
}
```

### 在 GitHub Copilot 中使用

参见 [copilot-guide.md](getting-started/copilot-guide.md) 获取详细设置。快速开始：

```bash
git clone https://github.com/zeromicro/zero-skills.git .ai-context/zero-skills
```

然后创建 `.github/copilot-instructions.md` 引用模式文件。

### 在 Cursor 中使用

参见 [cursor-guide.md](getting-started/cursor-guide.md) 获取详细设置。快速开始：

```bash
git clone https://github.com/zeromicro/zero-skills.git .ai-context/zero-skills
```

然后创建 `.cursorrules` 引用模式文件。

### 在 Windsurf 中使用

参见 [windsurf-guide.md](getting-started/windsurf-guide.md) 获取详细设置。快速开始：

```bash
git clone https://github.com/zeromicro/zero-skills.git .ai-context/zero-skills
```

然后创建 `.windsurfrules` 引用模式文件。

## 与 go-zero AI 生态集成

zero-skills 是 go-zero AI 辅助开发三件套生态的一部分：

| 工具 | 用途 | 大小 | 最适合 |
|------|------|------|--------|
| **[ai-context](https://github.com/zeromicro/ai-context)** | 工作流指令和决策树 | ~5KB | GitHub Copilot, Cursor, Windsurf |
| **zero-skills**（本仓库） | 完整知识库 | ~40KB | Claude Code，深度学习，参考 |
| **[mcp-zero](https://github.com/zeromicro/mcp-zero)** | 运行时工具（执行 goctl 命令） | MCP Server | Claude Desktop/Code 代码生成 |

### 它们如何协作

```
┌─────────────────────────────────────────────────────────────┐
│                     AI 助手                                  │
│  (Claude Code, GitHub Copilot, Cursor, 等)                  │
└────────────┬─────────────────────┬──────────────────────────┘
             │                     │
             ├─ 工作流层 ──────────┤
             │  ai-context         │  "做什么" - 快速决策
             │  (~5KB)             │  每次交互都加载
             │                     │
             ├─ 知识层 ────────────┤
             │  zero-skills        │  "如何和为什么" - 详细模式
             │  (~40KB)            │  需要时加载
             │                     │
             └─ 执行层 ────────────┘
                mcp-zero             "执行" - 运行 goctl 命令
                (MCP Server)          生成实际代码文件
```

### 使用场景

**场景 1: GitHub Copilot 用户**
- 使用：`ai-context`（通过 `.github/copilot-instructions.md` 加载）
- 优点：快速内联建议，工作流指导
- 限制：无代码执行，需手动运行 goctl 命令

**场景 2: Claude Code 用户（最佳体验）**
- 使用：`zero-skills`（本仓库）+ `mcp-zero` 工具
- 优点：
  - 来自模式指南的深度知识
  - 通过 goctl 自动生成代码
  - 实时项目数据的动态上下文
  - 复杂任务的子代理工作流
- 调用：`/zero-skills` 或处理 go-zero 时自动加载

**场景 3: Cursor/Windsurf 用户**
- 使用：`ai-context`（在项目规则中）+ `zero-skills` 链接
- 优点：IDE 原生体验加 go-zero 指导

参见 [入门指南](getting-started/) 获取每个工具的详细集成说明。

## 快速链接

**Skill 文档：**

- 📖 **[SKILL.md](SKILL.md)** - 主要 skill 入口和导航
- 📚 **[go-zero 快速开始](https://go-zero.dev/docs/quick-start)** - 官方 go-zero 框架教程
- 🎯 **[高级示例](skill-patterns/)** - 子代理，动态上下文等

**入门指南：**

- 💡 **[Claude Code](getting-started/claude-code-guide.md)** - 完整功能，子代理（推荐）
- 🖱️ **[Cursor](getting-started/cursor-guide.md)** - IDE 集成 .cursorrules
- 🤖 **[GitHub Copilot](getting-started/copilot-guide.md)** - VS Code 内联建议
- 🏄 **[Windsurf](getting-started/windsurf-guide.md)** - Cascade AI 集成
- 📋 **[工具对比](getting-started/README.md)** - 比较所有工具

## 贡献指南

欢迎贡献！请确保：
- 示例完整且经过测试
- 模式遵循官方 go-zero 约定
- 内容结构化，便于 AI 理解
- 包含正确（✅）和错误（❌）的示例对比
- 遵循 [Agent Skills 规范](https://agentskills.io/)

## 许可证

MIT License - 与 go-zero 框架相同
