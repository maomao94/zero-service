# BridgeModbus

`bridgemodbus` 是 Modbus TCP 协议桥接服务，把 gRPC 请求转换为 Modbus 读写操作，供内部服务和开放平台调用。

## 服务信息

| 项 | 值 |
| --- | --- |
| gRPC 端口 | 25004 |
| 契约 | [`bridgemodbus.proto`](./bridgemodbus.proto) |
| 服务名 | `bridge-modbus`（Nacos 注册名） |
| 配置 | [`etc/bridgemodbus.yaml`](./etc/bridgemodbus.yaml) |

## 能力范围

- **配置管理**：保存、删除、分页、按 code 查询、批量查询。
- **Bit Access**：读线圈 0x01、读离散输入 0x02、写单个线圈 0x05、写多个线圈 0x0F。
- **Register Access**：读输入寄存器 0x04、读保持寄存器 0x03、写单个寄存器 0x06、写多个寄存器 0x10、读写多个寄存器 0x17、掩码写 0x16、读 FIFO 0x18。
- **Device Identification**：0x2B/0x0E 读设备标识与指定对象。
- **本地换算**：16 位整数与 32 位（uint32/int32/float32、支持字序）数值和寄存器格式批量互转，不访问设备。

## 客户端复用与配置

按 `modbusCode` 复用长期客户端：同一 code 串行收发，不同 code 各自持有连接和固定从站号。

- `modbusCode` 为空：使用文件中的 `ModbusClientConf`。
- `modbusCode` 非空：读取 `modbus_slave_config` 中启用的配置；不存在返回 `100052`，未启用或 `slave` 不在 1–247 返回 `100140`，不回退文件配置。
- 保存/删除配置后立即失效对应客户端；直接改库时由后台定期校验后生效（通常 30 秒内），无需重启。数据库查询失败时保留现有连接。
- 服务退出时关闭全部客户端；写操作不会自动重试。

配置缺省值：`Timeout=10000ms`（上限 600000）、`IdleTimeout=60000ms`、`Slave=1`，恢复窗口与 `ConnectDelay` 为 0。需要长连接可设 `IdleTimeout=-1`；需要只读断链恢复或迟到响应处理，可分别设 `LinkRecoveryTimeout=3000`、`ProtocolRecoveryTimeout=2000`。

表中新建记录的默认值以 GORM 模型 `gormmodel.ModbusSlaveConfig` 为准，已有记录中的显式值不自动改写。TLS 证书文件内容变更需重启服务或改动配置后生效。

## 本地换算（不访问设备）

纯计算接口，不建立 Modbus 连接，用于数值与寄存器表示互转。

| 接口 | 方向 | 说明 |
| --- | --- | --- |
| `BatchConvertValue16ToRegisters` | 整数 → 1 个寄存器 | 16 位，无符号/有符号 |
| `BatchConvertValue32ToRegisters` | 数值 → 2 个寄存器 | 32 位，uint32/int32/float32，需指定字序；逐值结果另带展平的连续寄存器 |
| `BatchConvertRegistersToValue32` | 连续寄存器 → 数值 | 每 2 个寄存器一组，回显寄存器与字节，按 dataType 给出主结果及无符号/有符号/浮点视图 |

字序必须显式指定：`ABCD`（高字在前，最常见）、`CDAB`（字交换，国产仪表常见）、`BADC`、`DCBA`；不传会被拒绝。

四种字序的来源是 Modbus 只定义 16 位寄存器，32 位值由厂商自行拆成两个寄存器，"字顺序"与"字内字节序"两个自由度组合成 4 种。经验分布（同一厂商不同型号也可能不同，以手册或实测为准）：ABCD 为多数欧美 PLC/仪表默认形式；CDAB 常见于部分国产 PLC/仪表；BADC、DCBA 多见于电力/流量/温控等仪表的字节交换或小端配置。字序选错不会报错，只会得到错误数值，上线前应写入已知值读回验证。

`dataType` 是请求级字段，同一次请求内所有值必须使用同一类型；混排类型的点表按类型拆成多次调用。

编码与解码的结果结构对称，都包含该组的 `registers`（两个寄存器）、`bytes`（四字节）和 `hex`（两个寄存器的十六进制），三者统一按设备协议顺序（即寄存器地址顺序，每寄存器高字节在前）返回，可直接互相校验。

```json
// 220.5 按 float32、CDAB 编码
{"values":[220.5],"dataType":"VALUE32_FLOAT32","wordOrder":"WORD_ORDER_CDAB"}
// → results[0].registers [32768,17244]，bytes 0x8000 0x435C，hex "0x8000 0x435C"
// → registers [32768,17244]（全部值展平，可直接用于写多个寄存器）

// 反向解码，连续寄存器每 2 个一组
{"registers":[32768,17244],"dataType":"VALUE32_FLOAT32","wordOrder":"WORD_ORDER_CDAB"}
// → registers [32768,17244]，bytes 0x8000 0x435C
// → hex "0x8000 0x435C"，value 220.5，float32Value 220.5
```

`registers`/`bytes`/`hex` 是同一段字节的三种表示，顺序由设备协议决定（寄存器地址顺序），两端都不再按 `wordOrder` 二次重排：编码侧 `wordOrder` 只决定"编出哪两个寄存器"，解码侧 `wordOrder` 只决定"这两个寄存器还原成什么数值"。因此编码输出与解码回显完全同构：同一组寄存器原样读回后，`bytes`/`hex` 与当初编码写入时逐字符一致。需要"还原后的 32 位值"时看 `value`/`uint32Value`/`int32Value`/`float32Value`，不必从 `hex` 里再解析。寄存器落在哪个地址、怎么存储由业务侧决定，本服务只做数值与寄存器表示互转。

`value` 是按 `dataType` 选定的主结果：uint32/int32 为整数，float32 为该 float32 精确展开的 double。编码侧 `value` 回显同样按实际载荷给出（float32 会回落为 float32 精度，如 `0.1` 回显 `0.10000000149011612`），便于和设备读回值直接比对。`float32Value` 只在 `dataType=VALUE32_FLOAT32` 时填充，并按 float32 最短表示输出（如 `0.12`），因此整数比特不会以 `"NaN"`/`"Infinity"` 字符串出现在 JSON 中；解码到设备返回的 NaN/Inf 位模式时会原样输出 `"NaN"`/`"Infinity"` 字符串，属于预期行为。

多个值是连续地址时，按输入顺序取 `registers` 整块保存或下发；解码时把读回的寄存器按设备地址顺序整块传入，不要先按 `wordOrder` 自己重排（重排等于让字序生效两次），返回顺序与寄存器顺序一致。注意单次 Modbus 写多个寄存器最多 123 个，超过（约 61 个 32 位值）需按 123 个一组分片调用。

## 协议与参数约束

请求参数在服务入口统一校验，超出范围返回 `100022`，不会被截断后下发。

| 接口 / 参数 | 约束 |
| --- | --- |
| 保存、单条查询、批量查询的 `modbusCode` | 不能为空或纯空白，最多 128 个字符；按原值精确匹配，不自动去除首尾空格 |
| 保存的 `slaveAddress` | 必填，最多 64 个字符，格式为 `host:port`（IPv6 使用 `[地址]:port`），端口 1–65535 |
| 保存的 `slave` | 必须显式传入 1–247，0 不使用默认值 |
| 删除的 `ids` | 1–500 项，每项非空且不能是纯空白，最多 36 个字符；空数组返回参数错误 |
| 批量查询的 `modbusCode` | 1–500 项，逐项校验编码 |
| 分页的 `status` | 0=不过滤、1=启用、2=禁用 |
| 分页的 `page` / `pageSize` | 非正值分别使用 1 / 10，`pageSize` 超过 500 时限制为 500 |

| 参数 | 约束 |
| --- | --- |
| `address` / `readAddress` / `writeAddress` | 0–65535，超范围直接拒绝，不做截断 |
| `quantity`（0x01/0x02） | 1–2000 |
| `quantity`（0x03/0x04） | 1–125 |
| `quantity`（0x0F） | 1–1968，且必须等于 `values` 长度 |
| `quantity`（0x10） | 1–123，且必须等于 `values` 长度 |
| `readQuantity`（0x17） | 1–125 |
| `writeQuantity`（0x17） | 1–121，且必须等于 `values` 长度 |

设备标识 `readDeviceIdCode` 只接受 `1`–`3`；读取指定对象（0x04）使用 `readDeviceIdentificationSpecificObject`。

设备返回 Modbus 异常响应时返回 `100184`，`msg` 形如 `第三方接口错误，vendor=modbus，code=0x02，message=...非法数据地址`；通信超时返回 `100003`，其余通信故障返回 `100183`。

写操作（含 0x17）发生超时、断链或响应流错位时，失败不代表设备没有执行，错误提示为“写入执行结果未知”，需核实设备状态后再决定是否重试。读取发生响应流错位时可直接重新发起。

## 开放平台接入

`app/openapi` 通过 `BridgeModbusConf` 调用本服务，`opengtw` 对外开放以下 15 个 I/O method：

```text
modbus.readCoils
modbus.readDiscreteInputs
modbus.readInputRegisters
modbus.readHoldingRegisters
modbus.readFIFOQueue
modbus.readDeviceIdentification
modbus.readDeviceIdentificationSpecificObject
modbus.writeSingleCoil
modbus.writeMultipleCoils
modbus.writeSingleRegister
modbus.writeSingleRegisterWithDecimal
modbus.writeMultipleRegisters
modbus.writeMultipleRegistersWithDecimal
modbus.readWriteMultipleRegisters
modbus.maskWriteRegister
```

配置管理与本地换算工具暂不通过开放平台暴露。接口字段和示例见 [`docs/open-api/modbus.md`](../../docs/open-api/modbus.md)。

## 启动

```bash
cd app/bridgemodbus
go run . -f etc/bridgemodbus.yaml
```

依赖：

- 可访问的 Modbus TCP 设备或模拟器（默认 `127.0.0.1:5020`）。
- 关系数据库（配置持久化）。

## 相关文档

- [Bridge 协议桥接](../../docs/bridge/README.md)
- [开放 API 网关](../../docs/open-api/README.md)
- [Modbus 开放接口](../../docs/open-api/modbus.md)
