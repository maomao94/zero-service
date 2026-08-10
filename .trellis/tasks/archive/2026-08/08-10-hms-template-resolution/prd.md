# 校正 HMS 文案 Key 与模板填充

## Goal

使 HMS 文案 Key 查询和 args 模板填充严格符合 DJI 规则，并稳定返回最终实际命中的 Key。

## Requirements

- 机场只查询 `dock_tip_{code}`。
- 飞机地面查询 `fpv_tip_{code}`；飞机空中优先 `_in_the_sky`，缺失时回退普通 Key。
- `%alarmid` 原样填充；`%index` 和 `%component_index` 分别执行对应索引加一。
- `%component_index` 不绑定产品支持数量，不做云台数量范围限制。
- 电池、舱盖和充电杆按官方左右及前后左右规则转换。
- 缺参或非法类型保留占位符，不阻止事件继续处理。
- `imminent` 注释改为即时性/瞬时性告警，不描述为紧急程度。

## Acceptance Criteria

- [ ] 测试覆盖机场、飞机地面、飞机空中专用和空中回退，断言实际 Key 与 message。
- [ ] 测试覆盖所有指定占位符、超出当前产品数量的 component index、缺参及非法类型。
- [ ] `go test ./common/djisdk` 通过。

## Dependencies

- 无前置子任务，可与 `08-10-hms-device-placement` 并行。
