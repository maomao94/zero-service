# Research: holiday-data-audit-2023-2026

- **Query**: Audit `common/holiday/data/*.json` against authoritative Chinese State Council General Office holiday arrangements for 2023, 2024, 2025, and 2026; verify configured dates, holiday/workday type, festival-day flag, name/group, and note.
- **Scope**: mixed
- **Date**: 2026-07-21

## Findings

### Files Found

| File Path | Description |
|---|---|
| `common/holiday/data/2023.json` | 2023 holiday calendar configuration |
| `common/holiday/data/2024.json` | 2024 holiday calendar configuration |
| `common/holiday/data/2025.json` | 2025 holiday calendar configuration |
| `common/holiday/data/2026.json` | 2026 holiday calendar configuration |

### Code Patterns

The JSON files use a per-date object keyed by `YYYY-MM-DD`, with fields `name`, `type`, `isFestivalDay`, and `note`.

Examples from the current data set:

- `common/holiday/data/2023.json:2-33`
- `common/holiday/data/2024.json:2-37`
- `common/holiday/data/2025.json:2-34`
- `common/holiday/data/2026.json:2-40`

### External References

- [国务院办公厅关于2023年部分节假日安排的通知](https://www.gov.cn/zhengce/content/2022-12/08/content_5730844.htm) - Official 2023 arrangement.
- [国务院办公厅关于2024年部分节假日安排的通知](https://www.gov.cn/zhengce/content/202310/content_6911527.htm) - Official 2024 arrangement.
- [国务院办公厅关于2025年部分节假日安排的通知](https://www.gov.cn/zhengce/content/202411/content_6986382.htm) - Official 2025 arrangement.
- [国务院办公厅关于2026年部分节假日安排的通知](https://www.gov.cn/zhengce/zhengceku/202511/content_7047091.htm) - Official 2026 arrangement.

Official excerpts used for comparison:

- 2023: “元旦：2022年12月31日至2023年1月2日放假调休，共3天。二、春节：1月21日至27日放假调休，共7天。1月28日（星期六）、1月29日（星期日）上班。”
- 2024: “元旦：1月1日放假，与周末连休。春节：2月10日至17日放假调休，共8天。2月4日（星期日）、2月18日（星期日）上班。…中秋节：9月15日至17日放假调休，共3天。9月14日（星期六）上班。…国庆节：10月1日至7日放假调休，共7天。9月29日（星期日）、10月12日（星期六）上班。”
- 2025: “元旦：1月1日（周三）放假1天，不调休。春节：1月28日（农历除夕、周二）至2月4日（农历正月初七、周二）放假调休，共8天。1月26日（周日）、2月8日（周六）上班。…国庆节、中秋节：10月1日（周三）至8日（周三）放假调休，共8天。9月28日（周日）、10月11日（周六）上班。”
- 2026: “元旦：1月1日（周四）至3日（周六）放假调休，共3天。1月4日（周日）上班。春节：2月15日（农历腊月二十八、周日）至23日（农历正月初七、周一）放假调休，共9天。2月14日（周六）、2月28日（周六）上班。…国庆节：10月1日（周四）至7日（周三）放假调休，共7天。9月20日（周日）、10月10日（周六）上班。”

### Exact Discrepancy Report

#### `common/holiday/data/2023.json`

| Date | Current | Expected | Source |
|---|---|---|---|
| `2023-01-01` | missing | `type=holiday`, `name=元旦`, `isFestivalDay=true`, `note=元旦` | [gov.cn 2023 notice](https://www.gov.cn/zhengce/content/2022-12/08/content_5730844.htm) |
| `2023-01-02` | `type=holiday`, `name=元旦`, `isFestivalDay=true`, `note=元旦` | `type=holiday`, `name=元旦`, `isFestivalDay` should be the statutory festival day on `2023-01-01`, not `2023-01-02` | [gov.cn 2023 notice](https://www.gov.cn/zhengce/content/2022-12/08/content_5730844.htm) |

#### `common/holiday/data/2024.json`

- No exact date/type/festival-day mismatches found against the official 2024 notice.

#### `common/holiday/data/2025.json`

- No exact date/type/festival-day mismatches found against the official 2025 notice.

#### `common/holiday/data/2026.json`

- No exact date/type/festival-day mismatches found against the official 2026 notice.

### Semantic Naming / Note Caveats

- `2023.json` uses `name: "中秋国庆"` for the combined period. The official notice names this section as `中秋节、国庆节`.
- `2025.json` uses `name: "中秋国庆"` for the combined period. The official notice names this section as `国庆节、中秋节`.
- In both combined-period files, notes alternate between `中秋节` and `国庆节` across the same holiday block. That is semantically understandable, but it is not the exact wording used in the official title lines.

## Caveats / Not Found

- I found a primary gov.cn source for all four years, including 2026. No secondary source was needed for the final comparison.
- The only exact data mismatch is in `2023.json` around New Year’s Day: `2023-01-01` is missing and `2023-01-02` is incorrectly marked as the festival day.
