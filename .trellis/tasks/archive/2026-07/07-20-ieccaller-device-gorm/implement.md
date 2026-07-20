# Implementation Plan

## Checklist

- [ ] Add `app/ieccaller/model/gormmodel` GORM struct for `device_point_mapping`.
- [ ] Add service-local point mapping store/cache methods in `app/ieccaller/model/gormmodel` with the methods required by existing logic and `PushASDU`.
- [ ] Update `internal/config.Config.DB` to `gormx.Config` and remove `DisableStmtLog` if it is no longer needed by `ieccaller`.
- [ ] Update `internal/svc.ServiceContext` to open `gormx.DB`, optionally auto-migrate in dev/test, and inject the GORM store.
- [ ] Update query logic, page-list logic, cache-clear logic, MQTT broadcast, and `PushASDU` to use the new store/model types.
- [ ] Remove now-unused imports of root `zero-service/model`, `common/dbx`, and `sqlx` from `ieccaller` runtime files.
- [ ] Verify there is no old-model fallback or compatibility branch left in `ieccaller`.
- [ ] Update `app/ieccaller/etc/ieccaller.yaml` DB config example to match `gormx.Config` shape if needed.
- [ ] Run `gofmt` on changed Go files.
- [ ] Run focused verification commands.

## Validation Commands

- `go test ./app/ieccaller/...`
- `go test ./app/ispagent/...` if shared gormx behavior or config shape changes unexpectedly affect the reference service.
- `go build ./app/ieccaller/...`

## Risky Files

- `app/ieccaller/internal/svc/servicecontext.go`: central dependency initialization and ASDU push enrichment.
- `app/ieccaller/internal/logic/pagelistpointmappinglogic.go`: pagination and total semantics.
- `app/ieccaller/mqtt/broadcast.go`: cluster cache invalidation path.
- `app/ieccaller/internal/config/config.go`: config compatibility and go-zero tag defaults.

## Review Gates Before Start

- Confirm no `.proto` change is required.
- Confirm no data migration SQL is required for existing `device_point_mapping` schema.
- Confirm implementation can be confined to `app/ieccaller` and does not require changes to root generated model code.
