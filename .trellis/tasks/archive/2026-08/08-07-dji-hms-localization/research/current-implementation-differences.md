# Research: Current DJI HMS Implementation Differences

- **Query**: 对照 DJI 官方 HMS 和产品枚举，定位当前实现差异，给出符号级对齐项和测试矩阵。
- **Scope**: internal
- **Date**: 2026-08-07

## Findings

### Concrete Differences

| Location / symbol | Current behavior | Official evidence / observable impact | Alignment action |
|---|---|---|---|
| `common/djisdk/device_type.go:33-35`, `ParseDeviceType` | legacy `remote`/`rc` 被赋值 domain 1 | 官方 domain 1=负载，domain 2=遥控器 | legacy remote 如保留，应映射到 2；更稳妥是将 legacy 兼容与官方 parser 分开 |
| `common/djisdk/device_type.go:45-51`, `DeviceType.TipPrefix` | domain 1 -> `remote`，没有 domain 2 case | 官方 HMS 只定义 domain 0 飞机 `fpv`、domain 3 机场 `dock`；官方 JSON 无 `remote_tip_` | HMS 前缀函数仅对 0/3 返回 `fpv`/`dock`；1/2 返回无官方映射 |
| `common/djisdk/hms.go:22`, `hmsTipPrefixes` | 候选含 `remote` | 本地/官方 JSON 中 `remote_tip_` 为 0 个 | 从官方候选集合排除 `remote` |
| `common/djisdk/hms.go:118-129`, `orderedHmsPrefixes` | 未识别 domain 时跨 `dock/fpv/remote` 回退；已识别 domain 也跨设备类别回退 | 官方规则按告警设备类别拼接 key，没有跨类别 fallback 规则 | 按 0/3 精确选类；未知/1/2 不凭相同 code 猜类别。若产品需求坚持 fallback，应明确标记为非官方启发式并测试歧义 |
| `common/djisdk/protocol.go:1147`, `HmsItem.Args` | 使用 `map[string]any` | `.trellis/spec/backend/dji-guidelines.md:9,43` 要求 typed payload；未知字段保留需求与类型化可通过自定义 unmarshal 兼容 | 建模已知 `component_index`、`sensor_index`、`alarmid`、`gimbal_index`、`lidar_index`、`lte_index`，需要前向兼容时额外保留 raw/unknown fields |
| `common/djisdk/hms.go:187`, `renderHmsTemplate` | 从 `alarm_id` 读取 | 官方 HMS 功能页字段写 `alarmid` | 读取 `alarmid`；是否临时兼容 `alarm_id` 应显式处理，不能让非官方名优先 |
| `common/djisdk/hms.go:188` | 把 alarm ID 当整数并格式化 `0x%08X` | 官方示例是 args 中具体十六进制 `alarmid`，只要求替换，不要求数值重编码；字符串值会被当前 `HmsArgInt` 的十进制解析拒绝 | alarmid 类型应能无损承载官方十六进制文本；直接替换规范化后的官方值，覆盖大小写/长度/非法值测试 |
| `common/djisdk/hms.go:210-217`, `localizedSide` | 0=左、1=右，其他输出 index+1 | 官方 `%battery_index`/`%dock_cover_index` 为 0 左、否则右 | 对这些占位符按官方“否则右”规则；不要把 2/3 渲染成数字 3/4 |
| `common/djisdk/hms.go:220-228`, `localizedDirection` | 只支持 0=前、1=后；2/3 输出 3/4 | 官方 `%charging_rod_index`: 0前、1后、2左、3右 | 补齐四向映射；越界保留占位符或按明确未知策略处理 |
| `common/djisdk/hms.go:169-172` | component_index 无范围处理，直接 +1 | 官方文案写“最终范围限定在1和2之间”，但又说支持 1/2/3 号云台，官方原文矛盾 | 不静默截断；至少测试 0/1/2 和越界，记录官方矛盾，保留上报事实 |
| `common/djisdk/hms.go:19-20` | replacement regex 支持部分位置参数；unresolved regex 会把占位符后的字母一起吞入 | 官方 JSON 出现 `%1$sm`、`%lidar_indexAPD` 等相邻文本形态 | 用已知 token 的最长精确匹配与位置格式语法测试，避免误报/漏报 |
| `common/djisdk/hms.go:179-185` | 所有 dock `%s/%1$s/%d/%1$d` 都由同一个 sensor_index 替换 | 官方页面只定义命名占位符语义；JSON 中还有 `%2$s` 等，当前实现未支持 | 位置参数须逐类由有证据的模板/参数映射驱动；无法确认时保留原文，不作全局推断 |
| `common/djisdk/hms_test.go:54,64-68` | 人造并断言 `remote_tip_` 和 domain 1 remote fallback | 当前官方 JSON 和 HMS 页均无 remote 前缀 | 改为断言 domain 1/2 不选择 remote；增加 domain 0/3 官方行为 |
| `common/djisdk/hms_test.go:103` | 测试 payload 使用 `alarm_id` | 官方字段为 `alarmid` | fixture 改为官方拼写，并加 legacy/invalid 明确策略测试 |
| `.trellis/tasks/08-07-dji-hms-localization/prd.md:37` | 验收标准写 domain 1 -> `remote_tip_` | 官方 domain 与 HMS key 规则冲突 | 主代理应先更新任务需求/设计，再实现，避免代码继续服从错误验收项 |
| `.trellis/tasks/08-07-dji-hms-localization/design.md:22,32-33` | 设计重复 domain 1 remote、charging rod 仅前后、alarm 字段为 `alarm_id` | 均与当前官方证据冲突 | 设计需按研究结论修订 |

### Existing Correct Foundations

- `app/djicloud/model/gormmodel/dji_device.go:10-15` 的 domain 常量与官方一致。
- `common/djisdk/protocol.go:971,993` 的 domain 注释与官方一致。
- `common/djisdk/hms.go:106-113` 的 `_in_the_sky` 优先再回普通 key，符合飞行器规则。
- `common/djisdk/hms.go:169-176` 的 `%component_index`、`%index` 加一符合官方填充规则。
- `common/djisdk/hms.go:210-217` 对 index 0/1 的左右映射正确，只是 2+ 的行为不符合“否则右”。
- `app/djicloud/internal/hooks/event_notify_up.go:164-191` 已保存原始 `device_type`、解析三元组、message 与 `item_json`；修正 SDK 解析后可沿用该数据流。

### Test Matrix

| Area | Inputs | Required assertions |
|---|---|---|
| Official device parsing | `0-103-0`, `1-83-0`, `2-174-0`, `3-3-0` | domain/type/sub_type 精确；名称分别命中 Matrice 400、H30T、RC Plus 2、机场 3 |
| Strict parsing | 空串、`dock`、`0-67`、`0-x-1`、四段字符串、负数 | 官方 parser 返回错误；legacy parser 行为独立可见 |
| HMS category | domain 0、3 | 只查 `fpv_tip_`、`dock_tip_` |
| Unsupported HMS category | domain 1、2、未知、非法 | 不生成 `remote_tip_`；不跨类别误命中相同 code |
| Flight state | same fpv code, `in_the_sky=0/1`, sky variant present/absent | 0 普通；1 优先 sky；sky key 不存在时普通 |
| Dictionary inventory | embedded official JSON | remote=0、dock=258、fpv=3584；可用哈希作为资源更新提示而非永久协议断言 |
| Alarm ID | `alarmid="0x16100001"`、大小写变体、缺失、非法、legacy `alarm_id` | 无损替换；缺失/非法策略明确；官方名优先 |
| Component | `component_index=0,1,2,-1,3` | 0/1/2 的 +1；官方矛盾边界不被静默截断 |
| Sensor ordinal | `sensor_index=0,1,2` with `%index` | 1、2、3 |
| Battery/cover side | `sensor_index=0,1,2,3` | 左、右、右、右（中文；英文对应 left/right） |
| Charging rod | `sensor_index=0,1,2,3,4,-1` | 前、后、左、右；越界执行明确未知策略 |
| Other named args | gimbal/lidar/lte present, missing, numeric/string JSON | 已知值替换；缺失保留模板并可观测告警 |
| Positional placeholders | `%s`, `%d`, `%1$s`, `%1$d`, `%2$s`, `%1$.1f` | 只替换有已确认参数来源的格式；其他保留且不破坏相邻文本 |
| Adjacent text | `%1$sm`, `%lidar_indexAPD` | token 边界处理符合实际模板，不把后缀误作变量名 |
| Registry duplicates | `(1,39,0)`, `(1,165,0)` | 返回共用型号定义；不伪造宿主机型/机场代际；placement/context 单独查询 |
| Hook persistence | official HMS fixture with `alarmid`, domain 0/3 | message、raw item JSON、三元组及 args 派生字段持久化一致 |

## Files Found

| File Path | Description |
|---|---|
| `common/djisdk/device_type.go` | 当前三元组解析与 HMS 前缀映射 |
| `common/djisdk/hms.go` | 当前 key 查找、语言选择和模板渲染 |
| `common/djisdk/hms_test.go` | 当前 remote/domain/alarm_id 行为测试 |
| `common/djisdk/protocol.go:1130` | HMS 上报结构 |
| `app/djicloud/internal/hooks/event_notify_up.go:156` | HMS 入库数据流 |
| `app/djicloud/model/gormmodel/dji_device.go:10` | 正确的 domain 常量基线 |
| `app/djicloud/model/gormmodel/dji_event.go:16` | HMS 持久化字段 |

## Related Specs

- `.trellis/spec/backend/dji-guidelines.md:7` — SDK 协议所有权和 typed payload 约束。

## Caveats / Not Found

- 官方 API 表没有完整解释 `gimbal_index`、`lidar_index`、`lte_index` 的 args 类型/范围；当前只能确认官方 JSON 需要同名占位符，测试不应虚构范围。
- 官方未说明跨 `dock`/`fpv` code fallback。保留该行为属于产品决策，不应标注为 DJI 官方规则。
