# IEC 104 消息对接文档

本文档定义 ieccaller 推送的 IEC 60870-5-104 消息格式规范，供下游消费者通过 Kafka、MQTT、gRPC 对接使用。

平台架构、服务组件、配置管理、部署等内容参见：[IEC 104 数采平台](./iec104.md)。

## 目录

- [1. 消息格式规范](#1-消息格式规范)
  - [1.1 基础协议](#11-基础协议)
  - [1.2 消息结构](#12-消息结构)
  - [1.3 PointMapping 结构](#13-pointmapping-结构)
  - [1.4 设备点位映射表](#14-设备点位映射表)
- [2. ASDU 类型映射表](#2-asdu-类型映射表)
- [3. 信息体结构详解](#3-信息体结构详解)
- [4. 附录](#4-附录)
- [5. 数据消费指南](#5-数据消费指南)
- [6. 技术支持](#6-技术支持)
- [7. 控制命令接口](#7-控制命令接口)

## 1. 消息格式规范

### 1.1 基础协议

- **传输协议**：IEC 60870-5-104
- **传输载体**：Kafka Topic、MQTT Topic、gRPC 流式事件
- **数据格式**：JSON
- **编码格式**：UTF-8
- **时间格式**：`YYYY-MM-DD HH:mm:ss.SSSSSS`，UTC+8 时区

### 1.2 消息结构

```json
{
  "msgId": "消息id",
  "host": "127.0.0.1",
  "port": 2404,
  "asdu": "M_SP_NA_1",
  "typeId": 1,
  "dataType": 0,
  "coa": 1001,
  "body": {
    "ioa": 2001,
    "value": true,
    "qds": 0,
    "qdsDesc": "QDS(00000000)[]",
    "ov": false,
    "bl": false,
    "sb": false,
    "nt": false,
    "iv": false,
    "time": ""
  },
  "time": "2026-06-05 14:30:00.000000",
  "metaData": {
    "stationId": "station_001",
    "appId": "iec104"
  },
  "pm": {
    "deviceId": "device_123",
    "deviceName": "测试设备",
    "tdTableType": "yx,yc",
    "ext1": "alarm",
    "ext2": "",
    "ext3": "",
    "ext4": "",
    "ext5": ""
  }
}
```

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `msgId` | string | 消息 ID，由 ieccaller 为每条信息体生成。 |
| `host` | string | IEC 104 从站 IP。 |
| `port` | int | IEC 104 从站端口。 |
| `asdu` | string | ASDU 类型名称，来自 IEC 104 TypeId。 |
| `typeId` | int | ASDU 类型编号。完整映射见[第 2 节](#2-asdu-类型映射表)。 |
| `dataType` | int | ieccaller 内部信息体分类编号。 |
| `coa` | uint | 公共地址，范围通常为 1 至 65534，65535 为全局地址。 |
| `body` | object | 信息体对象，结构随 `typeId` 变化。完整结构见[第 3 节](#3-信息体结构详解)。 |
| `time` | string | ieccaller 推送时间，由服务端写入。 |
| `metaData` | object | 客户端配置中的应用级元数据，可为空。 |
| `pm` | object | 点位映射信息。未配置映射或映射禁用时可能缺省。 |

`body.time` 与 `time` 含义不同：

- `time` 是消息推送时间，所有成功推送的消息都有值。
- `body.time` 是 IEC 104 信息体携带的时标。对应不带时标的 ASDU 时，该字段为空字符串。

### 1.3 PointMapping 结构

PointMapping 来自 `device_point_mapping` 表，用于设备识别、动态 MQTT Topic 生成和业务侧点位归类。

```json
{
  "deviceId": "device_123",
  "deviceName": "测试设备",
  "tdTableType": "yx,yc",
  "ext1": "alarm",
  "ext2": "",
  "ext3": "",
  "ext4": "",
  "ext5": ""
}
```

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `deviceId` | string | 设备唯一标识。 |
| `deviceName` | string | 设备名称。 |
| `tdTableType` | string | TDengine 表类型，可用逗号分隔，如 `yx,yc`。 |
| `ext1` 至 `ext5` | string | 扩展字段，用于 Topic 拆分和业务透传。 |

扩展字段常见用途：

- 业务类型，如 `alarm`、`normal`、`control`。
- 设备类型，如 `switch`、`sensor`、`meter`。
- 区域或功能维度，如 `area1`、`power`、`communication`。

Topic 模板示例：

```text
{{.Pm.DeviceId}}/{{.Pm.Ext1}}/{{.Asdu}}
```

生成结果示例：

```text
device_123/alarm/M_SP_NA_1
```

### 1.4 设备点位映射表

`device_point_mapping` 是点位映射、弱校验推送和动态 Topic 生成的配置来源。

```sql
CREATE TABLE IF NOT EXISTS device_point_mapping (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    create_time TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    update_time TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    delete_time TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    del_state INTEGER NOT NULL DEFAULT 0,
    version INTEGER NOT NULL DEFAULT 0,

    tag_station VARCHAR(64) NOT NULL DEFAULT '',
    coa INTEGER NOT NULL DEFAULT 0,
    ioa INTEGER NOT NULL DEFAULT 0,
    device_id VARCHAR(64) NOT NULL DEFAULT '',
    device_name VARCHAR(128) NOT NULL DEFAULT '',
    td_table_type VARCHAR(255) NOT NULL DEFAULT '',
    enable_push INTEGER NOT NULL DEFAULT 1,
    enable_raw_insert INTEGER NOT NULL DEFAULT 1,
    description VARCHAR(256) NOT NULL DEFAULT '',

    ext_1 VARCHAR(64) NOT NULL DEFAULT '',
    ext_2 VARCHAR(64) NOT NULL DEFAULT '',
    ext_3 VARCHAR(64) NOT NULL DEFAULT '',
    ext_4 VARCHAR(64) NOT NULL DEFAULT '',
    ext_5 VARCHAR(64) NOT NULL DEFAULT '',

    UNIQUE(tag_station, coa, ioa)
);
```

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `tag_station` | string | 站点标识。默认由 `host` 和 `port` 生成，也可通过 `metaData.stationId` 覆盖。 |
| `coa` | int | IEC 104 公共地址。 |
| `ioa` | int | IEC 104 信息对象地址。 |
| `device_id` | string | 映射到 `pm.deviceId`。 |
| `device_name` | string | 映射到 `pm.deviceName`。 |
| `td_table_type` | string | 映射到 `pm.tdTableType`。 |
| `enable_push` | int | 是否允许推送，`0` 为不推送，`1` 为推送。 |
| `enable_raw_insert` | int | 是否允许写入原始数据，`0` 为不写入，`1` 为写入。 |
| `description` | string | 点位描述信息。 |
| `ext_1` 至 `ext_5` | string | 映射到 `pm.ext1` 至 `pm.ext5`。 |

唯一索引为 `UNIQUE(tag_station, coa, ioa)`，同一个站点、公共地址、信息对象地址只能绑定一条映射。

## 2. ASDU 类型映射表

下表列出 ieccaller 会解析并推送给下游的监视方向 ASDU。控制方向命令和回执见[第 7 节](#7-控制命令接口)。

| DataType | TypeId | ASDU | Body 结构体 | 说明 |
| --- | --- | --- | --- | --- |
| 0 | 1 | M_SP_NA_1 | `SinglePointInfo` | 单点遥信，不带时标。 |
| 0 | 2 | M_SP_TA_1 | `SinglePointInfo` | 单点遥信，带 CP24Time2a 时标。 |
| 0 | 30 | M_SP_TB_1 | `SinglePointInfo` | 单点遥信，带 CP56Time2a 时标。 |
| 1 | 3 | M_DP_NA_1 | `DoublePointInfo` | 双点遥信，不带时标。 |
| 1 | 4 | M_DP_TA_1 | `DoublePointInfo` | 双点遥信，带 CP24Time2a 时标。 |
| 1 | 31 | M_DP_TB_1 | `DoublePointInfo` | 双点遥信，带 CP56Time2a 时标。 |
| 2 | 11 | M_ME_NB_1 | `MeasuredValueScaledInfo` | 标度化遥测值，不带时标。 |
| 2 | 12 | M_ME_TB_1 | `MeasuredValueScaledInfo` | 标度化遥测值，带 CP24Time2a 时标。 |
| 2 | 35 | M_ME_TE_1 | `MeasuredValueScaledInfo` | 标度化遥测值，带 CP56Time2a 时标。 |
| 3 | 9 | M_ME_NA_1 | `MeasuredValueNormalInfo` | 归一化遥测值，不带时标。 |
| 3 | 10 | M_ME_TA_1 | `MeasuredValueNormalInfo` | 归一化遥测值，带 CP24Time2a 时标。 |
| 3 | 21 | M_ME_ND_1 | `MeasuredValueNormalInfo` | 归一化遥测值，不带品质描述。 |
| 3 | 34 | M_ME_TD_1 | `MeasuredValueNormalInfo` | 归一化遥测值，带 CP56Time2a 时标。 |
| 4 | 5 | M_ST_NA_1 | `StepPositionInfo` | 步位置信息，不带时标。 |
| 4 | 6 | M_ST_TA_1 | `StepPositionInfo` | 步位置信息，带 CP24Time2a 时标。 |
| 4 | 32 | M_ST_TB_1 | `StepPositionInfo` | 步位置信息，带 CP56Time2a 时标。 |
| 5 | 7 | M_BO_NA_1 | `BitString32Info` | 32 位比特串，不带时标。 |
| 5 | 8 | M_BO_TA_1 | `BitString32Info` | 32 位比特串，带 CP24Time2a 时标。 |
| 5 | 33 | M_BO_TB_1 | `BitString32Info` | 32 位比特串，带 CP56Time2a 时标。 |
| 6 | 13 | M_ME_NC_1 | `MeasuredValueFloatInfo` | 短浮点遥测值，不带时标。 |
| 6 | 14 | M_ME_TC_1 | `MeasuredValueFloatInfo` | 短浮点遥测值，带 CP24Time2a 时标。 |
| 6 | 36 | M_ME_TF_1 | `MeasuredValueFloatInfo` | 短浮点遥测值，带 CP56Time2a 时标。 |
| 7 | 15 | M_IT_NA_1 | `BinaryCounterReadingInfo` | 累计量，不带时标。 |
| 7 | 16 | M_IT_TA_1 | `BinaryCounterReadingInfo` | 累计量，带 CP24Time2a 时标。 |
| 7 | 37 | M_IT_TB_1 | `BinaryCounterReadingInfo` | 累计量，带 CP56Time2a 时标。 |
| 8 | 17 | M_EP_TA_1 | `EventOfProtectionEquipmentInfo` | 继电保护事件，带 CP16Time2a 时标。 |
| 8 | 38 | M_EP_TD_1 | `EventOfProtectionEquipmentInfo` | 继电保护事件，带 CP56Time2a 时标。 |
| 9 | 18 | M_EP_TB_1 | `PackedStartEventsOfProtectionEquipmentInfo` | 继电保护成组启动事件，带 CP16Time2a 时标。 |
| 9 | 39 | M_EP_TE_1 | `PackedStartEventsOfProtectionEquipmentInfo` | 继电保护成组启动事件，带 CP56Time2a 时标。 |
| 10 | 19 | M_EP_TC_1 | `PackedOutputCircuitInfoInfo` | 继电保护成组输出电路信息，带 CP16Time2a 时标。 |
| 10 | 40 | M_EP_TF_1 | `PackedOutputCircuitInfoInfo` | 继电保护成组输出电路信息，带 CP56Time2a 时标。 |
| 11 | 20 | M_PS_NA_1 | `PackedSinglePointWithSCDInfo` | 带变位检出的成组单点信息。 |
| 19 | 70 | M_EI_NA_1 | 不推送 | 初始化结束，仅记录日志。 |
| 20 | 其他 | UNKNOWN | 不推送 | 未识别 ASDU，仅记录日志。 |

## 3. 信息体结构详解

本节字段名与 `common/iec104/types/types.go` 的 JSON 标签保持一致。`qds`、`qdp`、`scd`、`oci` 的位含义见[第 4 节](#4-附录)。

### 3.1 单点信息 `SinglePointInfo`

适用 ASDU：`M_SP_NA_1`、`M_SP_TA_1`、`M_SP_TB_1`。

```json
{
  "ioa": 2001,
  "value": true,
  "qds": 160,
  "qdsDesc": "QDS(10100000)[Substituted|Invalid]",
  "ov": false,
  "bl": false,
  "sb": true,
  "nt": false,
  "iv": true,
  "time": "2026-06-05 14:30:00.000000"
}
```

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `ioa` | uint | 信息对象地址，十进制显示。 |
| `value` | bool | `true` 表示合或动作，`false` 表示分或未动作。 |
| `qds` | byte | 品质描述字。 |
| `qdsDesc` | string | 品质描述字符串。 |
| `ov` | bool | 溢出标志。 |
| `bl` | bool | 闭锁标志。 |
| `sb` | bool | 取代标志。 |
| `nt` | bool | 非当前值标志。 |
| `iv` | bool | 无效标志。 |
| `time` | string | 信息体时标。不带时标 ASDU 为空字符串。 |

### 3.2 双点信息 `DoublePointInfo`

适用 ASDU：`M_DP_NA_1`、`M_DP_TA_1`、`M_DP_TB_1`。

```json
{
  "ioa": 2002,
  "value": 2,
  "qds": 0,
  "qdsDesc": "QDS(00000000)[]",
  "ov": false,
  "bl": false,
  "sb": false,
  "nt": false,
  "iv": false,
  "time": ""
}
```

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `ioa` | uint | 信息对象地址。 |
| `value` | byte | 双点状态，`0` 为中间或不确定，`1` 为开，`2` 为合，`3` 为不确定。 |
| `qds` | byte | 品质描述字。 |
| `qdsDesc` | string | 品质描述字符串。 |
| `ov`、`bl`、`sb`、`nt`、`iv` | bool | 品质位，见[附录 A](#附录-a品质描述)。 |
| `time` | string | 信息体时标。不带时标 ASDU 为空字符串。 |

### 3.3 步位置信息 `StepPositionInfo`

适用 ASDU：`M_ST_NA_1`、`M_ST_TA_1`、`M_ST_TB_1`。

```json
{
  "ioa": 3001,
  "value": {
    "val": 63,
    "hasTransient": false
  },
  "qds": 0,
  "qdsDesc": "QDS(00000000)[]",
  "ov": false,
  "bl": false,
  "sb": false,
  "nt": false,
  "iv": false,
  "time": ""
}
```

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `ioa` | uint | 信息对象地址。 |
| `value.val` | int | 步位置值，范围为 `-64` 至 `63`。 |
| `value.hasTransient` | bool | `true` 表示设备处于瞬变状态。 |
| `qds`、`qdsDesc` | byte、string | 品质描述。 |
| `ov`、`bl`、`sb`、`nt`、`iv` | bool | 品质位，见[附录 A](#附录-a品质描述)。 |
| `time` | string | 信息体时标。不带时标 ASDU 为空字符串。 |

### 3.4 归一化遥测值 `MeasuredValueNormalInfo`

适用 ASDU：`M_ME_NA_1`、`M_ME_TA_1`、`M_ME_TD_1`、`M_ME_ND_1`。

```json
{
  "ioa": 4001,
  "value": 16384,
  "nva": 0.5,
  "qds": 0,
  "qdsDesc": "QDS(00000000)[]",
  "ov": false,
  "bl": false,
  "sb": false,
  "nt": false,
  "iv": false,
  "time": ""
}
```

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `ioa` | uint | 信息对象地址。 |
| `value` | int16 | 原始归一化值，范围为 `-32768` 至 `32767`。 |
| `nva` | float32 | 换算后的归一化浮点值，代码按 `value / 32768` 转换。 |
| `qds`、`qdsDesc` | byte、string | 品质描述。`M_ME_ND_1` 不带品质描述时由库返回默认值。 |
| `ov`、`bl`、`sb`、`nt`、`iv` | bool | 品质位，见[附录 A](#附录-a品质描述)。 |
| `time` | string | 信息体时标。不带时标 ASDU 为空字符串。 |

### 3.5 标度化遥测值 `MeasuredValueScaledInfo`

适用 ASDU：`M_ME_NB_1`、`M_ME_TB_1`、`M_ME_TE_1`。

```json
{
  "ioa": 4002,
  "value": 500,
  "qds": 0,
  "qdsDesc": "QDS(00000000)[]",
  "ov": false,
  "bl": false,
  "sb": false,
  "nt": false,
  "iv": false,
  "time": ""
}
```

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `ioa` | uint | 信息对象地址。 |
| `value` | int16 | 标度化值。 |
| `qds`、`qdsDesc` | byte、string | 品质描述。 |
| `ov`、`bl`、`sb`、`nt`、`iv` | bool | 品质位，见[附录 A](#附录-a品质描述)。 |
| `time` | string | 信息体时标。不带时标 ASDU 为空字符串。 |

### 3.6 比特位串信息 `BitString32Info`

适用 ASDU：`M_BO_NA_1`、`M_BO_TA_1`、`M_BO_TB_1`。

```json
{
  "ioa": 4003,
  "value": 220,
  "qds": 0,
  "qdsDesc": "QDS(00000000)[]",
  "ov": false,
  "bl": false,
  "sb": false,
  "nt": false,
  "iv": false,
  "time": ""
}
```

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `ioa` | uint | 信息对象地址。 |
| `value` | uint32 | 32 位比特串，每个比特位可表示一个状态。 |
| `qds`、`qdsDesc` | byte、string | 品质描述。 |
| `ov`、`bl`、`sb`、`nt`、`iv` | bool | 品质位，见[附录 A](#附录-a品质描述)。 |
| `time` | string | 信息体时标。不带时标 ASDU 为空字符串。 |

### 3.7 短浮点遥测值 `MeasuredValueFloatInfo`

适用 ASDU：`M_ME_NC_1`、`M_ME_TC_1`、`M_ME_TF_1`。

```json
{
  "ioa": 4004,
  "value": 220.5,
  "qds": 0,
  "qdsDesc": "QDS(00000000)[]",
  "ov": false,
  "bl": false,
  "sb": false,
  "nt": false,
  "iv": false,
  "time": ""
}
```

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `ioa` | uint | 信息对象地址。 |
| `value` | float32 | IEEE 754 短浮点数，通常直接表示工程值。 |
| `qds`、`qdsDesc` | byte、string | 品质描述。 |
| `ov`、`bl`、`sb`、`nt`、`iv` | bool | 品质位，见[附录 A](#附录-a品质描述)。 |
| `time` | string | 信息体时标。不带时标 ASDU 为空字符串。 |

### 3.8 累计量 `BinaryCounterReadingInfo`

适用 ASDU：`M_IT_NA_1`、`M_IT_TA_1`、`M_IT_TB_1`。

```json
{
  "ioa": 5001,
  "value": {
    "counterReading": 1000,
    "seqNumber": 5,
    "hasCarry": false,
    "isAdjusted": false,
    "isInvalid": false
  },
  "time": ""
}
```

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `ioa` | uint | 信息对象地址。 |
| `value.counterReading` | int32 | 计数器读数。 |
| `value.seqNumber` | byte | 顺序号，通常为 0 至 31。 |
| `value.hasCarry` | bool | `true` 表示计数器进位或溢出。 |
| `value.isAdjusted` | bool | `true` 表示计数量被人工调整。 |
| `value.isInvalid` | bool | `true` 表示累计量无效。 |
| `time` | string | 信息体时标。不带时标 ASDU 为空字符串。 |

### 3.9 继电保护事件 `EventOfProtectionEquipmentInfo`

适用 ASDU：`M_EP_TA_1`、`M_EP_TD_1`。

```json
{
  "ioa": 6001,
  "event": 1,
  "qdp": 145,
  "qdpDesc": "QDP(10010001)[Blocked|Invalid]",
  "ei": false,
  "bl": true,
  "sb": false,
  "nt": false,
  "iv": true,
  "msec": 500,
  "time": "2026-06-05 14:30:00.000000"
}
```

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `ioa` | uint | 信息对象地址。 |
| `event` | byte | 事件状态，见[附录 B](#附录-b单点和双点状态值)。 |
| `qdp` | byte | 保护事件品质描述字。 |
| `qdpDesc` | string | 保护事件品质描述字符串。 |
| `ei`、`bl`、`sb`、`nt`、`iv` | bool | QDP 品质位，见[附录 A](#附录-a品质描述)。 |
| `msec` | uint16 | 事件经过时间，单位毫秒。 |
| `time` | string | 信息体时标。 |

### 3.10 继电保护成组启动事件 `PackedStartEventsOfProtectionEquipmentInfo`

适用 ASDU：`M_EP_TB_1`、`M_EP_TE_1`。

```json
{
  "ioa": 6002,
  "event": 5,
  "qdp": 0,
  "qdpDesc": "QDP(00000000)[]",
  "ei": false,
  "bl": false,
  "sb": false,
  "nt": false,
  "iv": false,
  "msec": 120,
  "time": "2026-06-05 14:30:00.000000"
}
```

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `ioa` | uint | 信息对象地址。 |
| `event` | byte | 成组启动事件位串，按设备点表解释。 |
| `qdp`、`qdpDesc` | byte、string | 保护事件品质描述。 |
| `ei`、`bl`、`sb`、`nt`、`iv` | bool | QDP 品质位，见[附录 A](#附录-a品质描述)。 |
| `msec` | uint16 | 事件经过时间，单位毫秒。 |
| `time` | string | 信息体时标。 |

### 3.11 继电保护成组输出电路信息 `PackedOutputCircuitInfoInfo`

适用 ASDU：`M_EP_TC_1`、`M_EP_TF_1`。

```json
{
  "ioa": 6003,
  "oci": 10,
  "gc": false,
  "cl1": true,
  "cl2": false,
  "cl3": true,
  "qdp": 0,
  "qdpDesc": "QDP(00000000)[]",
  "ei": false,
  "bl": false,
  "sb": false,
  "nt": false,
  "iv": false,
  "msec": 120,
  "time": "2026-06-05 14:30:00.000000"
}
```

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `ioa` | uint | 信息对象地址。 |
| `oci` | byte | 输出电路信息位串，见[附录 D](#附录-d输出电路信息)。 |
| `gc` | bool | 总命令输出至输出电路。 |
| `cl1` | bool | A 相保护命令输出至输出电路。 |
| `cl2` | bool | B 相保护命令输出至输出电路。 |
| `cl3` | bool | C 相保护命令输出至输出电路。 |
| `qdp`、`qdpDesc` | byte、string | 保护事件品质描述。 |
| `ei`、`bl`、`sb`、`nt`、`iv` | bool | QDP 品质位，见[附录 A](#附录-a品质描述)。 |
| `msec` | uint16 | 事件经过时间，单位毫秒。 |
| `time` | string | 信息体时标。 |

### 3.12 带变位检出的成组单点信息 `PackedSinglePointWithSCDInfo`

适用 ASDU：`M_PS_NA_1`。

```json
{
  "ioa": 7001,
  "scd": 65295,
  "stn": "1111111100001111",
  "cdn": "0000000000000000",
  "qds": 0,
  "qdsDesc": "QDS(00000000)[]",
  "ov": false,
  "bl": false,
  "sb": false,
  "nt": false,
  "iv": false
}
```

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `ioa` | uint | 信息对象地址。 |
| `scd` | uint32 | 状态和状态变位检出，见[附录 C](#附录-c状态变位检出)。 |
| `stn` | string | `scd` 低 16 位的二进制字符串，表示当前状态。 |
| `cdn` | string | `scd` 高 16 位的二进制字符串，表示变位检出。 |
| `qds`、`qdsDesc` | byte、string | 品质描述。 |
| `ov`、`bl`、`sb`、`nt`、`iv` | bool | 品质位，见[附录 A](#附录-a品质描述)。 |

## 4. 附录

### 附录 A：品质描述

#### QDS

| 字段 | 位 | 说明 |
| --- | --- | --- |
| `ov` | bit 0 | Overflow，`true` 表示溢出。 |
| `bl` | bit 4 | Blocked，`true` 表示闭锁。 |
| `sb` | bit 5 | Substituted，`true` 表示取代。 |
| `nt` | bit 6 | NotTopical，`true` 表示非当前值。 |
| `iv` | bit 7 | Invalid，`true` 表示无效。 |

#### QDP

| 字段 | 位 | 说明 |
| --- | --- | --- |
| `ei` | bit 3 | ElapsedTimeInvalid，`true` 表示动作时间无效。 |
| `bl` | bit 4 | Blocked，`true` 表示闭锁。 |
| `sb` | bit 5 | Substituted，`true` 表示取代。 |
| `nt` | bit 6 | NotTopical，`true` 表示非当前值。 |
| `iv` | bit 7 | Invalid，`true` 表示无效。 |

### 附录 B：单点和双点状态值

| 值 | 单点含义 | 双点含义 |
| --- | --- | --- |
| 0 | 分或未动作 | 中间或不确定状态。 |
| 1 | 合或动作 | 开。 |
| 2 | 不适用 | 合。 |
| 3 | 不适用 | 不确定状态。 |

### 附录 C：状态变位检出

`scd` 是 32 位比特掩码：

| 位域 | 说明 |
| --- | --- |
| bit 0 至 bit 15 | 当前状态，16 个单点状态。 |
| bit 16 至 bit 31 | 变位检出，对应位为 1 表示发生变化。 |

示例：`scd = 0x0000FF0F`。

```go
scd := uint32(0x0000FF0F)
currentStatus := scd & 0xFFFF
statusChange := (scd >> 16) & 0xFFFF
```

解析结果：低 16 位为 `1111111100001111`，位 0 至 3 和位 8 至 15 为闭合状态；高 16 位为 0，无变位检出。

### 附录 D：输出电路信息

`oci` 是 8 位比特掩码。ieccaller 会按以下位掩码展开为 `gc`、`cl1`、`cl2`、`cl3`。

| 掩码值 | 字段 | 说明 |
| --- | --- | --- |
| 1 | `gc` | 总命令输出至输出电路。 |
| 2 | `cl1` | A 相保护命令输出至输出电路。 |
| 4 | `cl2` | B 相保护命令输出至输出电路。 |
| 8 | `cl3` | C 相保护命令输出至输出电路。 |

### 附录 E：公共地址规则

- 常规范围：`1` 至 `65534`。
- 全局地址：`65535`。
- 用途：标识 RTU、子站或子系统。

### 附录 F：业务单位枚举参考

以下枚举用于业务建模和点表配置，不属于 IEC 104 原始 ASDU 字段。

#### 遥测单位 `YcUnit`

| 枚举值 | 中文名称 | 符号 | 示例用途 |
| --- | --- | --- | --- |
| `VOLT` | 交流电压 | V | 电网电压。 |
| `VOLTAGE_DC` | 直流电压 | Vdc | 电池电压。 |
| `KILOVOLT` | 千伏 | kV | 高压线路。 |
| `AMPERE` | 电流 | A | 负载电流。 |
| `WATT` | 有功功率 | W | 瞬时功率。 |
| `VOLT_AMP_REACTIVE` | 无功功率 | var | 无功功率。 |
| `KILOWATT` | 千瓦 | kW | 有功功率。 |
| `KILOVOLT_AMP` | 千伏安 | kVA | 视在功率。 |
| `HERTZ` | 频率 | Hz | 电网频率。 |
| `DEGREE_CELSIUS` | 摄氏温度 | ℃ | 设备温度。 |
| `PERCENT` | 百分比 | % | 负载率或 SOC。 |
| `POWER_FACTOR` | 功率因数 | λ | 功率因数。 |
| `SECOND` | 秒 | s | 状态持续时长。 |
| `DIGITAL_STATUS` | 状态映射值 | 0/1 | 兼容旧系统。 |
| `NONE` | 无单位 | 空 | 自定义量。 |

#### 遥脉单位 `YmUnit`

| 枚举值 | 中文名称 | 符号 | 示例用途 |
| --- | --- | --- | --- |
| `KWH` | 有功电能 | kWh | 正向有功电度。 |
| `KVARH` | 无功电能 | kvarh | 无功电度。 |
| `MWH` | 兆瓦时 | MWh | 发电量统计。 |
| `M3` | 累计体积 | m³ | 燃气表或水表。 |
| `LITER` | 液体体积 | L | 液体流量。 |
| `COUNTER` | 累计次数 | cnt | 动作次数。 |
| `SECOND` | 累计时间，秒 | s | 运行秒数。 |
| `MINUTE` | 累计时间，分 | min | 停机时长。 |
| `HOUR` | 累计时间，时 | h | 运行小时。 |
| `NONE` | 无单位 | 空 | 自定义量。 |

#### 遥信单位 `YxUnit`

| 枚举值 | 中文名称 | 符号或取值 | 示例用途 |
| --- | --- | --- | --- |
| `DOUBLE_POINT` | 双点状态 | 0、1、2、3 | 断路器、隔离开关。 |
| `SWITCH_STATUS` | 单点分合闸 | 0、1 | 普通开关设备。 |
| `GENERIC_BOOL` | 通用状态 | 0、1、2 | 通用设备状态。 |
| `ALARM_STATUS` | 报警状态 | 0、1 | 故障信号。 |
| `CONTROL_MODE` | 控制权限 | 0、1 | 操作模式。 |
| `FAULT_STATUS` | 故障标志 | 0、1 | 系统级故障。 |
| `DOOR_STATUS` | 门禁状态 | 0、1 | 机柜门状态。 |
| `COMMUNICATION` | 通信状态 | 0、1 | 通信链路。 |
| `NONE` | 无单位 | 空 | 预留。 |

## 5. 数据消费指南

### 5.1 唯一键生成规则

ieccaller 生成 Kafka key 时使用 `MsgBody.GetKey()`，格式为：

```text
{host}_{coa}_{ioaHex}
```

`ioaHex` 使用 6 位十六进制小写格式。

```python
host = "127.0.0.1"
coa = 1
ioa = 2001
key = f"{host}_{coa}_0x{ioa:06x}"
print(key)  # 127.0.0.1_1_0x0007d1
```

### 5.2 Kafka 消费建议

- 使用消息 key 保证同一 `host`、`coa`、`ioa` 的事件落在同一分区。
- 以 `msgId` 做幂等处理，避免重试造成重复写入。
- 按 `typeId` 或 `asdu` 分流，再按 `body.ioa` 更新点位状态。

### 5.3 MQTT 消费建议

- Topic 可由 `pm`、`asdu`、`coa` 等字段模板生成。
- 消费者不能假设 `pm` 一定存在。没有点位映射时，仍可使用 `host`、`coa`、`body.ioa` 识别点位。
- 多 Topic 推送时，同一条消息的 JSON 内容一致。

### 5.4 gRPC 流事件消费建议

- gRPC 推送内容与 Kafka、MQTT 的 JSON 消息保持一致。
- 批量或流式消费方应逐条解析 `body`，不要按批次共享同一个信息体类型。

### 5.5 时区与时标处理

- `time` 为推送时间，UTC+8，微秒级字符串。
- `body.time` 为设备侧 ASDU 时标，只有带时标 ASDU 才有有效值。
- 下游排序优先使用 `body.time`。如果为空，再使用消息级 `time`。

### 5.6 异常数据处理

- 对带 QDS 的信息体，重点检查 `iv`、`ov`、`bl`、`nt`。
- 对累计量，重点检查 `value.isInvalid`、`value.hasCarry`、`value.isAdjusted`。
- 对保护事件，重点检查 QDP 展开的 `iv`、`ei`。
- 点表含义优先于通用说明。双点状态、比特串、SCD、保护事件位含义都可能因设备点表不同而变化。

### 5.7 Python 解析示例

```python
import json

raw = '''{
  "msgId": "msg-001",
  "host": "127.0.0.1",
  "port": 2404,
  "asdu": "M_SP_NA_1",
  "typeId": 1,
  "dataType": 0,
  "coa": 1,
  "body": {
    "ioa": 2001,
    "value": true,
    "qds": 0,
    "qdsDesc": "QDS(00000000)[]",
    "ov": false,
    "bl": false,
    "sb": false,
    "nt": false,
    "iv": false,
    "time": ""
  },
  "time": "2026-06-05 14:30:00.000000"
}'''

msg = json.loads(raw)
body = msg["body"]
point_key = f'{msg["host"]}_{msg["coa"]}_0x{body["ioa"]:06x}'

if body.get("iv"):
    raise ValueError(f"invalid IEC104 value: {point_key}")

print(point_key, body["value"])
```

## 6. 技术支持

- **文档版本**：v1.4.0（2026-06-05）
- **协议版本**：IEC 60870-5-104
- **联系支持**：[hehanpengyy@163.com](mailto:hehanpengyy@163.com)

注意：实际解析时需严格参照设备点表定义。部分字段，如双点信息 `value`、比特串 `value`、保护事件 `event`，具体含义可能因设备而异。

相关文档：

- [IEC 104 数采平台架构](./iec104.md)，服务组件、数据流、配置管理。
- [`ieccaller.proto`](../app/ieccaller/ieccaller.proto)，控制指令 RPC 接口。
- [`streamevent.proto`](../facade/streamevent/streamevent.proto)，统一流事件 gRPC 协议。

## 7. 控制命令接口

本节描述如何通过 ieccaller 向 IEC 104 从站下发控制命令。控制方向为主站到从站，与[第 2 节](#2-asdu-类型映射表)的监视方向相反。

### 7.1 接入方式

| 项目 | 说明 |
| --- | --- |
| 协议 | gRPC，支持 Endpoints 直连或 Nacos 服务发现。 |
| 服务名 | `ieccaller.IecCaller` |
| Proto | [`app/ieccaller/ieccaller.proto`](../app/ieccaller/ieccaller.proto) |
| 响应 | 7 个带类型命令返回各自类型的响应，包含从站回显的命令值。 |

### 7.2 推荐接口总览

7 个带类型接口封装了常用控制命令。调用方只需传 `host`、`port`、`coa`、`ioa`、`value`、`withTime`，无需手动查 `typeId`。

| RPC | value 类型 | withTime=false | withTime=true | 默认限定词 |
| --- | --- | --- | --- | --- |
| `SendSingleCommand` | bool | 45 `C_SC_NA_1` | 58 `C_SC_TA_1` | QOC 无附加定义，直接执行。 |
| `SendDoubleCommand` | `DoubleCommandValue` | 46 `C_DC_NA_1` | 59 `C_DC_TA_1` | QOC 无附加定义，直接执行。 |
| `SendStepCommand` | `StepCommandValue` | 47 `C_RC_NA_1` | 60 `C_RC_TA_1` | QOC 无附加定义，直接执行。 |
| `SendSetpointNormalized` | int32 | 48 `C_SE_NA_1` | 61 `C_SE_TA_1` | QOS `Qual=0`，直接执行。 |
| `SendSetpointScaled` | int32 | 49 `C_SE_NB_1` | 62 `C_SE_TB_1` | QOS `Qual=0`，直接执行。 |
| `SendSetpointFloat` | double | 50 `C_SE_NC_1` | 63 `C_SE_TC_1` | QOS `Qual=0`，直接执行。 |
| `SendBitstringCommand` | uint64 | 51 `C_BO_NA_1` | 64 `C_BO_TA_1` | 无限定词字段。 |

`withTime=true` 时，服务端使用当前时间填充 CP56Time2a 时标。所有带类型接口都采用直接执行，未启用选择执行。

### 7.3 公共请求字段

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `host` | string | 从站 IP。 |
| `port` | uint32 | 从站端口。 |
| `coa` | uint32 | 公共地址。 |
| `ioa` | uint32 | 信息对象地址。 |
| `value` | 各接口不同 | 命令值。 |
| `withTime` | bool | `false` 使用不带时标 TypeId，`true` 使用带 CP56Time2a 时标 TypeId。 |

### 7.4 单点命令 `SendSingleCommand`

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

### 7.5 双点命令 `SendDoubleCommand`

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

### 7.6 档位调节命令 `SendStepCommand`

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

### 7.7 归一化设点 `SendSetpointNormalized`

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

### 7.8 标度化设点 `SendSetpointScaled`

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

### 7.9 浮点设点 `SendSetpointFloat`

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
| `value` | double | gRPC 使用 double，发送前转换为 IEC 104 短浮点数 float32。 |

### 7.10 32 位位串命令 `SendBitstringCommand`

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

### 7.11 通用接口 `SendCommand`

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

常用控制 TypeId 见[第 7.14 节](#714-全量-typeid-双向对照表)。

### 7.12 其他控制接口

| RPC | TypeId | 说明 |
| --- | --- | --- |
| `SendInterrogationCmd` | 100 `C_IC_NA_1` | 总召唤。 |
| `SendCounterInterrogationCmd` | 101 `C_CI_NA_1` | 累计量召唤。 |
| `SendReadCmd` | 102 `C_RD_NA_1` | 读命令，需要 `ioa`。 |
| `SendTestCmd` | 107 `C_TS_TA_1` | 测试命令。 |

### 7.12.1 点位缓存管理接口

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

### 7.13 响应和回执机制

7 个带类型控制命令的 gRPC 响应包含从站回显的命令值（如 `SendSingleCommandRes.value`）。gRPC 调用成功只表示命令已由 ieccaller 发送到本地 IEC 104 client 或已在集群模式下广播给其他实例，不等于设备已执行成功。

**响应类型对照表**：

| RPC | 响应类型 | 响应值说明 |
| --- | --- | --- |
| `SendSingleCommand` | `SendSingleCommandRes` | `bool value` - 从站回显的单点命令值 |
| `SendDoubleCommand` | `SendDoubleCommandRes` | `DoubleCommandValue value` - 从站回显的双点命令值 |
| `SendStepCommand` | `SendStepCommandRes` | `int32 value` - 从站回显的档位调节命令值 |
| `SendSetpointNormalized` | `SendSetpointNormalizedRes` | `int32 value` - 从站回显的归一化设点值 |
| `SendSetpointScaled` | `SendSetpointScaledRes` | `int32 value` - 从站回显的标度化设点值 |
| `SendSetpointFloat` | `SendSetpointFloatRes` | `double value` - 从站回显的短浮点设点值 |
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

### 7.13.1 错误码

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

### 7.14 全量 TypeId 双向对照表

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
