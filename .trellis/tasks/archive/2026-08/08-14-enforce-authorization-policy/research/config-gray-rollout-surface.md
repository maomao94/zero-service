# Research: Config / Gray-rollout Surface

- **Query**: How services load config (go-zero conf, YAML), existing patterns for per-service feature flags / map flags, and where a "policy mode" (legacy/claims-only/none) + allowlist would naturally live
- **Scope**: internal
- **Date**: 2026-08-14

## Findings

### Config loading mechanism

- All services use the same go-zero pattern in `main`: `flag.String("f", "etc/<svc>.yaml", ...)` then `conf.MustLoad(*configFile, &c)` (YAML). Verified in: `zerorpc/zerorpc.go:30`, `socketapp/socketgtw/socketgtw.go:36`, `gtw/gtw.go:31`, `aiapp/aigtw/aigtw.go:39`, `aiapp/mcpserver/mcpserver.go:24`, `app/trigger/trigger.go:38`, `aiapp/aichat/aichat.go:28`, `aiapp/aisolo/aisolo.go:28`, and all other mains (21+ services).
- Config structs are per-service under `<svc>/internal/config/config.go`; `conf.MustLoad` maps YAML keys via go-zero `mapping` (field names = keys unless tagged).
- No central/remote config: services may register with Nacos for service discovery (`NacosConfig` in socketgtw/trigger configs, `common/nacosx`), but config itself is file-based `conf.MustLoad`.

### Existing feature-flag / policy-ish patterns (natural precedents)

| Pattern | Service | Config field | Location |
|---|---|---|---|
| Bool feature flag | socketgtw | `EnableStreamEventNotify bool` `json:",default=true"` | `socketapp/socketgtw/internal/config/config.go:27` |
| Bool feature flag | aigtw | `Knowledge.Enabled bool` | `aiapp/aigtw/internal/config/config.go:32` (embedded `einoxkb.Config`) |
| Bool feature flag | mcpserver | `Skills.AutoReload bool` | `aiapp/mcpserver/internal/config/config.go:17`; also `mcpx.McpServerConf.Skills` in `common/mcpx/server.go:14-17,22-28` |
| Bool + map | aisolo | `MCP.Enabled bool`, `MCP.ServiceToken string`; `DB.Enabled`; `Skills.Enabled/Strict` | `aiapp/aisolo/internal/config/config.go:150,187,194,201-207,227-231` |
| Map config | aigtw / mcpserver | `JwtAuth.ClaimMapping map[string]string` | `aiapp/aigtw/internal/config/config.go:26`; `common/mcpx/server.go:26` |
| Mode string | ieccaller | `DeployMode == "cluster"` (`IsBroadcast()`) | `app/ieccaller/internal/svc/servicecontext.go:359-361` (no config struct field shown; likely in config) |
| List config | socketgtw | `SocketMetaData []string` | `socketapp/socketgtw/internal/config/config.go:25`; `etc/socketgtw.yaml:29` |
| Slice of structs | aichat | `Providers []ProviderConfig`, `Models []ModelConfig` | `aiapp/aichat/internal/config/config.go:11-26,34-35` |

No service has an existing "authorization policy" config struct today.

### Candidate home for "policy mode" + allowlist

Per-service config structs (`<svc>/internal/config/config.go`) are the natural home, using the same `json:",optional"` + defaults conventions:

- **Sender services (stop raw-token propagation):**
  - `aiapp/aigtw/internal/config/config.go:22-33` — e.g. add `AuthPolicy struct { Mode string; AuthorizationSuppress bool; Allowlist map[string]bool }` at top level. aigtw is the main P1 sender (S1/S2).
  - `socketapp/socketgtw/internal/config/config.go:8-28` — P3 sender (S5) → StreamEvent.
  - `app/trigger/internal/config/config.go:11-27` — P7 sender for dynamic invokes (S6-S8).
  - `aiapp/mcpserver/internal/config/config.go:9-12` — P5/P6 `_meta` + nested client (S9).
- **Receiver services (duplicate/conflict policy):**
  - `facade/streamevent/internal/config/config.go` — R2 (raw-token receiver, L1).
  - `aiapp/aichat/internal/config/config.go:28-37` — R16.
  - `aiapp/aisolo/internal/config/config.go` — R20.
  - Shared chokepoint option: because `ExtractFromGrpcMD` (`common/grpcx/metadata.go:62-75`) and `LoggerInterceptor` (`common/grpcx/server_interceptor.go:12-31`) are shared across all 21 servers, a **package-level/global policy knobs in `common/grpcx`** could be configured per service, but that introduces global mutable state and cross-service coupling; per-service config passed into the interceptor (e.g. `grpcx.LoggerInterceptorWithPolicy(c.AuthPolicy)`) is a cleaner alternative — no such option exists today.
- **Shared common config type**: a new type in `common/` (e.g. `common/authpolicy` or under `common/grpcx`) could define `Mode` (`legacy`/`claims-only`/`none`) and `Allowlist map[string]bool`, embedded into each service's `Config` — consistent with existing pattern of shared config types (`einoxkb.Config`, `mcpx.Config`, `gormx.Config` embedded in service configs).

### YAML examples to mirror

- `aiapp/aigtw/etc/aigtw.yaml:12-18` shows the map-style `JwtAuth.ClaimMapping` block.
- `socketapp/socketgtw/etc/socketgtw.yaml:29-30` shows list flag + commented bool flag convention.
- `aiapp/aisolo/etc/aisolo.yaml` (not read) likely shows `Enabled`/`ServiceToken` blocks matching `MCPConfig`.

### Rollback / gray concerns (from audit §7.1)

- Receiver-first: dual-mode acceptance must be config-driven (e.g. mode flag on receivers) before senders switch.
- Config and code version independently; telemetry should record `policy version` (audit §7.3) — no existing telemetry records policy version today.
- Existing log-level config (`Log.Level` in every YAML) is the operational gate for L2/L3 exposure.

## Caveats / Not Found

- `ieccaller` `DeployMode` field location in config struct not verified (only usage at `servicecontext.go:359-361`).
- No remote/dynamic config reload mechanism exists; changing policy mode requires restart (config loaded once at `conf.MustLoad`).
- No method-level (per-RPC) policy registry exists anywhere in the repo; allowlist would be net-new.
