# Kafka IEC-104 消息对接文档

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
  "host": "从站 ip",
  "port": 2404,
  "asdu": "M_SP_NA_1",
  "typeId": 1,
  "coa": 1001,
  "body": {
    /* 信息体结构（不同typeId对应不同结构） */
  },
  "time": "2023-10-01 14:30:00"
}
```

| 字段     | 类型     | 说明                                    |
|--------|--------|---------------------------------------|
| host   | String | 设备唯一标识（如RTU/IP地址）                     |
| port   | int    | 设备端口号                                 |
| typeId | int    | ASDU类型标识符（见第2章类型映射表）                  |
| coa    | uint   | 公共地址（范围：1-65534，全局地址65535保留）          |
| body   | Object | 信息体对象（结构随typeId变化）                    |
| time   | String | 时间戳（格式：`YYYY-MM-DD HH:mm:ss`，UTC+8时区） |

---

## 2. 全量ASDU类型映射表

| TypeID | ASDU类型    | Body结构体                                      | 应用场景说明                 |
|--------|-----------|----------------------------------------------|------------------------|
| 1      | M_SP_NA_1 | `SinglePointInfo`                            | 单点遥信（不带时标）             |
| 2      | M_SP_TA_1 | `SinglePointInfo`                            | 单点遥信（带时标）              |
| 3      | M_DP_NA_1 | `DoublePointInfo`                            | 双点遥信（不带时标）             |
| 4      | M_DP_TA_1 | `DoublePointInfo`                            | 双点遥信（带时标）              |
| 5      | M_ST_NA_1 | `StepPositionInfo`                           | 步位置信息（不带时标）            |
| 6      | M_ST_TA_1 | `StepPositionInfo`                           | 步位置信息（带时标）             |
| 7      | M_BO_NA_1 | `BitString32Info`                            | 32位比特串（不带时标）           |
| 8      | M_BO_TA_1 | `BitString32Info`                            | 32位比特串（带时标）            |
| 9      | M_ME_NA_1 | `MeasuredValueNormalInfo`                    | 规一化遥测值（不带时标）           |
| 10     | M_ME_TA_1 | `MeasuredValueNormalInfo`                    | 规一化遥测值（带时标）            |
| 11     | M_ME_NB_1 | `MeasuredValueScaledInfo`                    | 标度化遥测值（不带时标）           |
| 12     | M_ME_TB_1 | `MeasuredValueScaledInfo`                    | 标度化遥测值（带时标）            |
| 13     | M_ME_NC_1 | `MeasuredValueFloatInfo`                     | 短浮点数遥测值（不带时标）          |
| 14     | M_ME_TC_1 | `MeasuredValueFloatInfo`                     | 短浮点数遥测值（带时标）           |
| 15     | M_IT_NA_1 | `BinaryCounterReadingInfo`                   | 累计量（不带时标）              |
| 16     | M_IT_TA_1 | `BinaryCounterReadingInfo`                   | 累计量（带时标）               |
| 17     | M_EP_TA_1 | `EventOfProtectionEquipmentInfo`             | 继电保护事件（带时标）            |
| 18     | M_EP_TB_1 | `PackedStartEventsOfProtectionEquipmentInfo` | 成组启动事件（带时标）            |
| 19     | M_EP_TC_1 | `PackedOutputCircuitInfoInfo`                | 成组输出电路信息（带时标）          |
| 20     | M_PS_NA_1 | `PackedSinglePointWithSCDInfo`               | 带变位检出的成组单点信息           |
| 21     | M_ME_ND_1 | `MeasuredValueNormalInfo`                    | 无品质描述的规一化遥测值           |
| 30     | M_SP_TB_1 | `SinglePointInfo`                            | 单点遥信（CP56Time2a时标）     |
| 31     | M_DP_TB_1 | `DoublePointInfo`                            | 双点遥信（CP56Time2a时标）     |
| 32     | M_ST_TB_1 | `StepPositionInfo`                           | 步位置信息（CP56Time2a时标）    |
| 33     | M_BO_TB_1 | `BitString32Info`                            | 32位比特串（CP56Time2a时标）   |
| 34     | M_ME_TD_1 | `MeasuredValueNormalInfo`                    | 规一化遥测值（CP56Time2a时标）   |
| 35     | M_ME_TE_1 | `MeasuredValueScaledInfo`                    | 标度化遥测值（CP56Time2a时标）   |
| 36     | M_ME_TF_1 | `MeasuredValueFloatInfo`                     | 短浮点数遥测值（CP56Time2a时标）  |
| 37     | M_IT_TB_1 | `BinaryCounterReadingInfo`                   | 累计量（CP56Time2a时标）      |
| 38     | M_EP_TD_1 | `EventOfProtectionEquipmentInfo`             | 继电保护事件（CP56Time2a时标）   |
| 39     | M_EP_TE_1 | `PackedStartEventsOfProtectionEquipmentInfo` | 成组启动事件（CP56Time2a时标）   |
| 40     | M_EP_TF_1 | `PackedOutputCircuitInfoInfo`                | 成组输出电路信息（CP56Time2a时标） |
| 70     | M_EI_NA_1 | `无Body`                                      | 初始化结束（仅公共地址）           |

---

## 3. 信息体结构详解

### 3.1 单点信息（SinglePointInfo）

```json
{
  "ioa": 2001,
  "value": true,
  "qds": 0,
  "time": "2023-10-01 14:30:00"
}
```

| 字段    | 类型     | 说明                                     |
|-------|--------|----------------------------------------|
| ioa   | uint   | 信息对象地址（范围：`0x000001`-`0xFFFFFF`，十进制显示） |
| value | bool   | `true`=合/动作，`false`=分/未动作              |
| qds   | byte   | 品质描述（见附录A）                             |
| time  | string | 时标（仅带时标的ASDU类型包含此字段）                   |

---

### 3.2 双点信息（DoublePointInfo）

```json
{
  "ioa": 2002,
  "value": 0,
  "qds": 0,
  "time": "2023-10-01 14:30:00"
}
```

| 字段    | 类型   | 说明                          |
|-------|------|-----------------------------|
| value | byte | `0`=不确定，`1`=开，`2`=合，`3`=不确定 |

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
  "time": "2023-10-01 14:30:00"
}
```

| 字段           | 类型   | 说明                    |
|--------------|------|-----------------------|
| val          | int  | 步位置值（范围：`-64` 至 `63`） |
| hasTransient | bool | `true`=设备处于瞬变状态       |

---

### 3.4 规一化遥测值（MeasuredValueNormalInfo）

```json
{
  "ioa": 4001,
  "value": 16384,
  "qds": 0,
  "time": "2023-10-01 14:30:00"
}
```

| 字段    | 类型    | 说明                                     |
|-------|-------|----------------------------------------|
| value | int16 | 归一化值（范围：`-32768` 至 `32767`，需按公式转换为工程值） |

---

### 3.5 比特位串信息（BitString32Info）

```json
{
  "ioa": 4002,
  "value": 220,
  "qds": 0,
  "time": "2023-10-01 14:30:00"
}
```

| 字段    | 类型     | 说明                                  |
|-------|--------|-------------------------------------|
| value | uint32 | 32 个独立设备状态（如开关、传感器、继电器），每个比特位对应一个设备 |

---

---

### 3.6 短浮点数遥测值（MeasuredValueFloatInfo）

```json
{
  "ioa": 4002,
  "value": 220.5,
  "qds": 0,
  "time": "2023-10-01 14:30:00"
}
```

| 字段    | 类型      | 说明                           |
|-------|---------|------------------------------|
| value | float32 | IEEE 754短浮点数（直接为工程值，如电压、电流等） |

---

### 3.7 累计量（BinaryCounterReadingInfo）

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

| 字段             | 类型    | 说明               |
|----------------|-------|------------------|
| counterReading | int32 | 计数器读数（32位有符号整数）  |
| seqNumber      | byte  | 顺序号（范围：`0`-`31`） |
| hasCarry       | bool  | `true`=计数器溢出     |
| isAdjusted     | bool  | `true`=计数量被人工调整  |
| isInvalid      | bool  | `true`=数据无效      |

---

### 3.8 继电保护事件（EventOfProtectionEquipmentInfo）

```json
{
  "ioa": 6001,
  "event": 1,
  "qdp": 0,
  "msec": 500,
  "time": "2023-10-01 14:30:00"
}
```

| 字段    | 类型     | 说明                         |
|-------|--------|----------------------------|
| event | byte   | 事件类型（见附录B）                 |
| msec  | uint16 | 事件发生的毫秒时间戳（范围：`0`-`59999`） |
| qdp   | byte   | 保护事件品质（见附录A）               |

---

---

### 3.9 继电器保护设备成组启动事件（PackedStartEventsOfProtectionEquipmentInfo）

```json
{
  "ioa": 6001,
  "event": 32,
  "qdp": 0,
  "msec": 500,
  "time": "2023-10-01 14:30:00"
}
```

| 字段    | 类型     | 说明                         |
|-------|--------|----------------------------|
| event | byte   | 事件类型（见附录B_2）               |
| msec  | uint16 | 事件发生的毫秒时间戳（范围：`0`-`59999`） |
| qdp   | byte   | 保护事件品质（见附录A）               |

---

---

### 3.10 继电器保护设备成组输出电路信息（PackedOutputCircuitInfoInfo）

```json
{
  "ioa": 6001,
  "oci": 7,
  "qdp": 0,
  "msec": 500,
  "time": "2023-10-01 14:30:00"
}
```

| 字段   | 类型     | 说明                         |
|------|--------|----------------------------|
| oci  | byte   | 事件类型（见附录D）                 |
| msec | uint16 | 事件发生的毫秒时间戳（范围：`0`-`59999`） |
| qdp  | byte   | 保护事件品质（见附录A）               |

---

---

### 3.11 带变位检出的成组单点信息（PackedSinglePointWithSCDInfo）

```json
{
  "ioa": 6001,
  "scd": 1,
  "qds": 0
}
```

| 字段   | 类型     | 说明                         |
|------|--------|----------------------------|
| scd  | byte   | 事件类型（见附录E）                 |
| msec | uint16 | 事件发生的毫秒时间戳（范围：`0`-`59999`） |

---

## 4. 附录

### 附录A：品质描述（QDS/QDP）

| 位 | 名称 | 描述   |
|---|----|------|
| 0 | 溢出 | 数据溢出 |
| 1 | 无效 | 数据无效 |

### 附录B：继电保护事件类型

| 值 | 事件类型     |
|---|----------|
| 0 | 不确定或中间状态 |
| 1 | 开        |
| 2 | 合        |
| 3 | 不确定      |

### 附录B_2：继电保护事件类型

#### 1. 字段定义

| 字段名   | 类型     | 位宽 | 描述                                     |
|:------|:-------|:---|:---------------------------------------|
| `oci` | `byte` | 8位 | 8位比特掩码，每一位表示一个输出电路的状态（`1`=启动，`0`=未启动）。 |

#### 2. 比特位映射规则

| 值  | 类型                       | 描述       |
|----|--------------------------|----------|
| 1  | SEPGeneralStart          | 总启动      |
| 2  | SEPStartL1               | A相保护启动   |
| 4  | SEPStartL2               | B相保护启动   |
| 8  | SEPStartL3               | C相保护启动   |
| 16 | SEPStartEarthCurrent     | 接地电流保护启动 |
| 32 | SEPStartReverseDirection | 反向保护启动   |

### 附录C：公共地址（COA）规则

- **范围**：`1`-`65534`（`65535`为全局广播地址）
- **用途**：标识RTU、子站或子系统

### 附录D：输出电路信息（OCI）定义

**概述**  
本附录定义了输出电路信息（OCI）的字段、比特位映射规则及示例解析。

#### 1. 字段定义

| 字段名   | 类型     | 位宽 | 描述                                                     |
|:------|:-------|:---|:-------------------------------------------------------|
| `oci` | `byte` | 8位 | 8位比特掩码，每一位表示一个输出电路的状态（`1`=无总命令输出至输出电路，`0`=总命令输出至输出电路）。 |

#### 2. 比特位映射规则

| 值 | 类型                | 描述             |
|---|-------------------|----------------|
| 1 | OCIGeneralCommand | 总命令输出至输出电路     |
| 2 | OCICommandL1      | A 相保护命令输出至输出电路 |
| 4 | OCICommandL2      | B 相保护命令输出至输出电路 |
| 8 | OCICommandL3      | C 相保护命令输出至输出电路 |

## 附录E：状态变位检出（SCD）定义--todo

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

- **文档版本**：v1.0.0（2025-05-06）
- **协议版本**：IEC 60870-5-104 Ed.2.0
- **联系支持**：[hehanpengyy@163.com](mailto:hehanpengyy@163.com)

---

> ⚠️ **注意**：实际解析时需严格参照设备点表定义，部分字段（如双点信息的value）的具体含义可能因设备而异。
>
> ⚠️ **注意**：代码需要测试
>
> ⚠️ **注意**：ieccaller.proto 包含常见控制指令（总召唤...）,接入方式 Endpoints,Nacos