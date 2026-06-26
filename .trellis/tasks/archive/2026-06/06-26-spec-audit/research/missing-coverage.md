# Research: Missing Coverage in Spec Files

- **Query**: Important packages or patterns not documented in specs
- **Scope**: internal (codebase)
- **Date**: 2026-06-26

## Findings

### 1. antsx: Reactor, Emitter, Stream, Tee, Unbounded, Select not documented

**What's missing**: The `common/antsx/` package has 19 .go files (including tests) but only 3 specs cover it:
- `antsx-invoke-guidelines.md` — covers Invoke/InvokeAllSettled
- `antsx-promise-guidelines.md` — covers Promise/All/AllSettled/Race/Any
- `antsx-replypool-guidelines.md` — covers ReplyPool

**Not documented**:
- `reactor.go` — Reactor (协程池), `Submit`, `Post`, `Go`, `NewReactor`, `WithReactor` options
- `emitter.go` — Emitter (事件发射器)
- `stream.go` — Stream (流式处理)
- `tee.go` — Tee (流分叉)
- `unbounded.go` — Unbounded channel
- `select.go` — Select utilities

**Recommendation**: Add spec coverage for at least Reactor and Stream, as they are used in production code. The others may be internal/experimental.

### 2. common/configx/ not documented

**What's missing**: `common/configx/` package (kqConfig.go, mockconfig.go) is referenced in `messaging-guidelines.md` but has no dedicated spec. It contains shared Kafka config types (`KafkaPushConf`, `KafkaMultiPushConf`, `KafkaConsumerConf`) used across multiple services.

**Recommendation**: Either add a brief `configx-guidelines.md` spec or add config type documentation to an existing spec.

### 3. common/carbonx/ not documented

**What's missing**: `common/carbonx/carbonx.go` sets global carbon defaults (Shanghai timezone, zh-CN locale). Referenced in `database-guidelines.md:112-113` but has no spec of its own.

**Recommendation**: Add a brief section to `database-guidelines.md` or create a small `carbonx-guidelines.md` documenting the global defaults.

### 4. common/tool/ not documented as a package

**What's missing**: `common/tool/` contains `errorutil.go` (`NewErrorByPbCode`, `NewErrorByPbCodeWrap`, `IsErrorByPbCode`), `SimpleUUID`, `DecimalBytes` and others. These are referenced across specs but there's no central documentation of what `common/tool/` provides.

**Recommendation**: Add a `tool-guidelines.md` spec documenting the package's public API.

### 5. common/trace/ not documented

**What's missing**: `common/trace/` is imported in `common/mqttx/client.go` and `common/djisdk/` but has no spec. It likely provides OpenTelemetry trace utilities.

**Recommendation**: Document `common/trace/` if it has public API; otherwise mark as internal.

### 6. common/dbx/ referenced but no spec

**What's missing**: Mentioned in `go-zero-conventions.md:86` as "数据库扩展和多库支持" but has no dedicated spec. If this package exists and is used, it needs documentation.

### 7. common/asynqx/ referenced but no spec

**What's missing**: Mentioned in `go-zero-conventions.md:87` as "asynq 任务队列扩展" but has no dedicated spec.

### 8. common/ssex/ referenced but no spec

**What's missing**: Mentioned in `go-zero-conventions.md:85` as "SSE Writer 和流式响应工具" but has no dedicated spec.

## Caveats / Not Found

- Some of these packages may be thin wrappers or have README files at the package level that serve as documentation. Check `common/*/README.md` for existing docs before creating spec files.
- `common/einox/`, `common/mcpx/`, `common/dockerx/` are also mentioned in `go-zero-conventions.md:82-88` without dedicated specs, but they may be complex enough to warrant their own specs.
