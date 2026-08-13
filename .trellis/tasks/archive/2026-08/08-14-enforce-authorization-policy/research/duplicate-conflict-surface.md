# Research: Duplicate / Conflict Current Behavior and Observability Surface

- **Query**: Exact current code in `metadata.go` `ExtractFromGrpcMD` — `values[0]` pick, empty-first skip, b64 decode, Authorization vs x-user-* independence; what info is currently DISCARDED and would need to be surfaced for content-free observability
- **Scope**: internal
- **Date**: 2026-08-14

## Findings

### Exact current behavior (`common/grpcx/metadata.go:62-75`)

```go
func ExtractFromGrpcMD(ctx context.Context) context.Context {
	md, _ := metadata.FromIncomingContext(ctx)
	for _, f := range metadataFields {
		if values := md.Get(f.grpcKey); len(values) > 0 && values[0] != "" {
			val := values[0]
			if strings.HasPrefix(val, base64Prefix) {
				val = cryptor.Base64StdDecode(val[len(base64Prefix):])
			}
			ctx = authctx.WithKey(ctx, f.contextKey, val)
		}
	}
	return ctx
}
```

Behavior per field (`metadataFields` at `metadata.go:27-34`: `authorization`, `x-user-id`, `x-user-name`, `x-dept-code`, `x-auth-type`):

| Case | Current behavior | Evidence |
|---|---|---|
| Multiple incoming values for one key | Only `values[0]` read; **all other values discarded** | `metadata.go:66`; test `metadata_test.go:73-80` |
| `values[0]` empty, later value non-empty | Whole field skipped; **later non-empty value discarded** | `metadata.go:66`; test `metadata_test.go:73-83` |
| Value starts with `b64:` | Base64-decoded; decode errors not surfaced (value used as-is or empty) | `metadata.go:68-70`; `metadata_test.go:44-47,98-101` |
| Duplicate values equal | First value used; no equality check; **duplicate count not observable** | — |
| Duplicate values conflict | First value used; no comparison; **conflict not observable** | — |
| Authorization vs `x-user-*` present together | Copied independently into ctx; no token parse, no cross-check, **mismatch not observable** | `metadata.go:27-34,62-74` |
| Incoming key absent | Existing outer ctx value retained (no overwrite) | `metadata.go:63-72` |
| Existing process ctx + incoming non-empty | Incoming first value overwrites by wrapping | `metadata.go:63-72` |

### What is currently DISCARDED (observability gap)

1. **Duplicate count** — if a proxy/mesh/client sends `authorization: a` and `authorization: b`, only `a` survives; the fact that a duplicate existed is lost. Audit §4.2 item 3 explicitly requires a signal to distinguish "no duplicates" vs "duplicates equal" vs "duplicates conflict".
2. **Empty-first with non-empty later** — field is silently skipped; a later visible value exists in raw MD but is ignored (audit §4.1: "can suppress propagated identity while leaving a later value in raw metadata").
3. **Value count per key beyond 1** — no `len(values)` inspection.
4. **Conflict between Authorization and derived claims** — no token-claim reconciliation; mismatch between `authorization` and `x-user-*` is invisible.
5. **b64 decode failures** — `cryptor.Base64StdDecode` result used without error check; malformed `b64:` input silently becomes empty/garbage.
6. **Non-printable / ASCII status** — whether value was transported via `b64:` wrapper is not recorded.

### Contract tests locking current behavior

- `common/grpcx/metadata_test.go:58-84` `TestMetadataOverwriteAndFirstValueContract` — locks: outgoing `Set` overwrite of existing key; other keys preserved; source MD not mutated; **first value wins on extraction**; **empty-first skips field**.
- `common/grpcx/metadata_test.go:33-56` `TestMetadataRoundTripAllFields` — locks field order and b64 round-trip.
- `common/grpcx/metadata_test.go:15-31` `TestMetadataFieldContract` — locks wire key names/order (must not change per must-not-combine boundary).
- `common/grpcx/metadata_test.go:86-102` `TestMetadataFilteringAndPrintableContract` — locks empty/non-string suppression and b64 encoding of control chars.

Any change to extraction semantics (reject duplicates, require equality, empty-first policy) will require updating these tests deliberately — they currently assert the legacy discard behavior.

### What content-free observability can add without changing behavior

At the single chokepoint `ExtractFromGrpcMD` (called by unary `LoggerInterceptor` and stream `StreamLoggerInterceptor`, and via tests), a report-only path can compute and emit **without token content**:

- per key: `len(values)` (duplicate count ≥1), `emptyFirst bool`, `nonEmptyLater bool`, `hasB64Prefix bool`, `firstNonEmptyIndex int`
- cross-field: `authPresent bool`, `claimsPresent bool` (any of x-user-id/x-user-name/x-dept-code), `claimsBitset` (which identity keys present)
- conflict signals: `duplicateCount>1`, `duplicatesEqual bool` (compare without logging values), `emptyFirst bool`, `authVsClaimsMismatch bool` (present without parse — boolean only)
- call context: `info.FullMethod` (available in `LoggerInterceptor` `grpc.UnaryServerInfo`), caller service name if available

All of these are booleans/counts — no token prefix/suffix/length/hash, no `_meta`, no headers (matches audit §7.2 constraints).

### Note on MCP-side extraction

`ExtractFromMeta` (`common/mcpx/context_meta.go:26-37`) has the same first-wins discard for `_meta` keys: `ClaimString(meta, key)` reads a single map value per key. `_meta` is a JSON object keyed by string, so duplicates cannot exist at that layer (JSON forbids duplicate keys in the SDK-parsed map), but **conflict between `authorization` and claim keys inside `_meta`** is equally unobservable today. The wrapper stores the raw map via `WithMeta` (`wrapper.go:241-245`), so a report-only inspection of the map is possible at `CallToolWrapper`.

## Caveats / Not Found

- No existing metric/telemetry instrumentation in `grpcx` or `mcpx` extraction paths; observability would be net-new.
- `cryptor.Base64StdDecode` error behavior not inspected at source; assumes permissiveness based on no error check in call site.
- Duplicate Authorization behavior has no dedicated test today (only the shared field-loop first-value test).
