# dtui docs packaging quality Implementation Plan

## Checklist

1. Load parent artifacts and verify all implementation children are complete or explicitly deferred.
2. Wait until module wiring and production workflows are implemented.
3. Compare README module list and keys against `main.go` and each module's `Bindings()`.
4. Rewrite README sections that still describe the app as a test host.
5. Document Docker daemon requirements only for operations that actually touch Docker.
6. Document safety model for destructive/deploy operations: default-visible actions, second confirmation, deploy backup/history, and recovery/rollback guidance.
7. Verify or update `build.sh` output behavior.
8. Run final validation commands and record any environment limitation in the parent wrap-up.

## Concurrency

Run last and alone. Do not update docs in parallel with feature children, or the README will drift from final behavior.

## Validation

```bash
go test ./cli/uix/... ./cli/dtui/...
go build ./cli/dtui
go vet ./cli/uix/... ./cli/dtui/...
git diff --check
```

## Risk / Rollback

- Risk: documenting planned features before they ship. Only document final implemented behavior.
- Risk: build artifacts dirty the repo. Keep outputs in `cli/dtui/bin/` and verify git expectations before final handoff.
