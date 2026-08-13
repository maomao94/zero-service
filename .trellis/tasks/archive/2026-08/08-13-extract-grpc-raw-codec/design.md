# Technical Design

## Boundaries

- `common/grpcx` owns project-wide gRPC transport helpers: raw-byte codec and context propagation interceptors.
- `common/authctx` owns authentication context values and claim normalization without importing a transport package.
- `common/grpcx` owns gRPC metadata keys and metadata encoding/decoding.
- `common/mcpx` owns MCP `_meta`, raw meta context storage, and trace extraction.
- Service packages remain responsible for interceptor registration order and zrpc client/server construction.
- No service proto, model, config, or generated package may be imported by `common/grpcx`.

## Package Shape

```text
common/grpcx/
  rawcodec.go
  rawcodec_test.go
  client_interceptor.go
  client_interceptor_test.go
  server_interceptor.go
  server_interceptor_test.go
  metadata.go
  metadata_test.go
common/authctx/
  context.go
  claims.go
  context_test.go
  claims_test.go
common/mcpx/
  context_meta.go
  context_meta_test.go
```

The four existing interceptor function names and signatures remain unchanged so call sites only change the import path/package qualifier.

## Data Flow

- Client unary/stream interceptors call package-local gRPC metadata injection before delegating to the supplied invoker/streamer.
- Server unary/stream interceptors call package-local gRPC metadata extraction before delegating to handlers.
- Server errors continue to be logged through `logx.WithContext(ctx)` and returned unchanged.
- Stream server context continues to be overridden through an internal `grpc.ServerStream` wrapper.
- JWT/HTTP boundaries use `authctx` claim/context helpers; MCP boundaries use `mcpx` adapters backed by `authctx`.

## Compatibility

- Source compatibility is intentionally broken for the repository-private old import paths; every repository caller migrates atomically.
- Runtime behavior, public function names, registration order, codec names, and error propagation remain unchanged.
- No compatibility forwarding package is retained because all consumers are in this repository and are migrated in the same change.
- Wire and request compatibility is strict: all context/claim keys, gRPC metadata keys, MCP `_meta` fields, `b64:` encoding and propagation order remain byte-for-byte equivalent.
- Existing raw Authorization propagation remains unchanged even though it warrants a separate security review.

## Risks And Rollback

- Main risk: missing one of the 35 old import paths. Search for `common/Interceptor` after migration and compile all direct callers.
- On case-insensitive filesystems, deleting the uppercase directory must be verified with Git status and repository search.
- Rollback is a direct import/file move reversal; no persisted data or external protocol changes are involved.
- Any required wire-key or request-behavior change blocks implementation and requires explicit user review.
