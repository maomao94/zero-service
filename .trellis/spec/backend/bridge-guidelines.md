# Bridge 协议网关规范

## 适用范围

修改 `app/bridgegtw`、`app/bridgekafka`、`app/bridgemodbus`、`app/bridgemqtt`、`app/bridgedump` 或公共库 `common/mqttx`、`common/modbusx`、`common/stream` 时读取。

## 服务总览

| 服务 | 协议方向 | 端口 | 服务器类型 | Nacos | 核心模式 |
| --- | --- | --- | --- | --- | --- |
| bridgegtw | HTTP → gRPC 代理 | 15001 | `gateway.Server` | 否 | REST 网关，YAML upstreams |
| bridgekafka | gRPC + Kafka 双工 | 21013 | `zrpc` + `kq` 消费者组 | 可选 | Pusher 发布 + StreamHandler 消费转发 |
| bridgemodbus | Modbus TCP 协议桥 | 25004 | `zrpc` | 可选 | 动态连接池（PoolManager）+ DB 配置 |
| bridgemqtt | gRPC + MQTT 双工 | 25005 | `zrpc` | 可选 | OnReady 注册 handler + 双路 fan-out |
| bridgedump | 文件落地 | 25003 | `zrpc` | 否 | 自定义报文格式，时间戳文件 |

依据：`app/bridgegtw`、`app/bridgekafka`、`app/bridgemodbus`、`app/bridgemqtt`、`app/bridgedump` 各服务源码与配置。

## 公共入口模式

所有 bridge 服务共享以下入口约定：

```go
var c config.Config
conf.MustLoad(*configFile, &c)
ctx := svc.NewServiceContext(c)
```

- Config 文件路径通过 `flag.String("f", "etc/<service>.yaml", ...)` 指定。
- `ServiceContext` 内按服务需要初始化 client、pool、DB 等。
- 启动日志使用 `logx.Field("app", c.Name)`，在 ServiceContext 构造中调用 `logx.Must(logx.SetUp(c.Log))`。
- gRPC 服务通过 `s.AddUnaryInterceptors(interceptor.LoggerInterceptor)` 注册（bridgegtw 和 bridgedump 除外）。

bridgekafka 是唯一使用 `service.NewServiceGroup()` 运行 gRPC + Kafka 消费者组的服务：

```go
serviceGroup := service.NewServiceGroup()
for _, kc := range c.KafkaConsumeConfig {
    h := handler.NewKafkaStreamHandler(kc.Topic, kc.Group, ctx.StreamEventCli)
    serviceGroup.Add(kq.MustNewQueue(fullConf, h))
}
s := zrpc.MustNewServer(c.RpcServerConf, func(grpcServer *grpc.Server) {
    bridgekafka.RegisterBridgeKafkaServer(grpcServer, server.NewBridgeKafkaServer(ctx))
})
serviceGroup.Add(s)
serviceGroup.Start()
```

依据：`app/bridgekafka/bridgekafka.go`。

## bridgegtw — HTTP → gRPC 代理

- 唯一 REST/gateway 服务，使用 `go-zero/gateway` 而非 `go-zero/rest` 直接启动。
- `ServiceContext` 仅持有 `Config`，无 DB、无连接池。
- YAML 中 `upstreams` 配置 HTTP 路径到 gRPC RPC 的映射，需要 proto descriptor 文件（如 `bridgedump.pb`）。
- 错误处理：`httpx.ErrorCtx` / `httpx.OkJsonCtx`。

```yaml
# etc/bridgegtw.yaml 中的上游映射
upstreams:
  - name: bridgedump
    grpc:
      endpoints: [0.0.0.0:25003]
    protoSets: [bridgedump.pb]
    mappings:
      - method: post
        path: /api/external/cable/workList
        rpcPath: bridgedump.BridgeDumpRpc/CableWorkList
```

新增 HTTP path 映射时：
1. 确认 `mappings` 中的 `rpcPath` 与对应 proto service 一致。
2. 确保 gateway Dockerfile 的 proto descriptor 文件同步拷贝。
3. 不能仅改 YAML 而忘记服务端新增对应 RPC 实现。

依据：`app/bridgegtw/bridgegtw.go`、`app/bridgegtw/etc/bridgegtw.yaml`。

## bridgekafka — Kafka 发布/消费

- 单一 gRPC service `BridgeKafka`，仅一个业务 RPC：`Publish(PublishReq) returns (PublishRes)`。
- `ServiceContext` 持有 `Pushers map[string]*kq.Pusher`（每个 topic 一个 pusher）和可选的 `StreamEventCli`。
- `PublishLogic` 按 `in.Topic` 查找 pusher，不存在返回 `fmt.Errorf("kafka topic %s not configured", in.Topic)`。
- Kafka 消费者 `KafkaStreamHandler` 通过 `threading.TaskRunner`（16 并发）将消息转发到 stream event gRPC。
- config 区分 `KafkaPushConfig`（发布用）和 `KafkaConsumeConfig`（消费用），两者独立配置。

依据：`app/bridgekafka/internal/logic/publishlogic.go`、`app/bridgekafka/internal/handler/kafkastreamhandler.go`。

## bridgemodbus — Modbus 协议桥

### 动态连接池（核心约定）

modbusCode 是动态连接解析的 key。逻辑收到请求后必须通过 `ServiceContext.GetModbusClientPool` 获取连接池，不得直接访问 `ModbusClientPool` 字段：

```
Request with modbusCode
  → svcCtx.GetModbusClientPool(ctx, modbusCode)
    → modbusCode == "": 返回 default pool（来自 config 文件）
    → modbusCode != "": PoolManager 查找
      → 找到：直接返回
      → 未找到：AddPool(ctx, modbusCode)
        → 查询 DB 中 ModbusSlaveConfig by modbusCode
        → Converter 将 DB model 转为 ModbusClientConf
        → PoolManager 创建新 ModbusClientPool
```

- `PoolManager` 使用 `sync.RWMutex` 保护 `pools map`。
- `AddPool` 先查再建，不重复创建已存在的 pool。
- pool 使用 `syncx.Pool` 实现，10 分钟最大连接年龄，自动清理。

### Modbus Client 生命周期

```
pool.Get() → mbCli (ModbusClient)
  → mbCli.ReadHoldingRegisters(或任意 FC)
  → defer pool.Put(mbCli)
```

- `pool.Put` 必须通过 defer 执行，确保异常路径也归还连接。
- 所有 Modbus RPC（18 个）接受的 modbusCode 为空时使用默认连接配置。

### Config CRUD

- `SaveConfig` 使用 DB transaction（`gormx.DB.Transact`）：检查 `modbusCode` 是否存在 → update 或 create。
- `GetConfigByCode` 返回 `gorm.ErrRecordNotFound` 作为受控错误，调用方按业务处理。
- DB model ↔ proto 使用 `copier.CopyWithOption`。

依据：`app/bridgemodbus/internal/svc/servicecontext.go`、`app/bridgemodbus/internal/logic/readholdingregisterslogic.go`、`common/modbusx/client.go`。

## bridgemqtt — MQTT 发布/订阅

### MQTT Client 初始化（核心约定）

`ServiceContext` 通过 `mqttx.MustNewClient` 创建 client，在 `OnReady` 回调中按订阅 topic 注册 handler：

```go
mqttCLi := mqttx.MustNewClient(c.MqttConfig,
    mqttx.WithOnReady(func(cli mqttx.Client) {
        for _, topic := range c.MqttConfig.SubscribeTopics {
            cli.AddHandler(topic, handler.NewMqttStreamHandler(...))
        }
    }),
)
```

- `OnReady` 在首次连接成功后执行一次（内部 `atomic.Bool` 保护），断线重连不重复调用。
- 不能在 `ServiceContext` 构造后、但在连接建立前发布或注册额外 handler。

### 双路 Fan-Out

`MqttStreamHandler.Consume` 将收到的 MQTT 消息同时转发到：
1. stream event gRPC — 非阻塞发送到 AI/实时事件服务
2. socket push gRPC — 广播到 WebSocket 房间

两条路径各自在独立 `threading.TaskRunner`（16 并发）中执行。

- `EventMapping` 配置按 `topicTemplate` 映射到 socket push event 名，fallback 到 `DefaultEvent`。
- `TopicLogManager` 提供 per-topic 日志控制（payload 开关、最小日志间隔）。

### Publish vs PublishWithTrace

- `Publish`：直接调用 `mqttx.Client.Publish()`，不返回 trace ID。
- `PublishWithTrace`：调用 `mqttx.Client.PublishWithTrace()`，返回 OTel trace ID。

依据：`app/bridgemqtt/internal/svc/servicecontext.go`、`app/bridgemqtt/internal/handler/mqttstreamhandler.go`。

## bridgedump — 文件落地

- 最简 bridge 服务：单 zrpc server，无 Nacos，无 interceptors。
- 核心逻辑 `DumpBridgeData(ctx, dumpPath, subDir, in)` 位于 `ServiceContext`，不是 Logic。
- 文件输出格式：`{DumpPath}/{subDir}/{YYYYMMDD_HHmmss}_{traceID}_json.txt`。
- 报文头固定为 `<!System=OMG Version=1.05 Code=utf-8 Data=1.0!>` + `<Bridge:=Free Size=N>` + JSON body。
- 所有 cable 逻辑文件遵循统一模式：调用 `DumpBridgeData` → 成功则返回 `Code: 200, Msg: "成功"`。

依据：`app/bridgedump/internal/svc/servicecontext.go`、`app/bridgedump/internal/logic/cableworklistlogic.go`。

## common/mqttx — MQTT 客户端库

- **`Client` 接口**：调用方唯一依赖，方法：`AddHandler`、`AddHandlerFunc`、`Publish`、`PublishWithTrace`、`Close`、`GetClientID`。
- **`ConsumeHandler` 接口**：`Consume(ctx, payload, topic, topicTemplate) error`。
- **`ReplyRouter[T]`**：泛型请求-应答匹配，按 TID 解析响应。
- **消息分发优先级**：先匹配 reply handler（若有），再分发普通 handler。
- **OTel 集成**：`Message.Headers` 传递 trace context，消费时提取 span context 创建 consumer span。
- **关闭**：`MustNewClient` 自动注册 `proc.AddWrapUpListener`，确保优雅关闭时断开连接。

新增 handler 时：
1. 实现 `ConsumeHandler` 接口。
2. 通过 `mqttx.AddHandler(topicTemplate, handler)` 注册，不与已注册 handler 共享可变状态。
3. 如果 handler 需要请求-应答，使用 `RequestReply[T]` 包级泛型函数。

依据：`common/mqttx/client.go`、`common/mqttx/dispatcher.go`、`common/mqttx/reply_router.go`。

## common/modbusx — Modbus 客户端库

- **`ModbusClient`**：封装 `grid-x/modbus.Client` + `TCPClientHandler`，实现全部标准 FC（0x01–0x18, 0x2B），支持 TLS。
- **`ModbusClientPool`**：`syncx.Pool` 连接池，Factory 创建 client，Destructor 关闭 TCP handler。
- **`PoolManager`**：管理多个 `ModbusClientPool`，key 为 `modbusCode`，线程安全（`sync.RWMutex`）。
- **`ModbusClientConf`**：TCP 配置（Address, Slave, Timeout 等），支持 TLS cert/key/CA。

新增 Modbus FC 使用：
1. 在 `ModbusClient` 中实现对应方法。
2. 在 bridgemodbus proto 添加 RPC，执行 `gen.sh`。
3. 新增 Logic 遵循 `GetModbusClientPool` → `pool.Get()` → operation → `defer pool.Put()` 模式。

依据：`common/modbusx/client.go`、`common/modbusx/config.go`。

## Scenario: bridgemodbus 动态连接池解析与竞争

### 1. Scope / Trigger

- 修改 `ServiceContext.GetModbusClientPool`、`PoolManager.AddPool`、任何 Modbus RPC Logic 中使用连接池的路径时适用。

### 2. Signatures

```go
// ServiceContext
func (s *ServiceContext) GetModbusClientPool(ctx context.Context, modbusCode string) (*modbusx.ModbusClientPool, error)

// PoolManager
func (p *PoolManager) GetPool(modbusCode string) *modbusx.ModbusClientPool
func (p *PoolManager) AddPool(modbusCode string, conf modbusx.ModbusClientConf, poolSize int) (*modbusx.ModbusClientPool, error)
```

### 3. Contracts

- `modbusCode` 为空字符串时返回 Config 中的默认 `ModbusClientPool`，不查询 DB，不经过 PoolManager。
- `modbusCode` 非空时通过 `PoolManager.GetPool` 查找；未找到时调用 `AddPool`。
- `AddPool` 在写锁内二次检查，避免并发重复创建 pool。
- `AddPool` 查询 DB 获取 `ModbusSlaveConfig` 记录，不存在时返回 error，不创建 pool。
- DB model 到 `ModbusClientConf` 的转换必须在 `ModbusConfigConverter` 中集中完成，Logic 不手工构造 conf。
- pool 创建成功后存储于 PoolManager，后续同 code 请求直接复用，不重复查 DB。

### 4. Validation & Error Matrix

- modbusCode 非空且 DB 无此 code → 返回 error（含 code 标记），不应创建空 pool。
- 并发 AddPool 同一 code → 写锁内第二次检查命中，返回现有 pool，不报错。
- default pool 未配置 → 应在 `ServiceContext` 初始化时 panic，不可在运行时返回 nil。
- ModbusClientConf 字段缺失（Address 为空等）→ `AddPool` 返回 error。

### 5. Good/Base/Bad Cases

- Good: 首次请求 `modbusCode="station1"`，PoolManager 无缓存 → 查 DB 获取配置 → 创建 pool → 返回；后续请求直接命中缓存。
- Base: `modbusCode=""` → 直接返回 default pool，不查 DB，不经过 PoolManager。
- Bad: Logic 中 `svcCtx.ModbusClientPool.Get()` 直接访问字段，绕过 `GetModbusClientPool` → 丢失动态解析能力，所有请求使用默认配置。

### 6. Tests Required

- 断言 `modbusCode=""` 返回 default pool 且不查 DB。
- 断言非空 code 命中缓存不查 DB。
- 断言非空 code 未命中时查 DB 并创建 pool。
- 断言并发 AddPool 同一 code 只创建一个 pool。
- 断言 code 在 DB 不存在时返回 error。
- 对 `common/modbusx` 和 bridgemodbus logic 包运行 race test。

### 7. Wrong vs Correct

#### Wrong

```go
mdCliPool := l.svcCtx.ModbusClientPool
mbCli := mdCliPool.Get()
```

#### Correct

```go
mdCliPool, err := l.svcCtx.GetModbusClientPool(l.ctx, in.ModbusCode)
if err != nil {
    return nil, err
}
mbCli := mdCliPool.Get()
defer mdCliPool.Put(mbCli)
```

依据：`app/bridgemodbus/internal/svc/servicecontext.go`、`app/bridgemodbus/internal/logic/readholdingregisterslogic.go`、`common/modbusx/client.go`。

## 反模式

- 在 bridgemqtt 的 `OnReady` 回调执行前发送 Publish 请求。
- 在 bridgemodbus Logic 中直接访问 `svcCtx.ModbusClientPool` 字段，不通过 `GetModbusClientPool`。
- 在 bridgegtw 的 upstreams YAML 中引用不存在的 proto service 或未提供的 proto descriptor。
- 在 bridgekafka 的 Pushers map 外直接创建 kq.Pusher，导致 topic 不可控。
- 用 `pool.Get()` 后忘记 `defer pool.Put()`。
- 在 bridgedump 的 `DumpBridgeData` 中修改报文头格式，导致下游解析不兼容。
- 把 `mqttx.ConsumeHandler.Consume` 中的耗时操作放在调用线程，不通过 TaskRunner 异步化。

## 验证

- bridgegtw：新增 HTTP 路径后执行 `gen.sh`，验证路由注册与 upstreams 映射。
- bridgekafka：Publish 测试覆盖 topic 存在/不存在、pusher 失败场景。
- bridgemodbus：所有 18 个 RPC 测试覆盖 `modbusCode` 为空/非空/不存在路径；执行 `go test -race ./app/bridgemodbus/internal/logic ./common/modbusx`。
- bridgemqtt：Publish/PublishWithTrace 测试覆盖断线重连场景；MqttStreamHandler 测试覆盖双路 fan-out。
- bridgedump：DumpBridgeData 测试覆盖目录创建、文件格式、trace ID 提取。
