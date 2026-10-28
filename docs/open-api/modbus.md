# Modbus 开放接口

本文档描述开放平台通过 `opengtw` 暴露的 Modbus I/O 能力。底层服务为 `app/bridgemodbus`（gRPC `bridgemodbus.BridgeModbus`，端口 25004），协议为 Modbus TCP。

## 调用方式

```http
POST /v1/gateway
Authorization: Basic base64(AppKey:AppSecret)
Content-Type: application/json
```

```json
{
  "method": "modbus.readHoldingRegisters",
  "data": {
    "modbusCode": "",
    "address": 0,
    "quantity": 4
  },
  "tid": "modbus-demo-1"
}
```

响应统一为：

```json
{"code": 0, "msg": "ok", "data": { ... }, "timestamp": 1761120000000}
```

## 公共字段

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `modbusCode` | string | 否 | Modbus 配置唯一编码；留空使用 bridgemodbus 默认连接配置 |
| `address` | uint32 | 是 | 起始地址，范围 0–65535 |
| `quantity` | uint32 | 视方法 | 读取/写入数量，需符合 Modbus 功能码限制 |

注意事项：

- 地址从 0 开始，具体寄存器映射以现场设备点表为准。
- `quantity` 限制：线圈/离散输入 1–2000，寄存器 1–125。
- 响应中的 `results` 等 `bytes` 字段在 protojson 中是 **base64 字符串**，不是十六进制。
- 读寄存器同时返回 `uintValues`、`intValues`、`hexValues`、`binaryValues`，可按业务选择。
- 所有响应 HTTP 状态为 200，业务结果看 `code`。

## 方法清单

### 读取

| method | 功能码 | 说明 |
| --- | --- | --- |
| `modbus.readCoils` | 0x01 | 读线圈 |
| `modbus.readDiscreteInputs` | 0x02 | 读离散输入 |
| `modbus.readInputRegisters` | 0x04 | 读输入寄存器 |
| `modbus.readHoldingRegisters` | 0x03 | 读保持寄存器 |
| `modbus.readFIFOQueue` | 0x18 | 读 FIFO 队列 |
| `modbus.readDeviceIdentification` | 0x2B/0x0E | 读设备标识 |
| `modbus.readDeviceIdentificationSpecificObject` | 0x2B/0x0E | 读指定设备对象 |

### 写入

| method | 功能码 | 说明 |
| --- | --- | --- |
| `modbus.writeSingleCoil` | 0x05 | 写单个线圈 |
| `modbus.writeMultipleCoils` | 0x0F | 写多个线圈 |
| `modbus.writeSingleRegister` | 0x06 | 写单个保持寄存器 |
| `modbus.writeSingleRegisterWithDecimal` | 0x06 | 按十进制数值写单个寄存器 |
| `modbus.writeMultipleRegisters` | 0x10 | 写多个保持寄存器 |
| `modbus.writeMultipleRegistersWithDecimal` | 0x10 | 按十进制数值写多个寄存器 |
| `modbus.readWriteMultipleRegisters` | 0x17 | 读写多个保持寄存器 |
| `modbus.maskWriteRegister` | 0x16 | 使用 AND/OR 掩码写寄存器 |

> Modbus 配置管理（保存/删除/查询配置）暂不对外开放，由平台管理员维护。

## 读取示例

### 读保持寄存器

```json
{
  "method": "modbus.readHoldingRegisters",
  "data": {"modbusCode": "", "address": 0, "quantity": 2}
}
```

响应 `data`：

```json
{
  "results": "AGT/nA==",
  "uintValues": [100, 65436],
  "intValues": [100, -100],
  "hexValues": ["0x0064", "0xFF9C"],
  "binaryValues": ["0b0000000001100100", "0b1111111110011100"]
}
```

### 读线圈

```json
{
  "method": "modbus.readCoils",
  "data": {"modbusCode": "line-01", "address": 0, "quantity": 4}
}
```

响应 `data`：

```json
{"results": "BQ==", "values": [true, false, true, false]}
```

### 读设备标识

```json
{
  "method": "modbus.readDeviceIdentification",
  "data": {"modbusCode": "", "readDeviceIdCode": 1}
}
```

`readDeviceIdCode`：`1` 基本、`2` 常规、`3` 扩展。

```json
{
  "results": {"0": "Vendor", "1": "Product", "2": "V1.0.0"},
  "hexResults": {"0x00": "Vendor", "0x01": "Product", "0x02": "V1.0.0"},
  "semanticResults": {"VendorName": "Vendor", "ProductCode": "Product", "MajorMinorRevision": "V1.0.0"}
}
```

指定对象：

```json
{
  "method": "modbus.readDeviceIdentificationSpecificObject",
  "data": {"modbusCode": "", "objectId": 0}
}
```

`objectId`：`0x00` 厂商名、`0x01` 产品代码、`0x02` 版本、`0x03` 厂商 URL、`0x04` 产品名、`0x05` 型号、`0x06` 应用名。

## 写入示例

### 写单个线圈

```json
{
  "method": "modbus.writeSingleCoil",
  "data": {"modbusCode": "", "address": 0, "value": true}
}
```

### 写多个线圈

```json
{
  "method": "modbus.writeMultipleCoils",
  "data": {
    "modbusCode": "",
    "address": 0,
    "quantity": 4,
    "values": [true, false, true, false]
  }
}
```

### 写单个寄存器（原始值）

```json
{
  "method": "modbus.writeSingleRegister",
  "data": {"modbusCode": "", "address": 0, "value": 100}
}
```

### 写单个寄存器（十进制）

```json
{
  "method": "modbus.writeSingleRegisterWithDecimal",
  "data": {"modbusCode": "", "address": 0, "value": -12, "unsigned": false}
}
```

### 写多个寄存器

```json
{
  "method": "modbus.writeMultipleRegisters",
  "data": {"modbusCode": "", "address": 0, "quantity": 2, "values": [100, 200]}
}
```

### 读写多个寄存器

```json
{
  "method": "modbus.readWriteMultipleRegisters",
  "data": {
    "modbusCode": "",
    "readAddress": 0,
    "readQuantity": 2,
    "writeAddress": 10,
    "writeQuantity": 2,
    "values": [100, 200]
  }
}
```

### 掩码写寄存器

```json
{
  "method": "modbus.maskWriteRegister",
  "data": {"modbusCode": "", "address": 0, "andMask": 65535, "orMask": 0}
}
```

## 写入返回

写方法通常只返回回显字节，例如：

```json
{"results": "AGQ="}
```

对应原始字节 `00 64`。第三方只需判断 `code === 0`，并按需记录返回的原始字节。

## 错误码

| code | 含义 |
| --- | --- |
| `100101` | data 不是合法 protojson，或字段类型/取值范围错误 |
| `100202` | 记录不存在（例如 modbusCode 对应配置不存在） |
| `100501` | 业务处理失败 |
| `100502` | 设备状态不允许 |
| `100601` | bridgemodbus 不可达（开放面折叠为 `100999`） |
| `130103` | method 未开放 |

## 安全提示

- Modbus 写操作会直接改变现场设备状态，调用前必须确认地址、数量和值。
- 生产环境建议只给可信应用开放写方法，并按应用/接口做权限控制。
- 设备通信超时通常表现为 `100601` 折叠后的 `100999`，需要查看服务端日志定位具体从站。
- `modbusCode` 由平台管理员配置，第三方不能通过开放接口创建或修改连接配置。
