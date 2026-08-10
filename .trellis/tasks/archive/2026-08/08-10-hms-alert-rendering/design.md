# 技术设计

## 边界与所有权

- `common/djisdk` 拥有 DJI 事件信封、HMS typed payload、文案 Key 选择和模板渲染规则。
- HMS 回调保持现有签名；SDK 事件入口通过 typed context 保存外层 `tid`、`bid`，避免 `app/djicloud` 从原始 JSON 或日志反向解析。
- `app/djicloud/internal/hooks` 负责从 resolver 取得实际命中的 `message_key`，从 context 提取平台 `trace_id`，并逐条追加告警历史。
- `common/djisdk` 的设备注册表负责三元组到产品名称；负载挂载位置转换属于独立设备拓扑能力，不接入 HMS 告警持久化。
- GORM model 拥有持久化字段，`djicloud.proto` 拥有查询响应契约；生成文件只能由 `app/djicloud/gen.sh` 更新。

## 数据流

1. MQTT 消费器建立 OTel consumer span，将上下文传入 `djisdk.Client.HandleEvents`。
2. `HandleEvents` 解码 `EventMessage`，把 `tid`、`bid`、method 等加入日志 context。
3. HMS 分发器从同一外层事件信封构造窄的事件元数据，连同 `HmsEventData` 传给 HMS handler。
4. hook 对每个 item 调用 `HmsResolver.Resolve`，得到实际命中的 Key、模板和最终文案。
5. hook 将同一推送的 `tid`、`bid`、context trace ID、实际 `message_key` 和每个 item 的原始 JSON追加写入 `dji_hms_alert`。
6. `ListHmsAlerts` 原样返回持久化关联字段、设备产品名和文案字段，不在查询阶段重新解析字典或 args。

## Key 与模板契约

- 机场只查 `dock_tip_{code}`。
- 飞机地面只查 `fpv_tip_{code}`；飞机空中先查 `fpv_tip_{code}_in_the_sky`，不存在时回退普通 Key。
- `message_key` 保存最终实际命中的 Key；未知设备、未知 code 或无对应字典项时为空。
- `%alarmid` 原样填充；`%index` 为 `sensor_index + 1`。
- `%component_index` 对任何可被 `HmsArgs.Int` 严格读取的整数执行加一，不根据当前设备产品支持的云台数量截断或拒绝。
- 电池和舱盖使用 `sensor_index=0` 左、其他右；充电杆仅接受 `0..3`，分别映射前、后、左、右。
- 缺参或非法值保留占位符并记录日志，不阻止事件入库。

## 持久化设计

`DjiHmsAlert` 新增：

- `MessageKey`：实际命中的 HMS 字典 Key。
- `Tid`：DJI 单次事件 ID。
- `Bid`：DJI 业务流程 ID。
- `TraceID`：服务端 OTel 消费链路 ID。

移除 `ComponentIndex`、`SensorIndex`、`GimbalIndex`、`LidarIndex`、`LteIndex`。原始值仍保存在 `ItemJSON.args`，HMS 不额外派生云台位置。现有测试使用 AutoMigrate 验证新 schema；生产环境删列仍由既有部署迁移流程负责，应用代码不依赖删列即时完成。

## 设备名称

- `device_type_name` 由完整三元组查询注册表；domain 1 的名称是负载产品名称，不将宿主型号拼入名称。
- `gimbal_index` 在 HMS 中仅用于 `%gimbal_index` 数值填充；设备拓扑中的位置解释不混入 HMS 历史。

`Imminent` 保留整数值，但模型和协议注释改为“是否为即时性告警，0 否、1 是”，不再表述为紧急程度。

## Proto 设计

`HmsAlertInfo` 按未发布契约处理。删除五个索引字段后不声明 reserved，全部保留和新增字段按照记录身份、推送关联、告警主体、设备身份、展示与原始数据、确认信息、上报时间连续编号。

## 兼容性与回滚

- 删除 RPC 字段对仍读取五个索引字段的新旧客户端属于契约收缩；字段号永久保留，不会被新语义复用。
- 新增字段对旧客户端按 Proto 未知字段规则兼容。
- 历史记录的新字段为空，不做无法可靠恢复的 `tid`、`bid`、`trace_id` 回填。
- 应用回滚后新增数据库列可保留；删列应在确认旧版本不再需要后由部署迁移执行，避免阻断应用回滚。
- 不修改 `hms.json`，出现解析回归时可回滚 SDK/hook/model/proto 代码而不迁移字典数据。
