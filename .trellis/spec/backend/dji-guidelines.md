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

## HMS 文案与设备型号契约

### 1. Scope / Trigger

- 修改 HMS `device_type`、`args`、文案字典、告警入库字段或 `ListHmsAlerts` 时适用。
- `common/djisdk` 拥有设备三元组、产品名称注册表、HMS key 和模板渲染；hook 只消费 SDK 结果并追加写入历史。

### 2. Signatures

- 设备型号格式固定为 `{domain}-{type}-{sub_type}`，domain 为 `0飞机/1负载/2遥控器/3机场`。
- 使用 `ParseDeviceType(string) (DeviceType, error)` 严格解析，使用 `LookupDeviceTypeName(string) (string, bool)` 查询官方中文产品名称。
- `gimbalindex` 不属于设备身份三元组；负载位置通过 `PayloadPlacement` / `PayloadGimbalIndex.Position()` 单独描述。
- `HmsItem.Args` 使用开放 `HmsArgs map[string]any`，读取已知参数时调用 `Args.Int(name)` 或 `Args.String(name)`，hook 不自行断言动态类型。

### 3. Contracts

- HMS key 仅有两类：domain 0 使用 `fpv_tip_{code}`，且 `in_the_sky=1` 时优先 `fpv_tip_{code}_in_the_sky`；domain 3 使用 `dock_tip_{code}`。
- domain 1/2 或非法 `device_type` 没有官方 HMS 前缀，不跨 `fpv`/`dock` 类别猜测同 code，不构造 `remote_tip_`。
- `Dji.Hms.Language` 空值规范化为 `zh`；非空语言只精确读取字典同名 key。该语言文案为空或缺失时返回该语言环境的未知告警，不静默切换语言。
- 官方 args 字段使用 `alarmid`；十六进制文本原样替换 `%alarmid`。`%index/%component_index` 使用索引加一，电池/舱盖为 0 左、其他右，充电杆 0/1/2/3 为前/后/左/右。
- HMS 历史同时保存原始 `device_type`、产品中文名、三元组、已知 args 派生字段、解析 message 和完整 `item_json`；未知产品名称保存空字符串。

### 4. Validation & Error Matrix

| 条件 | 行为 |
| --- | --- |
| `device_type` 不是三段非负整数或 domain 不在 0..3 | 解析返回 error；HMS 返回未知告警，不猜 key |
| 三元组合法但产品注册表未收录 | 型号解析成功，产品名称查询返回 `"", false` |
| domain 1/2 | 不查 HMS 字典，返回未知告警 |
| 配置语言文案缺失/空值 | 保留配置 language，返回包含 code 的未知告警 |
| args 整数是小数、溢出、NaN/Inf 或不支持类型 | `Args.Int` 返回 `false`，占位符保留并记录日志 |
| 模板参数缺失 | 不丢弃事件；保留占位符，继续入库 |

### 5. Good/Base/Bad Cases

- Good: `0-67-0` 查询到 `Matrice 30`，使用 `fpv_tip_...`；`3-3-0` 查询到 `大疆机场 3`，使用 `dock_tip_...`。
- Base: 未收录但格式合法的三元组保留数值字段，`device_type_name` 为空。
- Bad: 把 domain 1 当遥控器、生成 `remote_tip_`、跨类别按 code 回退，或配置 `fr` 时静默写入中文/英文文案。

### 6. Tests Required

- 设备注册表覆盖飞机、负载、遥控器、机场当前官方三元组和中文展示名；至少断言 `0-67-0`、`1-83-0`、`2-174-0`、`3-3-0`。
- HMS 测试断言 domain 0/3 key、domain 1/2 无 key、空中 key 优先与普通 key 回退。
- 文案测试断言语言精确匹配、`alarmid` 原值、索引加一、左右/前后方向和缺参保留。
- Hook/model/Logic 测试断言 message、device_type_name、三元组、args 派生字段、`item_json` 与 RPC 字段一致。

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
