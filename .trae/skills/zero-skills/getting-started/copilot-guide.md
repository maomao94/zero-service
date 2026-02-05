# Using zero-skills with GitHub Copilot

This guide explains how to use zero-skills with [GitHub Copilot](https://github.com/features/copilot) in VS Code.

## Installation

### Step 1: Clone zero-skills

```bash
cd your-gozero-project/

# Clone to a local directory
git clone https://github.com/zeromicro/zero-skills.git .ai-context/zero-skills
```

### Step 2: Create Copilot Instructions

Create `.github/copilot-instructions.md` in your project:

```markdown
# go-zero Development Instructions

You are an expert in go-zero microservices framework development.

## Architecture

Follow the three-layer architecture strictly:
- **Handler**: HTTP routing and request/response handling only
- **Logic**: All business logic goes here
- **Model**: Data access and database operations

## Code Patterns

### REST API Handler
```go
func (l *UserLogic) GetUser(req *types.GetUserReq) (*types.GetUserResp, error) {
    user, err := l.svcCtx.UserModel.FindOne(l.ctx, req.Id)
    if err != nil {
        return nil, err
    }
    return &types.GetUserResp{
        Id:   user.Id,
        Name: user.Name,
    }, nil
}
```

### Error Handling
- Use `httpx.Error(w, err)` for errors
- Use `httpx.OkJson(w, resp)` for success
- Never use `fmt.Fprintf()` directly

### Configuration
- Use `conf.MustLoad(&c, *configFile)`
- Never hard-code values

## Commands

```bash
# Generate API
goctl api go -api user.api -dir .

# Generate model
goctl model mysql datasource -url="..." -table="users" -dir="./model"
```

## Key Rules

1. Never put business logic in handlers
2. Always use ServiceContext for dependencies
3. Always pass ctx through all layers
4. Use goctl for code generation
```

## Usage

### Inline Suggestions

Copilot provides inline suggestions as you type. With the instructions file, it will follow go-zero patterns.

### Copilot Chat

Use Copilot Chat for more complex questions:

```
How do I add middleware to my go-zero API?
```

### Reference Pattern Files

For detailed patterns, ask Copilot to read files:

```
@workspace Read .ai-context/zero-skills/references/rest-api-patterns.md and help me create a user API
```

## Example Workflows

### Creating a New API

1. **Define the API file:**
   ```
   Help me create a user.api file for a user management service
   ```

2. **Generate code:**
   ```bash
   goctl api go -api user.api -dir .
   ```

3. **Implement logic:**
   ```
   Help me implement the GetUser logic following go-zero patterns
   ```

### Adding Database Support

1. **Ask for model generation:**
   ```
   What goctl command should I use to generate a model for the users table?
   ```

2. **Implement data access:**
   ```
   Help me add the user model to ServiceContext
   ```

### Troubleshooting

```
I'm getting "http: named cookie not present" error in my go-zero API
```

## Tips

### Keep Instructions Concise

GitHub Copilot has token limits. Keep `.github/copilot-instructions.md` focused:
- Key principles only
- Short code examples
- Reference external files for details

### Use @workspace

Reference your entire workspace:
```
@workspace What go-zero services do we have?
```

### Combine with ai-context

For minimal instructions, use [ai-context](https://github.com/zeromicro/ai-context):

```markdown
# .github/copilot-instructions.md

Follow go-zero patterns from .ai-context/ai-context/
For detailed patterns, see .ai-context/zero-skills/references/
```

### Open Pattern Files

Keep pattern files open in editor tabs. Copilot uses open files as context.

## Limitations

Compared to Claude Code, GitHub Copilot:
- No native skills support
- Limited instruction file size
- No automatic pattern loading
- No subagent workflows
- Primarily inline suggestions

## Workspace Settings

Optionally configure in `.vscode/settings.json`:

```json
{
  "github.copilot.chat.codeGeneration.instructions": [
    {
      "text": "Follow go-zero three-layer architecture: Handler → Logic → Model"
    },
    {
      "text": "Use httpx.Error() for errors and httpx.OkJson() for success responses"
    },
    {
      "text": "Always use goctl for code generation"
    }
  ]
}
```

## Troubleshooting

### Instructions Not Applied

**Problem:** Copilot ignores go-zero patterns.

**Solutions:**
1. Verify `.github/copilot-instructions.md` exists
2. Restart VS Code
3. Open pattern files in editor tabs
4. Use explicit @workspace references

### Suggestions Are Generic

**Problem:** Copilot gives generic Go code, not go-zero specific.

**Solutions:**
1. Add more go-zero examples to instructions
2. Reference pattern files explicitly
3. Keep go-zero files open in editor

### Context Limit Reached

**Problem:** Instructions file is too large.

**Solutions:**
1. Keep instructions under 200 lines
2. Use [ai-context](https://github.com/zeromicro/ai-context) for concise rules
3. Reference pattern files instead of inlining

## Additional Resources

- [GitHub Copilot Documentation](https://docs.github.com/en/copilot)
- [Copilot Chat](https://docs.github.com/en/copilot/using-github-copilot/asking-github-copilot-questions-in-your-ide)
- [go-zero Official Docs](https://go-zero.dev)
- [zero-skills Pattern Guides](../references/)
