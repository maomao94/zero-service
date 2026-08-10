# 实施计划

- [ ] 校对 `deviceTypeRegistry` 与官方产品支持表。
- [ ] 保持 `LookupDeviceTypeName` 以完整三元组查询当前告警设备名称。
- [ ] 重构负载位置 API，使宿主飞机型号成为必需输入。
- [ ] 删除不区分宿主的笼统 M300 描述，按宿主型号执行映射。
- [ ] 补充设备名称、M300、其他机型、FPV 和预留枚举测试。
- [ ] 运行 `gofmt`、`go test ./common/djisdk`、`git diff --check`。
