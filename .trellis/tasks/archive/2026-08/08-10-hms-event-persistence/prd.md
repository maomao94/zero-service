# 补全 HMS 推送关联与告警持久化

## Goal

将 HMS 外层推送、文案解析结果和设备展示信息完整关联到每条告警历史。

## Requirements

- HMS handler 保持现有签名；SDK 在事件入口将外层 `tid`、`bid` 写入 typed context，hook 通过 SDK helper 从 context 读取。`trace_id` 直接从同一消费 context 的 OTel span 提取。
- 每条告警保存实际 `message_key`、message、当前告警设备名称及完整 `item_json`。
- HMS 不派生 `gimbal_position`；`hms.json` 中 `%gimbal_index` 仅按 args 原始数值填充 message，原值保留在 `item_json.args`。
- 新增 `message_key`、`tid`、`bid`、`trace_id` 持久化字段。
- 移除五个 args 原始索引平铺字段；原值仍保留在 `item_json`。
- `imminent` 数据库注释表达即时性，不表达紧急程度。

## Acceptance Criteria

- [ ] 一个推送包含多个 item 时，每条记录保存相同且准确的推送关联标识。
- [ ] 实际命中、空中回退和未知告警分别保存正确或空的 `message_key`。
- [ ] `device_type` 返回当前告警设备产品名称。
- [ ] hook 不查询宿主拓扑，不持久化 HMS 协议不存在的 `gimbal_position`。
- [ ] hook/model 测试通过且事件历史仍逐条追加。

## Dependencies

- 依赖 `08-10-hms-template-resolution` 和 `08-10-hms-device-placement` 完成并通过检查。
- 本任务不得与上述两个子任务同时修改同一 SDK API；等待两者合入后实施。
