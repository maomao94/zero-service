# Using zero-skills with Cursor

This guide explains how to use zero-skills with [Cursor](https://cursor.sh), the AI-first code editor.

## Installation

### Step 1: Clone zero-skills

```bash
cd your-gozero-project/

# Clone to a local directory
git clone https://github.com/zeromicro/zero-skills.git .ai-context/zero-skills
```

### Step 2: Create .cursorrules

Create `.cursorrules` in your project root:

```markdown
# go-zero Development Rules

You are an expert in go-zero microservices framework development.

## Key Principles

Follow these go-zero patterns strictly:

### Architecture
- **Three-layer separation**: Handler (HTTP) → Logic (business) → Model (data)
- Never put business logic in handlers
- Always use ServiceContext for dependency injection

### Code Generation
- Use `goctl` for code generation, never hand-write boilerplate
- API definitions go in `.api` files
- RPC definitions go in `.proto` files

### Error Handling
- Use `httpx.Error(w, err)` for HTTP errors
- Use `httpx.OkJson(w, resp)` for success responses
- Never use `fmt.Fprintf()` or `w.Write()` directly

### Configuration
- Load config with `conf.MustLoad(&c, *configFile)`
- Never hard-code ports, hosts, or credentials
- Use environment-specific config files

### Context Propagation
- Always pass `ctx context.Context` through all layers
- Use context for tracing, cancellation, and timeouts

## Pattern References

When I need detailed patterns, I'll reference these files:
- REST APIs: .ai-context/zero-skills/references/rest-api-patterns.md
- RPC services: .ai-context/zero-skills/references/rpc-patterns.md
- Database: .ai-context/zero-skills/references/database-patterns.md
- Resilience: .ai-context/zero-skills/references/resilience-patterns.md
- Troubleshooting: .ai-context/zero-skills/troubleshooting/common-issues.md

## Common Commands

```bash
# Generate API code
goctl api go -api user.api -dir .

# Generate RPC code
goctl rpc protoc user.proto --go_out=. --go-grpc_out=. --zrpc_out=.

# Generate model from database
goctl model mysql datasource -url="user:pass@tcp(localhost:3306)/db" -table="users" -dir="./model"
```
```

## Usage

### Automatic Context

Cursor will apply these rules to all conversations in your project.

### Reference Pattern Files

When you need detailed patterns, ask Cursor to read the specific file:

```
Read .ai-context/zero-skills/references/rest-api-patterns.md and help me implement a user API
```

### Example Conversations

**Creating a REST API:**
```
Create a user management REST API with CRUD operations following go-zero patterns
```

**Adding middleware:**
```
Help me add JWT authentication middleware to my go-zero API
```

**Database integration:**
```
Read the database patterns and help me add MySQL support to my user service
```

## Tips

### Keep Rules Concise

Cursor has context limits. Keep `.cursorrules` focused on key principles and reference pattern files for details.

### Use @-mentions

Reference files directly in chat:
```
@.ai-context/zero-skills/references/database-patterns.md help me implement caching
```

### Combine with ai-context

For even more concise rules, use [ai-context](https://github.com/zeromicro/ai-context):

```bash
git clone https://github.com/zeromicro/ai-context.git .ai-context/ai-context
```

Then reference it in `.cursorrules`:
```markdown
Follow workflows from .ai-context/ai-context/
For detailed patterns, see .ai-context/zero-skills/
```

## Limitations

Compared to Claude Code, Cursor:
- No native skills support (YAML frontmatter ignored)
- No automatic loading by file type
- No subagent workflows
- No dynamic context injection
- Manual file references needed

## Troubleshooting

### Rules Not Applied

**Problem:** Cursor doesn't seem to follow go-zero patterns.

**Solutions:**
1. Check `.cursorrules` exists in project root
2. Restart Cursor after creating rules
3. Reference pattern files explicitly in chat

### Context Too Large

**Problem:** Rules file is too long.

**Solutions:**
1. Keep `.cursorrules` under 500 lines
2. Move detailed patterns to separate files
3. Reference files with @-mentions

### Outdated Patterns

**Problem:** Suggestions don't match latest go-zero.

**Solutions:**
1. Update zero-skills: `cd .ai-context/zero-skills && git pull`
2. Check go-zero version compatibility

## Additional Resources

- [Cursor Documentation](https://cursor.sh/docs)
- [go-zero Official Docs](https://go-zero.dev)
- [zero-skills Pattern Guides](../references/)
