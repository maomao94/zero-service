# ISP Agent Technical Design

## Overview

`ispagent` is a go-zero gRPC service that exposes a stable command API over gRPC and forwards commands over a long-lived TCP connection to the Java implementation of the regional substation remote inspection protocol.

Data flow:

```text
gRPC caller
  -> app/ispagent server
  -> app/ispagent logic
  -> internal ispclient Manager
  -> common/isp message + XML serializer
  -> common/gnetx Client
  -> Java TCP protocol service
```

## Boundaries

### `app/ispagent`

Owns the deployable go-zero service:

- `ispagent.proto`: gRPC contract source.
- `ispagent.go`: service entrypoint.
- `internal/config`: zrpc config and ISP TCP settings. Fixed local identity lives here; remote identity is learned from register response.
- `internal/svc`: initializes the ISP client manager.
- `internal/logic`: RPC orchestration and request validation.
- `internal/server`: generated server delegation.

### `common/isp`

Owns reusable protocol primitives:

- Message struct and message ID helpers.
- Item key-value XML model.
- XML build/parse functions for `PatrolHost` and `PatrolDevice` roots.
- gnetx Serializer/Codec constructor.
- Constants for confirmed command IDs.

`common/isp` must not depend on `app/ispagent`.

## Protocol Contract

TCP frame:

```text
+--------+------------------+------------------+-------+----------+--------------+--------+
| 0xEB90 | TransmitSeq      | ReceiveSeq       | src   | xmlLen   | XML Body     | 0xEB90 |
| 2B BE  | 8B LE           | 8B LE            | 1B    | 4B LE    | UTF-8        | 2B BE  |
+--------+------------------+------------------+-------+----------+--------------+--------+
```

XML body:

```xml
<PatrolHost>
  <SendCode>...</SendCode>
  <ReceiveCode>...</ReceiveCode>
  <Type>...</Type>
  <Code>...</Code>
  <Command>...</Command>
  <Time>...</Time>
  <Items>
    <Item key="value" />
  </Items>
</PatrolHost>
```

Root element is configurable as `PatrolHost` or `PatrolDevice`.

Message ID:

```text
messageId = (type << 16) | command
```

Confirmed initial command IDs:

| Name | Type | Command | messageId |
| --- | ---: | ---: | ---: |
| Register | 251 | 1 | 0xfb0001 |
| Heartbeat | 251 | 2 | 0xfb0002 |
| Generic response without Item | 251 | 3 | 0xfb0003 |
| Generic response with Item | 251 | 4 | 0xfb0004 |
| Patrol device status data | 1 | 0 | 0x10000 |
| Patrol device run data | 2 | 0 | 0x20000 |
| Patrol device coordinates | 3 | 0 | 0x30000 |

## Codec Design

Use `gnetx.NewLengthPrefixCodec` instead of `DelimiterCodec`. Although the frame has `0xEB90` at both start and end, `DelimiterCodec` scans the first delimiter and can emit empty frames when the next frame starts with the same delimiter. The protocol also contains an explicit `XMLLength`, so a length-field codec is more deterministic.

Codec configuration:

```go
gnetx.NewLengthPrefixCodec(
    4,
    binary.LittleEndian,
    serializer,
    gnetx.WithLeadingBytes([]byte{0xEB, 0x90}),
    gnetx.WithTrailingBytes([]byte{0xEB, 0x90}),
    gnetx.WithLengthOffset(19),
    gnetx.WithLengthAdjust(2),
    gnetx.WithStripBytes(2),
)
```

Why `WithStripBytes(2)`: strip only the leading flag, so the serializer receives the binary protocol header fields and can populate `TransmitSeq`, `ReceiveSeq`, and `SessionSource`.

Serializer payload after stripping:

```text
[TransmitSeq 8B LE][ReceiveSeq 8B LE][SessionSource 1B][XMLLength 4B LE][XML Body]
```

The serializer decodes XML metadata and Items into a Go message. It encodes a message by building XML first, then writing the 21-byte non-flag header and XML body. The length field bytes in the serializer output are reserved and overwritten by `LengthPrefixCodec` during encode.

## Message Model

`common/isp.Message` implements:

- `gnetx.Identifiable`: `MessageID()` returns `(Type << 16) | Command`.
- `gnetx.Correlatable`: `TID()` returns the transmit sequence.
- `gnetx.Response`: `ResponseTID()` returns the receive sequence.

Request-response correlation:

- Outbound request uses `TransmitSeq` as the TID.
- Java response returns the original request sequence in `ReceiveSeq`.
- gnetx `Client.Request` resolves the waiting gRPC call through `ReplyPool`.

## Client Lifecycle

`internal/ispclient.Manager` wraps `gnetx.Client` and owns protocol lifecycle:

1. Start one TCP client using configured server address.
2. On initial connection and reconnect, send register message `251-1` with configured local `SendCode` and the initial register target value.
3. Parse register response `251-4`; learn remote identity / subsequent `ReceiveCode` from that response. If the response contains heartbeat interval, use it, otherwise use configured default.
4. Send heartbeat `251-2` on ticker.
5. Expose `Execute(ctx, msg)` for gRPC logic.

Connection model is single Java protocol server connection for this task.

## gRPC Contract

Initial service methods:

- `ExecuteCommand(CommandReq) returns (CommandRes)`: generic command API.
- `SendPatrolDeviceRunData(PatrolItemsReq) returns (CommandRes)`: Type 2, Command 0.
- `SendPatrolDeviceStatusData(PatrolItemsReq) returns (CommandRes)`: Type 1, Command 0.
- `SendPatrolDeviceCoordinates(PatrolItemsReq) returns (CommandRes)`: Type 3, Command 0.
- Optional manual lifecycle methods: `SendRegister` and `SendHeartbeat`.

Response returns parsed data, not only raw XML:

- success flag.
- response code/message when present.
- send/receive sequence numbers.
- type/command/code/time.
- raw XML for diagnostics.
- parsed Items as repeated `map<string,string>` entries.

## Configuration

`app/ispagent/etc/ispagent.yaml`:

```yaml
Name: ispagent.rpc
ListenOn: 0.0.0.0:21006
Mode: dev

IspSetting:
  ServerAddr: 127.0.0.1:7100
  SendCode: testDog
  RegisterReceiveCode: Server01
  RootName: PatrolDevice
  HeartbeatInterval: 30s
  RequestTimeout: 10s
  ReconnectInterval: 3s
  MaxFrameLength: 1048576
```

## Error Handling

Logic layer returns errors and lets the existing RPC interceptor log them. Do not log-and-return in logic. External protocol failures should be wrapped with project error helpers where available; otherwise return stable gRPC errors with sanitized messages.

## Compatibility

- Protocol behavior follows Java `allcore-sip` implementation rather than assuming PDF-only semantics.
- Root element is configurable to handle upstream system differences.
- `ReceiveCode` for normal commands is not fixed configuration; it is learned from the register response. `RegisterReceiveCode` is only an initial bootstrap target for the register packet if the Java side requires one.
- Item model remains dynamic key-value to avoid locking the gRPC contract to incomplete protocol field coverage.

## Out Of Scope

- Multiple simultaneous Java protocol server connections.
- Type 11 model file synchronization.
- Type 41 task management.
- Persistence, Kafka, Redis, or callback delivery.
