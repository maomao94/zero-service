# Research: DJI HMS Key Selection and Args

- **Query**: 核实 HMS 文案 key 选择规则、`remote_tip_` 资源、HMS args 字段和填充规则。
- **Scope**: mixed
- **Date**: 2026-08-07

## Evidence Levels

- **A（官方直接证据）**：DJI 当前官方网页静态构建产物或网页直接链接的官方 `hms.json`。
- **B（官方资源统计）**：对官方 `hms.json` 的机器统计；可证明资源现状，不能单独扩展协议语义。
- **C（本地实现事实）**：仓库文件和测试的当前行为。
- **D（未证实）**：当前官方资料未给出足够依据，明确不作推断。

## Findings

### Official Key Selection

官方 HMS 功能页只定义两类 key（证据 A）：

1. 机场告警：`dock_tip_{code}`。
2. 飞行器告警：地面使用 `fpv_tip_{code}`；`in_the_sky=1` 时，空中使用 `fpv_tip_{code}_in_the_sky`，地面仍使用普通 key。

页面同时说明，`device_type` 格式为 `{domain-type-subtype}`，产品支持页定义 domain：`0=飞机`、`1=负载`、`2=遥控器`、`3=机场`。然而 HMS 页描述的上报内容和 key 选择只有“机场设备与飞行器”，没有为负载或遥控器定义独立 key（证据 A）。

因此 HMS key 选择不能采用通用设备 domain 到前缀的一一映射：

| HMS `device_type` domain | 官方产品域 | 官方 HMS key 规则 |
|---:|---|---|
| 0 | 飞机 | `fpv_tip_...` |
| 3 | 机场 | `dock_tip_...` |
| 1 | 负载 | HMS 页未定义独立前缀 |
| 2 | 遥控器 | HMS 页未定义独立前缀 |

对 domain 1/2 是否可能出现在未来 HMS 上报中，当前页面没有作绝对禁止；但即使出现，也没有官方证据支持 `remote_tip_` 或其他新前缀（证据 D）。当前规则下只应根据官方已定义的飞机/机场类别选择 key；不能把 domain 1 当遥控器。

### `remote_tip_` Audit

2026-08-07 下载 DJI HMS 页直接链接的官方 JSON，并与 `common/djisdk/hms.json` 比较：

```text
local SHA-256  = 283e2ad9151d36ba5f7491f00138c503661c9985958beecbadfcf98b3190baaa
remote SHA-256 = 283e2ad9151d36ba5f7491f00138c503661c9985958beecbadfcf98b3190baaa
```

两份文件逐字节一致。键统计（证据 B）：

| Prefix | Key count | Distinct hexadecimal code count | Lexical/numeric span |
|---|---:|---:|---|
| `dock_tip_` | 258 | 258 | `0x1910F003` to `0x19117481`（不代表连续区间） |
| `fpv_tip_` | 3,584 | 3,162 | `0x1A0100C0` to `0x26130808`（含同 code 的后缀变体） |
| `remote_tip_` | **0** | **0** | 不存在 |
| Total | 3,842 | - | - |

所以 `remote_tip_` 没有“实际 code 范围”和“文案内容”；它不是当前官方资源的一部分。官方 HMS 页面也未定义该前缀。能下的最高证据结论是：**当前 HMS 解析应忽略 `remote_tip_`**。无法证明它是“历史遗留”或“其他端资源”，因为当前仓库和官方资源里均无此类键（证据 D）。

### HMS Args and Rendering

官方 Dock 1/2/3 HMS API 页公开的 item 字段为：`level`、`module`、`in_the_sky`、`code`、`device_type`、`imminent`、`args`；Dock 1/2 的 `args` 明列 `component_index`、`sensor_index`，Dock 3 页面也列出这两个字段（证据 A）。

官方 HMS 功能页进一步明确 `args` 提供填充值，并点名 `alarmid`、`sensor_index`、`component_index`（证据 A）。注意协议/文档拼写是 **`alarmid`**，不是 `alarm_id`。

| Args field | Purpose / rendering | Evidence |
|---|---|---|
| `alarmid` | 替换 `%alarmid`，内容是具体十六进制告警 ID，例如 `0x16100001`；文档没有要求由整数重编码或补齐 8 位 | A |
| `sensor_index` | `%index` = `sensor_index + 1`；决定 `%battery_index`、`%dock_cover_index`、`%charging_rod_index` | A |
| `component_index` | `%component_index` = `component_index + 1`；官方原文称最终范围限定在 1 和 2 之间，但括号又称最多支持 1/2/3 号云台，原文自身存在矛盾，应以真实上报保留值并单测边界 | A |
| `gimbal_index` | 当前官方 JSON 包含 `%gimbal_index` 模板 | B；API 表未明列，不能声称协议页已完整定义 |
| `lidar_index` | 当前官方 JSON 包含 `%lidar_index` 模板 | B；同上 |
| `lte_index` | 当前官方 JSON 包含 `%lte_index` 模板 | B；同上 |

`sensor_index` 的官方映射（证据 A）：

| Placeholder | `sensor_index=0` | `1` | `2` | `3` |
|---|---|---|---|---|
| `%battery_index` | 左 | 右（官方规则是 0 左，否则右） | 右 | 右 |
| `%dock_cover_index` | 左 | 右（官方规则是 0 左，否则右） | 右 | 右 |
| `%charging_rod_index` | 前 | 后 | 左 | 右 |

`charging_rod_index` 是模板占位符，不是官方列出的 args 字段；其值由 `sensor_index` 映射。

### Dictionary Placeholder Reality

当前官方 JSON 的命名占位符统计包括 `%alarmid` 176、`%battery_index` 1004、`%component_index` 216、`%dock_cover_index` 4、`%gimbal_index` 9、`%index` 728、`%lidar_index` 105、`%lte_index` 45，以及位置占位符 `%s`、`%d`、`%1$s`、`%1$d`、`%2$s` 等（证据 B）。

JSON 还存在诸如 `%lidar_indexAPD`、`%1$sm` 等相邻文本造成的词法形态。通用“未解析占位符”正则若贪婪读取字母，会把合法占位符与后续文本合并，需在测试中覆盖。

## Files Found

| File Path | Description |
|---|---|
| `common/djisdk/hms.json` | 与当前 DJI 官方下载文件哈希一致的 HMS 文案资源 |
| `common/djisdk/device_type.go:41` | 当前按 domain 返回 `fpv`/`remote`/`dock` 前缀 |
| `common/djisdk/hms.go:19` | 当前占位符集合和替换实现 |
| `common/djisdk/protocol.go:1130` | 当前 HMS payload 使用 `map[string]any` 保存 args |

## External References

- [DJI HMS 功能](https://developer.dji.com/doc/cloud-api-tutorial/cn/feature-set/dock-feature-set/hms.html) — key 拼接与填充规则；静态 chunk `v-27dd068c.d30ee20d.js`，页面更新时间字段为 2025-03-19。
- [DJI 产品支持](https://developer.dji.com/doc/cloud-api-tutorial/cn/overview/product-support.html) — domain 定义和设备枚举。
- [DJI Dock 3 HMS API](https://developer.dji.com/doc/cloud-api-tutorial/cn/api-reference/dock-to-cloud/mqtt/dock/dock3/hms.html) — HMS 上报结构。
- [DJI 官方 hms.json](https://terra-1-g.djicdn.com/fee90c2e03e04e8da67ea6f56365fc76/SDK%20%E6%96%87%E6%A1%A3/CloudAPI/hms.json) — 页面直接链接的文案资源。

## Related Specs

- `.trellis/spec/backend/dji-guidelines.md` — DJI 协议应由 `common/djisdk` 拥有。

## Caveats / Not Found

- 官方 HMS API 表只显式列出 `component_index`、`sensor_index`；功能页点名 `alarmid`，而 `gimbal_index`、`lidar_index`、`lte_index` 只能由当前官方字典需求佐证。不能把字典占位符数量等同于协议字段完整定义。
- 未找到任何官方 `remote_tip_` 键、范围或文案，也未找到其历史来源，因此不能将其定性为历史遗留或其他端资源。
