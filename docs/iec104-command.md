# IEC 104 控制命令文档

本文档描述如何通过 ieccaller 向 IEC 104 从站下发控制命令。控制方向为主站到从站，与监视方向相反。

平台架构、服务组件、配置管理、部署等内容参见：[IEC 104 数采平台](./iec104.md)。

消息格式、信息体结构、数据消费等内容参见：[IEC 104 消息对接文档](./iec104-message.md)。

## 1. 接入方式

| 项目 | 说明 |
| --- | --- |
| 协议 | gRPC，支持 Endpoints 直连或 Nacos 服务发现。 |
| 服务名 | `ieccaller.IecCaller` |
| Proto | [`app/ieccaller/ieccaller.proto`](../app/ieccaller/ieccaller.proto) |
| 响应 | 7 个带类型命令返回各自类型的响应，包含从站回显的命令值。 |

## 2. 推荐接口总览

7 个带类型接口封装了常用控制命令。调用方只需传 `host`、`port`、`coa`、`ioa`、`value`、`withTime`，无需手动查 `typeId`。

| RPC | value 类型 | withTime=false | withTime=true | 默认限定词 |
| --- | --- | --- | --- | --- |
| `SendSingleCommand` | bool | 45 `C_SC_NA_1` | 58 `C_SC_TA_1` | QOC 无附加定义，直接执行。 |
| `SendDoubleCommand` | `DoubleCommandValue` | 46 `C_DC_NA_1` | 59 `C_DC_TA_1` | QOC 无附加定义，直接执行。 |
| `SendStepCommand` | `StepCommandValue` | 47 `C_RC_NA_1` | 60 `C_RC_TA_1` | QOC 无附加定义，直接执行。 |
| `SendSetpointNormalized` | int32 | 48 `C_SE_NA_1` | 61 `C_SE_TA_1` | QOS `Qual=0`，直接执行。 |
| `SendSetpointScaled` | int32 | 49 `C_SE_NB_1` | 62 `C_SE_TB_1` | QOS `Qual=0`，直接执行。 |
| `SendSetpointFloat` | float | 50 `C_SE_NC_1` | 63 `C_SE_TC_1` | QOS `Qual=0`，直接执行。 |
| `SendBitstringCommand` | uint64 | 51 `C_BO_NA_1` | 64 `C_BO_TA_1` | 无限定词字段。 |

`withTime=true` 时，服务端使用当前时间填充 CP56Time2a 时标。所有带类型接口都采用直接执行，未启用选择执行。

## 3. 公共请求字段

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `host` | string | 从站 IP。 |
| `port` | uint32 | 从站端口。 |
| `coa` | uint32 | 公共地址。 |
| `ioa` | uint32 | 信息对象地址。 |
| `value` | 各接口不同 | 命令值。 |
| `withTime` | bool | `false` 使用不带时标 TypeId，`true` 使用带 CP56Time2a 时标 TypeId。 |

## 4. 单点命令 `SendSingleCommand`

```json
{
  "host": "127.0.0.1",
  "port": 2404,
  "coa": 1,
  "ioa": 1001,
  "value": true,
  "withTime": false
}
```

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `value` | bool | `true` 表示合，`false` 表示分。 |

## 5. 双点命令 `SendDoubleCommand`

```json
{
  "host": "127.0.0.1",
  "port": 2404,
  "coa": 1,
  "ioa": 1002,
  "value": "DCO_ON",
  "withTime": false
}
```

| 枚举 | 数值 | 含义 |
| --- | --- | --- |
| `DCO_NOT_ALLOWED` | 0 | 不允许。 |
| `DCO_ON` | 1 | 合。 |
| `DCO_OFF` | 2 | 分。 |

## 6. 档位调节命令 `SendStepCommand`

```json
{
  "host": "127.0.0.1",
  "port": 2404,
  "coa": 1,
  "ioa": 1003,
  "value": "SCO_UP",
  "withTime": false
}
```

| 枚举 | 数值 | 含义 |
| --- | --- | --- |
| `SCO_NOT_ALLOWED` | 0 | 不允许。 |
| `SCO_DOWN` | 1 | 降一步。 |
| `SCO_UP` | 2 | 升一步。 |

## 7. 归一化设点 `SendSetpointNormalized`

```json
{
  "host": "127.0.0.1",
  "port": 2404,
  "coa": 1,
  "ioa": 4001,
  "value": 16384,
  "withTime": false
}
```

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `value` | int32 | 接口类型为 int32，发送前转换为 int16，取值应在 `-32768` 至 `32767`。 |

## 8. 标度化设点 `SendSetpointScaled`

```json
{
  "host": "127.0.0.1",
  "port": 2404,
  "coa": 1,
  "ioa": 4002,
  "value": 500,
  "withTime": false
}
```

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `value` | int32 | 接口类型为 int32，发送前转换为 int16，取值应在 `-32768` 至 `32767`。 |

## 9. 浮点设点 `SendSetpointFloat`

```json
{
  "host": "127.0.0.1",
  "port": 2404,
  "coa": 1,
  "ioa": 4003,
  "value": 220.5,
  "withTime": false
}
```

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `value` | float | IEEE 754 单精度浮点数，直接映射 IEC 104 短浮点值。 |

## 10. 32 位位串命令 `SendBitstringCommand`

```json
{
  "host": "127.0.0.1",
  "port": 2404,
  "coa": 1,
  "ioa": 4004,
  "value": 4294967295,
  "withTime": false
}
```

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `value` | uint64 | 接口类型为 uint64，发送前转换为 uint32，取值应在 `0` 至 `4294967295`。 |

## 11. 通用接口 `SendCommand`

`SendCommand` 接收显式 `typeId` 和字符串 `value`，适用于兼容旧调用方或调用未封装的控制命令。

```json
{
  "host": "127.0.0.1",
  "port": 2404,
  "coa": 1,
  "typeId": 50,
  "ioa": 4003,
  "value": "220.5"
}
```

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `typeId` | uint32 | 控制方向 ASDU TypeId。 |
| `value` | string | 通用字符串值，由服务端按 `typeId` 转换。 |

常用控制 TypeId 见[第 14 节](#14-全量-typeid-双向对照表)。

## 12. 其他控制接口

| RPC | TypeId | 说明 |
| --- | --- | --- |
| `SendInterrogationCmd` | 100 `C_IC_NA_1` | 总召唤。 |
| `SendCounterInterrogationCmd` | 101 `C_CI_NA_1` | 累计量召唤。 |
| `SendReadCmd` | 102 `C_RD_NA_1` | 读命令，需要 `ioa`。 |
| `SendTestCmd` | 107 `C_TS_TA_1` | 测试命令。 |

### 12.1 点位缓存管理接口

#### `ClearPointMappingCache`

清除点位映射缓存，支持批量操作。

**请求**：

```json
{
  "keys": ["192.168.1.100_1_0x000001"],
  "keyInfos": [
    {
      "tagStation": "192.168.1.100_2404",
      "coa": 1,
      "ioa": 1
    }
  ]
}
```

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `keys` | string[] | 缓存 key 列表，格式为 `{host}_{coa}_{ioaHex}`。 |
| `keyInfos` | CacheKeyInfo[] | 缓存 key 信息列表，用于根据 tagStation、coa、ioa 生成 key。 |

**响应**：

```json
{
  "clearedCount": 2
}
```

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `clearedCount` | int64 | 清除的缓存数量。 |

**CacheKeyInfo 结构**：

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `tagStation` | string | 站点标识。 |
| `coa` | int64 | 公共地址。 |
| `ioa` | int64 | 信息对象地址。 |

## 13. 响应和回执机制

7 个带类型控制命令的 gRPC 响应包含从站回显的命令值（如 `SendSingleCommandRes.value`）。gRPC 调用成功只表示命令已由 ieccaller 发送到本地 IEC 104 client 或已在集群模式下广播给其他实例，不等于设备已执行成功。

**响应类型对照表**：

| RPC | 响应类型 | 响应值说明 |
| --- | --- | --- |
| `SendSingleCommand` | `SendSingleCommandRes` | `bool value` - 从站回显的单点命令值 |
| `SendDoubleCommand` | `SendDoubleCommandRes` | `DoubleCommandValue value` - 从站回显的双点命令值 |
| `SendStepCommand` | `SendStepCommandRes` | `int32 value` - 从站回显的档位调节命令值 |
| `SendSetpointNormalized` | `SendSetpointNormalizedRes` | `int32 value` - 从站回显的归一化设点值 |
| `SendSetpointScaled` | `SendSetpointScaledRes` | `int32 value` - 从站回显的标度化设点值 |
| `SendSetpointFloat` | `SendSetpointFloatRes` | `float value` - 从站回显的短浮点设点值 |
| `SendBitstringCommand` | `SendBitstringCommandRes` | `uint64 value` - 从站回显的32位位串命令值 |

IEC 104 控制命令有两类后续反馈：

1. 协议级回执：从站返回同类 `C_*` ASDU，COT 通常为激活确认、停止激活确认、激活终止等。ieccaller 当前只记录日志，不推送到 Kafka 或 MQTT。
2. 状态更新：设备执行后，上报新的 `M_*` 监视数据。下游消费者应监听对应监视类型判断最终状态。

| 下发命令 | 控制 TypeId | 常见状态更新 TypeId | 状态 Body |
| --- | --- | --- | --- |
| `SendSingleCommand` | 45、58 | 1、2、30 | `SinglePointInfo` |
| `SendDoubleCommand` | 46、59 | 3、4、31 | `DoublePointInfo` |
| `SendStepCommand` | 47、60 | 5、6、32 | `StepPositionInfo` |
| `SendSetpointNormalized` | 48、61 | 9、10、34 | `MeasuredValueNormalInfo` |
| `SendSetpointScaled` | 49、62 | 11、12、35 | `MeasuredValueScaledInfo` |
| `SendSetpointFloat` | 50、63 | 13、14、36 | `MeasuredValueFloatInfo` |
| `SendBitstringCommand` | 51、64 | 7、8、33 | `BitString32Info` |

消费方建议：

1. 监听命令对应的监视 ASDU。
2. 匹配 `host`、`coa`、`body.ioa`。
3. 检查品质位，特别是 `iv`、`ov`、`bl`。
4. 将新 `body.value` 与期望状态对比。
5. 如果只收到 gRPC 成功但没有状态更新，应按业务超时处理。

### 13.1 错误码

gRPC 调用失败时，返回的错误码和 HTTP 状态码如下：

| 场景 | 业务错误码 | HTTP | gRPC | 说明 |
| --- | --- | --- | --- | --- |
| 设备拒绝命令（isNegative=true） | 105102 | 409 | FailedPrecondition | 从站明确拒绝，如 UnknownIOA、UnknownTypeID |
| ACK 超时 | 100997 | 504 | DeadlineExceeded | 从站未在规定时间内返回回执 |
| 同一点位重复下发 | 105103 | 409 | Aborted | 同一控制点已有未完成命令 |
| 找不到 IEC 客户端 | 106101 | 503 | Unavailable | 未连接到目标从站 |
| 获取客户端失败 | 106101 | 503 | Unavailable | ClientManager 返回错误 |
| 第三方服务异常 | 106102 | 503 | Unavailable | 其他 IEC 协议层错误 |

错误消息格式示例：

```
IEC命令被设备拒绝: command rejected: cot=UnknownTypeID isNegative=true typeId=46 coa=1 ioa=1001
```

## 14. 全量 TypeId 双向对照表

| TypeId | 方向 | ASDU | Body 结构体 | 推荐接口 |
| --- | --- | --- | --- | --- |
| 1 | 监视 | M_SP_NA_1 | `SinglePointInfo` | 不适用 |
| 2 | 监视 | M_SP_TA_1 | `SinglePointInfo` | 不适用 |
| 3 | 监视 | M_DP_NA_1 | `DoublePointInfo` | 不适用 |
| 4 | 监视 | M_DP_TA_1 | `DoublePointInfo` | 不适用 |
| 5 | 监视 | M_ST_NA_1 | `StepPositionInfo` | 不适用 |
| 6 | 监视 | M_ST_TA_1 | `StepPositionInfo` | 不适用 |
| 7 | 监视 | M_BO_NA_1 | `BitString32Info` | 不适用 |
| 8 | 监视 | M_BO_TA_1 | `BitString32Info` | 不适用 |
| 9 | 监视 | M_ME_NA_1 | `MeasuredValueNormalInfo` | 不适用 |
| 10 | 监视 | M_ME_TA_1 | `MeasuredValueNormalInfo` | 不适用 |
| 11 | 监视 | M_ME_NB_1 | `MeasuredValueScaledInfo` | 不适用 |
| 12 | 监视 | M_ME_TB_1 | `MeasuredValueScaledInfo` | 不适用 |
| 13 | 监视 | M_ME_NC_1 | `MeasuredValueFloatInfo` | 不适用 |
| 14 | 监视 | M_ME_TC_1 | `MeasuredValueFloatInfo` | 不适用 |
| 15 | 监视 | M_IT_NA_1 | `BinaryCounterReadingInfo` | 不适用 |
| 16 | 监视 | M_IT_TA_1 | `BinaryCounterReadingInfo` | 不适用 |
| 17 | 监视 | M_EP_TA_1 | `EventOfProtectionEquipmentInfo` | 不适用 |
| 18 | 监视 | M_EP_TB_1 | `PackedStartEventsOfProtectionEquipmentInfo` | 不适用 |
| 19 | 监视 | M_EP_TC_1 | `PackedOutputCircuitInfoInfo` | 不适用 |
| 20 | 监视 | M_PS_NA_1 | `PackedSinglePointWithSCDInfo` | 不适用 |
| 21 | 监视 | M_ME_ND_1 | `MeasuredValueNormalInfo` | 不适用 |
| 30 | 监视 | M_SP_TB_1 | `SinglePointInfo` | 不适用 |
| 31 | 监视 | M_DP_TB_1 | `DoublePointInfo` | 不适用 |
| 32 | 监视 | M_ST_TB_1 | `StepPositionInfo` | 不适用 |
| 33 | 监视 | M_BO_TB_1 | `BitString32Info` | 不适用 |
| 34 | 监视 | M_ME_TD_1 | `MeasuredValueNormalInfo` | 不适用 |
| 35 | 监视 | M_ME_TE_1 | `MeasuredValueScaledInfo` | 不适用 |
| 36 | 监视 | M_ME_TF_1 | `MeasuredValueFloatInfo` | 不适用 |
| 37 | 监视 | M_IT_TB_1 | `BinaryCounterReadingInfo` | 不适用 |
| 38 | 监视 | M_EP_TD_1 | `EventOfProtectionEquipmentInfo` | 不适用 |
| 39 | 监视 | M_EP_TE_1 | `PackedStartEventsOfProtectionEquipmentInfo` | 不适用 |
| 40 | 监视 | M_EP_TF_1 | `PackedOutputCircuitInfoInfo` | 不适用 |
| 45 | 控制 | C_SC_NA_1 | 不推送 | `SendSingleCommand`，`withTime=false` |
| 46 | 控制 | C_DC_NA_1 | 不推送 | `SendDoubleCommand`，`withTime=false` |
| 47 | 控制 | C_RC_NA_1 | 不推送 | `SendStepCommand`，`withTime=false` |
| 48 | 控制 | C_SE_NA_1 | 不推送 | `SendSetpointNormalized`，`withTime=false` |
| 49 | 控制 | C_SE_NB_1 | 不推送 | `SendSetpointScaled`，`withTime=false` |
| 50 | 控制 | C_SE_NC_1 | 不推送 | `SendSetpointFloat`，`withTime=false` |
| 51 | 控制 | C_BO_NA_1 | 不推送 | `SendBitstringCommand`，`withTime=false` |
| 58 | 控制 | C_SC_TA_1 | 不推送 | `SendSingleCommand`，`withTime=true` |
| 59 | 控制 | C_DC_TA_1 | 不推送 | `SendDoubleCommand`，`withTime=true` |
| 60 | 控制 | C_RC_TA_1 | 不推送 | `SendStepCommand`，`withTime=true` |
| 61 | 控制 | C_SE_TA_1 | 不推送 | `SendSetpointNormalized`，`withTime=true` |
| 62 | 控制 | C_SE_TB_1 | 不推送 | `SendSetpointScaled`，`withTime=true` |
| 63 | 控制 | C_SE_TC_1 | 不推送 | `SendSetpointFloat`，`withTime=true` |
| 64 | 控制 | C_BO_TA_1 | 不推送 | `SendBitstringCommand`，`withTime=true` |
| 70 | 监视 | M_EI_NA_1 | 不推送 | 初始化结束，仅日志记录。 |
| 100 | 控制 | C_IC_NA_1 | 不推送 | `SendInterrogationCmd` |
| 101 | 控制 | C_CI_NA_1 | 不推送 | `SendCounterInterrogationCmd` |
| 102 | 控制 | C_RD_NA_1 | 不推送 | `SendReadCmd` |
| 103 | 控制 | C_CS_NA_1 | 不推送 | `SendCommand`，时钟同步。 |
| 105 | 控制 | C_RP_NA_1 | 不推送 | `SendCommand`，复位进程。 |
| 107 | 控制 | C_TS_TA_1 | 不推送 | `SendTestCmd` |

图例：监视表示从站到主站，由 ieccaller 推送给下游；控制表示主站到从站，由 gRPC 调用触发。

---

- **文档版本**：v2.0.0（2026-06-15）
- **协议版本**：IEC 60870-5-104

相关文档：

- [IEC 104 数采平台架构](./iec104.md)，服务组件、数据流、配置管理。
- [IEC 104 消息对接文档](./iec104-message.md)，消息格式、信息体结构、数据消费指南。
- [`ieccaller.proto`](../app/ieccaller/ieccaller.proto)，控制指令 RPC 接口定义。
