# IEC 104 消息对接文档

本文档定义 ieccaller 推送的 IEC 60870-5-104 消息格式规范，供下游消费者（Kafka/MQTT/gRPC）对接使用。

平台架构、服务组件、配置管理、部署等内容参见：[IEC 104 数采平台](./iec104.md)

---

## 1. 消息格式规范

### 基础协议

- **传输协议**：IEC 60870-5-104
- **传输载体**：Kafka/MQTT Topic（具体名称协商约定）
- **数据格式**：JSON
- **编码格式**：UTF-8

### 消息结构

```json
{
  "msgId": "消息id",
  "host": "从站 ip",
  "port": 2404,
  "asdu": "M_SP_NA_1",
  "typeId": 1,
  "dataType": 0,
  "coa": 1001,
  "body": {
    /* 信息体结构（不同typeId对应不同结构） */
  },
  "time": "2023-10-01 14:30:00.000000",
  "metaData": {
    "key": "应用ID",
    "array": [
      "1",
      "2"
    ]
  },
  "pm": {
    "deviceId": "device_123",
    "deviceName": "测试设备",
    "tdTableType": "yx, yc",
    "ext1": "alarm",
    "ext2": "",
    "ext3": "",
    "ext4": "",
    "ext5": ""
  }
}
```

| 字段       | 类型     | 说明                                               |
|----------|--------|--------------------------------------------------|
| msgId    | String | 消息id                                             |
| host     | String | 采集设备地址                                           |
| port     | int    | 采集设备端口号                                          |
| asdu     | String | ASDU类型名称                                         |
| typeId   | int    | ASDU类型标识符                                        |
| dataType | int    | 信息体类型标识符                                         |
| coa      | uint   | 公共地址（范围：1-65534，全局地址65535保留）                     |
| body     | Object | 信息体对象（结构随typeId变化）                               |
| time     | String | 消息推送时间戳（格式：`YYYY-MM-DD HH:mm:ss.SSSSSS`，UTC+8时区） |
| metaData | Object | 应用级元数据（如：应用ID、用户信息、场站信息等）                        |
| pm       | Object | 点位映射信息，包含设备ID、名称和扩展字段，用于动态生成Topic                 |

### PointMapping 结构详细说明

PointMapping包含设备的详细信息和扩展字段，用于动态生成MQTT Topic和业务逻辑处理。

```json
{
  "deviceId": "device_123",
  "deviceName": "测试设备",
  "tdTableType": "yx, yc",
  "ext1": "alarm",
  "ext2": "",
  "ext3": "",
  "ext4": "",
  "ext5": ""
}
```

| 字段名         | 类型     | 说明                                         |
|-------------|--------|--------------------------------------------|
| deviceId    | String | 设备唯一标识符，用于设备识别和关联                          |
| deviceName  | String | 设备名称，用于业务显示和查询                              |
| tdTableType | String | TDengine表类型，逗号分隔，如：yx, yc，用于数据存储归类           |
| ext1        | String | 扩展字段1，用于主题拆分，如：alarm, normal, control等        |
| ext2        | String | 扩展字段2，用于主题拆分和业务逻辑                            |
| ext3        | String | 扩展字段3，用于主题拆分和业务逻辑                            |
| ext4        | String | 扩展字段4，用于主题拆分和业务逻辑                            |
| ext5        | String | 扩展字段5，用于主题拆分和业务逻辑                            |

### 扩展字段用途

扩展字段（ext1-ext5）主要用于主题拆分，允许根据业务需求灵活生成不同维度的MQTT Topic,也可以透传到 facade 层进行业务处理。

- **业务类型**：如alarm（告警）、normal（正常）、control（控制）
- **设备类型**：如switch（开关）、sensor（传感器）、meter（仪表）
- **区域划分**：如area1、building1、floor1
- **功能模块**：如power（电源）、communication（通信）、monitor（监控）
- **数据类型**：如analog（模拟量）、digital（数字量）、counter（计数器）

**示例**：
- Topic模板：`{{.Pm.DeviceId}}/{{.Pm.Ext1}}/{{.asdu}}`
- 实际生成：`device_123/alarm/M_SP_NA_1`

---

## 1.1 数据库表结构

### device_point_mapping 表

该表用于存储设备点位映射关系，是弱校验模式和动态Topic生成的核心配置。

```sql
CREATE TABLE IF NOT EXISTS device_point_mapping (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    create_time TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    update_time TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    delete_time TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    del_state INTEGER NOT NULL DEFAULT 0,
    version INTEGER NOT NULL DEFAULT 0,
    
    tag_station VARCHAR(64) NOT NULL DEFAULT '', -- 与 TDengine tag_station 对应
    coa INTEGER NOT NULL DEFAULT 0,              -- 与 TDengine coa 对应
    ioa INTEGER NOT NULL DEFAULT 0,              -- 与 TDengine ioa 对应
    device_id VARCHAR(64) NOT NULL DEFAULT '',   -- 设备编号/ID
    device_name VARCHAR(128) NOT NULL DEFAULT '',-- 设备名称
    td_table_type VARCHAR(255) NOT NULL DEFAULT '',-- TDengine表类型
    enable_push INTEGER NOT NULL DEFAULT 1,      -- 是否允许caller服务推送数据：0-不允许，1-允许
    enable_raw_insert INTEGER NOT NULL DEFAULT 1,-- 是否允许插入到raw原生数据中：0-不允许，1-允许
    description VARCHAR(256) NOT NULL DEFAULT '',-- 备注信息
    
    -- 扩展字段，用于存储额外的元数据
    ext_1 VARCHAR(64) NOT NULL DEFAULT '',      -- 扩展字段1，如：alarm, normal, control等
    ext_2 VARCHAR(64) NOT NULL DEFAULT '',      -- 扩展字段2
    ext_3 VARCHAR(64) NOT NULL DEFAULT '',      -- 扩展字段3
    ext_4 VARCHAR(64) NOT NULL DEFAULT '',      -- 扩展字段4
    ext_5 VARCHAR(64) NOT NULL DEFAULT '',      -- 扩展字段5
    
    UNIQUE(tag_station, coa, ioa)               -- 唯一索引，保证同一个点位只对应一个设备
);
```

**核心字段说明**：

| 字段名            | 类型     | 说明                                                 |
|----------------|--------|----------------------------------------------------|
| tag_station    | String | 与TDengine的tag_station对应，用于设备分组                     |
| coa            | int    | 公共地址，与IEC 104协议中的公共地址对应                            |
| ioa            | int    | 信息对象地址，与IEC 104协议中的信息对象地址对应                        |
| device_id      | String | 设备唯一标识符，映射到PointMapping.DeviceId                   |
| device_name    | String | 设备名称，映射到PointMapping.DeviceName                    |
| td_table_type  | String | TDengine表类型，逗号分隔，如：yx, yc                          |
| enable_push    | int    | 是否允许推送数据：0-不允许，1-允许，弱校验模式的核心控制字段                   |
| enable_raw_insert | int | 是否允许插入到raw原生数据中：0-不允许，1-允许                         |
| ext_1-ext_5    | String | 扩展字段，映射到PointMapping.Ext1-Ext5，用于动态生成Topic和自定义业务逻辑 |

**索引说明**：
- 唯一索引：`UNIQUE(tag_station, coa, ioa)`，保证同一个点位只对应一个设备配置

---

## 2. 全量ASDU类型映射表

| DataType | TypeId | ASDU类型    | Body结构体                                      | 应用场景说明                        |
|----------|--------|-----------|----------------------------------------------|-------------------------------|
| 0        | 1      | M_SP_NA_1 | `SinglePointInfo`                            | 单点遥信（不带时标）                    |
| 0        | 2      | M_SP_TA_1 | `SinglePointInfo`                            | 单点遥信（带时标）                     |
| 0        | 30     | M_SP_TB_1 | `SinglePointInfo`                            | 单点遥信（CP56Time2a时标）            |
| 1        | 3      | M_DP_NA_1 | `DoublePointInfo`                            | 双点遥信（不带时标）                    |
| 1        | 4      | M_DP_TA_1 | `DoublePointInfo`                            | 双点遥信（带时标）                     |
| 1        | 31     | M_DP_TB_1 | `DoublePointInfo`                            | 双点遥信（CP56Time2a时标）            |
| 2        | 11     | M_ME_NB_1 | `MeasuredValueScaledInfo`                    | 标度化遥测值（不带时标）                  |
| 2        | 12     | M_ME_TB_1 | `MeasuredValueScaledInfo`                    | 标度化遥测值（带时标）                   |
| 2        | 35     | M_ME_TE_1 | `MeasuredValueScaledInfo`                    | 标度化遥测值（CP56Time2a时标）          |
| 3        | 9      | M_ME_NA_1 | `MeasuredValueNormalInfo`                    | 规一化遥测值（不带时标）                  |
| 3        | 10     | M_ME_TA_1 | `MeasuredValueNormalInfo`                    | 规一化遥测值（带时标）                   |
| 3        | 34     | M_ME_TD_1 | `MeasuredValueNormalInfo`                    | 规一化遥测值（CP56Time2a时标）          |
| 3        | 21     | M_ME_ND_1 | `MeasuredValueNormalInfo`                    | 无品质描述的规一化遥测值                  |
| 4        | 5      | M_ST_NA_1 | `StepPositionInfo`                           | 步位置信息（不带时标）                   |
| 4        | 6      | M_ST_TA_1 | `StepPositionInfo`                           | 步位置信息（带时标）                    |
| 4        | 32     | M_ST_TB_1 | `StepPositionInfo`                           | 步位置信息（CP56Time2a时标）           |
| 5        | 7      | M_BO_NA_1 | `BitString32Info`                            | 32位比特串（不带时标）                  |
| 5        | 8      | M_BO_TA_1 | `BitString32Info`                            | 32位比特串（带时标）                   |
| 5        | 33     | M_BO_TB_1 | `BitString32Info`                            | 32位比特串（CP56Time2a时标）          |
| 6        | 13     | M_ME_NC_1 | `MeasuredValueFloatInfo`                     | 短浮点数遥测值（不带时标）                 |
| 6        | 14     | M_ME_TC_1 | `MeasuredValueFloatInfo`                     | 短浮点数遥测值（带时标）                  |
| 6        | 36     | M_ME_TF_1 | `MeasuredValueFloatInfo`                     | 短浮点数遥测值（CP56Time2a时标）         |
| 7        | 15     | M_IT_NA_1 | `BinaryCounterReadingInfo`                   | 累计量（不带时标）                     |
| 7        | 16     | M_IT_TA_1 | `BinaryCounterReadingInfo`                   | 累计量（带时标）                      |
| 7        | 37     | M_IT_TB_1 | `BinaryCounterReadingInfo`                   | 累计量（CP56Time2a时标）             |
| 8        | 17     | M_EP_TA_1 | `EventOfProtectionEquipmentInfo`             | 继电保护事件（带时标）                   |
| 8        | 38     | M_EP_TD_1 | `EventOfProtectionEquipmentInfo`             | 继电保护事件（CP56Time2a时标）          |
| 9        | 18     | M_EP_TB_1 | `PackedStartEventsOfProtectionEquipmentInfo` | 继电器保护设备成组启动事件（带时标）            |
| 9        | 39     | M_EP_TE_1 | `PackedStartEventsOfProtectionEquipmentInfo` | 继电器保护设备成组启动事件（CP56Time2a时标）   |
| 10       | 19     | M_EP_TC_1 | `PackedOutputCircuitInfoInfo`                | 继电器保护设备成组输出电路信息（带时标）          |
| 10       | 40     | M_EP_TF_1 | `PackedOutputCircuitInfoInfo`                | 继电器保护设备成组输出电路信息（CP56Time2a时标） |
| 11       | 20     | M_PS_NA_1 | `PackedSinglePointWithSCDInfo`               | 带变位检出的成组单点信息                  |
| 19       | 0      |           | `UNKNOWN`                                    | UNKNOWN 不发送                   |

---

## 3. 信息体结构详解

### 3.1 单点信息（SinglePointInfo）

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
  "time": "2023-10-01 14:30:00.000000"
}
```

| 字段      | 类型     | 说明                                     |
|---------|--------|----------------------------------------|
| ioa     | uint   | 信息对象地址（范围：`0x000001`-`0xFFFFFF`，十进制显示） |
| value   | bool   | `true`=合/动作,`false`=分/未动作              |
| qds     | byte   | 品质                                     |
| qdsDesc | string | 品质描述                                   |
| ov      | bool   | Overflow `true`=溢出,`false`=未溢出         |
| bl      | bool   | Blocked `true`=闭锁,`false`=未闭锁          |
| sb      | bool   | Substituted `true`=取代,`false`=未取代      |
| nt      | bool   | NotTopical `true`=非当前值,`false`=当前值     |
| iv      | bool   | Invalid `true`=无效,`false`=有效           |
| time    | string | 时标（仅带时标的ASDU类型包含此字段）                   |

---

### 3.2 双点信息（DoublePointInfo）

```json
{
  "ioa": 2002,
  "value": 0,
  "qds": 160,
  "qdsDesc": "QDS(10100000)[Substituted|Invalid]",
  "ov": false,
  "bl": false,
  "sb": true,
  "nt": false,
  "iv": true,
  "time": "2023-10-01 14:30:00.000000"
}
```

| 字段      | 类型     | 说明                                     |
|---------|--------|----------------------------------------|
| ioa     | uint   | 信息对象地址（范围：`0x000001`-`0xFFFFFF`，十进制显示） |
| value   | byte   | `0`=不确定或中间状态,`1`=开,`2`=合,`3`=不确定       |
| qds     | byte   | 品质                                     |
| qdsDesc | string | 品质描述                                   |
| ov      | bool   | Overflow `true`=溢出,`false`=未溢出         |
| bl      | bool   | Blocked `true`=闭锁,`false`=未闭锁          |
| sb      | bool   | Substituted `true`=取代,`false`=未取代      |
| nt      | bool   | NotTopical `true`=非当前值,`false`=当前值     |
| iv      | bool   | Invalid `true`=无效,`false`=有效           |
| time    | string | 时标（仅带时标的ASDU类型包含此字段）                   |

---

### 3.3 步位置信息（StepPositionInfo）

```json
{
  "ioa": 3001,
  "value": {
    "val": 63,
    "hasTransient": false
  },
  "qds": 160,
  "qdsDesc": "QDS(10100000)[Substituted|Invalid]",
  "ov": false,
  "bl": false,
  "sb": true,
  "nt": false,
  "iv": true,
  "time": "2023-10-01 14:30:00.000000"
}
```

| 字段             | 类型     | 说明                                     |
|----------------|--------|----------------------------------------|
| ioa            | uint   | 信息对象地址（范围：`0x000001`-`0xFFFFFF`，十进制显示） |
| value          | object | 步位置值对象                                 |
| ├ val          | int    | 步位置值（范围：`-64` 至 `63`）                  |
| ├ hasTransient | bool   | `true`=设备处于瞬变状态                        |
| qds            | byte   | 品质                                     |
| qdsDesc        | string | 品质描述                                   |
| ov             | bool   | Overflow `true`=溢出,`false`=未溢出         |
| bl             | bool   | Blocked `true`=闭锁,`false`=未闭锁          |
| sb             | bool   | Substituted `true`=取代,`false`=未取代      |
| nt             | bool   | NotTopical `true`=非当前值,`false`=当前值     |
| iv             | bool   | Invalid `true`=无效,`false`=有效           |
| time           | string | 时标（仅带时标的ASDU类型包含此字段）                   |

---

### 3.4 规一化遥测值（MeasuredValueNormalInfo）

```json
{
  "ioa": 4001,
  "value": 16384,
  "nva": 0.7355652,
  "qds": 160,
  "qdsDesc": "QDS(10100000)[Substituted|Invalid]",
  "ov": false,
  "bl": false,
  "sb": true,
  "nt": false,
  "iv": true,
  "time": "2023-10-01 14:30:00.000000"
}
```

| 字段      | 类型      | 说明                                       |
|---------|---------|------------------------------------------|
| ioa     | uint    | 信息对象地址（范围：`0x000001`-`0xFFFFFF`，十进制显示）   |
| value   | int16   | 原始归一化值（范围：`-32768` 至 `32767`，需按公式转换为工程值） |
| nva     | float32 | 规一化值 默认公式 f归一= 32768 * f真实 / 满码值         |
| qds     | byte    | 品质                                       |
| qdsDesc | string  | 品质描述                                     |
| ov      | bool    | Overflow `true`=溢出,`false`=未溢出           |
| bl      | bool    | Blocked `true`=闭锁,`false`=未闭锁            |
| sb      | bool    | Substituted `true`=取代,`false`=未取代        |
| nt      | bool    | NotTopical `true`=非当前值,`false`=当前值       |
| iv      | bool    | Invalid `true`=无效,`false`=有效             |
| time    | string  | 时标（仅带时标的ASDU类型包含此字段）                     |

---

### 3.5 标度化遥测值（MeasuredValueScaledInfo）

```json
{
  "ioa": 4001,
  "value": 16384,
  "qds": 160,
  "qdsDesc": "QDS(10100000)[Substituted|Invalid]",
  "ov": false,
  "bl": false,
  "sb": true,
  "nt": false,
  "iv": true,
  "time": "2023-10-01 14:30:00.000000"
}
```

| 字段      | 类型     | 说明                                     |
|---------|--------|----------------------------------------|
| ioa     | uint   | 信息对象地址（范围：`0x000001`-`0xFFFFFF`，十进制显示） |
| value   | int16  | 标度化值                                   |
| qds     | byte   | 品质                                     |
| qdsDesc | string | 品质描述                                   |
| ov      | bool   | Overflow `true`=溢出,`false`=未溢出         |
| bl      | bool   | Blocked `true`=闭锁,`false`=未闭锁          |
| sb      | bool   | Substituted `true`=取代,`false`=未取代      |
| nt      | bool   | NotTopical `true`=非当前值,`false`=当前值     |
| iv      | bool   | Invalid `true`=无效,`false`=有效           |
| time    | string | 时标（仅带时标的ASDU类型包含此字段）                   |

---

### 3.6 比特位串信息（BitString32Info）

```json
{
  "ioa": 4002,
  "value": 220,
  "qds": 160,
  "qdsDesc": "QDS(10100000)[Substituted|Invalid]",
  "ov": false,
  "bl": false,
  "sb": true,
  "nt": false,
  "iv": true,
  "time": "2023-10-01 14:30:00.000000"
}
```

| 字段      | 类型     | 说明                                     |
|---------|--------|----------------------------------------|
| ioa     | uint   | 信息对象地址（范围：`0x000001`-`0xFFFFFF`，十进制显示） |
| value   | uint32 | 32 个独立设备状态（如开关、传感器、继电器），每个比特位对应一个设备    |
| qds     | byte   | 品质                                     |
| qdsDesc | string | 品质描述                                   |
| ov      | bool   | Overflow `true`=溢出,`false`=未溢出         |
| bl      | bool   | Blocked `true`=闭锁,`false`=未闭锁          |
| sb      | bool   | Substituted `true`=取代,`false`=未取代      |
| nt      | bool   | NotTopical `true`=非当前值,`false`=当前值     |
| iv      | bool   | Invalid `true`=无效,`false`=有效           |
| time    | string | 时标（仅带时标的ASDU类型包含此字段）                   |

---

### 3.7 短浮点数遥测值（MeasuredValueFloatInfo）

```json
{
  "ioa": 4002,
  "value": 220.5,
  "qds": 160,
  "qdsDesc": "QDS(10100000)[Substituted|Invalid]",
  "ov": false,
  "bl": false,
  "sb": true,
  "nt": false,
  "iv": true,
  "time": "2023-10-01 14:30:00.000000"
}
```

| 字段      | 类型      | 说明                                     |
|---------|---------|----------------------------------------|
| ioa     | uint    | 信息对象地址（范围：`0x000001`-`0xFFFFFF`，十进制显示） |
| value   | float32 | 短浮点数值（直接为工程值，如电压、电流等）                  |
| qds     | byte    | 品质                                     |
| qdsDesc | string  | 品质描述                                   |
| ov      | bool    | Overflow `true`=溢出,`false`=未溢出         |
| bl      | bool    | Blocked `true`=闭锁,`false`=未闭锁          |
| sb      | bool    | Substituted `true`=取代,`false`=未取代      |
| nt      | bool    | NotTopical `true`=非当前值,`false`=当前值     |
| iv      | bool    | Invalid `true`=无效,`false`=有效           |
| time    | string  | 时标（仅带时标的ASDU类型包含此字段）                   |

---

### 3.8 累计量（BinaryCounterReadingInfo）

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
  "time": "2023-10-01 14:30:00.000000"
}
```

| 字段               | 类型     | 说明                                     |
|------------------|--------|----------------------------------------|
| ioa              | uint   | 信息对象地址（范围：`0x000001`-`0xFFFFFF`，十进制显示） |
| value            | object | 累计量信息                                  |
| ├ counterReading | int32  | 计数器读数（32位有符号整数）                        |
| ├ seqNumber      | byte   | 顺序号（范围：`0`-`31`）                       |
| ├ hasCarry       | bool   | `true`=计数器溢出                           |
| ├ isAdjusted     | bool   | `true`=计数量被人工调整                        |
| ├ isInvalid      | bool   | `true`=数据无效                            |
| time             | string | 时标（仅带时标的ASDU类型包含此字段）                   |

---

### 3.9 继电保护事件（EventOfProtectionEquipmentInfo）

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
  "time": "2023-10-01 14:30:00.000000"
}
```

| 字段      | 类型     | 说明                                              |
|---------|--------|-------------------------------------------------|
| ioa     | uint   | 信息对象地址（范围：`0x000001`-`0xFFFFFF`，十进制显示）          |
| event   | byte   | 事件类型（见附录B）                                      |
| qdp     | byte   | 保护事件品质（见附录A）                                    |
| qdpDesc | string | 保护事件品质描述                                        |
| ei      | bool   | ElapsedTimeInvalid `true`=动作时间无效,`false`=动作时间有效 |
| bl      | bool   | Blocked `true`=闭锁,`false`=未闭锁                   |
| sb      | bool   | Substituted `true`=取代,`false`=未取代               |
| nt      | bool   | NotTopical `true`=非当前值,`false`=当前值              |
| iv      | bool   | Invalid `true`=无效,`false`=有效                    |
| msec    | uint16 | 事件发生的毫秒时间戳（范围：`0`-`59999`）                      |
| time    | string | 时标（仅带时标的ASDU类型包含此字段）                            |

---

### 3.10 继电器保护设备成组启动事件（PackedStartEventsOfProtectionEquipmentInfo）

```json
{
  "ioa": 6001,
  "event": 32,
  "qdp": 145,
  "qdpDesc": "QDP(10010001)[Blocked|Invalid]",
  "ei": false,
  "bl": true,
  "sb": false,
  "nt": false,
  "iv": true,
  "msec": 500,
  "time": "2023-10-01 14:30:00.000000"
}
```

| 字段      | 类型     | 说明                                              |
|---------|--------|-------------------------------------------------|
| ioa     | uint   | 信息对象地址（范围：`0x000001`-`0xFFFFFF`，十进制显示）          |
| event   | byte   | 事件类型（见附录C）                                      |
| qdp     | byte   | 保护事件品质（见附录A）                                    |
| qdpDesc | string | 保护事件品质描述                                        |
| ei      | bool   | ElapsedTimeInvalid `true`=动作时间无效,`false`=动作时间有效 |
| bl      | bool   | Blocked `true`=闭锁,`false`=未闭锁                   |
| sb      | bool   | Substituted `true`=取代,`false`=未取代               |
| nt      | bool   | NotTopical `true`=非当前值,`false`=当前值              |
| iv      | bool   | Invalid `true`=无效,`false`=有效                    |
| msec    | uint16 | 事件发生的毫秒时间戳（范围：`0`-`59999`）                      |
| time    | string | 时标（仅带时标的ASDU类型包含此字段）                            |

---

### 3.11 继电器保护设备成组输出电路信息（PackedOutputCircuitInfoInfo）

```json
{
  "ioa": 6001,
  "oci": 10,
  "gc": false,
  "cl1": true,
  "cl2": false,
  "cl3": true,
  "qdp": 145,
  "qdpDesc": "QDP(10010001)[Blocked|Invalid]",
  "ei": false,
  "bl": true,
  "sb": false,
  "nt": false,
  "iv": true,
  "msec": 500,
  "time": "2023-10-01 14:30:00.000000"
}
```

| 字段      | 类型     | 说明                                              |
|---------|--------|-------------------------------------------------|
| ioa     | uint   | 信息对象地址（范围：`0x000001`-`0xFFFFFF`，十进制显示）          |
| oci     | byte   | 输出电路信息（见附录C）                                    |
| gc      | bool   | `true`=总命令输出至输出电路,`false`=无总命令输出至输出电路           |
| cl1     | bool   | `true`=命令输出至A相输出电路,`false`=无命令输出至A相输出电路         |
| cl2     | bool   | `true`=命令输出至B相输出电路,`false`=无命令输出至B相输出电路         |
| cl3     | bool   | `true`=命令输出至C相输出电路,`false`=无命令输出至C相输出电路         |
| qdp     | byte   | 保护事件品质（见附录A）                                    |
| qdpDesc | string | 保护事件品质描述                                        |
| ei      | bool   | ElapsedTimeInvalid `true`=动作时间无效,`false`=动作时间有效 |
| bl      | bool   | Blocked `true`=闭锁,`false`=未闭锁                   |
| sb      | bool   | Substituted `true`=取代,`false`=未取代               |
| nt      | bool   | NotTopical `true`=非当前值,`false`=当前值              |
| iv      | bool   | Invalid `true`=无效,`false`=有效                    |
| msec    | uint16 | 事件发生的毫秒时间戳（范围：`0`-`59999`）                      |
| time    | string | 时标（仅带时标的ASDU类型包含此字段）                            |

---

### 3.12 带变位检出的成组单点信息（PackedSinglePointWithSCDInfo）

```json
{
  "ioa": 6001,
  "scd": 0,
  "stn": "0000000000000000",
  "cdn": "0000000000000000",
  "qds": 160,
  "qdsDesc": "QDS(10100000)[Substituted|Invalid]",
  "ov": false,
  "bl": false,
  "sb": true,
  "nt": false,
  "iv": true
}
```

| 字段      | 类型     | 说明                                     |
|---------|--------|----------------------------------------|
| ioa     | uint   | 信息对象地址（范围：`0x000001`-`0xFFFFFF`，十进制显示） |
| scd     | byte   | 状态变位检出（见附录F）                           |
| qds     | byte   | 品质                                     |
| qdsDesc | string | 品质描述                                   |
| ov      | bool   | Overflow `true`=溢出,`false`=未溢出         |
| bl      | bool   | Blocked `true`=闭锁,`false`=未闭锁          |
| sb      | bool   | Substituted `true`=取代,`false`=未取代      |
| nt      | bool   | NotTopical `true`=非当前值,`false`=当前值     |
| iv      | bool   | Invalid `true`=无效,`false`=有效           |

---

## 4. 附录

### 附录A：品质描述（QDS/QDP）

#### 1. QDS

| 字段 | 位号 | 说明                                 |
|----|----|------------------------------------|
| ov | 1  | Overflow `true`=溢出,`false`=未溢出     |
| bl | 5  | Blocked `true`=闭锁,`false`=未闭锁      |
| sb | 6  | Substituted `true`=取代,`false`=未取代  |
| nt | 7  | NotTopical `true`=非当前值,`false`=当前值 |
| iv | 8  | Invalid `true`=无效,`false`=有效       |

#### 1.QDP

| 字段 | 位号 | 说明                                              |
|----|----|-------------------------------------------------|
| ei | 4  | ElapsedTimeInvalid `true`=动作时间无效,`false`=动作时间有效 |
| bl | 5  | Blocked `true`=闭锁,`false`=未闭锁                   |
| sb | 6  | Substituted `true`=取代,`false`=未取代               |
| nt | 7  | NotTopical `true`=非当前值,`false`=当前值              |
| iv | 8  | Invalid `true`=无效,`false`=有效                    |

### 附录B：继电保护事件类型

| 值 | 事件类型     |
|---|----------|
| 0 | 不确定或中间状态 |
| 1 | 开        |
| 2 | 合        |
| 3 | 不确定      |

### 附录C：继电保护事件类型

#### 1. 字段定义

| 字段名 | 类型   | 位宽 | 描述                                     |
|:----|:-----|:---|:---------------------------------------|
| oci | byte | 8位 | 8位比特掩码，每一位表示一个输出电路的状态（`1`=启动,`0`=未启动）。 |

#### 2. 比特位映射规则

| 字段  | 位号 | 说明                                      |
|-----|----|-----------------------------------------|
| gc  | 1  | `true`=总命令输出至输出电路,`false`=无总命令输出至输出电路   |
| cl1 | 2  | `true`=命令输出至A相输出电路,`false`=无命令输出至A相输出电路 |
| cl2 | 3  | `true`=命令输出至B相输出电路,`false`=无命令输出至B相输出电路 |
| cl3 | 4  | `true`=命令输出至C相输出电路,`false`=无命令输出至C相输出电路 |

### 附录D：公共地址（COA）规则

- **范围**：`1`-`65534`（`65535`为全局广播地址）
- **用途**：标识RTU、子站或子系统

### 附录E：输出电路信息（OCI）定义

**概述**  
本附录定义了输出电路信息（OCI）的字段、比特位映射规则及示例解析。

#### 1. 字段定义

| 字段名   | 类型     | 位宽 | 描述                                                     |
|:------|:-------|:---|:-------------------------------------------------------|
| `oci` | `byte` | 8位 | 8位比特掩码，每一位表示一个输出电路的状态（`1`=无总命令输出至输出电路,`0`=总命令输出至输出电路）。 |

#### 2. 比特位映射规则

| 值 | 类型                | 描述             |
|---|-------------------|----------------|
| 1 | OCIGeneralCommand | 总命令输出至输出电路     |
| 2 | OCICommandL1      | A 相保护命令输出至输出电路 |
| 4 | OCICommandL2      | B 相保护命令输出至输出电路 |
| 8 | OCICommandL3      | C 相保护命令输出至输出电路 |

### 附录F：状态变位检出（SCD）定义

**概述**
本附录定义了状态变位检出（SCD）的字段、比特位结构及示例解析。

#### 1. 字段定义

| 字段名   | 类型       | 位宽  | 描述                                    |
|-------|----------|-----|---------------------------------------|
| `scd` | `uint32` | 32位 | 32位比特掩码，低16位表示当前状态，高16位表示变位检出（=状态变化）。 |

#### 2. 比特位结构

| 位域        | 说明                    |
|-----------|-----------------------|
| 位0 ~ 位15  | 当前状态（16个单点状态） 0-开 1-合 |
| 位16 ~ 位31 | 变位检出（对应位=1表示变化）       |

#### 3. 示例解析

假设 `scd = 0x0000FF0F`：

##### Go 示例代码：

``` go
package main

import (
	"fmt"
)

func main() {
	scd := uint32(0x0000FF0F) // 十六进制值

	currentStatus := scd & 0xFFFF
	statusChange := (scd >> 16) & 0xFFFF

	var activePoints []int
	var changedPoints []int

	for i := 0; i < 16; i++ {
		if currentStatus&(1<<i) != 0 {
			activePoints = append(activePoints, i)
		}
		if statusChange&(1<<i) != 0 {
			changedPoints = append(changedPoints, i)
		}
	}

	fmt.Println("当前闭合的位:", activePoints)
	fmt.Println("状态变化的位:", changedPoints)
	// 输出:
	// 当前闭合的位: [0 1 2 3 8 9 10 11]
	// 状态变化的位: []
}
```

##### Java 示例代码：

``` java
public class Main {
    public static void main(String[] args) {
        int scd = 0x0000FF0F; // 十六进制值

        int currentStatus = scd & 0xFFFF;
        int statusChange = (scd >> 16) & 0xFFFF;

        StringBuilder activePoints = new StringBuilder();
        StringBuilder changedPoints = new StringBuilder();

        for (int i = 0; i < 16; i++) {
            if ((currentStatus & (1 << i)) != 0) {
                activePoints.append(i).append(" ");
            }
            if ((statusChange & (1 << i)) != 0) {
                changedPoints.append(i).append(" ");
            }
        }

        System.out.println("当前闭合的位: " + activePoints.toString().trim());
        System.out.println("状态变化的位: " + changedPoints.toString().trim());
        // 输出:
        // 当前闭合的位: 0 1 2 3 8 9 10 11
        // 状态变化的位: 
    }
}
```

• 解析结果：当前状态中，位0-3和位8-11为闭合状态，无变位检出。

#### 4. 使用场景

• 适用ASDU类型：`M_PS_NA_1`
• 典型场景：批量监测单点状态变化（如开关量输入模块）。

#### 5. 注意事项

• 低16位和高16位独立解析，变位检出仅在状态变化时置位。

### 附录G：枚举值汇总

> **设计原则**
> 1. 状态量优先使用遥信（YX）通道传输（高效、可靠）
> 2. 遥测（YC）中`DIGITAL_STATUS`仅用于兼容旧系统或特殊硬件
> 3. 单位符号遵循IEC国际标准（避免中英文混用）

---

#### 🧪 遥测单位（YcUnit）

| 枚举值               | 中文名称  | 符号  | 示例用途    | 补充说明            |
|-------------------|-------|-----|---------|-----------------|
| VOLT              | 交流电压  | V   | 电网电压    | 标准交流系统380V/10kV |
| VOLTAGE_DC        | 直流电压  | Vdc | 电池电压    | 光伏/储能系统专用       |
| KILOVOLT          | 千伏    | kV  | 高压线路    | 35kV/110kV等高压系统 |
| AMPERE            | 电流    | A   | 负载电流    | 交直流系统均适用        |
| WATT              | 有功功率  | W   | 瞬时功率    | 单相设备常用          |
| VOLT_AMP_REACTIVE | 无功功率  | var | 无功功率    | 功率补偿装置          |
| KILOWATT          | 千瓦    | kW  | 有功功率    | 三相设备常用单位        |
| KILOVOLT_AMP      | 千伏安   | kVA | 视在功率    | 变压器容量计算         |
| HERTZ             | 频率    | Hz  | 电网频率    | 中国50Hz/美国60Hz   |
| DEGREE_CELSIUS    | 摄氏温度  | ℃   | 设备温度    | 温控系统专用          |
| PERCENT           | 百分比   | %   | 负载率/SOC | 电池电量测量          |
| POWER_FACTOR      | 功率因数  | λ   | 功率因数    | 范围0.8~1.0       |
| SECOND            | 时间（秒） | s   | 状态持续时长  | 告警持续时间等         |
| DIGITAL_STATUS    | 状态映射值 | 0/1 | 非标设备状态  | **仅用于兼容旧系统**    |
| NONE              | 无单位   | -   | 特殊量     | 占位/自定义量         |

> 💡 注：正常情况下状态量应使用YX传输，`DIGITAL_STATUS`仅用于兼容历史系统

---

#### 🧮 遥脉单位（YmUnit）

| 枚举值     | 中文名称    | 符号    | 示例用途   | 补充说明    |
|---------|---------|-------|--------|---------|
| KWH     | 有功电能    | kWh   | 正向有功电度 | 电表常用单位  |
| KVARH   | 无功电能    | kvarh | 无功电度   | 力调电费计算  |
| MWH     | 兆瓦时     | MWh   | 发电量统计  | 电厂月度报表  |
| M3      | 累计体积    | m³    | 燃气表/水表 | 流量计量仪表  |
| LITER   | 液体体积    | L     | 液体流量   | 小型计量设备  |
| COUNTER | 累计次数    | cnt   | 动作次数   | 合闸/分闸次数 |
| SECOND  | 累计时间（秒） | s     | 运行秒数   | 设备运行时间  |
| MINUTE  | 累计时间（分） | min   | 停机时长   | 维护停机统计  |
| HOUR    | 累计时间（时） | h     | 设备运行小时 | 寿命计算依据  |
| NONE    | 无单位     | -     | 特殊计量   | 预留扩展位   |

---

#### 🟢 遥信单位（YxUnit）

| 枚举值           | 中文名称  | 符号                 | 示例用途     | 补充说明         |
|---------------|-------|--------------------|----------|--------------|
| DOUBLE_POINT  | 双点状态  | 0=中间,1=开,2=合,3=不确定 | 断路器/隔离开关 | IEC 104标准四状态 |
| SWITCH_STATUS | 单点分合闸 | 0=分,1=合            | 普通开关设备   | 简单开关信号       |
| GENERIC_BOOL  | 通用状态  | 0=关,1=开,2=未知       | 通用设备状态   | 三态支持质量判断     |
| ALARM_STATUS  | 报警状态  | 0=正常,1=报警          | 故障信号     | 优先级最高信号      |
| CONTROL_MODE  | 控制权限  | 0=本地,1=远程          | 操作模式     | 控制权切换标志      |
| FAULT_STATUS  | 故障标志  | 0=正常,1=故障          | 系统级故障    | 设备异常状态       |
| DOOR_STATUS   | 门禁状态  | 0=关,1=开            | 机柜门状态    | 安全防护信号       |
| COMMUNICATION | 通信状态  | 0=离线,1=在线          | 通信链路     | 设备在线监测       |
| NONE          | 无单位   | -                  | 预留位      | 扩展兼容位        |

---

## 5. 数据消费指南

### 5.1 唯一键生成规则

```python
# 格式：host_coa_0x{ioa}
key = f"{host}_{coa}_0x{ioa:06X}"  # 示例：127.0.0.1_1_0x0007D1
```

### 5.2 时区与时间处理

- **时区**：所有时间字段为 **UTC+8** 时区
- **精度**：微秒级（实际精度依赖设备能力）

### 5.3 异常数据处理

- **品质位检查**：标记 `IV`（无效）、`OV`（溢出）数据并告警
- **累计量异常**：记录 `isAdjusted`（人工调整）和 `hasCarry`（溢出）日志

---

## 6. 技术支持

- **文档版本**：v1.1.0（2026-03-19）
- **协议版本**：IEC 60870-5-104
- **联系支持**：[hehanpengyy@163.com](mailto:hehanpengyy@163.com)

---

> **注意**：实际解析时需严格参照设备点表定义，部分字段（如双点信息的 value）的具体含义可能因设备而异。

**相关文档**：
- [IEC 104 数采平台架构](./iec104.md) -- 服务组件、数据流、配置管理
- [`ieccaller.proto`](../app/ieccaller/ieccaller.proto) -- 控制指令 RPC 接口（总召唤、读定值等），支持 Endpoints / Nacos 接入
- [`streamevent.proto`](../facade/streamevent/streamevent.proto) -- 统一流事件 gRPC 协议