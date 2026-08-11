# Research: Bridge Services Architecture & Patterns

- **Query**: Analyze all bridge services (bridgegtw, bridgekafka, bridgemodbus, bridgemqtt, bridgedump) and common packages (mqttx, modbusx, stream)
- **Scope**: internal
- **Date**: 2026-08-11

---

## 1. app/bridgegtw — Bridge Gateway Service

### Directory Structure

```
app/bridgegtw/
├── bridgegtw.go                          # Main entry point
├── bridgegtw.api                         # go-zero API definition
├── doc/base.api                          # Base API types (TenantRequest, etc.)
├── gen.sh                                # Code generation script
├── Dockerfile                            # Multi-stage Docker build
├── etc/bridgegtw.yaml                    # Config (gateway + upstreams)
└── internal/
    ├── config/config.go                  # Config struct (embeds gateway.GatewayConf)
    ├── svc/servicecontext.go             # ServiceContext (minimal, Config-only)
    ├── types/types.go                    # Generated types (PingReply, TenantRequest, etc.)
    ├── handler/routes.go                 # Generated route registration
    ├── handler/bridgeGtw/pinghandler.go  # HTTP handler for /ping
    └── logic/bridgeGtw/pinglogic.go      # Business logic for ping
```

### Key Files and Their Roles

| File | Role |
|---|---|
| `bridgegtw.go` | Main entry: loads config, creates `gateway.Server`, registers routes |
| `bridgegtw.api` | go-zero API DSL: single `bridgeGtw` service with a `/ping` GET endpoint |
| `internal/config/config.go` | Config embeds `gateway.GatewayConf` from go-zero |
| `etc/bridgegtw.yaml` | Defines upstreams (bridgedump gRPC) with gRPC-to-HTTP mappings |

### Entry Point Pattern

**Unique**: This is the only REST/gateway service among the bridges. Uses `go-zero/gateway` instead of `go-zero/rest` directly.

```go
// bridgegtw.go
var c config.Config
conf.MustLoad(*configFile, &c)
server := gateway.MustNewServer(c.GatewayConf, ...)
ctx := svc.NewServiceContext(c)
handler.RegisterHandlers(server.Server, ctx)
server.Start()
```

- Starts on port `15001` (config: `0.0.0.0:15001`)
- Exposes `/bridge/v1/ping` as a health check
- Acts as an HTTP-to-gRPC proxy for bridgedump services
- Upstreams are configured in YAML, mapping HTTP paths to gRPC RPC paths using proto sets

### Proto/API Signatures

- **API DSL**: `bridgegtw.api` (go-zero `.api` format)
- **Service**: `bridgeGtw` with prefix `/bridge/v1`
- **Endpoint**: `GET /ping` → `PingReply { Msg string }`
- **Imported types**: `TenantRequest { Id, TenantId }`, `BaseRequest { Id }`, `EmptyReply`

### Proxy Convention (HTTP → gRPC)

The gateway proxies HTTP requests to gRPC services:
```
POST /api/external/cable/workList → bridgedump.BridgeDumpRpc/CableWorkList
POST /api/external/cable/fault    → bridgedump.BridgeDumpRpc/CableFault
POST /api/external/cable/faultWave → bridgedump.BridgeDumpRpc/CableFaultWave
```
Requires `app/bridgedump/bridgedump.pb` proto descriptor file.

### State Management

Minimal — `ServiceContext` holds only `Config`. No DB, no connections, no caches.

### Error Handling

Standard go-zero rest handler pattern: `httpx.ErrorCtx` on error, `httpx.OkJsonCtx` on success.

### Common Package Dependencies

- `common/tool` — `PrintGoVersion()`
- `github.com/zeromicro/go-zero/gateway` — gateway server
- `github.com/zeromicro/go-zero/rest` — middleware

---

## 2. app/bridgekafka — Bridge Kafka Service

### Directory Structure

```
app/bridgekafka/
├── bridgekafka.go                          # Main entry point
├── bridgekafka.proto                       # Proto definition (1 RPC)
├── gen.sh                                  # Code generation
├── deploy.sh                               # Deployment script
├── Dockerfile
├── etc/bridgekafka.yaml                    # Config (rpc + kafka + stream event)
├── bridgekafka/                            # Generated proto code
│   ├── bridgekafka.pb.go
│   └── bridgekafka_grpc.pb.go
└── internal/
    ├── config/config.go                    # Config struct
    ├── svc/servicecontext.go               # ServiceContext (pushers + stream client)
    ├── server/bridgekafkaserver.go         # gRPC server implementation
    ├── logic/publishlogic.go               # Publish business logic
    └── handler/kafkastreamhandler.go       # Kafka consumer handler
```

### Key Files and Their Roles

| File | Role |
|---|---|
| `bridgekafka.go` | Main: creates gRPC server + Kafka consumers via `service.NewServiceGroup()` |
| `bridgekafka.proto` | Single `BridgeKafka` service with `Publish` RPC |
| `internal/svc/servicecontext.go` | Initializes Kafka pushers map + `StreamEventClient` |
| `internal/handler/kafkastreamhandler.go` | `KafkaStreamHandler` — Kafka consumer that forwards to stream event gRPC |
| `internal/logic/publishlogic.go` | `PublishLogic` — looks up pusher by topic, pushes with optional key |

### Entry Point Pattern

**Unique**: Uses `service.NewServiceGroup()` to run both gRPC server and Kafka consumers together.

```go
// bridgekafka.go
var c config.Config
conf.MustLoad(*configFile, &c)
ctx := svc.NewServiceContext(c)

serviceGroup := service.NewServiceGroup()
defer serviceGroup.Stop()

// For each KafkaConsumeConfig, create a consumer and add to service group
for _, kc := range c.KafkaConsumeConfig {
    h := handler.NewKafkaStreamHandler(kc.Topic, kc.Group, ctx.StreamEventCli)
    serviceGroup.Add(kq.MustNewQueue(fullConf, h))
}

// gRPC server
s := zrpc.MustNewServer(c.RpcServerConf, func(grpcServer *grpc.Server) {
    bridgekafka.RegisterBridgeKafkaServer(grpcServer, server.NewBridgeKafkaServer(ctx))
    if dev/test mode { reflection.Register(grpcServer) }
})

// Nacos registration (optional)
serviceGroup.Add(s)
serviceGroup.Start()
```

- Starts on port `21013`
- Both a gRPC server (for publishing) and Kafka consumer (for consuming and forwarding)
- Uses `go-queue/kq` for Kafka integration

### Proto/API Signatures

```protobuf
service BridgeKafka {
  rpc Publish(PublishReq) returns (PublishRes);
}

message PublishReq {
  string topic = 1;
  string key = 2;
  bytes value = 3;
}
message PublishRes {}
```

### Handler/Logic Pattern

- **Server**: `BridgeKafkaServer.Publish(ctx, in)` → creates `logic.NewPublishLogic(ctx, svcCtx)` → calls `l.Publish(in)`
- **Logic**: `PublishLogic` looks up the pusher from `svcCtx.Pushers[in.Topic]`, pushes with `kq.Pusher.PushWithKey` or `kq.Pusher.Push`
- **Consumer**: `KafkaStreamHandler` implements `kq.ConsumeHandler` interface, forwards messages to stream event gRPC via `taskRunner.Schedule()` (16 concurrent workers)

### Kafka Consumer → gRPC Stream Pattern

When a Kafka message arrives:
1. `KafkaStreamHandler.Consume()` is called by `kq` framework
2. If `streamEventCli` is configured, calls `pushToStreamEvent()` via `threading.TaskRunner`
3. `pushToStreamEvent` creates a `streamevent.ReceiveKafkaMessageReq` with UUID, timestamp, topic, group, key, value
4. Calls `streamEventCli.ReceiveKafkaMessage()` gRPC (non-blocking)

### State Management

`ServiceContext` holds:
- `Pushers map[string]*kq.Pusher` — one pusher per configured topic
- `StreamEventCli streamevent.StreamEventClient` — optional gRPC client for stream forwarding

### Config Structure

```yaml
KafkaPushConfig:     # Topics the service can publish TO
  Brokers: [...]
  Topics: [asdu, alarm, event]
KafkaConsumeConfig:  # Topics the service consumes FROM (multiple entries)
  - Brokers: [...]
    Topic: asdu
    Group: bridge-kafka-asdu
    Conns: 3
    Consumers: 3
    Processors: 18
StreamEventConf:     # gRPC target for forwarding consumed messages
  Endpoints: [...]
```

### Common Package Dependencies

- `common/configx` — `KafkaMultiPushConf`, `KafkaConsumerConf`
- `common/Interceptor/rpcserver` — `LoggerInterceptor`
- `common/Interceptor/rpcclient` — `UnaryMetadataInterceptor`
- `common/nacosx` — Nacos service registration
- `common/carbonx` — carbon time library (blank import)
- `facade/streamevent/streamevent` — Stream event gRPC client
- `github.com/zeromicro/go-queue/kq` — Kafka queue

---

## 3. app/bridgemodbus — Bridge Modbus Service

### Directory Structure

```
app/bridgemodbus/
├── bridgemodbus.go                           # Main entry point
├── bridgemodbus.proto                        # Proto (18 RPCs)
├── gen.sh
├── deploy.sh
├── zgh_start.sh                              # Deploy startup script
├── Dockerfile
├── etc/bridgemodbus.yaml                     # Config (rpc + DB + modbus defaults)
├── bridgemodbus/                             # Generated proto
└── internal/
    ├── config/config.go                      # Config struct
    ├── svc/servicecontext.go                 # ServiceContext (DB, pool, manager)
    ├── server/bridgemodbusserver.go          # gRPC server (18 methods)
    └── logic/
        ├── saveconfiglogic.go                # Save/update modbus slave config
        ├── deleteconfiglogic.go              # Batch delete configs
        ├── pagelistconfiglogic.go            # Paginated config list
        ├── getconfigbycodelogic.go           # Get config by code
        ├── batchgetconfigbycodelogic.go      # Batch get configs by codes
        ├── readcoilslogic.go                 # FC 0x01
        ├── readdiscreteinputslogic.go        # FC 0x02
        ├── writesinglecoillogic.go           # FC 0x05
        ├── writemultiplecoilslogic.go        # FC 0x0F
        ├── readinputregisterslogic.go        # FC 0x04
        ├── readholdingregisterslogic.go      # FC 0x03
        ├── writesingleregisterlogic.go       # FC 0x06
        ├── writesingleregisterwithdecimallogic.go  # FC 0x06 (decimal variant)
        ├── writemultipleregisterslogic.go    # FC 0x10
        ├── writemultipleregisterswithdecimallogic.go # FC 0x10 (decimal variant)
        ├── readwritemultipleregisterslogic.go # FC 0x17
        ├── maskwriteregisterlogic.go         # FC 0x16
        ├── readfifoqueuelogic.go             # FC 0x18
        ├── readdeviceidentificationlogic.go  # FC 0x2B / 0x0E
        ├── readdeviceidentificationspecificobjectlogic.go # FC 0x2B / 0x0E (specific)
        └── batchconvertdecimaltoregisterlogic.go # Utility converter
```

### Key Files and Their Roles

| File | Role |
|---|---|
| `bridgemodbus.go` | Main: creates zrpc server, registers service, optionally registers with Nacos |
| `bridgemodbus.proto` | 18 RPCs covering Modbus config CRUD + all standard Modbus function codes |
| `internal/svc/servicecontext.go` | Initializes DB, ModbusClientPool (default), PoolManager (dynamic), ModbusConfigConverter |
| `internal/server/bridgemodbusserver.go` | gRPC server — one method per RPC, each delegates to a Logic |
| `internal/logic/*.go` | One file per RPC method; each has a Logic struct with `ctx`, `svcCtx`, `Logger` |

### Entry Point Pattern

Standard zrpc pattern with Nacos:

```go
// bridgemodbus.go
var c config.Config
conf.MustLoad(*configFile, &c)
ctx := svc.NewServiceContext(c)

s := zrpc.MustNewServer(c.RpcServerConf, func(grpcServer *grpc.Server) {
    bridgemodbus.RegisterBridgeModbusServer(grpcServer, server.NewBridgeModbusServer(ctx))
    if dev/test mode { reflection.Register(grpcServer) }
})

// Optional Nacos registration
if c.NacosConfig.IsRegister { ... }

s.AddUnaryInterceptors(interceptor.LoggerInterceptor)
s.Start()
```

- Starts on port `25004`
- Uses `s.Start()` directly (no `service.NewServiceGroup()` — single server)

### Proto/API Signatures

The service has 18 RPCs organized into 4 groups:

1. **Config Management** (5 RPCs): `SaveConfig`, `DeleteConfig`, `PageListConfig`, `GetConfigByCode`, `BatchGetConfigByCode`
2. **Bit Access** (4 RPCs): `ReadCoils`, `ReadDiscreteInputs`, `WriteSingleCoil`, `WriteMultipleCoils`
3. **16-bit Register Access** (7 RPCs): `ReadInputRegisters`, `ReadHoldingRegisters`, `WriteSingleRegister`, `WriteSingleRegisterWithDecimal`, `WriteMultipleRegisters`, `WriteMultipleRegistersWithDecimal`, `ReadWriteMultipleRegisters`, `MaskWriteRegister`, `ReadFIFOQueue`
4. **Device Identification** (2 RPCs): `ReadDeviceIdentification`, `ReadDeviceIdentificationSpecificObject`
5. **Utility** (1 RPC): `BatchConvertDecimalToRegister`

All Modbus RPCs accept `modbusCode` (config identifier) to dynamically resolve the Modbus connection.

### Connection Pool Pattern (Key Convention)

**Dynamic pool resolution** — the most architecturally significant pattern in bridgemodbus:

```
Request with modbusCode
  → ServiceContext.GetModbusClientPool(ctx, modbusCode)
    → If modbusCode is empty: return default pool (from config file)
    → If modbusCode is set: check PoolManager for existing pool
      → If found: return it
      → If not found: AddPool(ctx, modbusCode)
        → Query DB for ModbusSlaveConfig by modbusCode
        → Convert DB model to ModbusClientConf
        → Create new ModbusClientPool via PoolManager
```

The `ServiceContext` has:
- `ModbusClientPool` — default pool (from config file `ModbusClientConf`)
- `Manager *PoolManager` — dynamic pool manager (creates pools on-demand from DB configs)
- `ModbusConfigConverter` — converts `gormmodel.ModbusSlaveConfig` → `modbusx.ModbusClientConf`
- `DB` — gorm database connection

### Logic Pattern

Every logic file follows an identical structure:

```go
// readholdingregisterslogic.go
type ReadHoldingRegistersLogic struct {
    ctx    context.Context
    svcCtx *svc.ServiceContext
    logx.Logger
}

func NewReadHoldingRegistersLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ReadHoldingRegistersLogic {
    return &ReadHoldingRegistersLogic{
        ctx:    ctx,
        svcCtx: svcCtx,
        Logger: logx.WithContext(ctx),
    }
}

func (l *ReadHoldingRegistersLogic) ReadHoldingRegisters(in *bridgemodbus.ReadHoldingRegistersReq) (*bridgemodbus.ReadHoldingRegistersRes, error) {
    // 1. Get pool (dynamic or default)
    mdCliPool, err := l.svcCtx.GetModbusClientPool(l.ctx, in.ModbusCode)
    if err != nil { return nil, err }
    // 2. Get client from pool
    mbCli := mdCliPool.Get()
    defer mdCliPool.Put(mbCli)
    // 3. Execute Modbus operation
    results, err := mbCli.ReadHoldingRegisters(l.ctx, uint16(in.Address), uint16(in.Quantity))
    if err != nil { return nil, err }
    // 4. Convert results using bytex utilities
    bv := bytex.BytesToBinaryValues(results)
    return &bridgemodbus.ReadHoldingRegistersRes{...}, nil
}
```

### Config CRUD Pattern

`SaveConfigLogic` uses DB transactions (`gormx.DB.Transact`):
- Check if `modbusCode` exists → update; otherwise → create
- Uses `gormmodel.ModbusSlaveConfig` as the ORM model

`GetConfigByCodeLogic`:
- Queries DB by `modbus_code`
- Returns `gorm.ErrRecordNotFound` as a controlled error
- Copies DB model to proto using `copier.CopyWithOption`

### Config Structure

```yaml
ModbusPool: 32                                 # Default pool size
DB:
  DataSource: postgres://...                    # GORM DSN
ModbusClientConf:
  Address: 127.0.0.1:5020                      # Default Modbus TCP address
  Slave: 1                                      # Default slave ID
```

### Common Package Dependencies

- `common/modbusx` — `ModbusClient`, `ModbusClientPool`, `PoolManager`, `ModbusClientConf`
- `common/gormx` — GORM DB with config
- `common/bytex` — Byte-to-value conversion utilities (`BytesToBinaryValues`, `Uint16SliceToUint32Slice`, etc.)
- `common/copierx` — Copier options for DB model ↔ proto mapping
- `common/tool` — Error creation (`NewErrorByPbCode`, `NewErrorByPbCodeWrap`)
- `model/gormmodel` — `ModbusSlaveConfig` ORM model, `ModbusConfigConverter`
- `github.com/jinzhu/copier` — Struct copying
- `github.com/grid-x/modbus` — Modbus protocol library
- `gorm.io/gorm` — ORM

---

## 4. app/bridgemqtt — Bridge MQTT Service

### Directory Structure

```
app/bridgemqtt/
├── bridgemqtt.go                           # Main entry point
├── bridgemqtt.proto                        # Proto (2 RPCs)
├── gen.sh
├── deploy.sh
├── Dockerfile
├── etc/bridgemqtt.yaml                     # Config (rpc + mqtt + stream/socket clients)
├── bridgemqtt/                             # Generated proto
└── internal/
    ├── config/config.go                    # Config struct (with EventMapping)
    ├── svc/servicecontext.go               # ServiceContext (mqtt client + stream clients)
    ├── server/bridgemqttserver.go          # gRPC server (2 methods)
    ├── logic/
    │   ├── publishlogic.go                 # Publish to MQTT
    │   └── publishwithtracelogic.go        # Publish with trace ID
    └── handler/
        └── mqttstreamhandler.go            # MQTT → Stream/Socket forwarder
```

### Key Files and Their Roles

| File | Role |
|---|---|
| `bridgemqtt.go` | Main: creates zrpc server, registers service, optional Nacos |
| `bridgemqtt.proto` | `BridgeMqtt` service with `Publish` and `PublishWithTrace` |
| `internal/svc/servicecontext.go` | Initializes MQTT client with `OnReady` callback to register stream handler |
| `internal/handler/mqttstreamhandler.go` | `MqttStreamHandler` — forwards MQTT messages to stream event gRPC and/or socket push |
| `internal/logic/publishlogic.go` | `PublishLogic` — delegates to `mqttx.Client.Publish()` |
| `internal/logic/publishwithtracelogic.go` | `PublishWithTraceLogic` — delegates to `mqttx.Client.PublishWithTrace()` |

### Entry Point Pattern

Standard zrpc pattern, same as bridgemodbus:

```go
// bridgemqtt.go
var c config.Config
conf.MustLoad(*configFile, &c)
ctx := svc.NewServiceContext(c)

s := zrpc.MustNewServer(c.RpcServerConf, func(grpcServer *grpc.Server) {
    bridgemqtt.RegisterBridgeMqttServer(grpcServer, server.NewBridgeMqttServer(ctx))
    if dev/test mode { reflection.Register(grpcServer) }
})

// Optional Nacos registration
s.AddUnaryInterceptors(interceptor.LoggerInterceptor)
s.Start()
```

- Starts on port `25005`

### Proto/API Signatures

```protobuf
service BridgeMqtt {
  rpc Publish(PublishReq) returns (PublishRes);
  rpc PublishWithTrace(PublishWithTraceReq) returns (PublishWithTraceRes);
}

message PublishReq     { string topic = 1; bytes payload = 2; }
message PublishRes     {}
message PublishWithTraceReq  { string topic = 1; bytes payload = 2; }
message PublishWithTraceRes  { string traceId = 1; }
```

### MQTT Client Initialization Pattern (Key Convention)

The `ServiceContext` creates the MQTT client with an `OnReady` callback:

```go
mqttCLi := mqttx.MustNewClient(c.MqttConfig,
    mqttx.WithOnReady(func(cli mqttx.Client) {
        // On first connect, register stream forward handler for each subscribed topic
        for _, topic := range c.MqttConfig.SubscribeTopics {
            cli.AddHandler(topic, handler.NewMqttStreamHandler(
                cli.GetClientID(), streamEventCli, socketPushCli,
                c.EventMapping, c.DefaultEvent, c.LogConfig))
        }
    }),
)
```

### MQTT → Stream/Socket Forwarding Pattern

`MqttStreamHandler` implements `mqttx.ConsumeHandler`:

```
MQTT message received on subscribed topic
  → MqttStreamHandler.Consume(ctx, payload, topic, topicTemplate)
    → logMessage() — conditionally log based on TopicLogManager config
    → pushToStreamEvent() — via taskRunner (16 workers), sends to stream event gRPC
    → pushToSocket() — via taskRunner, broadcasts to WebSocket rooms by topicTemplate
```

**Dual fan-out**: Messages are forwarded to BOTH stream event and socket push, each in its own goroutine via `taskRunner`.

**Event mapping**: `EventMapping` config maps `topicTemplate` → `event` name for socket push. Falls back to `DefaultEvent` ("mqtt").

### State Management

`ServiceContext` holds:
- `MqttClient mqttx.Client` — the central MQTT client (connect, publish, subscribe, handler dispatch)

The MQTT client internally manages:
- `handlerMgr` — handlers per topic template
- `dispatcher` — message dispatch with reply handler priority
- `subscribed` — tracked subscriptions for reconnection recovery

### Config Structure

```yaml
MqttConfig:
  Broker: ["tcp://localhost:1883"]
  Qos: 0
  Timeout: 30000
  KeepAlive: 60000
  SubscribeTopics: ["test", "test/topic2", "iec/#"]
EventMapping:                          # Optional: map topic to event name for socket push
  - topicTemplate: "iec/#"
    event: "iecEvent"
DefaultEvent: mqtt                     # Default event name
LogConfig:                             # Optional: per-topic log control
  defaultLogPayload: false
  topicSettings: [...]
StreamEventConf:                       # Optional: stream event gRPC client
SocketPushConf:                        # Optional: socket push gRPC client
```

### Common Package Dependencies

- `common/mqttx` — Full MQTT client library (client, config, dispatcher, message, reply_router, request_replyer, topic_log)
- `common/Interceptor/rpcserver` — `LoggerInterceptor`
- `common/Interceptor/rpcclient` — `UnaryMetadataInterceptor`
- `common/carbonx` — carbon time library
- `common/nacosx` — Nacos registration
- `facade/streamevent/streamevent` — Stream event gRPC client
- `socketapp/socketpush/socketpush` — Socket push gRPC client
- `github.com/eclipse/paho.mqtt.golang` — MQTT protocol library

---

## 5. app/bridgedump — Bridge Dump Service

### Directory Structure

```
app/bridgedump/
├── bridgedump.go                           # Main entry point
├── bridgedump.proto                        # Proto (4 RPCs)
├── gen.sh
├── Dockerfile
├── etc/bridgedump.yaml                     # Config (rpc + DumpPath)
├── bridgedump/                             # Generated proto
└── internal/
    ├── config/config.go                    # Config struct (RpcServerConf + DumpPath)
    ├── svc/servicecontext.go               # ServiceContext (Config + DumpBridgeData method)
    ├── server/bridgedumprpcserver.go       # gRPC server (4 methods)
    └── logic/
        ├── pinglogic.go                    # Health check
        ├── cableworklistlogic.go           # Cable device run data dump
        ├── cablefaultlogic.go              # Cable fault data dump
        └── cablefaultwavelogic.go          # Cable fault wave data dump
```

### Key Files and Their Roles

| File | Role |
|---|---|
| `bridgedump.go` | Main: standard zrpc server, no Nacos |
| `bridgedump.proto` | `BridgeDumpRpc` service: Ping + 3 cable data dump RPCs |
| `internal/svc/servicecontext.go` | Contains `DumpBridgeData()` — the core file-writing logic, lives on ServiceContext |
| `internal/logic/*.go` | Each calls `svcCtx.DumpBridgeData(ctx, dumpPath, subDir, in)` |

### Entry Point Pattern

Simplest of all bridges — standard zrpc, no Nacos, no interceptors:

```go
// bridgedump.go
var c config.Config
conf.MustLoad(*configFile, &c)
ctx := svc.NewServiceContext(c)

s := zrpc.MustNewServer(c.RpcServerConf, func(grpcServer *grpc.Server) {
    bridgedump.RegisterBridgeDumpRpcServer(grpcServer, server.NewBridgeDumpRpcServer(ctx))
    if dev/test mode { reflection.Register(grpcServer) }
})
defer s.Stop()
s.Start()
```

- Starts on port `25003`
- No Nacos registration, no interceptors, minimal dependencies

### Proto/API Signatures

```protobuf
service BridgeDumpRpc {
  rpc Ping(Req) returns (Res);
  rpc CableWorkList(CableWorkListReq) returns (CableWorkListRes);
  rpc CableFault(CableFaultReq) returns (CableFaultRes);
  rpc CableFaultWave(CableFaultWaveReq) returns (CableFaultWaveRes);
}
```

Three cable data types:
- `CableWorkListReq` — `repeated DeviceRunData` (DTU ID, currents, voltages, GPS, temperature)
- `CableFaultReq` — `repeated FaultData` (fault location, type, category, distance)
- `CableFaultWaveReq` — `repeated FaultWaveData` (wave data, sampling rate, timestamps)

All three cable responses are `{ code int32, msg string }`.

### Dump Mechanism (Core Convention)

`DumpBridgeData()` creates timestamped files with a custom wire format:

```go
func (s *ServiceContext) DumpBridgeData(ctx context.Context, dumpPath, subDir string, in any) (string, error) {
    // 1. Extract trace ID from context
    // 2. Create directory: dumpPath/subDir/
    // 3. Marshal request body to JSON
    // 4. Wrap in BridgeMsgBody { TraceId, Body, Time, FilePath }
    // 5. Write file with format:
    //    <!System=OMG Version=1.05 Code=utf-8 Data=1.0!>
    //    <Bridge:=Free Size=N>
    //    <JSON body>
    //    </Bridge:=Free>
    // 6. Filename: Ymd_His_<traceID>_json.txt
    return writeFilePath, nil
}
```

File path pattern: `{DumpPath}/{subDir}/{YYYYMMDD_HHmmss}_{traceID}_json.txt`

SubDirs: `cable_work_list`, `cable_fault`, `cable_fault_wave`

### Logic Pattern

All three cable logic files follow an identical pattern:

```go
func (l *CableWorkListLogic) CableWorkList(in *bridgedump.CableWorkListReq) (*bridgedump.CableWorkListRes, error) {
    _, err := l.svcCtx.DumpBridgeData(l.ctx, l.svcCtx.Config.DumpPath, cableWorkListDataFile, in)
    if err != nil {
        return nil, err
    }
    return &bridgedump.CableWorkListRes{Code: 200, Msg: "成功"}, nil
}
```

### State Management

Minimal — `ServiceContext` holds only `Config`. The dump method is a stateless function on the context.

### Config Structure

```yaml
Name: bridgedump.rpc
ListenOn: 0.0.0.0:25003
DumpPath: /opt/bridgedump
```

### Common Package Dependencies

- `model` — `BridgeMsgBody` struct
- `github.com/dromara/carbon/v2` — Time formatting
- `github.com/duke-git/lancet/v2/fileutil` — File I/O utilities

---

## 6. Common Packages

### 6.1 common/mqttx — MQTT Client Library

#### Directory Structure

```
common/mqttx/
├── client.go           # Client interface + mqttClient implementation (~420 lines)
├── config.go           # MqttConfig, ClientOptions, ClientOption
├── message.go          # Message wrapper (Topic, Payload, Headers for tracing)
├── dispatcher.go       # messageDispatcher, ConsumeHandler interface, handlerManager
├── errors.go           # Sentinels: ErrNilDecoder, ErrEmptyReplyTid, ErrNoReplyRouter, etc.
├── reply_router.go     # ReplyRouter[T] — generic request/reply matching
├── request_replyer.go  # RequestReply[T] — typed request/reply API
├── topic_log.go        # TopicLogManager — per-topic log control with rate limiting
├── config_test.go      # Tests
└── reply_router_test.go
```

#### Architecture

```
                   ┌─────────────────────────┐
                   │   Client (interface)     │
                   │   - Publish              │
                   │   - PublishWithTrace     │
                   │   - AddHandler           │
                   │   - AddHandlerFunc       │
                   │   - Close                │
                   │   - GetClientID          │
                   └─────────┬───────────────┘
                             │
                   ┌─────────▼───────────────┐
                   │   mqttClient (impl)      │
                   │   - client (paho)        │
                   │   - handlerMgr           │
                   │   - dispatcher           │
                   │   - tracer (OTel)        │
                   │   - metrics              │
                   └──────┬──────────┬───────┘
                          │          │
              ┌───────────▼──┐  ┌────▼──────────────┐
              │ handlerManager│  │ messageDispatcher │
              │ - handlers    │  │ - dispatch()      │
              │ - replyHandlers│ │ - reply priority  │
              └───────────────┘  └───────────────────┘
```

#### Key Abstractions

1. **`Client` interface** — Caller-facing: `Publish`, `PublishWithTrace`, `AddHandler`, `AddHandlerFunc`, `Close`, `GetClientID`
2. **`ConsumeHandler` interface** — `Consume(ctx, payload, topic, topicTemplate) error`; adapters: `ConsumeHandlerFunc`
3. **`ReplyRouter[T]`** — Generic request/reply matching: register decoder, wait for matching reply by TID, resolve pending requests
4. **`TopicLogManager`** — Per-topic log control: rate limiting (`MinLogInterval`), payload logging toggle
5. **`Message`** — JSON wrapper with `Headers` for OTel trace context carrier

#### Dispatch Flow

```
MQTT message arrives
  → processMessage(msg, topicTemplate)
    → tryUnwrapPayload() — parse JSON wrapper for trace context
    → extractTraceContext() — extract OTel span context from headers
    → startSpan() — create consumer span
    → dispatcher.dispatch(ctx, payload, topic, topicTemplate)
      → getReplyHandler(topicTemplate) — if exists, call reply handler first
      → getHandlers(topicTemplate) — then call all normal handlers
      → if no handlers at all: onNoHandler (log)
```

#### Connection Lifecycle

- `MustNewClient()` — creates client, connects, registers `proc.AddWrapUpListener` for graceful shutdown
- `onConnect()` — triggers `onReady` callback (once via `atomic.Bool`), restores subscriptions
- `onConnectionLost()` — clears subscription tracking, paho auto-reconnects
- Auto client ID generation via UUID if not provided

---

### 6.2 common/modbusx — Modbus Client Library

#### Directory Structure

```
common/modbusx/
├── client.go    # ModbusClient, ModbusClientPool, PoolManager (~217 lines)
└── config.go    # ModbusClientConf, PoolManager, device ID constants
```

#### Key Abstractions

1. **`ModbusClient`** — Wraps `grid-x/modbus.Client` + `modbus.TCPClientHandler`. Implements all standard Modbus function codes (0x01–0x18, 0x2B). Supports TLS.
2. **`ModbusClientPool`** — `syncx.Pool`-based connection pool with:
   - Factory: creates `ModbusClient` via `NewModbusClient()`
   - Destructor: closes TCP handler
   - Auto-cleanup: 10-minute max age
   - Get/Put pattern: `pool.Get()` → use → `pool.Put(cli)`
3. **`PoolManager`** — Manages multiple `ModbusClientPool` instances keyed by `modbusCode`. Thread-safe with `sync.RWMutex`. Methods: `AddPool`, `GetPool`.
4. **`ModbusClientConf`** — TCP connection config: Address, Slave, Timeout, IdleTimeout, LinkRecoveryTimeout, ProtocolRecoveryTimeout, ConnectDelay, TLS
5. **`ModbusLogger`** — Custom logger for the modbus library, logs with address MD5 and session ID in context fields

#### PoolManager → ModbusClient Flow

```
Request with modbusCode
  → PoolManager.GetPool(modbusCode)
    → Found: return existing ModbusClientPool
    → Not found: AddPool(modbusCode, conf, poolSize)
      → Validate params
      → If exists: return existing (no duplicate)
      → Create new ModbusClientPool(conf, poolSize) using syncx.Pool
      → Store in pools map
    → pool.Get() returns a ModbusClient
    → Execute modbus operation
    → pool.Put(client) returns to pool
```

#### TLS Support

`NewModbusClient()` supports TLS configuration:
- Loads certificate/key pair
- Optionally loads CA cert for mutual TLS
- Configures `modbus.WithTLSConfig()`

---

### 6.3 common/stream — Stream Sender Library

#### Directory Structure

```
common/stream/
├── stream.go       # Sender interface, StreamEvent types, event helpers
└── grpc_sender.go  # GRPCSender, GRPCStreamSender, chunk helpers
```

#### Key Abstractions

1. **`Sender` interface** — `io.Writer` + `SendJSON(v any)`, `SendDone()`, `SendError(err error)`
2. **`StreamEvent`** — Typed events: `text`, `tool_call`, `tool_result`, `thinking`, `interrupt`, `error`, `done`
3. **`GRPCSender`** — Implements `Sender` for gRPC streams using a `sendFunc(interface{}) error`
4. **`GRPCStreamSender`** — Typed version using `GRPCStreamChunk{SessionID, Data, IsFinal, Error}`

#### Note on Usage

This package is **not directly used** by any of the five bridge services. It appears to be a general-purpose streaming utility likely used by other services in the project (e.g., AI/chat services). Noted here for completeness.

---

## 7. Cross-Cutting Patterns

### Entry Point Patterns Summary

| Service | Server Type | Group? | Nacos? | Port |
|---|---|---|---|---|
| bridgegtw | `gateway.Server` | No | No | 15001 |
| bridgekafka | `zrpc` + `kq.Consumers` | `service.NewServiceGroup()` | Yes (optional) | 21013 |
| bridgemodbus | `zrpc` | No | Yes (optional) | 25004 |
| bridgemqtt | `zrpc` | No | Yes (optional) | 25005 |
| bridgedump | `zrpc` | No | No | 25003 |

### ServiceContext Patterns

| Service | Holds |
|---|---|
| bridgegtw | `Config` only |
| bridgekafka | `Config`, `Pushers map[string]*kq.Pusher`, `StreamEventCli` |
| bridgemodbus | `Config`, `DB`, `ModbusConfigConverter`, `ModbusClientPool` (default), `Manager` (dynamic pools) |
| bridgemqtt | `Config`, `MqttClient` |
| bridgedump | `Config`, `DumpBridgeData()` method |

### Interceptor Pattern

All non-gateway services use `interceptor.LoggerInterceptor` (from `common/Interceptor/rpcserver`) when they have a gRPC server. The gateway service has no interceptor.

### Config Flag Pattern

All services use `flag.String("f", "etc/<service>.yaml", "the config file")` and `conf.MustLoad(*configFile, &c)`.

### Logging Pattern

All services set up logging via `logx.Must(logx.SetUp(c.Log))` in `ServiceContext` construction, and add a global field `logx.Field("app", c.Name)` in main.

### Dockerfile Pattern

All five services share an identical Dockerfile pattern:
- Builder: `golang:1.23-alpine3.22`
- Runtime: `FROM scratch`
- Copy binary, config, and any needed files (e.g., proto descriptors for gateway)
- CMD: `./<binary> -f etc/<binary>.yaml`

### Nacos Registration Pattern

Used by bridgekafka, bridgemodbus, bridgemqtt (not bridgegtw, bridgedump):
```go
if c.NacosConfig.IsRegister {
    sc := []constant.ServerConfig{*constant.NewServerConfig(c.NacosConfig.Host, c.NacosConfig.Port)}
    cc := &constant.ClientConfig{...}
    m := map[string]string{
        "gRPC_port": strutil.After(c.RpcServerConf.ListenOn, ":"),
        "preserved.register.source": "go-zero",
    }
    opts := nacosx.NewNacosConfig(c.NacosConfig.ServiceName, c.ListenOn, sc, cc, nacosx.WithMetadata(m))
    _ = nacosx.RegisterService(opts)
}
```

### Error Handling Patterns

- bridgemodbus: Uses `common/tool.NewErrorByPbCode()` / `NewErrorByPbCodeWrap()` with proto error codes
- bridgekafka: Returns `fmt.Errorf("kafka topic %s not configured", in.Topic)` for missing topics; native kq errors for push failures
- bridgemqtt: Delegates to `mqttx.Client.Publish()` which returns native paho errors
- bridgedump: Returns errors from file operations directly, wraps in `BridgeMsgBody`
- bridgegtw: Uses `httpx.ErrorCtx` and `httpx.OkJsonCtx`

### Common Package Dependencies Matrix

| Package | bridgegtw | bridgekafka | bridgemodbus | bridgemqtt | bridgedump |
|---|---|---|---|---|---|
| `common/tool` | ✓ | ✓ | ✓ | ✓ | ✓ |
| `common/Interceptor/rpcserver` | | ✓ | ✓ | ✓ | |
| `common/Interceptor/rpcclient` | | ✓ | | ✓ | |
| `common/nacosx` | | ✓ | ✓ | ✓ | |
| `common/carbonx` | | ✓ | | ✓ | ✓ |
| `common/mqttx` | | | | ✓ | |
| `common/modbusx` | | | ✓ | | |
| `common/gormx` | | | ✓ | | |
| `common/bytex` | | | ✓ | | |
| `common/configx` | | ✓ | | | |
| `facade/streamevent` | | ✓ | | ✓ | |
| `socketapp/socketpush` | | | | ✓ | |
| `model` | | | ✓ | | ✓ |

---

## 8. Service Relationships & Data Flow

```
External Systems (IoT, Kafka, MQTT)
        │
        ├── Kafka ──→ bridgekafka (consumer)
        │                │
        │                ├── gRPC Publish → Kafka (producer)
        │                └── gRPC → stream-event service
        │
        ├── MQTT ──→ bridgemqtt (subscriber)
        │                │
        │                ├── gRPC Publish/PublishWithTrace → MQTT (publisher)
        │                ├── gRPC → stream-event service
        │                └── gRPC → socket-push service (WebSocket broadcast)
        │
        ├── Modbus TCP ──→ bridgemodbus (protocol bridge)
        │                     │
        │                     ├── gRPC: all Modbus FCs (0x01–0x18, 0x2B)
        │                     └── Config management via DB
        │
        └── External HTTP ──→ bridgegtw (gateway)
                                │
                                └── gRPC proxy → bridgedump
                                                    │
                                                    └── File dump to disk
```

### bridgegtw ↔ bridgedump Integration

`bridgegtw` proxies HTTP requests to `bridgedump` gRPC:
- The gateway's `etc/bridgegtw.yaml` maps HTTP POST paths to `bridgedump.BridgeDumpRpc/*` RPCs
- Requires `bridgedump.pb` proto descriptor file at `app/bridgedump/bridgedump.pb`
- Uses `NonBlock: true` for the gRPC upstream

---

## 9. Caveats / Not Found

- **bridgegtw proxy mechanism**: The gateway-to-dump proxy is configured but commented-out HTTP upstreams suggest a possible migration path from HTTP to gRPC. The `app/bridgedump/bridgedump.pb` file copied in Dockerfile is a compiled proto descriptor (not `.proto` source).
- **bridgemodbus env files**: Multiple env files (`test.env`, `test_105.env`) exist suggesting environment-specific configs, but these were not analyzed in detail.
- **common/stream**: Not used by any bridge service — appears designed for other parts of the project (likely AI/chat streaming).
- **bridgekafka deploy.sh**: Contains a deployment script not analyzed.
- **bridgemodbus zgh_start.sh**: Custom startup script suggesting a specific deployment environment.
- **bridgegtw gen.sh vs api**: The gen.sh likely generates `internal/handler/routes.go` and `internal/types/types.go` from the `.api` file using `goctl`.
