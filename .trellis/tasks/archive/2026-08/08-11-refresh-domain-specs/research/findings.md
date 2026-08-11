# Domain Specs Refresh Findings

**Date**: 2026-08-11

---

## File: trigger-guidelines.md

**Status**: NEEDS UPDATE

**Issues**:

1. **Missing CronExecLog model reference**: `CronExecLog` exists in the codebase (`app/trigger/model/gormmodel/cron_exec_log.go`, `app/trigger/internal/cronjob/handler.go:84`, `app/trigger/internal/svc/servicecontext.go:76`) but is **not mentioned anywhere** in trigger-guidelines.md. The `NewLoggingEventHandler` function in `handler.go:69-102` wraps event handlers and writes `CronExecLog` records after each cron job execution (with fields: job_id, task_code, task_name, scheduled_time, start_time, end_time, cost_ms, status, error_message). The spec's "CronJob 适配" section should mention:
   - The `CronExecLog` model and its purpose (per-execution audit logging)
   - The `NewLoggingEventHandler` wrapper pattern
   - That execution log write failures are silently ignored and don't affect handler return values

2. **All file paths verified exist**:
   - `app/trigger/trigger.proto` ✓
   - `docs/trigger.md` ✓
   - `app/trigger/gen.sh` ✓
   - `app/trigger/internal/logic` (directory, 64 entries) ✓
   - `app/trigger/internal/cronjob` (directory, 7 entries) ✓
   - `app/trigger/cron/cronservice.go` ✓
   - `app/trigger/model/gormmodel/plan.go` ✓
   - `app/trigger/model/gormmodel/plan_test.go` ✓
   - `app/trigger/internal/logic/terminateplanlogic.go` ✓
   - `app/trigger/internal/logic/terminateplanbatchlogic.go` ✓
   - `app/trigger/internal/logic/callbackplanexecitemlogic.go` ✓
   - `common/crontask` (directory, 9 entries) ✓
   - `app/trigger/internal/planscope` (package) ✓
   - `./crontask-guidelines.md` (`.trellis/spec/backend/crontask-guidelines.md`) ✓

3. **Code patterns still match**:
   - `set.All()` in `calcplantaskdatelogic.go:48` ✓
   - `crontask.DescribeRRule(schedule.RRuleStr)` in `calcplantaskdatelogic.go:53` ✓
   - `cronjob.CompileSchedule(in.Rule, ...)` in `calcplantaskdatelogic.go:39` ✓
   - `CalcPlanTaskDateRes` proto fields: `1=planDates, 2=scheduleDescription, 3=rruleStr` ✓
   - `planscope.Scope.LogMessage` with `[cron-plan]` prefix via `EntryCron` ✓
   - `planscope.CronPlanLogMessage` for lifecycle logs ✓
   - Plan/Batch termination pattern (CAS check, running count) consistent ✓

**Summary**: One actionable gap — CronExecLog model and NewLoggingEventHandler pattern need to be added to the "CronJob 适配" section. All paths and code patterns are current.

---

## File: isp-guidelines.md

**Status**: OK

**Verification results**:

- `common/isp/message.go` ✓
- `common/isp/serializer.go` ✓
- `common/isp/client.go` ✓
- `common/isp/client_test.go` ✓
- `common/isp/errors.go` ✓
- `common/isp/*_test.go` (4 test files found: client, model_writer, serializer, errors) ✓
- `app/ispagent/internal/handler` (directory, 8 entries) ✓
- `app/ispagent/internal/crontask` (directory, 8 entries) ✓
- `app/ispagent/internal/svc/servicecontext.go` ✓
- `./crontask-guidelines.md` ✓

**Summary**: No issues. All referenced paths exist.

---

## File: iec104-guidelines.md

**Status**: OK

**Verification results**:

- `app/ieccaller/ieccaller.proto` ✓
- `common/iec104/client/interface.go` ✓
- `common/iec104/client/command_reply.go` ✓
- `common/iec104/client/clientmanager_test.go` ✓
- `common/iec104/trace.go` ✓
- `app/ieccaller/internal/iec` (directory, 2 entries: clienthandler.go, clienthandler_test.go) ✓

**Summary**: No issues. All referenced paths exist.

---

## File: dji-guidelines.md

**Status**: OK

**Verification results**:

- `common/djisdk/topic.go` ✓
- `common/djisdk/method.go` ✓
- `common/djisdk/protocol.go` ✓
- `common/djisdk/protocol_drc.go` ✓
- `common/djisdk/client.go` ✓ (verified line ranges: 37-49 MustNewClient, 80-117 SendCommand, 1042-1078 EnableDrc)
- `common/djisdk/option.go` ✓ (verified line ranges: 21-217, file ends exactly at 217)
- `common/djisdk/handler.go` ✓ (verified line ranges: 43-78 replyRouters, 90-567 HandleEvents/tryDispatchEventNotify, file ends at 567)
- `common/djisdk/device_type.go` ✓
- `common/djisdk/hms.go` ✓
- `common/djisdk/hms.json` ✓
- `common/djisdk/drc.go` ✓ (verified line ranges: 73-460, file ends at 460)
- `common/djisdk/drc_test.go` ✓
- `app/djicloud/internal/hooks` (directory, 11 entries) ✓
- `app/djicloud/internal/hooks/sys_status_up.go` ✓
- `app/djicloud/internal/hooks/event_notify_up.go` ✓
- `app/djicloud/internal/logic/helper.go` ✓
- `app/djicloud/model/gormmodel` (directory, 6 entries) ✓
- `app/djicloud/model/gormmodel/dji_device.go` ✓
- `app/djicloud/djicloud.proto` ✓
- `./gis-guidelines.md` ✓

All handler option names verified in `option.go`:
- `WithFlightTaskProgressHandler` (line 81) ✓
- `WithFlightTaskReadyHandler` (line 87) ✓
- `WithHmsEventNotifyHandler` (line 111) ✓
- `WithUpdateTopoHandler` (line 141) ✓
- `WithOsdHandler` (line 147) ✓
- `WithStateHandler` (line 153) ✓
- `WithStatusHandler` (line 159) ✓
- `WithRequestHandler` (line 165) ✓
- `WithDrcUpHandler` (line 171) ✓
- `WithOnlineChecker` (line 177) ✓

**Summary**: No issues. All paths, line ranges, and handler option names are accurate.

---

## File: gis-guidelines.md

**Status**: OK

**Verification results**:

- `common/gisx/doc.go` ✓
- `app/gis/internal/logic/helper.go` ✓
- `common/gisx/geos` (directory, 17 entries) ✓
- `common/gisx/geos/orbconv` (directory, 2 entries) ✓
- `common/gisx/store.go` ✓
- `app/gis/model/fencestore.go` ✓

**Summary**: No issues. All referenced paths exist.

---

## File: realtime-guidelines.md

**Status**: OK

**Verification results**:

- `common/socketiox/server.go` ✓
- `common/socketiox/container.go` ✓
- `socketapp/socketgtw` (directory) ✓
- `socketapp/socketgtw/internal` (directory, 6 subdirectories) ✓
- `socketapp/socketpush/internal/logic/broadcastroomlogic.go` ✓
- `socketapp/socketpush/internal/logic/broadcastgloballogic.go` ✓
- `facade/streamevent/streamevent.proto` ✓
- `app/bridgekafka/internal` (directory, 5 subdirectories) ✓
- `common/mqttx` (directory, 10 entries) ✓

**Summary**: No issues. All referenced paths exist.

---

## File: ai-guidelines.md

**Status**: OK

**Verification results**:

- `common/einox/runtime/runner.go` ✓
- `common/einox/tool/kit.go` ✓
- `aiapp/aisolo/internal/turn/executor.go` ✓
- `aiapp/aisolo/internal/turn` (directory, 2 entries) ✓
- `common/mcpx/client.go` ✓
- `common/mcpx/server.go` ✓
- `aiapp/mcpserver` (directory, 4 entries) ✓

**Summary**: No issues. All referenced paths exist.

---

## Final Summary

| Spec File | Status | Action Needed |
|---|---|---|
| trigger-guidelines.md | NEEDS UPDATE | Add CronExecLog model + NewLoggingEventHandler to CronJob 适配 section |
| isp-guidelines.md | OK | None |
| iec104-guidelines.md | OK | None |
| dji-guidelines.md | OK | None |
| gis-guidelines.md | OK | None |
| realtime-guidelines.md | OK | None |
| ai-guidelines.md | OK | None |

**Only actionable finding**: `trigger-guidelines.md` is missing coverage of the `CronExecLog` model (defined in `app/trigger/model/gormmodel/cron_exec_log.go`, used in `app/trigger/internal/cronjob/handler.go:69-102` via `NewLoggingEventHandler`, and auto-migrated in `app/trigger/internal/svc/servicecontext.go:76`). The "CronJob 适配" section should document:
- The `CronExecLog` GORM model and table (`cron_exec_log`)
- The `NewLoggingEventHandler` wrapper that records per-execution logs (job_id, task_code, task_name, scheduled_time, start_time, end_time, cost_ms, status, error_message)
- The design decision that log write failures are silently skipped (do not affect handler return)
