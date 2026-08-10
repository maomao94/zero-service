# Persist Topology Device Name and Type

## Goal

在 DJI `update_topo` 上行入库时，为子设备主记录和每条拓扑记录保存规范设备三元组和对应产品名称，避免消费者重复拼接型号或查询产品注册表。

## Background

- `DjiDeviceTopo` 当前分别保存 `Domain`、`SubDeviceType`、`SubDeviceSubType`，但未保存组合后的设备型号和产品名称。
- 拓扑快照唯一身份仍为 `gateway_sn + sub_device_sn`；本任务不改变蛙跳场景和软删除恢复语义。
- `common/djisdk.DeviceType.String()` 已定义规范 `{domain}-{type}-{sub_type}` 格式。
- `common/djisdk.LookupDeviceTypeName(string)` 已按该三元组查询产品注册表；合法但未收录的型号返回空名称。
- 拓扑的唯一写入链路位于 `app/djicloud/internal/hooks/sys_status_up.go`，现有 `FirstOrCreate + Assign` 会同时处理新增和重复上报更新。

## Requirements

- `DjiDeviceTopo` 新增持久化字段 `DeviceType` 和 `DeviceName`。
- `DjiDevice` 同步新增 `DeviceType` 和 `DeviceName`，字段顺序紧跟 `GatewaySn`。
- `DeviceType` 必须由每条 `sub_device` 的 `domain/type/sub_type` 组合生成，格式固定为 `{domain}-{type}-{sub_type}`，例如 `0-60-0`。
- `DeviceName` 必须复用 `common/djisdk` 产品注册表查询能力，不在 hook 或 model 中复制产品映射。
- 已收录型号保存对应产品名称；未收录型号保存空字符串，但仍保存规范 `DeviceType`。
- 重复 `update_topo` 上报必须更新 `device_type` 和 `device_name`，软删除后恢复也必须写入最新值。
- RPC `DeviceTopoInfo` 必须返回持久化的 `device_type` 和 `device_name`，设备列表与设备详情保持一致。
- RPC `DeviceInfo` 必须在 `gateway_sn` 后返回 `device_type` 和 `device_name`；该契约未发布，字段按语义顺序连续编号，不保留旧编号兼容。
- `DjiDevice` 必须按 `device_sn = sub.SN` 唯一定位并始终刷新 `device_type`、`device_name`；`gateway_sn` 保持原逻辑，domain 0/1 不由 `update_topo` 覆盖，其他 domain 更新为当前网关。
- 保持现有事务、差量删除和 `DjiDeviceTopo` 蛙跳多绑定语义不变。

## Acceptance Criteria

- [ ] 已收录的拓扑子设备入库后，`device_type` 为三元组字符串且 `device_name` 为注册表产品名称。
- [ ] 未收录的合法三元组入库后，`device_type` 仍正确且 `device_name` 为空。
- [ ] 同一拓扑关系再次上报不同 `domain/type/sub_type` 时，两个派生字段与原始字段一起更新。
- [ ] 软删除拓扑恢复时，两个派生字段写入最新上报值。
- [ ] `DeviceTopoInfo` 新增字段后，设备列表和详情均返回数据库中的 `device_type` 与 `device_name`。
- [ ] `DjiDevice` 主记录同步保存两个字段，`DeviceInfo` 在 `gateway_sn` 后按连续字段号返回。
- [ ] 执行 `app/djicloud/gen.sh` 后，Proto 源与生成产物一致。
- [ ] 相关 hook/model/Logic 测试和 `go test ./app/djicloud/...` 通过，`git diff --check` 通过。

## Out of Scope

- 修改产品名称注册表内容或名称文案。
- 改变 `DjiDevice` 在线状态语义。
- 从 `sub_device_index` 推导云台位置。
- 主动回填已有拓扑历史行；旧行保持空值，后续收到 `update_topo` 时自然补齐。

## Notes

- 当前工作区已有不属于本任务的 `common/djisdk/hms.go` 修改，实施和提交时不得包含或覆盖。
