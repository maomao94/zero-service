# review holiday trigger

## Goal

继续审阅 `app/trigger` 与 `common/holiday` 中新增的中国大陆节假日能力，修正发现的明确问题，确保接口契约、生成代码、GORM 数据源、内存日历查询和测试覆盖保持一致。

## Requirements

- `QueryHolidayGroupReq.name` 支持为空；为空时查询全年节假日分组汇总。
- 保持 `common/holiday` API 边界清晰，业务侧通过 `ServiceContext` 使用日历和可编辑数据源。
- 核对 `HolidayDayPb.kind`、源配置、分组响应的字段语义，避免保存派生字段。
- 审阅 GORM 初始化、保存、启停、列表、缓存刷新路径，修正明显缺陷。
- 修改后执行相关 Go 测试。

## Acceptance Criteria

- [ ] 相关 proto 与生成代码一致。
- [ ] `common/holiday.Group(year, "")` 可返回全年特殊日期汇总。
- [ ] 保存、启停和查询路径没有发现阻断性问题。
- [ ] `go test -count=1 ./common/holiday ./app/trigger/...` 通过。

## Notes

- 本任务为轻量审阅任务，PRD-only。
