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
- **本地换算**：十进制数值与 Modbus 寄存器格式批量转换。

## 客户端复用与配置

按 `modbusCode` 复用长期客户端：同一 code 串行收发，不同 code 各自持有连接和固定从站号。

- `modbusCode` 为空：使用文件中的 `ModbusClientConf`。
- `modbusCode` 非空：读取 `modbus_slave_config` 中启用的配置；不存在返回 `100052`，未启用或 `slave` 不在 1–247 返回 `100140`，不回退文件配置。
- 保存/删除配置后立即失效对应客户端；直接改库时由后台定期校验后生效（通常 30 秒内），无需重启。数据库查询失败时保留现有连接。
- 服务退出时关闭全部客户端；写操作不会自动重试。

配置缺省值：`Timeout=10000ms`（上限 600000）、`IdleTimeout=60000ms`、`Slave=1`，恢复窗口与 `ConnectDelay` 为 0。需要长连接可设 `IdleTimeout=-1`；需要只读断链恢复或迟到响应处理，可分别设 `LinkRecoveryTimeout=3000`、`ProtocolRecoveryTimeout=2000`。

表中新建记录的默认值以 GORM 模型 `gormmodel.ModbusSlaveConfig` 为准，已有记录中的显式值不自动改写。TLS 证书文件内容变更需重启服务或改动配置后生效。

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
