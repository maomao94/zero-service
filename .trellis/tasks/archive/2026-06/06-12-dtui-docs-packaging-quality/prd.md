# dtui docs packaging quality

## Goal

Make `dtui` production delivery repeatable: documentation, build script, validation commands, module key references, safety notes, and quality gates must match the final shipped terminal app.

## Confirmed Facts

- `README.md` currently describes `dtui` as a lightweight `uix` test host and says Docker modules are legacy/not wired.
- `main.go` already wires Docker modules (`images`, `compose`, `deploy`) and should eventually wire more production modules.
- `build.sh` builds local plus darwin/linux amd64/arm64 binaries into `cli/dtui/bin/`.
- Existing spec requires validation with `go test ./cli/uix/... ./cli/dtui/...`, `go build`, `go vet`, and `git diff --check` after shell/module contract changes.

## Requirements

- This child runs last, after implementation children finish, and must not be used as the active implementation target before production behavior exists.
- README must describe the production app accurately after all child implementation tasks finish.
- README must include startup behavior, module list, command palette, module keys, Docker requirements, config path, safety model, and validation commands.
- Packaging/build scripts must be safe, reproducible, and documented.
- Quality gates must be executable by future agents/users without hidden local assumptions.
- Docs must not claim a module is legacy/unwired if it is shipped.

## Acceptance Criteria

- [ ] `cli/dtui/README.md` matches `cli/dtui/main.go` module registration.
- [ ] README includes a complete module/key reference for all shipped modules.
- [ ] README documents no-Docker startup and Docker-required operations separately.
- [ ] README documents that destructive/overwrite-style operations are available by default but require second confirmation, and deploy overwrite requires backup/history/recovery behavior.
- [ ] README documents config file path and package/build outputs.
- [ ] `build.sh` is verified or updated so build outputs are predictable and validation-friendly.
- [ ] Final validation commands are documented and executed before parent completion.

## Out Of Scope

- Publishing a GitHub release or tag.
- Installing binaries globally.
- Adding external package managers such as Homebrew unless requested later.

## Resolved Decisions

- Safety docs must state that destructive and overwrite-style operations are visible/available by default but require second confirmation. Deploy docs must describe backup, history, and recovery/rollback behavior.

## Notes

- This child should run after implementation children so docs reflect final behavior rather than planned behavior.
