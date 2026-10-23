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

## 连接池

请求携带 `modbusCode` 时，服务从 `PoolManager` 解析对应连接池；未命中时查数据库配置并创建。`modbusCode` 为空时使用 `ModbusClientConf` 的默认连接。连接使用后归还池。

配置项：

- `ModbusPool`：连接池数量。
- `ModbusClientConf`：默认 Modbus 地址与从站号。
- `DB`：配置持久化数据库。

## 开放平台接入

`facade/openapi` 通过 `BridgeModbusConf` 调用本服务，`opengtw` 对外开放以下 15 个 I/O method：

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

- [Bridge 协议桥接](../bridge/README.md)
- [开放 API 网关](../../docs/open-api/README.md)
- [Modbus 开放接口](../../docs/open-api/modbus.md)
