# Backend Development Guidelines

> Best practices for backend development in this project (go-zero microservices + eino AI framework).

---

## Overview

This project uses go-zero for microservices and eino for AI capabilities. All services follow the Handler → Logic → Model three-layer architecture. Code generation via `gen.sh` is mandatory.

---

## Guidelines Index

| Guide | Description | Status |
|-------|-------------|--------|
| [Directory Structure](./directory-structure.md) | go-zero service layout and module organization | To fill |
| [Database Guidelines](./database-guidelines.md) | ORM patterns, queries, migrations | To fill |
| [Error Handling](./error-handling.md) | Error types, handling strategies | To fill |
| [Quality Guidelines](./quality-guidelines.md) | Code standards, forbidden patterns | To fill |
| [Logging Guidelines](./logging-guidelines.md) | Structured logging, log levels | To fill |

Also read:
- [Coding Standards](../coding-standards.md) — Naming, style, Git conventions
- [go-zero Conventions](../go-zero-conventions.md) — Directory structure, code generation, service types

---

## Pre-Development Checklist

Before writing any backend code, verify:

- [ ] Read `../coding-standards.md` — naming conventions (API: xxxRequest/xxxResponse, gRPC: xxxReq/xxxRes)
- [ ] Read `../go-zero-conventions.md` — directory structure, gen.sh workflow
- [ ] Check `common/` for reusable components (mqttx, djisdk, antsx)
- [ ] Identify the target service's `.api` or `.proto` file
- [ ] Confirm the code generation workflow: modify `.api`/`.proto` → run `gen.sh` → write Logic
- [ ] Read the relevant guideline files listed above (if filled)

---

## Quality Check

After completing backend code, verify:

- [ ] `go build ./...` compiles without errors
- [ ] `go mod tidy` — no unused dependencies
- [ ] `go vet ./...` — no static analysis warnings
- [ ] Utility functions have unit tests (`*_test.go`)
- [ ] `.proto` / `.api` files have complete, consistent comments
- [ ] API→gRPC comment alignment: comments match between `.api` and `.proto`
- [ ] No Java-style patterns (unnecessary getters/setters, over-encapsulation)
- [ ] No skipped `gen.sh` — Handler/Types are generated, not hand-written
- [ ] Request/Response naming follows convention (API: xxxRequest/xxxResponse, gRPC: xxxReq/xxxRes)

---

## How to Fill These Guidelines

For each guideline file:

1. Document your project's **actual conventions** (not ideals)
2. Include **code examples** from your codebase
3. List **forbidden patterns** and why
4. Add **common mistakes** your team has made

---

**Language**: All documentation should be written in **English**.
