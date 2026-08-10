# Design: Persist Topology Device Name and Type

## Boundaries

- `common/djisdk` 继续拥有设备三元组格式和产品名称注册表，本任务只消费现有 `DeviceType.String` 与 `LookupDeviceTypeName`。
- `app/djicloud/internal/hooks/sys_status_up.go` 是拓扑派生字段的唯一写入者。
- `DjiDeviceTopo` 保存每个网关绑定关系的原始三元组字段和派生查询字段；`DjiDevice` 保存设备自身最近一次已知的型号和名称。
- `djicloud.proto` 是查询契约源；生成文件只通过 `app/djicloud/gen.sh` 更新。

## Data Flow

1. `update_topo.sub_devices[]` 提供 `domain`、`type`、`sub_type`。
2. Hook 使用上游 `domain/type/sub_type` 生成规范三元组字符串 `device_type`。
3. Hook 使用 `djisdk.LookupDeviceTypeName(deviceType)` 查询产品名称；该函数负责严格解析三元组，非法或未收录型号均得到空字符串。
4. `FirstOrCreate + Assign` 在同一事务中写入拓扑原始字段、派生字段，并按 `device_sn = sub.SN` 唯一更新 `DjiDevice` 的型号和名称；`gateway_sn` 仅对非 domain 0/1 由 `update_topo` 覆盖。
5. 设备列表和详情分别从 `DjiDevice`、`DjiDeviceTopo` 映射两个字段到 `DeviceInfo`、`DeviceTopoInfo`。

## Contracts

- `DjiDevice` 和 `DjiDeviceTopo` 均增加：`device_type varchar(32) not null default ''`，`device_name varchar(128) not null default ''`；Go 字段紧跟各自 `GatewaySn`。
- `DjiDevice.device_sn` 是设备主记录唯一键；`DjiDeviceTopo.gateway_sn + sub_device_sn` 是多机巢绑定关系唯一键，两者不可混用。
- `device_type` 只使用 `{domain}-{type}-{sub_type}`，不包含 `sub_device_index`。
- `device_name` 是注册表查询结果；合法但未知型号使用空字符串，不阻止拓扑入库。
- `DeviceTopoInfo` 追加 `device_type = 8`、`device_name = 9`，保留现有字段号 `1..7`。
- `DeviceInfo` 为未发布契约，`device_type = 4`、`device_name = 5`，后续字段顺延并保持连续，不使用 `reserved`。

## Compatibility And Migration

- 新 Proto 字段对旧客户端是向后兼容的可选缺省字段。
- 服务启动时现有 AutoMigrate 增加数据库列；不增加一次性迁移或启动回填。
- 历史记录在下一次对应 `update_topo` 上报时通过现有 Assign 路径补齐。

## Trade-offs

- 选择持久化而非查询时动态计算，确保 API 直接返回入库快照，并避免每个消费者重复解析。
- 保留原始三字段与派生字段，存在有限冗余，但可直接审计上游值并满足查询需求。
- 不新增公共 helper；现有 SDK 查询函数已覆盖严格解析和产品查询逻辑。

## Rollback

- 代码回滚可停止写入和返回新字段；数据库新增列保留为空闲列，不需要破坏性回滚。
- 若生成结果出现无关噪声，停止实施并检查 `gen.sh` 环境，不手工修生成文件。
