# Update Templates

Copy-paste templates for each update type. Examples use Go, matching this
repo's go-zero backend stack.

---

## Mandatory Template for Infra/Cross-Layer Work

```markdown
## Scenario: <name>

### 1. Scope / Trigger
- Trigger: <why this requires code-spec depth>

### 2. Signatures
- Go func / gRPC method / SQL model signature(s)

### 3. Contracts
- Request fields (name, type, constraints) — proto/api definitions
- Response fields (name, type, constraints)
- Config keys in etc/*.yaml (required/optional)

### 4. Validation & Error Matrix
- <condition> -> <error>

### 5. Good/Base/Bad Cases
- Good: ...
- Base: ...
- Bad: ...

### 6. Tests Required
- Unit/Integration/E2E with assertion points

### 7. Wrong vs Correct
#### Wrong
...
#### Correct
...
```

---

## Adding a Design Decision

```markdown
### Design Decision: [Decision Name]

**Context**: What problem were we solving?

**Options Considered**:
1. Option A - brief description
2. Option B - brief description

**Decision**: We chose Option X because...

**Example**:
\`\`\`go
// How it's implemented, e.g. logic layer delegating to common/ package
func (l *SendLogic) Send(in *pb.SendReq) (*pb.SendResp, error) {
    if err := l.svcCtx.MqttClient.Publish(l.ctx, in.Topic, in.Payload); err != nil {
        return nil, errors.Wrapf(err, "publish topic: %s", in.Topic)
    }
    return &pb.SendResp{}, nil
}
\`\`\`

**Extensibility**: How to extend this in the future...
```

---

## Adding a Project Convention

```markdown
### Convention: [Convention Name]

**What**: Brief description of the convention.

**Why**: Why we do it this way in this project.

**Example**:
\`\`\`go
// How to follow this convention, e.g. constructor + ServiceContext wiring
func NewFooLogic(ctx context.Context, svcCtx *svc.ServiceContext) *FooLogic {
    return &FooLogic{
        ctx:    ctx,
        svcCtx: svcCtx,
        Logger: logx.WithContext(ctx),
    }
}
\`\`\`

**Related**: Links to related conventions or specs.
```

---

## Adding a New Pattern

```markdown
### Pattern Name

**Problem**: What problem does this solve?

**Solution**: Brief description of the approach.

**Example**:
\`\`\`go
// Good
code example

// Bad
code example
\`\`\`

**Why**: Explanation of why this works better.
```

---

## Adding a Forbidden Pattern

```markdown
### Don't: Pattern Name

**Problem**:
\`\`\`go
// Don't do this
bad code example
\`\`\`

**Why it's bad**: Explanation of the issue.

**Instead**:
\`\`\`go
// Do this instead
good code example
\`\`\`
```

---

## Adding a Common Mistake

```markdown
### Common Mistake: Description

**Symptom**: What goes wrong

**Cause**: Why this happens

**Fix**: How to correct it

**Prevention**: How to avoid it in the future
```

---

## Adding a Gotcha

```markdown
> **Warning**: Brief description of the non-obvious behavior.
>
> Details about when this happens and how to handle it.
```
