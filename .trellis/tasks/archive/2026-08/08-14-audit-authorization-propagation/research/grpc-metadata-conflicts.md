# Research: gRPC Metadata Duplicate and Conflict Audit

- **Query**: Audit duplicate/conflicting Authorization and identity metadata, including Set, first-value, empty-first, and b64 behavior
- **Scope**: internal
- **Date**: 2026-08-14

## Findings

### Current Contract

| Case | Current behavior | Evidence | Compatibility / attack implication |
|---|---|---|---|
| Existing outgoing Authorization plus context Authorization | `md.Copy()` then `md.Set("authorization", contextValue)` replaces all existing values | `common/grpcx/metadata.go:45-59`; test `metadata_test.go:58-70` | Context value wins at interceptor hop. Existing values survive only when context Authorization is absent/empty. |
| Duplicate incoming values | Receiver reads only `values[0]` | `metadata.go:62-74`; test `metadata_test.go:73-83` | Metadata ordering selects identity. No duplicate rejection or equality check. Proxy/client ordering differences can change selected principal. |
| Empty first + non-empty later | Entire field is skipped; later values ignored | `metadata.go:66`; test lines 73-83 | Can suppress propagated identity/Authorization while leaving a later visible value in raw metadata for another component using different semantics. |
| Conflicting Authorization vs `x-user-*` claims | Both are independently copied to context; no token parse or cross-check | `metadata.go:27-34,62-74` | A trusted/untrusted RPC caller can assert a token and unrelated identity metadata. Business code typically consumes claims directly, so identity can diverge from token. |
| Existing process context plus incoming metadata | Non-empty first metadata value overwrites by wrapping context; absent/empty leaves prior value visible | `metadata.go:63-72` | Middleware composition can retain stale outer identity when incoming metadata is empty/absent. |
| Empty/non-string outgoing context | Not emitted; existing outgoing metadata is not deleted because `Set` is not called | `metadata.go:49-58`; test lines 86-101 | A caller may expect empty context to clear inherited metadata, but inherited metadata remains. |
| Non-printable value | Sender encodes entire string as `b64:` + standard Base64; receiver decodes any value beginning `b64:` | `metadata.go:36-56,68-70` | Literal printable values beginning `b64:` are decoded even if not encoded. Invalid Base64 decode behavior depends on `cryptor.Base64StdDecode`; no error is surfaced. |
| Non-ASCII Authorization | Encoded using same `b64:` mechanism | Authorization is first `metadataFields` entry at lines 27-34 and generic encoding at lines 54-56 | Wire compatibility relies on every receiver using this helper. External gRPC consumers may see an opaque `b64:` credential. |

### Metadata Schema

Stable ordered pairs are:

```text
authorization -> authorization
user-id       -> x-user-id
user-name     -> x-user-name
dept-code     -> x-dept-code
auth-type     -> x-auth-type
```

Evidence: `common/grpcx/metadata.go:13-34`; contract test `metadata_test.go:13-23`.

### Verification Boundary

`LoggerInterceptor` and `StreamLoggerInterceptor` call `ExtractFromGrpcMD` then invoke handlers; they do not authenticate the RPC peer, parse Authorization, validate claims, or reject conflicts (`common/grpcx/server_interceptor.go:10-40`). Errors are logged with the enriched context but no metadata content is explicitly formatted there.

### Archived Contract

The extraction task explicitly kept metadata overwrite, first-value reads, empty filtering, field order, `b64:`, and raw Authorization propagation unchanged (`.trellis/tasks/archive/2026-08/08-13-extract-grpc-raw-codec/prd.md:65-80`). These are deliberate compatibility constraints for this audit, not inferred accidents.

### Policy Inputs, Not Decisions

Future enforcement must obtain approval for each conflict class: reject duplicates; accept only one non-empty value; require all duplicates equal; define whether Authorization-derived claims override metadata; and define behavior when no token is present. The current implementation provides no signal/metric to distinguish these cases because extraction discards later values.

## Caveats / Not Found

- No test specifically covers duplicate Authorization; the generic user-ID first-value test demonstrates the shared field-loop behavior.
- No repository code was found that reads raw incoming gRPC Authorization directly from `metadata.MD`, so divergent first/last-value readers inside this repository were not found.
- Proxy, mesh, and load-balancer duplicate-header behavior is not represented in repository config inspected by this audit.
