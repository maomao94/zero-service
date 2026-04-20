# Advanced Skill Examples

This directory contains examples demonstrating advanced Claude Code skills features for go-zero development.

## What Are These Examples?

These are **example skill configurations** showing how to leverage Claude Code's advanced capabilities:
- Dynamic context injection with `!`command"` syntax
- Subagent workflows with `context: fork`
- Argument passing with `$ARGUMENTS`
- Tool restrictions with `allowed-tools`

These are **not production-ready skills** but templates you can adapt for your needs.

## Available Examples

### 1. [analyze-project.md](analyze-project.md)
**Use case**: Automated project analysis and architecture validation

**Features**:
- `context: fork` with Explore agent for isolated analysis
- Dynamic context with live file discovery
- Read-only tools for safe inspection
- Comprehensive reporting

**When to use**:
- Initial codebase audit
- Architecture compliance checking
- Onboarding new team members
- Code review automation

### 2. [generate-service.md](generate-service.md)
**Use case**: Generate go-zero services with proper structure

**Features**:
- Argument passing (`$0` = service name, `$1` = port)
- `disable-model-invocation: true` for manual control
- Dynamic context showing existing services
- Integration with goctl commands

**When to use**:
- Creating new API services
- Avoiding service name conflicts
- Ensuring consistent project structure
- Manual control over code generation

### 3. [plan-architecture.md](plan-architecture.md)
**Use case**: Design microservices architecture before coding

**Features**:
- `context: fork` with Plan agent for architecture design
- No execution tools (planning only)
- Multiple service coordination
- go-zero specific recommendations

**When to use**:
- Starting new projects
- Adding complex features
- Architecture reviews
- Team technical discussions

## How to Use These Examples

### Option 1: Create Custom Skills

Copy an example to `.claude/skills/` and customize:

```bash
# Create a new skill based on analyze-project
mkdir -p ~/.claude/skills/analyze-gozero
cp analyze-project.md ~/.claude/skills/analyze-gozero/SKILL.md

# Edit and customize
code ~/.claude/skills/analyze-gozero/SKILL.md
```

### Option 2: Learn the Patterns

Study these examples to understand:
- How to structure skill frontmatter
- When to use different agent types
- How dynamic context improves skills
- Best practices for argument handling

### Option 3: Adapt for Your Project

Create project-specific skills in `.claude/skills/`:

```
your-project/
└── .claude/
    └── skills/
        ├── analyze-myproject/
        │   └── SKILL.md          # Based on analyze-project.md
        └── deploy-myproject/
            └── SKILL.md          # Custom deployment workflow
```

## Key Concepts Demonstrated

### Dynamic Context Injection

```yaml
Existing services: !`find . -name "*.api" -type f`
```

- Executes **before** Claude sees the prompt
- Injects live project data
- Prevents stale or hardcoded information

### Subagent Contexts

```yaml
context: fork
agent: Explore
```

- Isolated execution (no conversation history)
- Specialized agent types (Explore, Plan)
- Clean separation of concerns

### Argument Handling

```yaml
argument-hint: [service-name] [port]

Generate service: $0 on port $1
```

- `$0`, `$1`: Positional arguments
- `$ARGUMENTS`: All arguments
- `${CLAUDE_SESSION_ID}`: Session tracking

### Tool Restrictions

```yaml
allowed-tools:
  - Read
  - Grep
  - Glob
```

- Limit Claude's capabilities for safety
- Prevent accidental modifications
- Enable auto-execution without approval

## Integration with zero-skills

These advanced skills **complement** the main zero-skills knowledge base:

```
┌─────────────────────────────────────────────┐
│  Main Skill (zero-skills)                   │
│  - Comprehensive knowledge base             │
│  - Pattern guides and best practices        │
│  - Automatic loading for go-zero work       │
└─────────────────┬───────────────────────────┘
                  │ References
┌─────────────────▼───────────────────────────┐
│  Advanced Skills (these examples)           │
│  - Project analysis workflows               │
│  - Service generation automation            │
│  - Architecture planning                    │
└─────────────────────────────────────────────┘
```

**Usage together**:
1. `zero-skills` provides knowledge: "Here's how to build go-zero services"
2. Advanced skills provide automation: "Analyze my project structure"
3. Both reference the same pattern guides

## Creating Your Own Skills

### Checklist

- [ ] Valid YAML frontmatter with `name` and `description`
- [ ] Clear `argument-hint` if arguments are expected
- [ ] Appropriate `context` and `agent` for the task
- [ ] `allowed-tools` specified for safety
- [ ] `disable-model-invocation: true` for side-effect operations
- [ ] References to supporting files (pattern guides, docs)
- [ ] Clear instructions for Claude to follow
- [ ] Dynamic context for live project data
- [ ] Example invocations and expected output

### Template

```yaml
---
name: my-gozero-skill
description: Brief description of what this skill does and when to use it
argument-hint: [arg1] [arg2]
context: fork                          # Optional: run in subagent
agent: Explore                         # Optional: which agent type
disable-model-invocation: true         # Optional: manual invoke only
user-invocable: true                   # Optional: show in /menu
allowed-tools:                         # Optional: tool restrictions
  - Read
  - Grep
---

# Your skill instructions here

## Context
- Project info: !`command to gather data`

## Task
1. Step one
2. Step two
3. Step three

## Output
Describe expected output format
```

## Testing Your Skills

1. **Install**: Place in `.claude/skills/your-skill/SKILL.md`
2. **Verify**: Ask "What skills are available?"
3. **Invoke**: Try `/your-skill` or let Claude load automatically
4. **Debug**: Check if commands execute, arguments work, etc.
5. **Iterate**: Refine based on actual usage

## Further Reading

- [Claude Code Skills Documentation](https://code.claude.com/docs/en/skills)
- [Agent Skills Specification](https://agentskills.io/)
- [zero-skills Main Documentation](../SKILL.md)
- [Claude Code Guide](../getting-started/claude-code-guide.md)
- [go-zero Official Documentation](https://go-zero.dev)

## Contributing

Have a useful skill pattern? Share it!
1. Test thoroughly in your projects
2. Document clearly with examples
3. Submit a PR to zero-skills
4. Help others build better skills

---

**Remember**: These are examples, not production skills. Adapt them to your specific needs and workflows.
