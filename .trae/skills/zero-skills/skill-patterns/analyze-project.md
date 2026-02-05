# Example: Deep Project Analysis with Subagent

This example demonstrates using `context: fork` with the Explore agent to perform thorough codebase analysis.

## Skill Configuration

```yaml
---
name: analyze-gozero-project
description: Analyze a go-zero project structure and identify issues
context: fork
agent: Explore
allowed-tools:
  - Read
  - Grep
  - Glob
---

Analyze the go-zero project in the current workspace:

1. **Project Structure Analysis**
   - Find all .api files: !`find . -name "*.api" -type f`
   - Find all .proto files: !`find . -name "*.proto" -type f`
   - Identify service entry points: !`find . -name "main.go" -type f`

2. **Architecture Validation**
   - Check if three-layer architecture is followed
   - Verify Handler → Logic → Model separation
   - Look for business logic in handlers (anti-pattern)

3. **Configuration Review**
   - Find all config files: !`find . -name "*-api.yaml" -o -name "*-rpc.yaml"`
   - Check for hardcoded values in source files
   - Verify proper use of conf.MustLoad

4. **Common Issues Detection**
   - Search for `fmt.Errorf` in handlers (should use httpx.Error)
   - Check if all handlers have corresponding logic files
   - Verify proper error handling patterns

5. **Generate Report**
   Provide a summary with:
   - Service count and types (API/RPC)
   - Architecture compliance score
   - Specific issues found with file references
   - Recommendations for improvements
```

## How Dynamic Context Works

The `!`command"` syntax executes shell commands **before** the skill content is sent to Claude. The output replaces the placeholder:

- `!`find . -name "*.api"`` → Lists all API definition files
- `!`find . -name "*.proto"`` → Lists all Proto files
- `!`find . -name "*-api.yaml"`` → Lists all config files

This preprocessing happens instantly, giving Claude real project data instead of generic instructions.

## Expected Behavior

1. Subagent runs in isolated context (no conversation history)
2. Shell commands execute and inject actual file paths
3. Explore agent uses read-only tools to analyze files
4. Results summarized and returned to main conversation

## Use Cases

- **Initial project audit**: Understand a new codebase structure
- **Code review automation**: Check for common anti-patterns
- **Architecture compliance**: Verify go-zero conventions
- **Onboarding**: Generate project documentation for new developers
