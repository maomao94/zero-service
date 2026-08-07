# 深化 DJI HMS 本地化与告警持久化

## 目标

将 DJI HMS 原始告警码转换为配置语言下可直接展示的告警文案，并在告警上报时持久化该文案，使查询接口不再要求调用方自行解析 `item_json`。

## 背景

- `common/djisdk/hms.json` 已包含 HMS 多语言文案，但当前代码未加载或使用该字典。
- `common/djisdk/protocol.go` 的 HMS args 需要保留开放字段并由 SDK 提供安全读取能力。
- `app/djicloud/internal/hooks/event_notify_up.go` 当前逐条保存 HMS 事件字段和 `item_json`，不保存可展示文案。
- `app/djicloud/model/gormmodel/dji_event.go` 与 `app/djicloud/djicloud.proto` 当前均无 HMS `message` 字段。

## 需求

1. `common/djisdk` 提供 HMS 文案解析能力，启动时将 `hms.json` 反序列化到内存。
2. HMS 语言可通过 `Dji.Hms.Language` 配置，默认使用 `zh`。
3. SDK 严格解析 `device_type` 的 `{domain}-{type}-{sub_type}` 固定格式，并提供产品注册表查询对应的大疆产品名称。
4. 产品注册表覆盖当前产品支持文档中的飞机、负载、遥控器和机场枚举；负载挂载位置 `gimbalindex` 与设备三元组分开建模。
5. HMS 只按官方规则选择 key：domain 0 使用 `fpv_tip_`，domain 3 使用 `dock_tip_`；不为 domain 1/2 构造未定义前缀，也不跨设备类别猜测同 code。
6. 支持普通 key 和飞行器 `_in_the_sky` key。
7. `HmsItem.Args` 使用 `map[string]any` 保留未来字段，SDK 提供安全参数读取方法，hook 不自行做类型断言。
8. 解析器按官方规则替换 `alarmid`、索引和方向占位符；缺失参数不得导致事件处理失败，未解析占位符保留原文并记录告警日志。
9. HMS 上报 hook 同时保存 `message`、`device_type_name`、设备三元组和已知 args 派生字段，并继续保留 `item_json` 原文。
10. `ListHmsAlerts` 返回持久化的 HMS message、device_type_name 和平铺字段。
11. Proto 生成物必须通过 `app/djicloud/gen.sh` 生成，不手工编辑。

## 范围外

- 不根据飞行器固件或产品型号选择 `_pm440`、`_index_1`、`_n_mode` 等专用字典变体。
- 不回填历史 HMS 记录的 message。
- 不改变 HMS 告警追加写入、确认状态或查询过滤语义。
- 不移除 `item_json` 字段。

## 验收标准

- [ ] 配置未指定语言时，已知 HMS code 能生成非空中文 message。
- [ ] 文案按配置语言精确匹配；该语言文案为空或缺失时返回该语言环境下的未知告警，不静默切换语言。
- [ ] `0-67-0` 可解析为 Matrice 30，当前官方设备枚举可通过三元组查询产品名称。
- [ ] `device_type=3-3-0` 选择 `dock_tip_`，domain 0 选择 `fpv_tip_`；domain 1/2 不构造 HMS 前缀。
- [ ] `in_the_sky=1` 时优先匹配 `_in_the_sky`，不存在时回退普通键。
- [ ] 官方 `alarmid` 和索引占位符被正确替换，充电杆索引 0/1/2/3 分别为前/后/左/右；缺失参数不会阻止告警入库。
- [ ] HMS 新记录同时包含原始 `item_json`、`message`、`device_type_name` 和设备三元组字段。
- [ ] `ListHmsAlerts` 返回新增 HMS 字段，旧客户端未读取新增字段时保持兼容。
- [ ] HMS 解析单元测试、hook/model 测试和 `app/djicloud` 目标测试通过。
- [ ] `app/djicloud/gen.sh`、`gofmt` 和 `git diff --check` 通过。
