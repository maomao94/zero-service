# Refine IEC104 server config

## Goal

Refine the IEC104 server construction API so the package keeps the upstream-style `Settings` for low-level IEC104 runtime configuration while exposing a go-zero friendly server config for service YAML.

## Requirements

- Keep `Settings` with `Host`, `Port`, `Cfg104`, `Params`, and `LogProvider` as the low-level server settings shape.
- Provide a go-zero config type that contains only YAML-loadable fields, including `LogEnable` with default true.
- `New(cfg Settings, handler CommandHandler)` remains the low-level constructor.
- Provide a higher-level `NewServer(config, options...)` constructor for go-zero service use.
- Options should reliably cover runtime-only settings: IEC104 config, ASDU params, custom log provider, and connection callbacks if appropriate.
- ASDU `Params` are protocol encoding settings and stay out of go-zero YAML; override them only through code options.
- Default server logging uses the project go-zero log provider.
- `app/iecagent` uses the go-zero friendly config constructor.

## Acceptance Criteria

- [x] `common/iec104/server` compiles with both `Settings` and go-zero `ServerConfig` defined.
- [x] go-zero config structs do not embed runtime-only pointer/interface fields.
- [x] IEC104 defaults are applied when optional settings are omitted.
- [x] `LogEnable` is available in go-zero config with default true.
- [x] `app/iecagent` config and startup path compile and use `NewServer`.
- [x] Related IEC104 and iecagent tests pass.

## Notes

- User suggested this shape; implementation may adjust naming/details to match Go and go-zero conventions.
