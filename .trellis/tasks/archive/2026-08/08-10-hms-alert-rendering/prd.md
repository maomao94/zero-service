# 检查并修正 HMS 告警文案查询与填充

## Goal

检查并修正 HMS 告警从协议字段生成文案 Key、查询 `hms.json`、填充模板参数并写入告警历史的完整行为，使其符合 DJI HMS 规则。

## Background

- 协议入口为 `HmsItem`，文案解析由 `common/djisdk.HmsResolver` 负责，`app/djicloud/internal/hooks.NewHmsEventNotifyHandler` 消费解析结果并追加写入告警历史。
- HMS 是 `thing/.../events` 上行推送。外层 `EventMessage` 已包含 `tid`、`bid`，MQTT 消费上下文承载服务端 OTel trace；同一次推送中的多个 HMS item 共享这组三类关联标识。
- 当前实现根据 `device_type` 的 domain 区分飞机（0）和机场（3），不为负载、遥控器或非法设备类型猜测文案类别。
- 当前测试已覆盖主要 Key 查询、参数填充、缺参保留和告警入库链路，但需按本次补充规则重新核对边界。

## Requirements

- `imminent` 表示告警是否具有即时性/瞬时性：`0` 否、`1` 是；即时性告警可随触发条件消失而自动消失。不得描述为“是否紧急”，告警严重程度由 `level` 表达。
- 机场告警只查询 `dock_tip_{code}`，不受 `in_the_sky` 影响。
- 飞机告警以 `fpv_tip_{code}` 为普通 Key；`in_the_sky=1` 时优先查询 `fpv_tip_{code}_in_the_sky`，该 Key 不存在时回退普通 Key；`in_the_sky=0` 时直接使用普通 Key。
- `%alarmid` 使用 args 中的 `alarmid` 十六进制文本原样替换。
- `%index` 使用 `sensor_index + 1` 替换。
- `%component_index` 只按协议执行 `component_index + 1` 后替换，不按当前产品支持的云台数量做范围限制。平台负责通用文案转换，设备型号支持几个云台不固化到 HMS 渲染器中；仅在参数缺失或类型非法时保留占位符。
- `%battery_index` 和 `%dock_cover_index` 在 `sensor_index=0` 时替换为“左”，其他值替换为“右”。
- `%charging_rod_index` 根据 `sensor_index=0/1/2/3` 分别替换为“前/后/左/右”；范围外值不臆测方向。
- args 缺失或类型非法时不丢弃告警，保留未解析占位符并继续写入原始 item 和已解析 message。
- hook 不重复实现 Key 或模板规则，只消费 `HmsResolver.Resolve` 的结果。
- `component_index`、`sensor_index`、`gimbal_index`、`lidar_index`、`lte_index` 均属于开放 `args` 的模板参数；完整原始值由 `item_json` 保存。仓库当前不存在依赖这些数据库平铺列进行筛选、排序或业务判断的读取方，因此从 HMS 数据库模型和 RPC 响应中移除这些平铺字段。
- `device_type_name` 继续由告警 `device_type={domain}-{type}-{sub_type}` 查询官方产品注册表得到。对于负载告警，产品名称只表示负载型号，不混入宿主飞机或挂载位置。
- `hms.json` 仅将 `%gimbal_index` 作为数值模板变量使用，没有 `gimbal_position` 或主/上下/左右云台位置字段，因此 HMS 告警不派生、不持久化、不返回 `gimbal_position`。原始 `gimbal_index` 保留在 `item_json.args` 并按数值填充 message。
- HMS 告警记录需要关联原始推送的 `tid`、`bid` 和服务端 `trace_id`，用于从告警历史追踪到 DJI 事件和内部消费链路；三者不得混用：`tid` 是单次 DJI 事件 ID，`bid` 是 DJI 业务流程 ID，`trace_id` 是平台 OTel 链路 ID。
- `tid`、`bid` 必须从 HMS 外层事件信封传入 hook，不得从 item 或日志文本反向解析；`trace_id` 从消费 `context.Context` 提取。
- 每条 HMS 告警保存 `message_key`，其值必须是 `HmsResolver` 实际命中的字典 Key，而不是仅按输入预先推导的候选 Key。飞机空中专用 Key 不存在而回退普通 Key 时保存普通 Key；机场始终保存普通 Key；未命中字典时保存空值。
- `message_key`、`tid`、`bid`、`trace_id` 同时写入告警历史并由 `HmsAlertInfo` 返回，支持调用方展示来源和排查推送链路。
- `HmsAlertInfo` 按未发布契约整理：声明按记录身份、DJI 推送关联标识、告警主体、设备身份、展示文案与原始条目、确认信息、上报时间分组；移除字段后不保留 `reserved`，全部字段按语义顺序连续重编号。

## Acceptance Criteria

- [ ] 单元测试证明机场、飞机地面、飞机空中和空中专用 Key 缺失回退均符合规则。
- [ ] 单元测试证明 `%alarmid`、`%index`、不限制产品数量的 `%component_index + 1`、左右和前后左右方向替换符合规则及边界。
- [ ] 单元测试证明缺参、非法索引和未知 Key 不导致事件丢失或错误类别回退。
- [ ] hook 测试证明最终解析文案及原始 HMS item 正确写入告警历史。
- [ ] 一次包含多个 item 的 HMS 推送中，每条告警记录均保存相同且准确的 `tid`、`bid`、`trace_id`，日志也可按这些标识关联。
- [ ] 已知机场、飞机地面、飞机空中专用和飞机空中回退告警均保存实际命中的 `message_key`；未知告警的 `message_key` 为空。
- [ ] 协议、模型或接口中对 `imminent` 的说明准确表达即时性，不再表达紧急程度。
- [ ] `HmsAlertInfo` 字段声明顺序清晰，生成代码来自 `app/djicloud/gen.sh`，且没有重编号或复用已发布字段号。
- [ ] `ListHmsAlerts` 返回 `message_key`、`tid`、`bid`、`trace_id`，不再返回五个 args 索引平铺字段。
- [ ] 飞机、机场、遥控器和负载 `device_type` 均按注册表返回准确产品名称；未知三元组名称为空。
- [ ] HMS 告警不查询宿主拓扑或产生 `gimbal_position`；`%gimbal_index` 仍按原始数值正确填充 message。
- [ ] `go test ./common/djisdk`、`go test ./app/djicloud/internal/hooks` 和 `go test ./app/djicloud/...` 通过。

## Task Map

- `08-10-hms-template-resolution`：文案 Key、实际命中 Key、占位符和 `imminent` 协议语义。
- `08-10-hms-device-placement`：`device_type` 产品名称及按宿主机型解释负载云台位置。
- `08-10-hms-event-persistence`：事件 `tid`/`bid`/`trace_id`、message key、设备名称、位置与模型持久化。
- `08-10-hms-proto-query`：Proto 字段排序、reserved、生成代码和查询响应。

前两个子任务无相互依赖，可由不同 agent 并行开发并分别验证；持久化任务等待前两项完成；Proto 查询任务等待持久化模型稳定；父任务最后执行全链路集成检查。

## Out Of Scope

- 修改 DJI 提供的 `hms.json` 文案内容或翻译。
- 为 domain 1/2 或非法 `device_type` 新增非官方 Key 前缀。
- 修改 HMS 告警确认、分页、排序、过滤和语言回退策略。
- 回填既有历史记录的 `message_key`、`tid`、`bid` 或 `trace_id`。
