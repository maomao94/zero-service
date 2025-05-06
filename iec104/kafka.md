Kafka ASDU 消息对接文档（全量版）

---

**1. 消息格式概述**
• 传输协议：IEC 60870-5-104 报文解析后的结构化数据

• 传输载体：Kafka Topic（具体Topic名称由双方约定）

• 数据格式：JSON

• 编码：UTF-8

• 字段说明：

  ```json
  {
  "host": "设备标识（如RTU-01）",
  "port": 2404,
  "typeId": 1,
  "coa": 1001,
  "body": {
    /* ASDU信息体 */
  },
  "time": "2023-10-01 14:30:00"
}
  ```

---

**2. 全量ASDU类型映射表**

| TypeID | ASDU类型                          | Body结构体                           | 用途描述                           |
|--------|-----------------------------------|--------------------------------------|------------------------------------|
| 1      | `M_SP_NA_1`                      | `SinglePointInfo`                   | 单点遥信（不带时标）               |
| 2      | `M_SP_TA_1`                      | `SinglePointInfo`                   | 单点遥信（带时标）                 |
| 3      | `M_DP_NA_1`                      | `DoublePointInfo`                   | 双点遥信（不带时标）               |
| 4      | `M_DP_TA_1`                      | `DoublePointInfo`                   | 双点遥信（带时标）                 |
| 5      | `M_ST_NA_1`                      | `StepPositionInfo`                  | 步位置信息（不带时标）             |
| 6      | `M_ST_TA_1`                      | `StepPositionInfo`                  | 步位置信息（带时标）               |
| 7      | `M_BO_NA_1`                      | `MeasuredValueFloatInfo`            | 32位比特串（不带时标）             |
| 8      | `M_BO_TA_1`                      | `MeasuredValueFloatInfo`            | 32位比特串（带时标）               |
| 9      | `M_ME_NA_1`                      | `MeasuredValueNormalInfo`           | 规一化遥测值（不带时标）           |
| 10     | `M_ME_TA_1`                      | `MeasuredValueNormalInfo`           | 规一化遥测值（带时标）             |
| 11     | `M_ME_NB_1`                      | `MeasuredValueScaledInfo`           | 标度化遥测值（不带时标）           |
| 12     | `M_ME_TB_1`                      | `MeasuredValueScaledInfo`           | 标度化遥测值（带时标）             |
| 13     | `M_ME_NC_1`                      | `MeasuredValueFloatInfo`            | 短浮点数遥测值（不带时标）         |
| 14     | `M_ME_TC_1`                      | `MeasuredValueFloatInfo`            | 短浮点数遥测值（带时标）           |
| 15     | `M_IT_NA_1`                      | `BinaryCounterReadingInfo`          | 累计量（不带时标）                 |
| 16     | `M_IT_TA_1`                      | `BinaryCounterReadingInfo`          | 累计量（带时标）                   |
| 17     | `M_EP_TA_1`                      | `EventOfProtectionEquipmentInfo`    | 继电保护事件（带时标）             |
| 18     | `M_EP_TB_1`                      | `PackedStartEventsOfProtectionEquipmentInfo` | 成组启动事件（带时标）       |
| 19     | `M_EP_TC_1`                      | `PackedOutputCircuitInfoInfo`       | 成组输出电路信息（带时标）         |
| 20     | `M_PS_NA_1`                      | `PackedSinglePointWithSCDInfo`      | 带变位检出的成组单点信息           |
| 21     | `M_ME_ND_1`                      | `MeasuredValueNormalInfo`           | 无品质描述的规一化遥测值           |
| 30     | `M_SP_TB_1`                      | `SinglePointInfo`                   | 单点遥信（带CP56Time2a时标）       |
| 31     | `M_DP_TB_1`                      | `DoublePointInfo`                   | 双点遥信（带CP56Time2a时标）       |
| 32     | `M_ST_TB_1`                      | `StepPositionInfo`                  | 步位置信息（带CP56Time2a时标）     |
| 33     | `M_BO_TB_1`                      | `MeasuredValueFloatInfo`            | 32位比特串（带CP56Time2a时标）     |
| 34     | `M_ME_TD_1`                      | `MeasuredValueNormalInfo`           | 规一化遥测值（带CP56Time2a时标）   |
| 35     | `M_ME_TE_1`                      | `MeasuredValueScaledInfo`           | 标度化遥测值（带CP56Time2a时标）   |
| 36     | `M_ME_TF_1`                      | `MeasuredValueFloatInfo`            | 短浮点数遥测值（带CP56Time2a时标） |
| 37     | `M_IT_TB_1`                      | `BinaryCounterReadingInfo`          | 累计量（带CP56Time2a时标）         |
| 38     | `M_EP_TD_1`                      | `EventOfProtectionEquipmentInfo`    | 继电保护事件（带CP56Time2a时标）   |
| 39     | `M_EP_TE_1`                      | `PackedStartEventsOfProtectionEquipmentInfo` | 成组启动事件（带CP56Time2a时标） |
| 40     | `M_EP_TF_1`                      | `PackedOutputCircuitInfoInfo`       | 成组输出电路信息（带CP56Time2a时标） |
| 70     | `M_EI_NA_1`                      | 无Body（仅公共地址）                | 初始化结束                         |

**3. 信息体结构详解**
**3.1 单点信息 (`SinglePointInfo`)**

```json
{
  "ioa": 2001,
  "value": true,
  "qds": 0,
  "time": "2023-10-01 14:30:00"
}
```

• ioa : 信息对象地址（唯一标识测点，范围 `0x000001-0xFFFFFF`）

• value : `true`=合/动作，`false`=分/未动作

• qds : 品质描述（见附录A）

• time : 时标（仅带时标的ASDU类型包含此字段）

**3.2 双点信息 (`DoublePointInfo`)**

```json
{
  "ioa": 2002,
  "value": true,
  "qds": 0,
  "time": "2023-10-01 14:30:00"
}
```

• value : `true`=中间态，`false`=确定态（需结合点表定义具体含义）

**3.3 步位置信息 (`StepPositionInfo`)**

```json
{
  "ioa": 3001,
  "value": {
    "val": 63,
    "hasTransient": false
  },
  "qds": 0,
  "time": "2023-10-01 14:30:00"
}
```

• val : 步位置值（范围 `-64` 至 `63`）

• hasTransient : `true`=设备处于瞬变状态

**3.4 规一化遥测值 (`MeasuredValueNormalInfo`)**

```json
{
  "ioa": 4001,
  "value": 16384,
  "qds": 0,
  "time": "2023-10-01 14:30:00"
}
```

• value : 归一化值（范围 `-32768` 至 `32767`，需按公式转换为实际工程值）

**3.5 短浮点数遥测值 (`MeasuredValueFloatInfo`)**

```json
{
  "ioa": 4002,
  "value": 220.5,
  "qds": 0,
  "time": "2023-10-01 14:30:00"
}
```

• value : IEEE 754 短浮点数（直接为工程值，如电压、电流）

**3.6 累计量 (`BinaryCounterReadingInfo`)**

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
  "time": "2023-10-01 14:30:00"
}
```

• counterReading : 计数器读数（32位有符号整数）

• seqNumber : 顺序号（范围 `0-31`）

• hasCarry : `true`=计数器溢出

• isAdjusted : `true`=计数量被人工调整

• isInvalid : `true`=数据无效

**3.7 继电保护事件 (`EventOfProtectionEquipmentInfo`)**

```json
{
  "ioa": 6001,
  "event": 1,
  "qdp": 0,
  "msec": 500,
  "time": "2023-10-01 14:30:00"
}
```

• event : 事件类型（见附录B）

• msec : 事件发生的毫秒时间戳（范围 `0-59999`）

• qdp : 保护事件品质（见附录A）


---

**4. 附录**
**附录A：品质描述（QDS/QDP）**  
| 位 | 名称 | 描述 |
|--------|------------|---------------------------------------|
| 0 | 溢出（OV） | `1`=数据溢出 |
| 1 | 无效（IV） | `1`=数据无效 |
| 2 | 旧数据（SB）| `1`=数据未更新（如通信中断后补传） |
| 3 | 替代（BL） | `1`=人工替代值 |

**附录B：继电保护事件类型**  
| 值 | 事件类型 |
|--------|------------------------|
| 0 | 无事件 |
| 1 | 启动（Start） |
| 2 | 跳闸（Trip） |
| 3 | 重合闸（Reclose） |

**附录C：公共地址（COA）规则**  
• 范围：`1-65534`（全局地址 `65535` 保留用于广播）

• 用途：标识RTU、子站或子系统


---

**5. 数据消费建议**

1. 唯一键生成：  
   使用 `host_coa_0x{ioa}` 格式（如 `RTU-01_1001_0x0007D1`）作为数据点唯一标识。
2. 时区处理：  
   所有时间字段为 UTC+8 时区，格式 `YYYY-MM-DD HH:mm:ss`。
3. 异常处理：  
   • 检查 `qds` 字段，标记 `OV`（溢出）、`IV`（无效）等异常状态。

   • 累计量的 `isAdjusted` 和 `hasCarry` 需记录日志。

---

**6. 版本与支持**
• 文档版本：v1.0.0（2025-05-06）

• 协议版本：IEC 60870-5-104 Ed.2

• 联系支持：hehanpeng@163.com