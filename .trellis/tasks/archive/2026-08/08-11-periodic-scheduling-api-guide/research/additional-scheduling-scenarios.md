# Research: Additional CronJob scheduling scenarios

- **Query**: Inspect the current `PlanRulePb`, scheduler compiler, tests, rrule usage, and CronJob API guide; identify additional practical scheduling scenarios that are directly and accurately expressible, identify unsupported or misleading scenarios caused by absent interval/count/setpos/holiday semantics, and prioritize additions without making the guide repetitive.
- **Scope**: mixed
- **Date**: 2026-08-11

## Findings

### Files Found

| File Path | Description |
|---|---|
| `app/trigger/trigger.proto:733` | Defines the complete public `PlanRulePb` surface and field validation. |
| `app/trigger/trigger.proto:1341` | Defines CronJob request boundaries, date formats, and `exclude_dates`. |
| `app/trigger/internal/cronjob/schedule.go:34` | Compiles the request into an Asia/Shanghai RRULE Set and computes the first run. |
| `app/trigger/internal/cronjob/schedule.go:89` | Directly maps `PlanRulePb` arrays to `rrule.ROption` `BY*` fields. |
| `app/trigger/internal/cronjob/schedule_test.go:15` | Verifies DAILY scheduling, date exclusion, timezone preservation, inclusive current occurrence, range defaults, and exhaustion. |
| `app/trigger/internal/logic/calcplantaskdatelogic_test.go:14` | Verifies compiled dates and descriptions use the same RRULE Set and EXDATE behavior. |
| `common/crontask/describe_test.go:704` | Provides occurrence-level evidence for date filters, intersections, and frequency semantics in the pinned rrule library. |
| `docs/trigger-rrule-api-guide.md` | Current field and boundary documentation. |
| `docs/trigger-rrule-api-guide.md` | Current scenarios: every minute, fixed 10-minute marks, hourly, fixed quarter months, first day monthly, M/W/F, bounded daily with one excluded date, and one-shot. |
| `go.mod:65` | Pins `github.com/teambition/rrule-go v1.8.2`. |

### Code Patterns

`ConvertToRRuleOption` is a direct mapping with no hidden recurrence dimensions:

```go
opts := rrule.ROption{
    Freq:     rrule.Frequency(planRule.Freq),
    Dtstart:  startTime,
    Until:    endTime,
    Bysecond: []int{0},
}
opts.Byhour = int32sToInts(planRule.Hours)
opts.Byminute = int32sToInts(planRule.Minutes)
opts.Bymonth = int32sToInts(planRule.Month)
opts.Bymonthday = int32sToInts(planRule.Day)
```

Source: `app/trigger/internal/cronjob/schedule.go:89-100`. Week values are translated individually from `1..7` to `MO..SU` at `schedule.go:100-121`. There is no assignment to `Interval`, `Count`, `Bysetpos`, ordinal weekdays, holiday calendars, `RDATE`, or `Wkst`.

The practical composition semantics are:

- `month`, `day`, and `week` are simultaneous filters. A candidate must satisfy every non-empty date filter. This is corroborated by the occurrence differential using `BYMONTH`, `BYMONTHDAY`, and `BYDAY` together in `common/crontask/describe_test.go:782-792`.
- Explicit `hours` and `minutes` produce every matching hour/minute combination at second zero. Both fields are required to contain at least one value by `trigger.proto:754-766`.
- Negative `day` values are passed directly as negative `BYMONTHDAY`; `-1` is the last calendar day, `-2` the penultimate calendar day, and so on (`trigger.proto:742-746`, `schedule.go:99`).
- `start_time` and `end_time` become inclusive `DTSTART` and `UNTIL`; the compiler rejects reverse ranges and ranges longer than three years (`schedule.go:126-152`).
- Each `exclude_dates` value is expanded to an `EXDATE` for every configured `hours × minutes` pair on that date (`schedule.go:55-65`). It is a static date exclusion, not a calendar policy.

### Prioritized Expressible Scenarios

Each fragment below is the exact `rule` object plus any range/exclusion fields needed to state the scenario accurately. All times are Asia/Shanghai and execute at second zero.

#### P1: Weekdays at one fixed time

Practical label: Monday through Friday at 09:00.

```json
{
  "rule": {
    "freq": 2,
    "month": [],
    "day": [],
    "week": [1, 2, 3, 4, 5],
    "hours": [9],
    "minutes": [0]
  },
  "exclude_dates": []
}
```

Semantic caveat: this means calendar weekdays, not working days. Statutory holidays and adjusted weekend workdays are not recognized. This is a high-value addition because it is common and makes the `week` mapping immediately legible, while being distinct from the existing M/W/F example.

#### P1: Last calendar day of every month

Practical label: month-end close at 23:30.

```json
{
  "rule": {
    "freq": 1,
    "month": [],
    "day": [-1],
    "week": [],
    "hours": [23],
    "minutes": [30]
  },
  "exclude_dates": []
}
```

Semantic caveat: this is the last **calendar** day, including weekends and holidays. It does not mean last working day. This should be added because negative `day` is documented but not demonstrated, and it handles variable month length without enumerating dates.

#### P1: Multiple fixed times each day

Practical label: daily at 09:00 and 17:30.

The API creates a Cartesian product, so one rule cannot express only `(09:00, 17:30)` by setting `hours: [9,17]` and `minutes: [0,30]`; that fragment would execute at 09:00, 09:30, 17:00, and 17:30. A directly accurate same-minute scenario is:

```json
{
  "rule": {
    "freq": 3,
    "month": [],
    "day": [],
    "week": [],
    "hours": [9, 17],
    "minutes": [30]
  },
  "exclude_dates": []
}
```

This means exactly 09:30 and 17:30 daily. Likewise `hours: [9,17], minutes: [0]` means exactly 09:00 and 17:00. For mismatched minute pairs such as 09:00 and 17:30 only, use two CronJobs. A short example or explicit note should be added because it reveals the array-composition rule and prevents a likely request-construction mistake.

#### P2: Fixed annual date

Practical label: every year on December 31 at 18:00.

```json
{
  "rule": {
    "freq": 0,
    "month": [12],
    "day": [31],
    "week": [],
    "hours": [18],
    "minutes": [0]
  },
  "exclude_dates": []
}
```

Semantic caveat: execution is bounded by the request's `start_time`/`end_time`, whose maximum span is three years. A February 29 version (`month: [2], day: [29]`) runs only in leap years within that range; it does not shift to February 28. This is a useful compact addition if the guide needs a clear YEARLY example simpler than the existing fixed-quarter-month set.

#### P2: Quarter-end on the last calendar day

Practical label: March, June, September, and December month-end at 23:00.

```json
{
  "rule": {
    "freq": 0,
    "month": [3, 6, 9, 12],
    "day": [-1],
    "week": [],
    "hours": [23],
    "minutes": [0]
  },
  "exclude_dates": []
}
```

Semantic caveat: fixed quarter-ending months are selected; this is not an `INTERVAL=3` recurrence anchored to an arbitrary start month. It means last calendar day, not last working day. This can replace or be a brief variant beside the existing quarter-start example; a second full request would be repetitive.

#### P2: Selected calendar days every month

Practical label: the 1st and 15th of every month at 10:00.

```json
{
  "rule": {
    "freq": 1,
    "month": [],
    "day": [1, 15],
    "week": [],
    "hours": [10],
    "minutes": [0]
  },
  "exclude_dates": []
}
```

Semantic caveat: each listed calendar date is independent. A day that does not exist in a month is skipped rather than moved; for example, `day: [31]` has no occurrence in April. This is directly expressible but close to the existing monthly-first-day example, so it is better as a concise variant than a full request.

#### P3: Weekend-only schedule

Practical label: Saturday and Sunday at 08:00.

```json
{
  "rule": {
    "freq": 2,
    "month": [],
    "day": [],
    "week": [6, 7],
    "hours": [8],
    "minutes": [0]
  },
  "exclude_dates": []
}
```

Semantic caveat: this is a straightforward variant of the existing M/W/F scenario. It is accurate but should not receive another full request example unless weekend scheduling is a target use case.

#### P3: Bounded campaign with several times and explicit blackout dates

```json
{
  "start_time": "2027-11-01 00:00:00",
  "end_time": "2027-11-30 23:59:59",
  "rule": {
    "freq": 3,
    "month": [],
    "day": [],
    "week": [],
    "hours": [9, 18],
    "minutes": [0]
  },
  "exclude_dates": ["2027-11-11", "2027-11-20"]
}
```

Semantic caveat: both 09:00 and 18:00 are excluded on each listed date because exclusions expand across all hour/minute combinations. This is directly expressible but overlaps the existing bounded-daily exclusion example; retain it as explanatory evidence, not another full guide example.

### Unsupported or Misleading Scenarios

| Requested scenario | Why it is not directly expressible by `PlanRulePb` | Misleading near-match to avoid |
|---|---|---|
| Every N periods from an arbitrary anchor, such as every 2 days, every 3 hours, or every 90 minutes | There is no `interval`; `ConvertToRRuleOption` never sets `ROption.Interval`. | Enumerated fixed wall-clock values can represent selected phases such as hours `[0,2,...,22]`, but that resets to those clock positions and is not a general rolling interval from `start_time`. |
| Stop after exactly N generated occurrences | There is no `count`; only inclusive `end_time`/`UNTIL` bounds exist. | Choosing an estimated end date is not equivalent when months vary, filters skip dates, or exclusions remove candidates. A one-shot is possible only by collapsing the range to one matching second, as already documented. |
| First Monday, second Tuesday, last Friday, first weekday, or last weekday of each month | `week` carries only plain weekday numbers; no ordinal weekday is representable, and there is no `BYSETPOS`. | `day: [1,2,3,4,5,6,7]` plus `week: [1]` does express the first Monday because the date/weekday intersection has exactly one Monday, but general "first weekday" with `week: [1,2,3,4,5]` produces every weekday among days 1-7, not only the first candidate. It also does not solve last business day. |
| Nth candidate from a set, such as the last weekday or second matching time each month | No `setpos`; the compiler never sets `ROption.Bysetpos`. | Multiple `day`, `week`, `hours`, and `minutes` values retain all matching combinations rather than selecting a position. |
| Chinese statutory working day, holiday skip, adjusted workday, or last business day | CronJob compilation never calls the holiday APIs or holiday package. `exclude_dates` is only caller-supplied static dates. | `week: [1,2,3,4,5]` means Monday-Friday, not legal workdays; it omits weekend make-up workdays and includes weekday public holidays unless manually excluded. |
| Automatically skip every holiday forever | Exclusions are concrete `yyyy-MM-dd` values stored with the task; there is no holiday rule or dynamic calendar lookup during recurrence calculation. | Precomputing and submitting known dates works only for that supplied date set and must fit the at-most-three-year rule range. |
| Move an occurrence falling on a holiday/weekend to the previous or next working day | `EXDATE` removes candidates only; there is no replacement-date or shift semantic and no public `RDATE` field. | Excluding the original date does not create a shifted occurrence. A separately managed task/date is required to add one. |
| Exactly 09:00 and 17:30 in one rule | `hours × minutes` is a Cartesian product. | `hours: [9,17], minutes: [0,30]` creates four daily times, not two. |
| Clamp day 29/30/31 to month end | Positive `BYMONTHDAY` values that do not exist are absent. | `day: [31]` skips short months; use `day: [-1]` only when the intended semantic really is last calendar day. |
| More than three years of recurrence in one submitted rule | `normalizeRange` rejects `end_time > start_time + 3 years`. | Omitting `end_time` defaults only to the end of the start year; it does not create an unbounded rule. |

### Recommended Guide Selection

To add breadth without repeating full request envelopes:

1. Add one full example for **last calendar day of every month**. It demonstrates the currently undocumented-in-examples negative `day` capability and variable month lengths.
2. Add one full or medium example for **Monday-Friday at a fixed time**, with an explicit “calendar weekdays, not statutory working days” caveat. It is the most common practical weekly filter.
3. Add a compact subsection for **multiple daily times and Cartesian products**, showing one accurate same-minute fragment and the inaccurate `hours: [9,17], minutes: [0,30]` interpretation. This is semantically important and does not need a full CronJob request.
4. Add **fixed annual date** as a compact fragment, optionally including the February 29 skip caveat. This rounds out the frequency examples without duplicating the quarter-month request.
5. Mention **quarter-end** and **1st-and-15th** as one-line variants of existing quarter/monthly examples, not additional full JSON requests.
6. Do not add separate full examples for weekend-only or another bounded exclusion campaign; their mechanics are already represented by the M/W/F and bounded-daily examples.

### External References

- [rrule-go package documentation](https://pkg.go.dev/github.com/teambition/rrule-go) — documentation for the pinned recurrence engine; its `ROption` surface includes `Interval`, `Count`, `Bysetpos`, and ordinal `Weekday` capabilities that the current protobuf does not expose.
- [rrule-go `ROption` source](https://github.com/teambition/rrule-go/blob/master/rrule.go) — confirms defaulting and `BYMONTHDAY`/`BYDAY` handling in the library family used by the project; the repository pins v1.8.2, so local compiler code and tests remain the primary evidence.
- [RFC 5545 recurrence rule](https://datatracker.ietf.org/doc/html/rfc5545#section-3.8.5.3) — normative background for RRULE filtering, `COUNT`, `INTERVAL`, and `BYSETPOS`; only the subset mapped by `ConvertToRRuleOption` is public through this API.

### Related Specs

- `.trellis/spec/backend/trigger-guidelines.md:136` — CronJob adaptation contract; line 166 records Asia/Shanghai, complete RRULE Set, fixed second zero, and one-occurrence catch-up semantics.
- `.trellis/spec/backend/crontask-guidelines.md:88` — complete RRULE Set and next-run contract.
- `.trellis/spec/backend/crontask-guidelines.md:146` — schedule descriptions must derive from the parsed persisted RRULE Set.
- `.trellis/spec/guides/cross-layer-thinking-guide.md:32` — Plan and CronJob may share pure rule compilation but are different runtime state machines.

## Caveats / Not Found

- Trigger-specific scheduler tests currently exercise DAILY rules and exclusions but do not contain direct month-end, annual fixed-date, or multi-time test cases. Their expressibility follows from the direct mapping plus rrule-go behavior; common crontask occurrence tests provide additional filter/intersection evidence.
- No CronJob holiday-calendar integration was found in `CompileSchedule`; holiday RPCs exist in the same protobuf service but are separate APIs.
- No `interval`, `count`, `setpos`, ordinal weekday, dynamic holiday, replacement-date, or caller-provided `RDATE` field exists in `PlanRulePb` or the CronJob request messages.
- The `.trellis/big-question/` search returned no scheduling limitation that changes these findings.
