# 整理 HMS Proto 与查询响应

## Goal

使 `HmsAlertInfo` 按 HMS 语义有序返回最终持久化数据，并安全收缩冗余索引字段。

## Requirements

- Proto 声明按记录身份、推送关联、告警主体、设备身份、展示与原始数据、确认信息、上报时间分组。
- 本次按未发布契约处理，不保留旧字段号兼容性；删除五个索引字段后不声明 `reserved`。
- 所有保留及新增字段按当前语义声明顺序从 1 开始连续重编号；不新增 HMS 协议不存在的 `gimbal_position`。
- `device_type_name` 表示 `device_type` 对应的当前告警设备产品名称。
- `ListHmsAlerts` 返回持久化快照，不查询时重算字典、拓扑或 args。

## Acceptance Criteria

- [ ] `app/djicloud/gen.sh` 成功且生成 diff 无无关噪声。
- [ ] 查询映射返回全部新增字段，不再引用五个删除字段。
- [ ] Proto 无编号空洞和 `reserved`，字段按语义顺序连续编号，注释准确说明 `imminent`、设备名称和关联 ID。
- [ ] `go test ./app/djicloud/...` 通过。

## Dependencies

- 依赖 `08-10-hms-event-persistence` 完成，模型字段稳定后实施。
- 不与持久化子任务并行修改 `ListHmsAlerts` 或 HMS model。
