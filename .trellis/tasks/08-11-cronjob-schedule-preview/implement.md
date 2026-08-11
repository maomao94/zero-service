# Implementation Plan

- [ ] Add and test bounded future occurrence preview in `common/crontask`, including `EXDATE`, exhaustion, count, and `InvalidTimeFilter`.
- [ ] Add Trigger proto RPC/messages, regenerate code, and inspect generated diff.
- [ ] Implement Trigger Logic using persisted `RRuleStr` and `CronJobScheduler.PreviewNextRuns`.
- [ ] Add Trigger Logic tests for defaults, exclusions, disabled tasks, exhaustion, malformed rule, and missing job.
- [ ] Add concise API documentation for the preview endpoint.
- [ ] Run focused tests, generated-contract checks, formatting, and `git diff --check`.
