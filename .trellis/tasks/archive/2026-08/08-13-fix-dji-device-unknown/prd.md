# 修复 GaussDB 设备型号空串约束

## Goal

避免 GaussDB 兼容模式将设备型号或名称空字符串转换为 SQL NULL，导致 `dji_device` / `dji_device_topo` 的非空约束失败。

## Requirements

- 保留 `device_type`、`device_name` 的数据库 `NOT NULL` 约束，不改为可空字段，不使用 `sql.NullString`。
- 设备型号或名称暂时未知时，数据库显式保存非空哨兵值 `unknown`，不得依赖空字符串默认值。
- OSD 或 State 先于 `update_topo` 到达时，仍可创建并更新设备主记录和遥测快照。
- 后续收到 `update_topo` 时，真实设备三元组和产品名称必须覆盖已有哨兵值。
- 合法但产品注册表未收录的设备三元组保留真实 `device_type`，`device_name` 保存 `unknown`。
- RPC 沿用持久化值，未知型号或名称返回 `unknown`。
- 不根据设备 SN 猜测设备型号。

## Acceptance Criteria

- [x] OSD/State 首次创建的设备不会向型号或名称列写入空字符串或 SQL NULL。
- [x] OSD/State 首次创建的设备保存 `device_type=unknown`、`device_name=unknown`。
- [x] `update_topo` 可将设备主表中的 `unknown` 覆盖为真实三元组和产品名称。
- [x] 未收录产品名称的拓扑记录和设备主记录保存 `device_name=unknown`，不触发非空约束。
- [x] 现有设备归属、在线状态、快照和蛙跳场景行为保持不变。
- [x] DJI hook 和 model 相关测试通过。

## Notes

- 根因是 GaussDB 当前兼容模式将 `''` 视为 `NULL`；日志中的 GORM INSERT 已确认应用侧原始值为空字符串。
- 本任务为轻量修复，采用 PRD-only。
