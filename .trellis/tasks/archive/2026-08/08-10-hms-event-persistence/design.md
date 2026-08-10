# 技术设计

- HMS handler 保持现有签名。`HandleEvents` 在已有日志 context 上叠加 SDK 私有 typed event context，保存 `tid`、`bid`；hook 通过只读 helper 获取，不从 logx 字段或 JSON 反向解析。
- hook 从 context 提取 OTel trace ID，不把 `tid`、`bid` 当 trace ID。
- 告警 `device_type` 直接查询当前告警设备名称。
- HMS hook 不查询宿主拓扑；`gimbal_index` 仅由 resolver 用于 message 数值填充并保留在 `item_json`。
- 所有派生结果在上报时持久化，查询阶段不重新读取字典或拓扑，保留历史快照语义。
- 旧索引列的物理删除与应用发布解耦，应用先停止读写，部署迁移后续清理。
