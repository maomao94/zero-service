# 后端开发规范

> zero-service 的后端开发规范，覆盖 go-zero 微服务、gRPC/API 契约、Eino AI 能力、数据库、日志和质量门禁。

---

## 总览

本项目使用 go-zero 构建微服务，使用 Eino 承载 AI Agent、知识库、工具调用和流式会话能力。后端代码遵循 Handler/Server → Logic → Model/SDK 的分层，接口契约变更必须经过 `gen.sh` 生成流程。

---

## 规范索引

| Guide | Description | Status |
|-------|-------------|--------|
| [Directory Structure](./directory-structure.md) | go-zero 服务布局、模块边界和复用位置 | Active |
| [Database Guidelines](./database-guidelines.md) | Model、SQL、事务、缓存和持久化约定 | Active |
| [Error Handling](./error-handling.md) | HTTP/RPC 错误返回、错误码和传播策略 | Active |
| [Quality Guidelines](./quality-guidelines.md) | 禁止模式、测试要求和交付检查 | Active |
| [Logging Guidelines](./logging-guidelines.md) | logx 使用、日志内容和敏感信息边界 | Active |
| [Trellis Template Policy](./trellis-template-policy.md) | Trellis 模板侧跟随最新版、用户数据区定位、更新后验证流程 | Active |

同时阅读：
- [Coding Standards](../coding-standards.md) — 命名、编码、AI 协作和 Git 约定
- [go-zero Conventions](../go-zero-conventions.md) — 服务结构、代码生成、ServiceContext 和公共组件约定
- [错误码规范](../../../code.md) — google.rpc.Code 与 HTTP/gRPC 错误码映射

---

## 开发前检查

写后端代码前，按任务范围确认：

- [ ] Read `../coding-standards.md` — naming conventions (API: xxxRequest/xxxResponse, gRPC: xxxReq/xxxRes)
- [ ] Read `../go-zero-conventions.md` — directory structure, gen.sh workflow
- [ ] Check `common/` for reusable components (mqttx, djisdk, einox, mcpx, ssex, dbx, asynqx, dockerx)
- [ ] Identify the target service's `.api` or `.proto` file
- [ ] Confirm the code generation workflow: modify `.api`/`.proto` → run `gen.sh` → implement Logic
- [ ] Read only the relevant guideline files listed above; do not load every spec by default
- [ ] Confirm whether database, Redis, queue, MQTT, OSS, Docker, Eino, DJI SDK or external API boundaries are involved

---

## 质量检查

完成后端改动后，按实际影响范围验证：

- [ ] `go build ./...` compiles without errors
- [ ] `go mod tidy` only when dependencies changed
- [ ] `go vet ./...` — no static analysis warnings
- [ ] Targeted `go test` for changed packages or related modules
- [ ] Utility functions have unit tests (`*_test.go`)
- [ ] `.proto` / `.api` files have complete, consistent comments
- [ ] API→gRPC comment alignment: comments match between `.api` and `.proto`
- [ ] No Java-style patterns (unnecessary getters/setters, over-encapsulation)
- [ ] No skipped `gen.sh` — Handler/Types are generated, not hand-written
- [ ] Request/Response naming follows convention (API: xxxRequest/xxxResponse, gRPC: xxxReq/xxxRes)
- [ ] Generated code diff has been inspected after `gen.sh`
- [ ] Secrets, local paths and internal infrastructure details are not logged or committed

---

## 使用原则

1. Index first：先读本索引，再读当前任务相关的具体规范。
2. Task scoped：只加载当前任务需要的文件，避免上下文膨胀。
3. Existing pattern first：先读相邻 Handler、Logic、svc、model、config、types 和 `common/` 封装。
4. Generated code boundary：契约源文件和生成流程优先，非必要不手写生成物。
5. Spec update：发现稳定规则或踩坑经验时，沉淀到 `.trellis/spec/**`。
