# Design

## Boundaries

- `common/crontask` owns recurrence iteration and Scheduler-level invalid-time filtering.
- Trigger owns `job_id` lookup, transport validation, response formatting, and Trigger error mapping.
- The preview path does not call Store mutation methods, Scheduler execution paths, or handlers.

## Common API

Add a `Scheduler.PreviewNextRuns(task, after, count)` read-only method. It parses `task.RRuleStr` as a complete RRULE Set, repeatedly selects the first candidate strictly after the cursor, applies the Scheduler's configured `InvalidTimeFilter`, and stops at `count` or exhaustion.

The filter may advance across multiple RRULE candidates. The preview method treats the returned non-zero time as the accepted candidate and advances the cursor to it. A zero result means no valid future occurrence remains.

## Trigger API

`PreviewCronJobSchedule(job_id, count)` loads the current `TaskConfig` through `CronJobStore.GetByID`, defaults zero count to 10, invokes `CronJobScheduler.PreviewNextRuns` from the current time, formats results in the existing Trigger date-time format, and returns persisted RRULE text plus its description.

## Compatibility

The change only adds an RPC and messages. No existing field numbers or behavior change. Generated files are produced from `app/trigger/trigger.proto` using `app/trigger/gen.sh`.
