# Implementation Plan

1. Add client and server interceptor files under `common/grpcx`, preserving existing signatures and behavior.
2. Add focused tests for unary and stream metadata/context propagation, handler delegation, response/error passthrough, and stream context replacement.
3. Replace all `common/Interceptor/rpcclient` and `common/Interceptor/rpcserver` imports with `common/grpcx`; preserve registration order.
4. Delete the two old interceptor source files and empty `common/Interceptor` hierarchy.
5. Run `gofmt` only on changed Go files.
6. Verify no old imports or private Raw Codec types remain.
7. Run `go test ./common/grpcx`, then tests/builds for all direct caller packages; run `go vet` over affected packages and `git diff --check`.
8. Review the final diff to ensure unrelated datetime and module-file changes were not modified.
9. Add `common/authctx` by moving identity keys/getters and claims helpers without changing key values or conversion behavior.
10. Move gRPC metadata fields and conversion from `ctxprop/grpc.go` into `common/grpcx`; update interceptors to use package-local helpers.
11. Move MCP context collection, extraction, raw `_meta` storage and trace extraction into `common/mcpx`.
12. Migrate all `ctxdata` and `ctxprop` imports/calls atomically, using Go-style API names where safe while preserving string contracts.
13. Delete `common/ctxdata` and `common/ctxprop`; search all production/test Go files for old imports and symbols.
14. Add contract tests that assert every context/claim/gRPC metadata key literal, field order, Unicode `b64:` behavior, metadata overwrite/first-value behavior, claims mapping direction, MCP `_meta` shape, and existing Authorization propagation.
15. Run target package race tests and compile/test all 74 direct import files' packages; run affected `go vet` and `git diff --check`.
16. 修正 AIGTW/MCP ClaimMapping 注释方向；将 grpcx metadata schema 收窄为实际使用字段并补充不改变 wire 行为的契约测试。

## Rollback Points

- Before deleting old files, repository search must show all callers migrated.
- If direct-caller compilation reveals package cycles, restore interceptor ownership and keep only Raw Codec in `grpcx`; do not add forwarding wrappers without explicit review.
- If preserving request behavior requires changing a wire key or transport policy, stop and request user approval instead of implementing the behavior change.
