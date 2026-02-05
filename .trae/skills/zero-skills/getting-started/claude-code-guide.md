# Using zero-skills with Claude Code

This guide explains how to use zero-skills effectively with Claude Code, leveraging its advanced skills capabilities.

## Table of Contents
- [Quick Reference](#quick-reference)
- [Installation](#installation)
- [Basic Usage](#basic-usage)
- [Advanced Features](#advanced-features)
- [Skill Pattern Examples](#skill-pattern-examples)
- [Example Workflows](#example-workflows)
- [Troubleshooting](#troubleshooting)
- [Best Practices](#best-practices)

---

## Quick Reference

### Commands

| Command | Description |
|---------|-------------|
| `/zero-skills` | Load skill for go-zero help |
| `/zero-skills [query]` | Load skill with specific question |
| Ask "What skills are available?" | Check loaded skills |

### Key Principles

**Always Do:**
- Handler → Logic → Model separation
- Use `httpx.Error()` for HTTP errors
- Load config with `conf.MustLoad`
- Pass `ctx` through all layers
- Generate code with `goctl`

**Never Do:**
- Put business logic in handlers
- Hard-code configuration
- Skip error handling
- Bypass ServiceContext injection

### Pattern Guides

| Guide | When to Use |
|-------|-------------|
| [rest-api-patterns.md](../references/rest-api-patterns.md) | REST APIs, handlers, middleware |
| [rpc-patterns.md](../references/rpc-patterns.md) | gRPC services, service discovery |
| [database-patterns.md](../references/database-patterns.md) | SQL, MongoDB, Redis, caching |
| [resilience-patterns.md](../references/resilience-patterns.md) | Circuit breakers, rate limiting |
| [common-issues.md](../troubleshooting/common-issues.md) | Debugging errors |
| [overview.md](../best-practices/overview.md) | Production hardening |

### Integration with Other Tools

| Tool | Purpose | Command/Usage |
|------|---------|---------------|
| **mcp-zero** | Execute goctl commands | Claude calls automatically |
| **ai-context** | Quick workflows | GitHub Copilot integration |
| **goctl** | Code generation | `goctl api go`, `goctl model`, etc. |

---

## Installation

### Option 1: Project-Level (Recommended for go-zero projects)

Add zero-skills to your project for automatic discovery:

```bash
cd your-gozero-project/

# Create skills directory
mkdir -p .claude/skills

# Clone zero-skills
git clone https://github.com/zeromicro/zero-skills.git .claude/skills/zero-skills
```

Claude Code automatically discovers skills in `.claude/skills/` directories.

### Option 2: Personal-Level (Available across all projects)

Install to your personal skills directory to use with any go-zero project:

```bash
# Create personal skills directory
mkdir -p ~/.claude/skills

# Clone zero-skills
git clone https://github.com/zeromicro/zero-skills.git ~/.claude/skills/zero-skills
```

### Option 3: Enterprise-Level (For organizations)

Distribute via managed settings (requires Claude for Enterprise):
1. Add skill to your organization's managed settings
2. All team members get the skill automatically
3. See [Claude Code IAM documentation](https://code.claude.com/docs/en/iam#managed-settings)

## Basic Usage

### Automatic Invocation

Claude automatically loads the skill when you:
- Open or edit `.api` files (REST API definitions)
- Open or edit `.proto` files (gRPC definitions)
- Work with `go.mod` that includes `github.com/zeromicro/go-zero`
- Ask questions about go-zero

**Example:**
```
You: How do I create a REST API with go-zero?
```
Claude loads zero-skills and provides detailed guidance from [references/rest-api-patterns.md](../references/rest-api-patterns.md).

### Manual Invocation

Invoke directly with `/zero-skills`:

```
/zero-skills
```

Or with arguments:
```
/zero-skills Create a user management API with authentication
/zero-skills How do I implement rate limiting?
/zero-skills Explain the three-layer architecture
```

### Check Skill Availability

```
You: What skills are available?
```

Claude lists all loaded skills. Look for `zero-skills` in the output.

## Advanced Features

### Dynamic Context Injection

Skills can execute shell commands to gather live project data using `!`command`` syntax.

**Example: Find existing services**
```yaml
---
name: check-services
---

Current services in this project:
- API services: !`find . -name "*.api" -type f`
- RPC services: !`find . -name "*.proto" -type f`
- Config files: !`find . -name "*-api.yaml" -o -name "*-rpc.yaml"`
```

The commands execute before Claude sees the prompt, injecting actual file paths.

See [skill-patterns/analyze-project.md](../skill-patterns/analyze-project.md) for a complete example.

### Subagent Workflows

Use `context: fork` to run skills in isolated subagent contexts:

#### Explore Agent (Read-Only Analysis)
```yaml
---
name: analyze-gozero-project
context: fork
agent: Explore
---

Analyze the go-zero project structure and identify issues...
```

Benefits:
- Isolated context (no conversation history leakage)
- Read-only tools prevent accidental modifications
- Focused analysis without distractions

See [skill-patterns/analyze-project.md](../skill-patterns/analyze-project.md) for details.

#### Plan Agent (Architecture Design)
```yaml
---
name: plan-microservices
context: fork
agent: Plan
---

Plan a microservices architecture for: $ARGUMENTS
```

Benefits:
- Optimized for planning and design
- No code execution (prevents premature implementation)
- Fresh perspective on architecture

See [skill-patterns/plan-architecture.md](../skill-patterns/plan-architecture.md) for details.

### Tool Restrictions

Control which tools Claude can use with `allowed-tools`:

```yaml
---
name: safe-gozero-review
allowed-tools:
  - Read
  - Grep
  - Glob
---

Review go-zero code without making changes...
```

Available tool categories:
- `Read`: Read files
- `Grep`: Search file contents
- `Glob`: Find files by pattern
- `Bash(goctl *)`: Run goctl commands only
- `Write`: Create/modify files

### Argument Passing

Skills can accept arguments for customization:

```yaml
---
name: generate-api-service
argument-hint: [service-name] [port]
---

Generate service: $0 on port $1
```

Usage:
```
/generate-api-service user 8080
/generate-api-service order 8081
```

Access arguments:
- `$0` or `$ARGUMENTS[0]`: First argument
- `$1` or `$ARGUMENTS[1]`: Second argument
- `$ARGUMENTS`: All arguments combined
- `${CLAUDE_SESSION_ID}`: Current session ID

See [skill-patterns/generate-service.md](../skill-patterns/generate-service.md) for details.

## Skill Pattern Examples

See [skill-patterns/](../skill-patterns/) for advanced skill patterns:

- **[analyze-project.md](../skill-patterns/analyze-project.md)** - Explore agent with dynamic context
- **[generate-service.md](../skill-patterns/generate-service.md)** - Argument passing patterns
- **[plan-architecture.md](../skill-patterns/plan-architecture.md)** - Plan agent for design
- **[README.md](../skill-patterns/README.md)** - Full guide with best practices

## Example Workflows

### Workflow 1: Building a New REST API

**Step 1: Manual invocation for planning**
```
/zero-skills Create a user management REST API with CRUD operations
```

Claude:
1. Loads [references/rest-api-patterns.md](../references/rest-api-patterns.md)
2. Explains the `.api` file structure
3. Shows example definitions
4. Guides you through Handler → Logic → Model setup

**Step 2: Implement with guidance**
```
You: How do I handle authentication in middleware?
```

Claude references [references/rest-api-patterns.md](../references/rest-api-patterns.md#middleware-patterns) and provides examples.

**Step 3: Troubleshoot issues**
```
You: I'm getting "http: named cookie not present" error
```

Claude loads [troubleshooting/common-issues.md](../troubleshooting/common-issues.md) and diagnoses the problem.

### Workflow 2: Analyzing an Existing Project

**Use the analyze-project skill (in subagent):**

```
You: Analyze this go-zero project for issues
```

Claude (automatically or if you have the analyze skill):
1. Forks to Explore agent
2. Finds all `.api` and `.proto` files
3. Checks architecture compliance
4. Identifies anti-patterns
5. Returns summary to main conversation

See [skill-patterns/analyze-project.md](../skill-patterns/analyze-project.md) for this skill template.

### Workflow 3: Planning Microservices Architecture

**Use the plan-microservices skill:**

```
/plan-microservices e-commerce platform with user, product, cart, and order services
```

Claude (using Plan agent):
1. Designs service boundaries
2. Specifies API vs RPC communication
3. Plans data storage strategy
4. Provides `.api` and `.proto` examples
5. Suggests implementation order

See [skill-patterns/plan-architecture.md](../skill-patterns/plan-architecture.md) for this skill template.

### Workflow 4: Combined with mcp-zero

If you have [mcp-zero](https://github.com/zeromicro/mcp-zero) installed:

```
You: Create a user API service with database operations
```

Claude:
1. Uses zero-skills for patterns and structure
2. Calls mcp-zero `create_api_service` to generate files
3. Calls mcp-zero `generate_model` for database models
4. Validates generated code against patterns
5. Shows you how to run and test

This combines **knowledge** (zero-skills) with **execution** (mcp-zero).

## Troubleshooting

### Skill Not Loading

**Problem**: Claude doesn't seem to have go-zero knowledge.

**Solutions**:
1. Check if skill is available: `What skills are available?`
2. Verify installation:
   ```bash
   ls -la ~/.claude/skills/zero-skills/SKILL.md
   # or
   ls -la .claude/skills/zero-skills/SKILL.md
   ```
3. Manually invoke: `/zero-skills`
4. Check frontmatter in SKILL.md (must have valid YAML)

### Skill Not Triggering Automatically

**Problem**: Have to manually invoke with `/zero-skills` every time.

**Solutions**:
1. Improve the description in SKILL.md to match your queries
2. Use go-zero specific keywords: "api", "rpc", "goctl", "handler", "logic"
3. Work with `.api` or `.proto` files (triggers automatic loading)

### Commands in Skill Not Executing

**Problem**: Dynamic context (`!`command``) not working.

**Possible causes**:
1. Commands execute before Claude sees them (this is by design)
2. Shell command errors are silent - check command syntax
3. Working directory might not be what you expect

**Debug**:
```yaml
Current directory: !`pwd`
Go version: !`go version`
Files: !`ls -la`
```

### Subagent Not Working

**Problem**: `context: fork` skill doesn't run in isolation.

**Solutions**:
1. Verify `context: fork` in frontmatter
2. Specify agent type: `agent: Explore` or `agent: Plan`
3. Check allowed-tools (subagents need explicit tool permissions)

### Skill Triggers Too Often

**Problem**: zero-skills loads even when not working with go-zero.

**Solutions**:
1. Make description more specific in SKILL.md
2. Add `disable-model-invocation: true` to prevent automatic loading
3. Only invoke manually with `/zero-skills` when needed

### Quick Troubleshooting Table

| Problem | Solution |
|---------|----------|
| Skill not loading | Check: `What skills are available?` |
| Not auto-triggering | Invoke manually: `/zero-skills` |
| Need specific pattern | Ask: "Show me REST API middleware patterns" |
| Want to analyze project | Say: "Analyze this go-zero project" |

## Best Practices

### 1. Use Specific Invocations

Instead of:
```
/zero-skills Help me
```

Be specific:
```
/zero-skills Implement rate limiting for my API service
```

### 2. Leverage Supporting Files

Don't load everything. Reference specific guides:
```
You: How do I implement database transactions?
```

Claude loads just [references/database-patterns.md](../references/database-patterns.md), not the entire skill.

### 3. Combine with mcp-zero

Use zero-skills for knowledge, mcp-zero for execution:
- zero-skills: "What pattern should I use?"
- mcp-zero: "Generate the code for me"

### 4. Create Custom Skills

Build project-specific skills in `.claude/skills/` that extend zero-skills:

```
.claude/
└── skills/
    ├── zero-skills/         # Base knowledge
    └── myproject-gozero/    # Project-specific
        └── SKILL.md
```

Example custom skill:
```yaml
---
name: myproject-gozero
description: Project-specific go-zero patterns for MyProject
---

This project uses zero-skills patterns with these customizations:
- All APIs use JWT authentication (see internal/middleware/auth.go)
- Database models use soft deletes
- Service discovery via Kubernetes (not etcd)

For general patterns, see zero-skills. This skill covers project-specific overrides.
```

### 5. Use Subagents for Complex Tasks

Create skills with `context: fork` for:
- **Analysis**: Use Explore agent for codebase review
- **Planning**: Use Plan agent for architecture design
- **Isolation**: Keep experimental or risky operations separate

## Learning Path

1. **New to go-zero?** → [Official Quick Start](https://go-zero.dev/docs/quick-start)
2. **Building APIs?** → [references/rest-api-patterns.md](../references/rest-api-patterns.md)
3. **Adding database?** → [references/database-patterns.md](../references/database-patterns.md)
4. **Production ready?** → [best-practices/overview.md](../best-practices/overview.md)

## Additional Resources

- **Official docs**: [code.claude.com/docs/en/skills](https://code.claude.com/docs/en/skills)
- **Agent Skills spec**: [agentskills.io](https://agentskills.io/)
- **go-zero docs**: [go-zero.dev](https://go-zero.dev)
- **mcp-zero tools**: [github.com/zeromicro/mcp-zero](https://github.com/zeromicro/mcp-zero)
- **ai-context**: [github.com/zeromicro/ai-context](https://github.com/zeromicro/ai-context)

## Feedback and Contributions

Found an issue or want to improve zero-skills?
- Open an issue: [github.com/zeromicro/zero-skills/issues](https://github.com/zeromicro/zero-skills/issues)
- Submit a PR: [github.com/zeromicro/zero-skills/pulls](https://github.com/zeromicro/zero-skills/pulls)
- Join the community: See main go-zero repository

---

**Tips:**
- Be specific: "Create a user API with authentication" > "Help me"
- Reference files: ".api files" or "REST API" trigger automatic loading
- Use mcp-zero: Combine knowledge (zero-skills) + execution (mcp-zero)
- Create custom skills: Extend for project-specific patterns
- Check examples: See [skill-patterns/](../skill-patterns/) for advanced usage

**Need help?** Just ask Claude: "How do I [task] with go-zero?"
