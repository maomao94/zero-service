# dtui Docker resource modules Implementation Plan

## Checklist

1. Load parent artifacts and apply the recorded second-confirmation safety decision.
2. Confirm host navigation exposes `/containers` and `/images` before expanding workflows.
3. Decide the smallest seam for testing Docker module logic without a daemon.
4. Extend container module key bindings and help for detail/stats/log/action flows.
5. Add container detail view using existing `InspectContainer` data.
6. Add container stats view using existing `StreamStats` or a bounded sample approach.
7. Extend image module with history, tag, and save/export workflows using existing Docker internals.
8. Add focused tests for state transitions, confirmations, and no-daemon error recovery.
9. Run focused validation and then broader `dtui` validation.

## Concurrency

Implement after host navigation. Do not run concurrently with config/compose/deploy workflow work because both touch shared host/module UX and Docker safety rules.

## Validation

```bash
go test ./cli/dtui/plugins/containers ./cli/dtui/plugins/images ./cli/dtui/internal/docker
go test ./cli/dtui/...
go build ./cli/dtui
git diff --check
```

## Risk / Rollback

- Risk: streaming stats/log commands can leak goroutines. Keep close behavior explicit and bounded in tests.
- Risk: image save can write large files. Require explicit path confirmation and test with fakes rather than real saves.
- Risk: destructive actions affect real Docker resources. Do not test real remove/prune against user daemon.
- Risk: accidental destructive action from key press. Require a second confirmation modal before executing remove/prune operations.
