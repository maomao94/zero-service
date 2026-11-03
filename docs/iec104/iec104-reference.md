# IEC 104 附录与数据消费参考

本文件补充 [IEC 104 消息对接文档](./iec104-message.md) 未展开的参考内容：品质描述、状态值、地址规则、业务单位枚举，以及 Kafka / MQTT / gRPC 三种通道的消费建议。

## 附录

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

## 数据消费指南

### 唯一键生成规则

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

### Kafka 消费建议

- 使用消息 key 保证同一 `host`、`coa`、`ioa` 的事件落在同一分区。
- 以 `msgId` 做幂等处理，避免重试造成重复写入。
- 按 `typeId` 或 `asdu` 分流，再按 `body.ioa` 更新点位状态。
- 链路追踪信息位于 Kafka message headers，同时 JSON payload 顶层 `headers` 也为同一内容。

### MQTT 消费建议

- Topic 可由 `pm`、`asdu`、`coa` 等字段模板生成。
- 消费者不能假设 `pm` 一定存在。没有点位映射时，仍可使用 `host`、`coa`、`body.ioa` 识别点位。
- 多 Topic 推送时，同一条消息的 JSON 内容一致。
- 如需恢复链路追踪上下文，从顶层 `headers.traceparent` / `headers.tracestate` 读取 W3C trace context；不要从 `metaData` 读取追踪传输字段。

### gRPC 流事件消费建议

- gRPC 推送内容与 Kafka/MQTT payload 的 JSON 保持一致。`streamevent.MsgBody` 已包含 `headers` 字段（map<string, string>），服务端可直接读取 `headers["traceparent"]` 还原链路上下文。
- 批量或流式消费方应逐条解析 `body`，不要按批次共享同一个信息体类型。

### 时区与时标处理

- `time` 为推送时间，UTC+8，微秒级字符串。
- `body.time` 为设备侧 ASDU 时标，只有带时标 ASDU 才有有效值。
- 下游排序优先使用 `body.time`。如果为空，再使用消息级 `time`。

### 异常数据处理

- 对带 QDS 的信息体，重点检查 `iv`、`ov`、`bl`、`nt`。
- 对累计量，重点检查 `value.isInvalid`、`value.hasCarry`、`value.isAdjusted`。
- 对保护事件，重点检查 QDP 展开的 `iv`、`ei`。
- 点表含义优先于通用说明。双点状态、比特串、SCD、保护事件位含义都可能因设备点表不同而变化。

### Python 解析示例

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

## 技术支持

- **文档版本**：v2.0.0（2026-06-15）
- **协议版本**：IEC 60870-5-104
- **联系支持**：[hehanpengyy@163.com](mailto:hehanpengyy@163.com)

注意：实际解析时需严格参照设备点表定义。部分字段，如双点信息 `value`、比特串 `value`、保护事件 `event`，具体含义可能因设备而异。

相关文档：

- [IEC 104 数采平台架构](./iec104.md)，服务组件、数据流、配置管理。
- [IEC 104 控制命令文档](./iec104-command.md)，控制指令 RPC 接口。
- [`streamevent.proto`](../../facade/streamevent/streamevent.proto)，统一流事件 gRPC 协议。
