# DJI Cloud 与 DRC 规范

## 适用范围

修改 `common/djisdk`、`app/djicloud`、DJI Cloud MQTT topic/method/payload、设备在线状态、OSD/State/Event 持久化、DRC 或飞行区同步时读取。

## SDK 与协议所有权

- `common/djisdk` 的协议包拥有 topic 构造、method、typed payload 和结果解析；`app/djicloud` Logic/hook 不手工拼 topic 或通用 map。
- client option 只写构造配置；`NewClient` 负责连接、handler、reply router 和运行态初始化。
- `SendCommand` 表达需要设备响应的 request/reply；fire-and-forget publish 是另一种语义，不能因返回 error 形式相似而互换。
- 请求应答使用 TID/BID 等协议关联标识，先注册后发送，区分 broker publish、设备回复和 reply result。

依据：`common/djisdk/client.go`、`common/djisdk/option.go`、`common/djisdk/handler.go`、`common/djisdk` 协议目录与测试。

## 上行处理与数据所有权

- `app/djicloud` 的 event、telemetry、DRC、request hooks 按 DJI 上行类型分工；新增 method 放到对应注册边界，不建立第二套总分发器。
- 设备/拓扑/最新状态属于快照，使用查询后更新或现有 `FirstOrCreate` + `Assign` 模式；事件/告警历史使用追加 `Create`，不得 Upsert 覆盖历史。
- 在线语义由当前设备/topology/heartbeat 处理链路拥有；单条旧消息或迟到 callback 不能覆盖更新 session 的在线状态。
- Store/model 决定事务和更新字段。不要从 hook 整行 `Save`，也不要以方言专属 Upsert SQL 替换已经测试的写入模式。

依据：`app/djicloud/internal/hooks`、`app/djicloud/model/gormmodel` 及其测试。

## 设备主表与拓扑型号契约

### 1. Scope / Trigger

- 修改 `update_topo`、`DjiDevice`、`DjiDeviceTopo`、`DeviceInfo` 或 `DeviceTopoInfo` 的设备型号、产品名称和网关归属时适用。
- `common/djisdk` 拥有 `{domain}-{type}-{sub_type}` 格式与产品名称注册表；DJI hook 只负责从 typed topology payload 派生并持久化。

### 2. Signatures

- `DjiDevice` 以 `device_sn` 唯一，保存设备最近快照；`DeviceType`、`DeviceName` 紧跟 `GatewaySn`。
- `DjiDeviceTopo` 以 `gateway_sn + sub_device_sn` 唯一，允许同一 `sub_device_sn` 出现在多个网关绑定中。
- 型号使用 `ParseDeviceType(string)` 规范化，名称使用 `LookupDeviceTypeName(string)` 查询；格式固定为 `{domain}-{type}-{sub_type}`。
- 未发布的 `DeviceInfo` 按语义顺序连续编号：`gateway_sn` 后为 `device_type`、`device_name`，不为旧编号增加 `reserved`。

### 3. Contracts

- 每次 `update_topo` 都同时更新拓扑记录和按 `device_sn = sub.SN` 定位的主设备记录中的 `device_type/device_name`。
- domain 0/1 的主设备 `gateway_sn` 不由 `update_topo` 覆盖，继续由 OSD/State 表达当前通信网关；domain 2/3 由 `update_topo` 写当前 `gateway_sn`。
- `DjiDeviceTopo` 始终保存本次上报的 `gateway_sn`，因此蛙跳场景可保留多个绑定；不能用主表 `device_sn` 唯一语义替代 topology pair。
- 网关自身使用 `update_topo` 顶层 `domain/type/sub_type` 更新主设备 `device_type/device_name`。
- 合法但产品注册表未收录的三元组保存规范 `device_type` 和空 `device_name`，不得阻止事务。

### 4. Validation & Error Matrix

| 条件 | 行为 |
| --- | --- |
| 已收录三元组 | 主表与拓扑保存规范型号和产品名称 |
| 合法但未收录三元组 | 保存规范型号，名称为空 |
| 非法 domain 或三元组 | 保留上游组合字符串，名称为空，不中断拓扑快照入库 |
| domain 0/1 子设备重复上报 | 更新型号和名称，不覆盖主表 `gateway_sn` |
| domain 2/3 子设备重复上报 | 更新型号、名称和主表 `gateway_sn` |
| 同一子设备由多个网关上报 | 主表仍只有一个 `device_sn` 记录；拓扑按 pair 保留多条绑定 |
| 软删除 topology pair 再上报 | 恢复 pair 并写入最新原始字段、型号和名称 |

### 5. Good/Base/Bad Cases

- Good: `sub.SN=drone-1, domain=0, type=60, sub_type=0` 更新主表为 `0-60-0/Matrice 300 RTK`，保留其 OSD/State `gateway_sn`，并写入当前网关 topology pair。
- Base: `0-999-7` 同时写入主表和拓扑，`device_name` 为空。
- Bad: 因 domain 0/1 不更新 `gateway_sn` 而连带跳过 `device_type/device_name`；或用 `sub_device_sn` 单列唯一覆盖其他网关的 topology pair。

### 6. Tests Required

- Hook 测试断言 domain 0/1 保留主表 `gateway_sn`，但更新型号和名称；domain 2/3 同时更新三者。
- Hook 测试覆盖已知产品、未知产品、重复上报、软删除恢复和网关顶层型号。
- Model 测试断言两张表的列类型、非空和空字符串默认值。
- Logic 测试断言 `DeviceInfo`、`DeviceTopoInfo` 返回持久化值；Proto 变更执行 `app/djicloud/gen.sh` 并检查生成 diff。

### 7. Wrong vs Correct

```go
// Wrong: domain 0/1 连型号字段也不更新。
updateData := map[string]any{}
if domain != "0" && domain != "1" {
	updateData["gateway_sn"] = gatewaySn
}

// Correct: 型号始终按设备 SN 更新，仅 GatewaySn 保留 domain 所有权差异。
updateData := map[string]any{
	"device_type": deviceType,
	"device_name": deviceName,
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

- Manager map 由 manager 锁保护，session 可变字段由 session 锁保护；统一按 manager -> session 的顺序取得，不允许反向。
- 取得 session 后的网络写入只持 session 所需锁，避免长期占用 manager 锁阻塞其他设备。
- 每次新会话分配新的 session ID/version；旧 read/heartbeat/cleanup goroutine 在修改状态前验证自己仍是当前 session。
- `heartbeatCancel` 属于对应 session，只能由该 session 的替换/关闭路径调用和清除。
- 清理采用锁内标记/快照、锁外关闭与回调，避免持锁 I/O 和 callback 重入死锁。

依据：`common/djisdk/drc.go`、`common/djisdk/protocol_drc_test.go`。

## 飞行区与外部资源

- DJI 自定义飞行区 payload、坐标和文件字段复用现有 protocol/model helper；GIS 规则见 [gis-guidelines.md](./gis-guidelines.md)。
- OSS 上传、设备同步和数据库记录是多个阶段。失败时保留可重试状态，删除按现有 device/file ID 所有权执行，不把部分成功报告为全部完成。
- 外部设备返回原始结构在 SDK 边界转换，业务 model 不直接绑定所有上游字段。

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
