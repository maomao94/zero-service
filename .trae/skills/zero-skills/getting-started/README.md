# Getting Started with zero-skills

This directory contains guides for using zero-skills with different AI coding tools.

## Choose Your Tool

| Tool | Guide | Skills Support | Best For |
|------|-------|----------------|----------|
| **Claude Code** | [claude-code-guide.md](claude-code-guide.md) | Native | Full features, subagents, dynamic context |
| **Cursor** | [cursor-guide.md](cursor-guide.md) | Via rules | IDE integration, fast responses |
| **GitHub Copilot** | [copilot-guide.md](copilot-guide.md) | Via instructions | VS Code users, inline suggestions |
| **Windsurf** | [windsurf-guide.md](windsurf-guide.md) | Via rules | IDE integration, Cascade AI |

## Feature Comparison

| Feature | Claude Code | Cursor | Copilot | Windsurf |
|---------|-------------|--------|---------|----------|
| Native skills support | Yes | No | No | No |
| YAML frontmatter | Yes | No | No | No |
| Subagent workflows | Yes | No | No | No |
| Dynamic context (`!cmd`) | Yes | No | No | No |
| Project rules | `.claude/` | `.cursorrules` | `.github/` | `.windsurfrules` |
| Auto-load by file type | Yes | Manual | Manual | Manual |
| Tool restrictions | Yes | No | No | No |

## Quick Comparison

### Claude Code (Recommended for go-zero)

**Pros:**
- Native Agent Skills support
- Automatic skill loading
- Subagent workflows (Explore, Plan)
- Dynamic context injection
- Tool restrictions for safety

**Cons:**
- Requires Claude Code CLI
- Learning curve for advanced features

**Best for:** Comprehensive go-zero development with full automation.

### Cursor

**Pros:**
- Fast IDE integration
- Good context awareness
- Project-wide rules
- Popular among developers

**Cons:**
- No native skills support
- Manual context loading
- Rules file can get large

**Best for:** Developers who prefer IDE-based AI assistance.

### GitHub Copilot

**Pros:**
- Deep VS Code integration
- Inline suggestions
- Widely adopted
- Chat interface

**Cons:**
- Limited instruction size
- No structured skills
- Manual context management

**Best for:** VS Code users who want inline go-zero suggestions.

### Windsurf

**Pros:**
- Cascade AI for complex tasks
- Good file context
- Project rules support

**Cons:**
- No native skills support
- Newer tool, evolving features

**Best for:** Developers who like Cascade AI's approach.

## Installation Overview

All tools follow a similar pattern:

```bash
# 1. Clone zero-skills to your project
git clone https://github.com/zeromicro/zero-skills.git .ai-context/zero-skills

# 2. Configure your tool (see specific guide)
```

Then configure based on your tool:

| Tool | Configuration |
|------|---------------|
| Claude Code | Clone to `.claude/skills/zero-skills/` |
| Cursor | Reference in `.cursorrules` |
| Copilot | Reference in `.github/copilot-instructions.md` |
| Windsurf | Reference in `.windsurfrules` |

## Key Principles (All Tools)

Regardless of which tool you use, these go-zero principles apply:

**Always:**
- Handler → Logic → Model separation
- Use `httpx.Error()` for HTTP errors
- Load config with `conf.MustLoad`
- Pass `ctx` through all layers
- Generate code with `goctl`

**Never:**
- Put business logic in handlers
- Hard-code configuration
- Skip error handling
- Bypass ServiceContext injection

## Additional Resources

- **[SKILL.md](../SKILL.md)** - Main skill entry point
- **[Pattern guides](../references/)** - Detailed patterns
- **[go-zero docs](https://go-zero.dev)** - Official documentation
- **[ai-context](https://github.com/zeromicro/ai-context)** - Lightweight workflow instructions
