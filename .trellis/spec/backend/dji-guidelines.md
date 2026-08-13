# DJI Cloud 与 DRC 规范

## 适用范围

修改 `common/djisdk`、`app/djicloud`、DJI Cloud MQTT topic/method/payload、设备在线状态、OSD/State/Event 持久化、DRC 或飞行区同步时读取。

## SDK 与协议所有权

- `common/djisdk` 的协议包拥有 topic 构造（`topic.go`）、method 常量（`method.go`）、typed payload（`protocol.go`）、DRC 载荷（`protocol_drc.go`）和结果解析；`app/djicloud` Logic/hook 不手工拼 topic 或 `map[string]any`。
- `MustNewClient(cfg Config, opts ...ClientOption)` 为 go-zero 风格构造，接收 `mqttx.MqttConfig`、`PendingTTL`、`ReplyConfig`、`DrcConfig`、`HmsConfig`；`NewClient(mqttx.Client, opts ...ClientOption)` 接收已建好的 MQTT client。
- `Config.PendingTTL` 默认 30s，控制在 `mqttx.RequestReply` 中等待设备 `services_reply` / `property/set_reply` 的超时。
- handler 注册通过 `With*Handler` 系列 option：`WithFlightTaskProgressHandler`、`WithHmsEventNotifyHandler`、`WithUpdateTopoHandler`、`WithOsdHandler`、`WithStateHandler`、`WithStatusHandler`、`WithRequestHandler`、`WithDrcUpHandler`、`WithOnlineChecker` 等。handler 无返回值的方法也可用 `WithFlightTaskReadyHandler` 等注册。
- `SendCommand` 经 `services` 下发并等待 `services_reply`，应答中 `result != 0` 转换为 `*DJIError`；`SendCommandFireAndForget` 只发布不等待应答；两者语义不同，不能互换。
- `PropertySet` 经 `property/set` 下发并等待 `property/set_reply`，与 `SendCommand` 共用同一 request/reply 机制但走独立 topic。
- 请求应答使用 TID/BID 关联，在 `mqttx.ReplyRouter` 中通过 TID 匹配请求与回复。
- `onlineChecker` 在 `SendCommand` 入口提前拦截离线设备，不经 MQTT 发布直接返回错误。

依据：`common/djisdk/client.go:37-49`、`common/djisdk/option.go:21-217`、`common/djisdk/handler.go:43-78`、`common/djisdk/client.go:80-117`。

## 上行处理与数据所有权

- SDK 通配订阅 `SubscribeAll()` 注册六路 handler：`HandleOsd`、`HandleState`、`HandleEvents`、`HandleRequests`、`HandleStatus`、`HandleDrcUp`。`app/djicloud` 的 hook 各自注册到对应 handler option，不建立第二套总分发器。
- `HandleEvents` 内按 method 预分发强类型处理（`tryDispatchEventNotify`）：`flighttask_progress` → `OnFlightTaskProgress`、`flighttask_ready` → `OnFlightTaskReady`、`hms` → `OnHmsEventNotify`、`return_home_info` → `OnReturnHomeInfo`、`ota_progress` → `OnOtaProgress`、`remote_log_file_upload_progress` → `OnRemoteLogFileUploadProgress`、`flight_areas_sync_progress` → `OnFlightAreasSyncProgress`、`flight_areas_drone_location` → `OnFlightAreasDroneLocation`、`custom_data_transmission_from_psdk` / `_from_esdk` 等；未命中 method 打印日志不报错。`need_reply` 由 `ReplyConfig.EnableEventReply` 控制。
- `HandleStatus` 内按 method 预分发 `update_topo` → `OnUpdateTopo`，未命中则由 `OnStatus` 全局回调兜底；`ReplyConfig.EnableStatusReply` 控制 `status_reply` 发送。
- `HandleRequests` 将整条 `RequestMessage` 交给 `OnRequest` handler，返回 `output` 写入 `requests_reply.data.output`；handler 返回 `ErrSkipRequestReply` 时不发送回复。`ReplyConfig.EnableRequestReply` 控制回复行为。
- `HandleDrcUp` 解析为 `DrcUpMessage` + typed data（`DrcUnmarshalUpData`），未知 method 保留 `*DrcUnknownUpData`；`drc/up` 上的心跳 `heart_beat` 自动路由到 DRC manager 的 `OnDeviceHeartbeat`。
- handler 返回的 error 若为 `*PlatformError`，其 `Code` 作为 `status_reply` / `events_reply` / `requests_reply` 的 `result`；否则默认 `PlatformResultHandlerError`（1）。
- 设备/拓扑/最新状态属于快照，使用查询后更新或 `FirstOrCreate` + `Assign` 模式；事件/告警历史使用追加 `Create`，不得 Upsert 覆盖历史。
- 在线语义由当前 OSD/拓扑/heartbeat 处理链路拥有；单条旧消息或迟到 callback 不能覆盖更新 session 的在线状态。内存缓存 (`OnlineCache`) 做在线判断短路，缓存未命中时回退数据库 `last_online_at` 懒过期。
- Store/model 决定事务和更新字段。不要从 hook 整行 `Save`，也不要以方言专属 Upsert SQL 替换已经测试的写入模式。

依据：`app/djicloud/internal/hooks`、`app/djicloud/internal/logic/helper.go`、`common/djisdk/handler.go:90-567`、`app/djicloud/model/gormmodel` 及其测试。

## 设备主表与拓扑型号契约

### 1. Scope / Trigger

- 修改 `update_topo`、`DjiDevice`、`DjiDeviceTopo`、`DeviceInfo` 或 `DeviceTopoInfo` 的设备型号、产品名称和网关归属时适用。
- `common/djisdk` 拥有 `{domain}-{type}-{sub_type}` 格式与产品名称注册表；DJI hook 只负责从 typed topology payload 派生并持久化。

### 2. Signatures

- `DjiDevice` 以 `device_sn` 唯一，保存设备最近快照；`DeviceType`、`DeviceName` 紧跟 `GatewaySn`。
- `DjiDeviceTopo` 以 `gateway_sn + sub_device_sn` 唯一，允许同一 `sub_device_sn` 出现在多个网关绑定中。
- 型号使用 `ParseDeviceType(string)` 规范化，名称使用 `LookupDeviceTypeName(string)` 查询；格式固定为 `{domain}-{type}-{sub_type}`。
- `DjiDeviceUnknown == "unknown"` 是设备身份尚未补齐的非空哨兵值；两张表的 `device_type/device_name` 保持 `NOT NULL DEFAULT 'unknown'`。
- 未发布的 `DeviceInfo` 按语义顺序连续编号：`gateway_sn` 后为 `device_type`、`device_name`，不为旧编号增加 `reserved`。

### 3. Contracts

- 每次 `update_topo` 都同时更新拓扑记录和按 `device_sn = sub.SN` 定位的主设备记录中的 `device_type/device_name`。
- domain 0/1 的主设备 `gateway_sn` 不由 `update_topo` 覆盖，继续由 OSD/State 表达当前通信网关；domain 2/3 由 `update_topo` 写当前 `gateway_sn`。
- `DjiDeviceTopo` 始终保存本次上报的 `gateway_sn`，因此蛙跳场景可保留多个绑定；不能用主表 `device_sn` 唯一语义替代 topology pair。
- 网关自身使用 `update_topo` 顶层 `domain/type/sub_type` 更新主设备 `device_type/device_name`。
- OSD/State 先于 `update_topo` 创建主设备时显式写 `device_type/device_name=unknown`；后续合法拓扑覆盖哨兵，遥测更新不得把真实身份覆盖回 `unknown`。
- 合法但产品注册表未收录的三元组保存规范 `device_type` 和 `device_name=unknown`，不得阻止事务。
- 非法拓扑身份不得制造伪三元组或覆盖已有真实身份；新记录使用 `unknown`，已有记录保留原型号和名称。`gateway_sn` 独立按 domain 所有权更新，不与身份解析结果绑定。
- GaussDB 兼容模式可能把空字符串转换为 SQL `NULL`；设备身份列不得依赖空字符串表达未知状态，也不改用与 `NOT NULL` 冲突的 `sql.NullString`。

### 4. Validation & Error Matrix

| 条件 | 行为 |
| --- | --- |
| 已收录三元组 | 主表与拓扑保存规范型号和产品名称 |
| OSD/State 先于 topology | 主表保存 `unknown/unknown`，继续保存遥测和在线状态 |
| 合法但未收录三元组 | 保存规范型号，名称为 `unknown` |
| 非法 domain 或三元组，新记录 | 型号和名称保存 `unknown`，不制造伪三元组 |
| 非法 domain 或三元组，已有记录 | 保留已有型号和名称；非 domain 0/1 仍更新 `gateway_sn` |
| domain 0/1 子设备重复上报 | 更新型号和名称，不覆盖主表 `gateway_sn` |
| domain 2/3 子设备重复上报 | 更新型号、名称和主表 `gateway_sn` |
| 同一子设备由多个网关上报 | 主表仍只有一个 `device_sn` 记录；拓扑按 pair 保留多条绑定 |
| 软删除 topology pair 再上报 | 恢复 pair 并写入最新原始字段、型号和名称 |

### 5. Good/Base/Bad Cases

- Good: `sub.SN=drone-1, domain=0, type=60, sub_type=0` 更新主表为 `0-60-0/Matrice 300 RTK`，保留其 OSD/State `gateway_sn`，并写入当前网关 topology pair。
- Base: OSD 首次发现设备时保存 `unknown/unknown`；后续 `0-999-7` 同时写入主表和拓扑，`device_name=unknown`。
- Bad: 用空字符串表达未知身份、从 SN 猜型号、把非法输入保存为 `9-60-0`，或因 domain 0/1 不更新 `gateway_sn` 而连带跳过合法 `device_type/device_name`。

### 6. Tests Required

- Hook 测试断言 domain 0/1 保留主表 `gateway_sn`，但更新型号和名称；domain 2/3 同时更新三者。
- Hook 测试覆盖 OSD/State 首次创建、合法 topology 覆盖哨兵、已知产品、未知产品、非法身份保留、重复上报、软删除恢复和网关顶层型号。
- Model 测试断言两张表的身份列类型、非空和 `unknown` 默认值。
- Logic 测试断言 `DeviceInfo`、`DeviceTopoInfo` 原样返回持久化的 `unknown`；Proto 变更执行 `app/djicloud/gen.sh` 并检查生成 diff。

### 7. Wrong vs Correct

```go
// Wrong: 空字符串在 GaussDB 兼容模式下可能成为 NULL，且非法输入会制造伪三元组。
device := DjiDevice{DeviceType: "", DeviceName: ""}
deviceType := fmt.Sprintf("%s-%d-%d", domain, typ, subType)

// Correct: 新记录使用非空哨兵；合法拓扑才覆盖身份，网关独立按 domain 更新。
device := DjiDevice{
	DeviceType: DjiDeviceUnknown,
	DeviceName: DjiDeviceUnknown,
}
updateData := map[string]any{}
if parsed, err := djisdk.ParseDeviceType(rawDeviceType); err == nil {
	updateData["device_type"] = parsed.String()
	updateData["device_name"] = resolvedNameOrUnknown
}
if domain != "0" && domain != "1" {
	updateData["gateway_sn"] = gatewaySn
}
db.Where("device_sn = ?", sub.SN).Assign(updateData).FirstOrCreate(&device)
```

依据：`common/djisdk/device_type.go`、`app/djicloud/internal/hooks/sys_status_up.go`、`app/djicloud/model/gormmodel/dji_device.go`、`app/djicloud/djicloud.proto` 及对应测试。

## HMS 文案与设备型号契约

### 1. Scope / Trigger

- 修改 HMS `device_type`、`args`、文案字典、告警入库字段或 `ListHmsAlerts` 时适用。
- `common/djisdk` 拥有设备三元组、产品名称注册表、HMS key 和模板渲染；hook 只消费 SDK 结果并追加写入历史。

### 2. Signatures

- 设备型号格式固定为 `{domain}-{type}-{sub_type}`，domain 为 `0飞机/1负载/2遥控器/3机场`。
- 使用 `ParseDeviceType(string) (DeviceType, error)` 严格解析，使用 `LookupDeviceTypeName(string) (string, bool)` 查询官方中文产品名称。
- `gimbalindex` 不属于设备身份三元组；设备拓扑需要解释负载位置时使用显式宿主飞机型号的 `PayloadPlacement` / `PayloadGimbalIndex.Position(hostAircraftType)`，但 HMS 仅将 `gimbal_index` 作为文案数值参数。
- `HmsItem.Args` 使用开放 `HmsArgs map[string]any`；读取已知参数时先用 map lookup 确认 key 存在且值非 `nil`，再使用 `cast.ToIntE`、`cast.ToStringE` 等转换函数，hook 不自行断言动态类型。
- HMS handler 保持 `func(context.Context, string, *HmsEventData) error`；外层事件 `tid/bid` 由 SDK typed context 注入并通过 `EventCorrelationFromContext` 读取，trace 使用 `trace.TraceIDFromContext(ctx)`。

### 3. Contracts

- HMS key 仅有两类：domain 0 使用 `fpv_tip_{code}`，且 `in_the_sky=1` 时优先 `fpv_tip_{code}_in_the_sky`；domain 3 使用 `dock_tip_{code}`。
- domain 1/2 或非法 `device_type` 没有官方 HMS 前缀，不跨 `fpv`/`dock` 类别猜测同 code，不构造 `remote_tip_`。
- `Dji.Hms.Language` 空值规范化为 `zh`；非空语言只精确读取字典同名 key。该语言文案为空或缺失时返回该语言环境的未知告警，不静默切换语言。
- 官方 args 字段使用 `alarmid`；十六进制文本原样替换 `%alarmid`。`%index/%component_index` 使用索引加一，其中 component 不按当前产品数量截断；电池/舱盖为 0 左、其他右，充电杆 0/1/2/3 为前/后/左/右。
- `cast` 在 HMS 中仅负责转换已存在的参数值，不能代替字段存在性判断；例如 `cast.ToIntE(nil)` 可返回有效零值，若直接转换缺失的 `component_index` 会错误渲染为索引 1。`cast.ToIntE` 对浮点数采用宽松转换，是否接受小数截断必须由对应 HMS 参数契约决定。
- HMS 历史保存外层 `tid/bid/trace_id`、实际命中的 `message_key`、原始 `device_type`、产品中文名、三元组、解析 message 和完整 `item_json`；未知产品名称和未命中 Key 保存空字符串。
- `component_index/sensor_index/gimbal_index/lidar_index/lte_index` 不在 HMS model/RPC 平铺；原始值只保留在 `item_json.args`，resolver 仍按字典需要填充 message。`hms.json` 没有 `gimbal_position`，HMS hook 不查询宿主拓扑或派生位置。
- 当前 `HmsAlertInfo` 按未发布契约管理，字段按语义顺序连续编号；删除字段不保留 `reserved`。若该契约发布给外部消费者，后续变更必须重新按 Proto 兼容规则评估。

### 4. Validation & Error Matrix

| 条件 | 行为 |
| --- | --- |
| `device_type` 不是三段非负整数或 domain 不在 0..3 | 解析返回 error；HMS 返回未知告警，不猜 key |
| 三元组合法但产品注册表未收录 | 型号解析成功，产品名称查询返回 `"", false` |
| domain 1/2 | 不查 HMS 字典，返回未知告警 |
| 配置语言文案缺失/空值 | 保留配置 language，返回包含 code 的未知告警 |
| args key 缺失、值为 `nil` 或无法由对应 `cast.To*E` 转换 | 占位符保留并记录日志 |
| 模板参数缺失 | 不丢弃事件；保留占位符，继续入库 |
| context 无 `tid/bid` 或有效 trace | 对应持久化字段为空，不把三者相互替代 |
| `gimbal_index` 出现在 args | 仅填充 message 并保留 item JSON，不生成 `gimbal_position` |

### 5. Good/Base/Bad Cases

- Good: `0-67-0` 查询到 `Matrice 30`，使用 `fpv_tip_...`；`3-3-0` 查询到 `大疆机场 3`，使用 `dock_tip_...`。
- Base: 未收录但格式合法的三元组保留数值字段，`device_type_name` 为空；缺少关联 context 时告警仍追加写入。
- Bad: 把 domain 1 当遥控器、生成 `remote_tip_`、跨类别按 code 回退、配置 `fr` 时静默写入中文/英文文案，或从 HMS `gimbal_index` 猜宿主云台位置。

### 6. Tests Required

- 设备注册表覆盖飞机、负载、遥控器、机场当前官方三元组和中文展示名；至少断言 `0-67-0`、`1-83-0`、`2-174-0`、`3-3-0`。
- HMS 测试断言 domain 0/3 key、domain 1/2 无 key、空中 key 优先与普通 key 回退。
- 文案测试断言语言精确匹配、`alarmid` 原值、索引加一、component 不限产品数量、左右/前后方向和缺参保留。
- HMS 动态参数测试必须覆盖 key 缺失和显式 `nil`，确保两者不会经 `cast` 变成有效零值；若字段契约要求严格整数，再单独覆盖小数拒绝。
- SDK 分发测试断言 HMS handler 签名不变且 `tid/bid` 可从 context 读取。
- Hook/model/Logic 测试断言 `tid/bid/trace_id`、实际 `message_key`、message、device_type_name、三元组、`item_json` 与 RPC 字段一致，并断言不存在五个平铺索引和 `gimbal_position`。

### 7. Wrong vs Correct

```go
// Wrong: 通用 domain 被错误当成 HMS 前缀，并隐式切换语言。
prefix := map[int]string{0: "fpv", 1: "remote", 3: "dock"}[domain]
message := translations[preferred]
if message == "" {
	message = translations["zh"]
}

// Correct: HMS 只接受官方飞机/机场分类，语言只做精准读取。
deviceType, err := ParseDeviceType(raw)
if err != nil {
	return unknown
}
prefix := hmsTipPrefix(deviceType.Domain) // 仅 domain 0/3 非空
message := strings.TrimSpace(translations[language])

// Wrong: 从 HMS args 推断拓扑位置，并把协议关联 ID 当 trace。
position := PayloadGimbalIndex(args["gimbal_index"].(int)).Position(DeviceType{})
traceID := tid

// Correct: HMS 只保存原始 args 与解析文案，关联标识各自从 context 获取。
tid, bid := EventCorrelationFromContext(ctx)
traceID := trace.TraceIDFromContext(ctx)
message := resolver.Resolve(item).Message
```

依据：`common/djisdk/device_type.go`、`common/djisdk/hms.go`、`common/djisdk/hms.json`、`app/djicloud/internal/hooks/event_notify_up.go` 及对应测试。

## DRC 并发模型

- Manager map 由 `drcManager.mu` 保护，session 可变字段由 `drcDeviceSession.mu` 保护；统一按 manager -> session 的顺序取得，不允许反向。
- 取得 session 后的网络写入只持 session 所需锁，避免长期占用 manager 锁阻塞其他设备。
- 每次新会话分配新的 `sessionID`（UUID）；旧 heartbeat/cleanup goroutine 在修改状态前验证自己的 `sessionID` 与当前 session 一致。
- `heartbeatCancel()` 属于对应 session，只能由该 session 的替换 (`DeleteSession`) 或关闭 (`Disable`) 路径调用和清除。
- 清理采用锁内标记/快照、锁外关闭与回调，避免持锁 I/O 和 callback 重入死锁。
- `cleanupExpiredSessions` 在 heartbeat goroutine 内每秒执行：锁内遍历所有 session，收集过期项（`lastHb + timeout < now`），锁外调用 `DeleteSession`。
- `Enable` 支持 `max_deadline` 时间上限（`WithDrcMaxTimeout`），到期自动退场。
- `GetNextSeq` 原子增 1，`seq` 在 session 替换后重置为 1。
- DRC manager 的 lifecycle hook 通过 `WithDrcSessionEnabled` / `WithDrcSessionDisabled` / `WithDrcSessionExpired` 注册到 client option。

依据：`common/djisdk/drc.go:73-460`、`common/djisdk/client.go:1042-1078`、`common/djisdk/drc_test.go`。

## 飞行区与外部资源

- DJI 自定义飞行区 payload、坐标和文件字段复用 `djisdk.GeofenceFeature`、`djisdk.NewGeofencePolygonFeature`、`djisdk.NewGeofenceCircleFeature`、`djisdk.NewGeofenceFeatureCollection`；GIS 规则见 [gis-guidelines.md](./gis-guidelines.md)。
- 飞行区交互时序：Logic 将 proto 几何参数转为 DJI GeoJSON → 上传 OSS → 写入 `DjiDeviceFlyRegion` 记录 → `FlightAreasUpdate` 通知设备 → 设备经 `flight_areas_get` requests 拉取文件 → 云平台 `requests_reply` 返回 OSS URL。
- OSS 上传失败、数据库写入失败和 `FlightAreasUpdate` 通知失败都是平台阶段错误，使用 gRPC error（`tool.NewErrorByPbCode*`）。`FlightAreasUpdate` 等待设备 ACK 后的 DJI rejection 保留在响应的 `tid/reason_code` 中。
- 删除按现有 fly region record ID 所有权执行；`DeleteCustomFlyRegionByFileId` 按 fileId 精确删除。不把部分成功报告为全部完成。
- 飞行区同步进度（`flight_areas_sync_progress`）和告警（`flight_areas_drone_location`）由 events hook 处理。
- 外部设备返回原始结构在 SDK 边界转换，业务 model 不直接绑定所有上游字段。

## DJI RPC 错误边界

### 1. Scope / Trigger

- 修改 `app/djicloud/djicloud.proto`、返回 `CommonRes` 的 Logic、`drc/down` 即发即忘接口或平台组合接口时适用。
- 目标是区分“设备已 ACK 但拒绝执行”和“平台调用链失败”，避免将数据库、MQTT、超时等错误伪装为正常 gRPC 响应。

### 2. Signatures

- 等待设备 ACK 的命令返回 `(*djicloud.CommonRes, error)`。
- `CommonRes` 固定包含 `code/message/tid/reason_code`；平台专用响应按 `<RpcName>Res` 命名。
- `AckHmsAlert` 返回 `(*djicloud.AckHmsAlertRes, error)`；成功响应为空，失败返回 extproto error。
- `commandRes(tid string, err error) (*djicloud.CommonRes, error)` 仅把 `djisdk.DJIError` 转为 `CommonRes`。

### 3. Contracts

- DJI 设备 ACK 的 `result != 0`：返回 `CommonRes{Code:-1, Message, Tid, ReasonCode}` 和 `nil` error。
- 设备 ACK 成功：返回 `CommonRes{Code:0, Message:"success", Tid}`。
- 参数、数据库、OSS、配置和 DRC 序列分配失败：返回 `tool.NewErrorByPbCode*`，响应为 `nil`。
- 等待 DJI ACK 的 SDK 调用若返回非 `DJIError`（如设备离线、MQTT 发布或等待超时），直接返回 SDK 原始 error，保留其具体错误信息。
- `drc/down` 接口无设备 ACK；成功响应只表达已分配/发布的 `seq`，不得使用 `CommonRes` 推断设备执行成功。
- 自定义飞行区写接口的平台阶段失败走 extproto error；末尾 `FlightAreasUpdate` 的 typed DJI 拒绝保留在专用响应的 `tid/reason_code`。

### 4. Validation & Error Matrix

| 条件 | 返回行为 |
| --- | --- |
| `djisdk.IsDJIError(err)` 成功 | `code=-1`、原始 `reason_code`、对应 `tid`，gRPC error 为 `nil` |
| 等待 ACK 命令的 MQTT/request-reply/超时/离线错误 | 直接返回 SDK error，响应为 `nil` |
| 平台参数或 JSON 非法 | `_1_01_PARAM_INVALID` / `_1_01_PARAM_MISSING` |
| 平台数据库失败 | `_1_02_DB` |
| 平台记录不存在 | `_1_02_RECORD_NOT_EXIST` |
| OSS 未配置或上传/签名失败 | `_1_00_INTERNAL` 或 `_1_06_THIRD_PARTY` |
| DRC fire-and-forget 发布成功 | 返回对应 `*Res{Seq}`，不等待设备结果 |

### 5. Good/Base/Bad Cases

- Good: 设备回复 DJI 错误码时，调用方收到 `tid/reason_code`，可按 DJI 业务原因处理。
- Base: 平台查询无记录时按该 RPC 契约返回空数据或 `_1_02_RECORD_NOT_EXIST`，不构造 `CommonRes`。
- Bad: MQTT 超时、设备离线或数据库错误返回 `CommonRes{Code:-1, Message:err.Error()}`，导致调用方误判为设备业务拒绝。

### 6. Tests Required

- Helper 单测断言 typed `DJIError` 产生 `CommonRes`，并保留 `tid/reason_code`。
- Helper 单测断言普通 error 原样返回，且 `errors.Is` 仍能识别原 cause。
- 平台 Logic 测试覆盖数据库错误、记录不存在和空成功响应。
- DRC fire-and-forget 测试覆盖序列分配失败与 publish 失败，断言响应为 `nil` 且错误为 `_1_06_THIRD_PARTY`。
- Proto 变更执行 `app/djicloud/gen.sh`，并运行 `go test ./app/djicloud/...`。

### 7. Wrong vs Correct

```go
// Wrong: 将所有 SDK 错误都包装成设备业务结果。
if err != nil {
	return &djicloud.CommonRes{Code: -1, Message: err.Error(), Tid: tid}, nil
}

// Correct: 只有 typed DJI ACK 错误进入响应，其余错误走 gRPC error。
if err != nil {
	return commandRes(tid, err)
}
```

## 反模式

- 在 hook 中硬编码 topic/method 或用 `map[string]any` 绕过 typed payload。
- 把事件历史做 Upsert，或把设备快照每次 Insert 成历史。
- manager/session 锁顺序不一致，或持锁执行 callback/MQTT/数据库。
- 旧 session goroutine 关闭新 session 的 heartbeat。
- publish 成功即更新为设备命令成功。

## 验证

- SDK 测试覆盖 topic、handler 路由、快速回复、设备错误、超时和 fire-and-forget 差异。
- Model/hook 测试分别断言快照更新、事件追加、在线状态和重复/迟到消息。
- DRC 修改运行 `go test -race ./common/djisdk -count=10`，覆盖替换、心跳、清理和回调重入。
