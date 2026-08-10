# 完善 DJI 设备名称与负载云台位置

## Goal

通过 HMS `device_type={domain}-{type}-{sub_type}` 得到当前告警设备的准确产品名称，并结合宿主飞机型号解释负载 `gimbal_index` 的挂载位置。

## Requirements

- 完整三元组唯一查询告警设备产品名称，覆盖飞机、负载、遥控器和机场；未知三元组返回空名称。
- 负载 `device_type` 得到的是负载产品名，不拼入宿主飞机或云台位置。
- 位置转换显式接收宿主飞机 `DeviceType` 与 `gimbal_index`。
- M300 RTK：0 下方左、1 下方右、2 上方；所有机型的 7 为 FPV。
- 非 M300 RTK 的 0 为主云台；1、2 和其他预留值不套用 M300 专属位置。

## Acceptance Criteria

- [ ] 注册表测试覆盖当前全部官方三元组及代表性四类设备名称。
- [ ] 位置测试覆盖 M300 RTK 0/1/2/7、其他机型 0/7 和预留值。
- [ ] SDK API 不在缺少宿主型号时猜测 M300 专属位置。
- [ ] `go test ./common/djisdk` 通过。

## Dependencies

- 无前置子任务，可与 `08-10-hms-template-resolution` 并行。
