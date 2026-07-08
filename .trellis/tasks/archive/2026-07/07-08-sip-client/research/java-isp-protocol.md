# Java ISP Protocol Research

## Sources

- PDF path provided by user: `/Users/hehanpeng/Desktop/工作/众芯汉创/项目/青海无人值守/2-区域型变电站远程智能巡视系统技术规范（试行）0506.pdf`
- Java service path: `/Users/hehanpeng/IdeaProjects/qinghai-biz/allcore-business/allcore-sip`
- Transport reference: `/Users/hehanpeng/IdeaProjects/qinghai-biz/allcore-business/allcore-sip/src/main/java/com/allcore/sip/transport`
- User-provided Go gnet demo has been verified against Java parsing behavior.

## Frame Format

The Java TCP service uses a Netty delimiter-based transport with `0xEB90`, but the payload itself includes both start and end flags. For Go/gnetx implementation, prefer length-field framing using the XML length field instead of raw delimiter splitting.

Full frame:

```text
+--------+------------------+------------------+-------+----------+--------------+--------+
| 0xEB90 | sendSerialNo     | receiveSerialNo  | src   | xmlLen   | XML Body     | 0xEB90 |
| 2B BE  | 8B LE           | 8B LE            | 1B    | 4B LE    | UTF-8        | 2B BE  |
+--------+------------------+------------------+-------+----------+--------------+--------+
```

Java encoder writes:

- `0xEB90` as start flag.
- `sendSerialNo` as little-endian long.
- `receiveSerialNo` as little-endian long.
- `sessionSource` as byte (`0x00` client, `0x01` server).
- XML byte length as little-endian int.
- XML UTF-8 body.
- `0xEB90` as end flag.

## XML Model

Canonical XML shape:

```xml
<PatrolHost>
  <SendCode>...</SendCode>
  <ReceiveCode>...</ReceiveCode>
  <Type>...</Type>
  <Code>...</Code>
  <Command>...</Command>
  <Time>...</Time>
  <Items>
    <Item attr="value" />
  </Items>
</PatrolHost>
```

Root name must be configurable. User confirmed Java side switches root based on upstream system attributes, so Go `ispagent` must support both `PatrolHost` and `PatrolDevice`.

`Item` is dynamic: Java maps XML attributes to a map-like object, not fixed structs for most T2513 messages.

## Message ID Rules

Java `TSip.encode(type, command)`:

```text
messageId = (type << 16) | command
```

Confirmed constants from `TSip.java`:

| Name | Type | Command | Hex |
| --- | ---: | ---: | --- |
| 注册指令 | 251 | 1 | `0xfb0001` |
| 心跳指令 | 251 | 2 | `0xfb0002` |
| 通用应答_无Item | 251 | 3 | `0xfb0003` |
| 通用应答_有Item | 251 | 4 | `0xfb0004` |
| 巡视设备状态数据 | 1 | 0 | `0x10000` |
| 巡视设备运行数据 | 2 | 0 | `0x20000` |
| 巡视设备坐标 | 3 | 0 | `0x30000` |

## Registration And ReceiveCode

User corrected that normal command `ReceiveCode` should not be fixed configuration. The client sends registration first; the service response determines what target code should be used later.

Design implication:

- Configure local `SendCode`.
- Optionally configure `RegisterReceiveCode` only for bootstrap registration packet if required by the Java service.
- Store the learned remote code from `251-4` and use it as `ReceiveCode` for normal commands and heartbeat.

## Initial gRPC Business Methods

The user wants a generic command interface plus three specific test-oriented business methods:

- `SendPatrolDeviceStatusData`: Type `1`, Command `0`.
- `SendPatrolDeviceRunData`: Type `2`, Command `0`.
- `SendPatrolDeviceCoordinates`: Type `3`, Command `0`.

These are client-to-server reports used to verify the reliability of the TCP client. Do not implement all protocol commands in the first iteration.
