# Optimize pending registry stats loop

## Goal

`common/antsx.PendingRegistry` should follow the go-zero `collection.Cache` stats-loop style: stats collection is owned by the registry internals and starts from construction when configured, so callers do not need to manually call `StartStatsLoop` for normal usage.

## Confirmed facts

- `common/antsx/pending.go` currently exposes `默认内置 statLoop(interval)` but `NewPendingRegistry` does not use `cfg.statsInterval`.
- `StartStatsLoop(ctx, interval, logFn)` currently requires callers to provide context, interval, and logger, then keep and call the returned stop function.
- go-zero `core/collection/cache.go` creates `cache.stats = newCacheStat(...)` inside `NewCache`; `newCacheStat` starts `go st.statLoop()` internally.
- Existing tests cover manual `StartStatsLoop`, no-activity suppression, stats counters, close accuracy, and concurrent stats updates.
- Backend specs require minimal scope, no broad refactor, `logx`-style logging for stats, and targeted tests for public component behavior.

## Requirements

1. `PendingRegistry` must provide a default internal stats-loop path through `默认内置 statLoop(interval)`.
2. Callers who use `NewPendingRegistry(..., 默认内置 statLoop(interval))` must not need to call `StartStatsLoop` or manage the returned stop function.
3. The internal stats loop must stop when `PendingRegistry.Close()` is called.
4. The stats loop should preserve existing semantics: only emit/log when interval activity exists, and include cumulative counters, interval deltas, pending size, and interval duration.
5. Existing manual `StartStatsLoop` behavior should remain available for tests or custom logging unless implementation evidence shows it conflicts with requirement 1.
6. Changes should be limited to `common/antsx` implementation, tests, and only necessary documentation/spec updates.

## Acceptance criteria

- `NewPendingRegistry[T](默认内置 statLoop(interval))` starts the stats loop automatically.
- No public API requires users to manually start the default stats loop.
- `Close()` stops the default stats loop and remains idempotent.
- Manual `StartStatsLoop` tests continue to pass or are replaced by equivalent coverage if the API is intentionally changed.
- `go test ./common/antsx` passes.
- `lsp_diagnostics` on changed Go files reports no new errors.

## Out of scope

- Reworking `Promise` APIs or `RequestReply` semantics.
- Changing TimingWheel TTL derivation.
- Adding external metrics backends.
- Broad docs rewrite outside necessary `PendingRegistry` examples.
