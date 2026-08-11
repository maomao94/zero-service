# 新增 bridge/alarm/file domain Spec

参考父任务 [prd.md](../08-11-refresh-all-specs/prd.md)。

## Scope

为以下 service 新增 domain spec：

| 新 Spec 文件 | 涵盖 service/包 |
|-------------|----------------|
| bridge-guidelines.md | app/bridgegtw, bridgekafka, bridgemodbus, bridgemqtt, bridgedump + common/mqttx, modbusx |
| alarm-guidelines.md | app/alarm + common/alarmx |
| file-guidelines.md | app/file + common/filex, ossx |

## Acceptance Criteria

- [ ] 每个 spec 包含标准的 7 section 场景
- [ ] 有真实 source file 引用
- [ ] backend/index.md 已更新
