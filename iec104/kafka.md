# Kafka ASDU 消息对接文档（全量版）

---

## 1. 消息格式规范

### 基础协议
- **传输协议**：IEC 60870-5-104
- **传输载体**：Kafka Topic（具体名称协商约定）
- **数据格式**：JSON
- **编码格式**：UTF-8

### 消息结构
```json
{
  "host": "设备标识（如RTU-01）",
  "port": 2404,
  "typeId": 1,
  "coa": 1001,
  "body": {
    /* 信息体结构（不同typeId对应不同结构） */
  },
  "time": "2023-10-01 14:30:00.123456"
}
```

| 字段     | 类型         | 说明                                                                 |
|----------|--------------|----------------------------------------------------------------------|
| host     | String       | 设备唯一标识（如RTU/IP地址）                                         |
| port     | int          | 设备端口号                                                           |
| typeId   | int          | ASDU类型标识符（见第2章类型映射表）                                  |
| coa      | uint         | 公共地址（范围：1-65534，全局地址65535保留）                         |
| body     | Object       | 信息体对象（结构随typeId变化）                                       |
| time     | String       | 时间戳（格式：`YYYY-MM-DD HH:mm:ss.u`，UTC+8时区，带微秒精度）       |

---

## 2. 全量ASDU类型映射表

| TypeID | ASDU类型                | Body结构体                           | 应用场景说明                         |
|--------|-------------------------|--------------------------------------|--------------------------------------|
| 1      | M_SP_NA_1              | `SinglePointInfo`                   | 单点遥信（不带时标）                 |
| 2      | M_SP_TA_1              | `SinglePointInfo`                   | 单点遥信（带时标）                   |
| 3      | M_DP_NA_1              | `DoublePointInfo`                   | 双点遥信（不带时标）                 |
| 4      | M_DP_TA_1              | `DoublePointInfo`                   | 双点遥信（带时标）                   |
| 5      | M_ST_NA_1              | `StepPositionInfo`                  | 步位置信息（不带时标）               |
| 6      | M_ST_TA_1              | `StepPositionInfo`                  | 步位置信息（带时标）                 |
| 7      | M_BO_NA_1              | `MeasuredValueFloatInfo`            | 32位比特串（不带时标）               |
| 8      | M_BO_TA_1              | `MeasuredValueFloatInfo`            | 32位比特串（带时标）                 |
| 9      | M_ME_NA_1              | `MeasuredValueNormalInfo`           | 规一化遥测值（不带时标）             |
| 10     | M_ME_TA_1              | `MeasuredValueNormalInfo`           | 规一化遥测值（带时标）               |
| 11     | M_ME_NB_1              | `MeasuredValueScaledInfo`           | 标度化遥测值（不带时标）             |
| 12     | M_ME_TB_1              | `MeasuredValueScaledInfo`           | 标度化遥测值（带时标）               |
| 13     | M_ME_NC_1              | `MeasuredValueFloatInfo`            | 短浮点数遥测值（不带时标）           |
| 14     | M_ME_TC_1              | `MeasuredValueFloatInfo`            | 短浮点数遥测值（带时标）             |
| 15     | M_IT_NA_1              | `BinaryCounterReadingInfo`          | 累计量（不带时标）                   |
| 16     | M_IT_TA_1              | `BinaryCounterReadingInfo`          | 累计量（带时标）                     |
| 17     | M_EP_TA_1              | `EventOfProtectionEquipmentInfo`    | 继电保护事件（带时标）               |
| 18     | M_EP_TB_1              | `PackedStartEventsOfProtectionEquipmentInfo` | 成组启动事件（带时标）       |
| 19     | M_EP_TC_1              | `PackedOutputCircuitInfoInfo`       | 成组输出电路信息（带时标）           |
| 20     | M_PS_NA_1              | `PackedSinglePointWithSCDInfo`      | 带变位检出的成组单点信息             |
| 21     | M_ME_ND_1              | `MeasuredValueNormalInfo`           | 无品质描述的规一化遥测值             |
| 30     | M_SP_TB_1              | `SinglePointInfo`                   | 单点遥信（CP56Time2a时标）           |
| 31     | M_DP_TB_1              | `DoublePointInfo`                   | 双点遥信（CP56Time2a时标）           |
| 32     | M_ST_TB_1              | `StepPositionInfo`                  | 步位置信息（CP56Time2a时标）         |
| 33     | M_BO_TB_1              | `MeasuredValueFloatInfo`            | 32位比特串（CP56Time2a时标）         |
| 34     | M_ME_TD_1              | `MeasuredValueNormalInfo`           | 规一化遥测值（CP56Time2a时标）       |
| 35     | M_ME_TE_1              | `MeasuredValueScaledInfo`           | 标度化遥测值（CP56Time2a时标）       |
| 36     | M_ME_TF_1              | `MeasuredValueFloatInfo`            | 短浮点数遥测值（CP56Time2a时标）     |
| 37     | M_IT_TB_1              | `BinaryCounterReadingInfo`          | 累计量（CP56Time2a时标）             |
| 38     | M_EP_TD_1              | `EventOfProtectionEquipmentInfo`    | 继电保护事件（CP56Time2a时标）       |
| 39     | M_EP_TE_1              | `PackedStartEventsOfProtectionEquipmentInfo` | 成组启动事件（CP56Time2a时标） |
| 40     | M_EP_TF_1              | `PackedOutputCircuitInfoInfo`       | 成组输出电路信息（CP56Time2a时标）   |
| 70     | M_EI_NA_1              | `无Body`                            | 初始化结束（仅公共地址）             |

---

## 3. 信息体结构详解

### 3.1 单点信息（SinglePointInfo）
```json
{
  "ioa": 2001,
  "value": true,
  "qds": 0,
  "time": "2023-10-01 14:30:00.123456"
}
```
| 字段   | 类型    | 说明                                                                 |
|--------|---------|----------------------------------------------------------------------|
| ioa    | uint    | 信息对象地址（范围：`0x000001`-`0xFFFFFF`，十进制显示）              |
| value  | bool    | `true`=合/动作，`false`=分/未动作                                   |
| qds    | byte    | 品质描述（见附录A）                                                 |
| time   | string  | 时标（仅带时标的ASDU类型包含此字段）                                |

---

### 3.2 双点信息（DoublePointInfo）
```json
{
  "ioa": 2002,
  "value": false,
  "qds": 0,
  "time": "2023-10-01 14:30:00.123456"
}
```
| 字段   | 类型    | 说明                                                                 |
|--------|---------|----------------------------------------------------------------------|
| value  | bool    | `true`=中间态，`false`=确定态（具体含义参考点表）                   |

---

### 3.3 步位置信息（StepPositionInfo）
```json
{
  "ioa": 3001,
  "value": {
    "val": 63,
    "hasTransient": false
  },
  "qds": 0,
  "time": "2023-10-01 14:30:00.123456"
}
```
| 字段          | 类型    | 说明                                                                 |
|---------------|---------|----------------------------------------------------------------------|
| val           | int     | 步位置值（范围：`-64` 至 `63`）                                     |
| hasTransient  | bool    | `true`=设备处于瞬变状态                                             |

---

### 3.4 规一化遥测值（MeasuredValueNormalInfo）
```json
{
  "ioa": 4001,
  "value": 16384,
  "qds": 0,
  "time": "2023-10-01 14:30:00.123456"
}
```
| 字段   | 类型    | 说明                                                                 |
|--------|---------|----------------------------------------------------------------------|
| value  | int16   | 归一化值（范围：`-32768` 至 `32767`，需按公式转换为工程值）          |

---

### 3.5 短浮点数遥测值（MeasuredValueFloatInfo）
```json
{
  "ioa": 4002,
  "value": 220.5,
  "qds": 0,
  "time": "2023-10-01 14:30:00.123456"
}
```
| 字段   | 类型    | 说明                                                                 |
|--------|---------|----------------------------------------------------------------------|
| value  | float32 | IEEE 754短浮点数（直接为工程值，如电压、电流等）                    |

---

### 3.6 累计量（BinaryCounterReadingInfo）
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
  "time": "2023-10-01 14:30:00.123456"
}
```
| 字段            | 类型    | 说明                                                                 |
|-----------------|---------|----------------------------------------------------------------------|
| counterReading  | int32   | 计数器读数（32位有符号整数）                                        |
| seqNumber       | byte    | 顺序号（范围：`0`-`31`）                                            |
| hasCarry        | bool    | `true`=计数器溢出                                                   |
| isAdjusted      | bool    | `true`=计数量被人工调整                                             |
| isInvalid       | bool    | `true`=数据无效                                                     |

---

### 3.7 继电保护事件（EventOfProtectionEquipmentInfo）
```json
{
  "ioa": 6001,
  "event": 1,
  "qdp": 0,
  "msec": 500,
  "time": "2023-10-01 14:30:00.123456"
}
```
| 字段   | 类型    | 说明                                                                 |
|--------|---------|----------------------------------------------------------------------|
| event  | byte    | 事件类型（见附录B）                                                 |
| msec   | uint16  | 事件发生的毫秒时间戳（范围：`0`-`59999`）                           |
| qdp    | byte    | 保护事件品质（见附录A）                                             |

---

## 4. 附录

### 附录A：品质描述（QDS/QDP）
| 位 | 名称   | 描述                                |
|----|--------|-------------------------------------|
| 0  | 溢出   | `1`=数据溢出                       |
| 1  | 无效   | `1`=数据无效                       |
| 2  | 旧数据 | `1`=数据未更新（如通信中断后补传） |
| 3  | 替代   | `1`=人工替代值                     |

### 附录B：继电保护事件类型
| 值 | 事件类型       |
|----|----------------|
| 0  | 无事件         |
| 1  | 启动（Start）  |
| 2  | 跳闸（Trip）   |
| 3  | 重合闸（Reclose） |

### 附录C：公共地址（COA）规则
- **范围**：`1`-`65534`（`65535`为全局广播地址）
- **用途**：标识RTU、子站或子系统

---

## 5. 数据消费指南

### 5.1 唯一键生成规则
```python
# 格式：host_coa_0x{ioa}
key = f"{host}_{coa}_0x{ioa:06X}"  # 示例：RTU-01_1001_0x0007D1
```

### 5.2 时区与时间处理
- **时区**：所有时间字段为 **UTC+8** 时区
- **精度**：微秒级（实际精度依赖设备能力）

### 5.3 异常数据处理
- **品质位检查**：标记 `IV`（无效）、`OV`（溢出）数据并告警
- **累计量异常**：记录 `isAdjusted`（人工调整）和 `hasCarry`（溢出）日志

---

## 6. 技术支持
- **文档版本**：v1.1.0（2023-11-20）
- **协议版本**：IEC 60870-5-104 Ed.2.0
- **联系支持**：[hehanpeng@163.com](mailto:hehanpeng@163.com)

---

> ⚠️ **注意**：实际解析时需严格参照设备点表定义，部分字段（如双点信息的value）的具体含义可能因设备而异。