# Example: Multi-Service Architecture Planning

This example uses the Plan agent with `context: fork` to design multi-service architectures.

## Skill Configuration

```yaml
---
name: plan-microservices
description: Plan a microservices architecture for a feature
argument-hint: [feature-description]
context: fork
agent: Plan
---

Plan a microservices architecture for: $ARGUMENTS

## Project Context
- Existing API services: !`find . -name "*-api.yaml" -type f | xargs grep "Name:" | cut -d: -f3 | sort -u`
- Existing RPC services: !`find . -name "*-rpc.yaml" -type f | xargs grep "Name:" | cut -d: -f3 | sort -u`
- Database models: !`find . -path "*/model/*.go" -type f | xargs grep "type.*Model interface" | wc -l`

## Planning Requirements

Create a comprehensive microservices plan including:

### 1. Service Decomposition
- Identify the core services needed
- Define service boundaries and responsibilities
- Specify API (REST) vs RPC (gRPC) for each service

### 2. Data Flow
- Show how services communicate
- Identify synchronous vs asynchronous patterns
- Define service dependencies

### 3. Data Storage
- Database per service or shared?
- Caching strategy (Redis)
- Data consistency approach

### 4. API Gateway Design
- External API endpoints (REST)
- Internal RPC communication
- Authentication and authorization flow

### 5. go-zero Implementation Plan
For each service, specify:
- `.api` or `.proto` file structure
- Required models and tables
- Configuration requirements
- Middleware needed
- Resilience patterns (circuit breaker, rate limiting)

### 6. Development Order
Suggest implementation sequence:
1. Core services first (user, auth)
2. Dependent services
3. Integration points
4. Testing strategy

## Output Format

Provide:
1. **Architecture Diagram** (text-based)
2. **Service Specifications** (one per service)
3. **File Structure** (complete project layout)
4. **Implementation Steps** (ordered list)
5. **Potential Issues** (risks and mitigation)

Ensure the design follows go-zero best practices and the three-layer architecture.
```

## Why Use Plan Agent?

The `agent: Plan` configuration gives Claude:
- **Planning-optimized prompts**: Better at architecture design
- **No execution tools**: Prevents accidental code generation
- **Isolated context**: Fresh perspective without conversation bias

## Usage Example

```
/plan-microservices e-commerce platform with user management, product catalog, shopping cart, and order processing
```

Output will include:
- Detailed service breakdown
- go-zero specific file structures
- Communication patterns
- Implementation roadmap

## When to Use

- **Starting new projects**: Before writing any code
- **Feature planning**: Adding complex features to existing systems
- **Architecture review**: Validating or improving current design
- **Team discussions**: Generate starting point for technical discussions

## Integration with Other Skills

1. **Planning phase** (this skill):
   - `/plan-microservices [feature]`
   - Review and refine the plan

2. **Implementation phase**:
   - Use `/generate-api-service` for each planned service
   - Reference pattern guides for implementation details

3. **Validation phase**:
   - Use `/analyze-gozero-project` to verify architecture compliance
   - Check against [best-practices/overview.md](../best-practices/overview.md)

## Advanced: Custom Agent

You can also create a custom agent for go-zero planning. Create `.claude/agents/gozero-architect/AGENT.md`:

```yaml
---
name: gozero-architect
description: Specialized agent for go-zero architecture design
skills:
  - zero-skills
---

You are an expert go-zero architect. Design microservices following:
- Three-layer architecture (Handler → Logic → Model)
- Service discovery with etcd
- API Gateway pattern for external access
- RPC for internal communication
- Redis for caching and pub/sub
- Circuit breakers and rate limiting

Always provide `.api` and `.proto` file examples.
```

Then use: `@gozero-architect Design a user management system`
