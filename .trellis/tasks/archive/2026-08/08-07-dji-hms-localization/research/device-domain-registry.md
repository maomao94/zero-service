# Research: DJI Device Domains and Registry

- **Query**: 整理官方 domain 和当前完整设备枚举，给出适合 Go SDK 的 `{domain}-{type}-{sub_type}` 查询模型，并分析负载 gimbalindex。
- **Scope**: mixed
- **Date**: 2026-08-07

## Findings

### Domain Definitions

DJI 产品支持页定义（官方直接证据）：

| Domain | Meaning |
|---:|---|
| 0 | 飞机类 |
| 1 | 负载类 |
| 2 | 遥控器类 |
| 3 | 机场类 |

`domain` 是命名空间；`domain + type + sub_type` 唯一确定一款设备。官方字段名为 `sub_type`，而 Go 字段可使用 `SubType`，序列化时保持 `sub_type`。

### Aircraft / Remote Controller / Dock Enumeration

以下是 2026-08-07 当前官方产品支持页的完整“飞行器/遥控器/机场枚举值”：

| Name | Domain | Type | SubType | Note |
|---|---:|---:|---:|---|
| Matrice 400 | 0 | 103 | 0 | - |
| Matrice 350 RTK | 0 | 89 | 0 | - |
| Matrice 300 RTK | 0 | 60 | 0 | - |
| Matrice 30 | 0 | 67 | 0 | - |
| Matrice 30T | 0 | 67 | 1 | - |
| Mavic 3 行业系列（M3E 相机） | 0 | 77 | 0 | - |
| Mavic 3 行业系列（M3T 相机） | 0 | 77 | 1 | - |
| Mavic 3 行业系列（M3TA 相机） | 0 | 77 | 3 | - |
| Matrice 3D | 0 | 91 | 0 | - |
| Matrice 3TD | 0 | 91 | 1 | - |
| Matrice 4D | 0 | 100 | 0 | - |
| Matrice 4TD | 0 | 100 | 1 | - |
| DJI Matrice 4 系列（M4E 相机） | 0 | 99 | 0 | - |
| DJI Matrice 4 系列（M4T 相机） | 0 | 99 | 1 | - |
| DJI 带屏遥控器行业版 | 2 | 56 | 0 | 搭配 Matrice 300 RTK |
| DJI RC Plus | 2 | 119 | 0 | 搭配 M350/M300/M30/M30T |
| DJI RC Plus 2 | 2 | 174 | 0 | 页面写明搭配 DJI Matrice 4 系列 |
| DJI RC Pro 行业版 | 2 | 144 | 0 | 搭配 Mavic 3 行业系列 |
| 大疆机场 | 3 | 1 | 0 | - |
| 大疆机场 2 | 3 | 2 | 0 | - |
| 大疆机场 3 | 3 | 3 | 0 | - |

### Payload Enumeration

官方“相机枚举值”表将以下设备放在 domain 1。`type-subtype-gimbalindex` 中前两段标识负载型号，第三段标识挂载位置。

| Product group | Name | Domain | Type | SubType | Supported gimbalindex / context |
|---|---|---:|---:|---:|---|
| 飞行器 FPV | M300 RTK FPV | 1 | 39 | 0 | 7 |
| 飞行器 FPV | M350 RTK FPV | 1 | 39 | 0 | 7 |
| 飞行器 FPV | M30 FPV | 1 | 39 | 0 | 7 |
| 飞行器 FPV | M30T FPV | 1 | 39 | 0 | 7 |
| 飞行器 FPV | Matrice 400 FPV | 1 | 39 | 0 | 7 |
| 飞行器 FPV | Matrice 3D 辅助影像 | 1 | 176 | 0 | 0 |
| 飞行器 FPV | Matrice 3TD 辅助影像 | 1 | 176 | 0 | 0 |
| 飞行器 FPV | Matrice 4D 辅助影像 | 1 | 176 | 0 | 0 |
| 飞行器 FPV | Matrice 4TD 辅助影像 | 1 | 176 | 0 | 0 |
| 相机 | 禅思 Z30 | 1 | 20 | 0 | 0/1/2 |
| 相机 | 禅思 XT2 | 1 | 26 | 0 | 0/1/2 |
| 相机 | 禅思 XTS | 1 | 41 | 0 | 0/1/2 |
| 相机 | 禅思 H20 | 1 | 42 | 0 | 0/1/2 |
| 相机 | 禅思 H20T | 1 | 43 | 0 | 0/1/2 |
| 相机 | 禅思 H20N | 1 | 61 | 0 | 0/1/2 |
| 相机 | 禅思 H30 | 1 | 82 | 0 | 0/1/2 |
| 相机 | 禅思 H30T | 1 | 83 | 0 | 0/1/2 |
| 相机 | Matrice 30 Camera | 1 | 52 | 0 | 0 |
| 相机 | Matrice 30T Camera | 1 | 53 | 0 | 0 |
| 相机 | DJI Matrice 4E Camera | 1 | 88 | 0 | 0 |
| 相机 | DJI Matrice 4T Camera | 1 | 89 | 0 | 0 |
| 相机 | Mavic 3E Camera | 1 | 66 | 0 | 0 |
| 相机 | Mavic 3T Camera | 1 | 67 | 0 | 0 |
| 相机 | Mavic 3TA Camera | 1 | 129 | 0 | 0 |
| 相机 | Matrice 3D Camera | 1 | 80 | 0 | 0 |
| 相机 | Matrice 3TD Camera | 1 | 81 | 0 | 0 |
| 相机 | Matrice 4D Camera | 1 | 98 | 0 | 0 |
| 相机 | Matrice 4TD Camera | 1 | 99 | 0 | 0 |
| 机场相机 | DJI Dock 舱外相机 | 1 | 165 | 0 | 7；`camera_position=1` |
| 机场相机 | DJI Dock 2 舱内相机 | 1 | 165 | 0 | 7；`camera_position=0` |
| 机场相机 | DJI Dock 2 舱外相机 | 1 | 165 | 0 | 7；`camera_position=1` |
| 机场相机 | DJI Dock 3 舱内相机 | 1 | 165 | 0 | 7；`camera_position=0` |
| 机场相机 | DJI Dock 3 舱外相机 | 1 | 165 | 0 | 7；`camera_position=1` |

相同 `(domain,type,sub_type)` 可对应多个上下文名称，例如 `(1,39,0)` 不能区分是哪款飞机的 FPV，相同 `(1,165,0,7)` 也不能区分机场代际/舱内外；官方表依赖所属飞行平台或 `camera_position` 补充语境。因此注册表应把官方共用设备名与“展示上下文”分开，不能承诺仅凭三元组得到所有上下文细节。

### gimbalindex Semantics

官方规则：

| gimbalindex | Meaning |
|---:|---|
| 0 | M300 RTK 为机身下方左云台；其他机型为主云台 |
| 1 | M300 RTK 机身下方右云台 |
| 2 | M300 RTK 机身上方云台 |
| 7 | FPV 相机 |
| other | 预留，不必关注 |

`gimbalindex` **不属于设备身份三元组**。官方分别表述：三元组唯一确定设备；`type + sub_type + gimbalindex` 唯一确定负载及其挂载云台口。适合单独建模为 placement/value object，而非塞入 `DeviceType`：

```go
type DeviceDomain uint8

const (
    DomainAircraft DeviceDomain = 0
    DomainPayload  DeviceDomain = 1
    DomainRemote   DeviceDomain = 2
    DomainDock     DeviceDomain = 3
)

type DeviceModel struct {
    Domain  DeviceDomain
    Type    int
    SubType int
}

type DeviceDefinition struct {
    Model DeviceModel
    Name  string
}

type PayloadPlacement struct {
    Model       DeviceModel // DomainPayload
    GimbalIndex int
}
```

建议的只读注册表 API 形状：

```go
func ParseDeviceModel(raw string) (DeviceModel, error)
func (m DeviceModel) String() string
func LookupDevice(m DeviceModel) (DeviceDefinition, bool)
func LookupDeviceName(raw string) (string, bool)
func DescribePayload(p PayloadPlacement) (DeviceDefinition, GimbalPosition, bool)
```

注册表 key 应使用可比较 struct `DeviceModel`，避免字符串拼接差异；`ParseDeviceModel` 严格接受 `{domain}-{type}-{sub_type}`。旧词形兼容若业务仍需要，应放在独立 compatibility parser，不使官方模型出现伪造的 type/subtype。

## Files Found

| File Path | Description |
|---|---|
| `common/djisdk/device_type.go:8` | 已有三元组结构，但使用 `Subtype` 拼写且混入 HMS 前缀职责 |
| `common/djisdk/protocol.go:969` | `TopoUpdateData`/`TopoSubDevice` 已按官方 domain 说明建模 |
| `app/djicloud/model/gormmodel/dji_device.go:10` | 数据层 domain 常量正确：0/1/2/3 = 飞机/负载/遥控器/机场 |
| `app/djicloud/model/gormmodel/dji_device.go:62` | 拓扑保存 domain/type/sub_type/index |

## External References

- [DJI 产品支持](https://developer.dji.com/doc/cloud-api-tutorial/cn/overview/product-support.html) — 全部枚举和 gimbalindex 语义；静态 chunk `v-4dbe2c9a.8eb912af.js`。

## Related Specs

- `.trellis/spec/backend/dji-guidelines.md` — 设备协议类型由 `common/djisdk` 持有。

## Caveats / Not Found

- 官方页面 `git.updatedTime` 数值对应未来时间，且页面内容可能继续更新；上述表是 2026-08-07 抓取快照。
- 三元组不能表达某些重复负载条目的宿主机型或机场相机 `camera_position`，这些是关联/运行上下文，不应伪装成设备型号字段。
