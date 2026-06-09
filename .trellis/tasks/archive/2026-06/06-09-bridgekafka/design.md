# bridgekafka 技术设计

## 整体架构

```
                        gRPC 调用方
                            |
                     +------v------+
                     | bridgekafka |
                     |  gRPC 服务   |
                     +------+------+
                            |
               +------------+------------+
               |                         |
        +------v------+          +-------v-------+
        | Kafka 生产者 |          | Kafka 消费者   |
        | (kq.Pusher) |          | (kq.MustNewQueue) |
        +------+------+          +-------+-------+
               |                         |
          Kafka 集群                 Kafka 集群
                                       |
                                +------v------+
                                | 转发到 gRPC  |
                                | streamevent  |
                                | ReceiveKafkaMessage |
                                +-------------+
```

## 目录结构

```
app/bridgekafka/
├── bridgekafka.go              # main 入口
├── bridgekafka.proto           # proto 定义
├── bridgekafka/                # goctl 生成的 pb 代码（编译产物）
├── deploy.sh                   # 部署脚本
├── Dockerfile
├── env/
│   └── test.env
├── etc/
│   └── bridgekafka.yaml        # 配置文件
├── gen.sh                      # proto 编译脚本
└── internal/
    ├── config/
    │   └── config.go           # 配置结构体
    ├── handler/
    │   └── kafkastreamhandler.go  # Kafka 消费转发处理器
    ├── logic/
    │   ├── publishlogic.go         # Publish 业务逻辑
    │   └── publishwithtracelogic.go # PublishWithTrace 业务逻辑
    ├── server/
    │   └── bridgekafkaserver.go    # gRPC server 实现
    └── svc/
        └── servicecontext.go       # 服务上下文
```

## 核心设计

### 1. Proto 定义

```protobuf
syntax = "proto3";

package bridgekafka;
option go_package = "./bridgekafka";

service BridgeKafka {
  // 推送消息到 Kafka
  rpc Publish(PublishReq) returns (PublishRes);
  // 带 traceId 的推送
  rpc PublishWithTrace(PublishWithTraceReq) returns (PublishWithTraceRes);
}

message PublishReq {
  string topic = 1;
  string key = 2;
  bytes value = 3;
}

message PublishRes {}

message PublishWithTraceReq {
  string topic = 1;
  string key = 2;
  bytes value = 3;
}

message PublishWithTraceRes {
  string traceId = 1;
}
```

### 2. 配置结构体

```go
type Config struct {
    zrpc.RpcServerConf
    NacosConfig struct { ... }           // Nacos 注册配置
    KafkaPushConfig KafkaPushConfig      // 推送配置（多 topic）
    KafkaConsumeConfig kq.KqConf         // 消费配置（go-queue 标准）
    StreamEventConf zrpc.RpcClientConf   // streamevent gRPC 客户端
    SocketPushConf zrpc.RpcClientConf    // socket-push gRPC 客户端（可选）
}

type KafkaPushConfig struct {
    Brokers []string
    Topics  []string    // 预配置的 topic 列表，每个 topic 创建一个 Pusher
}
```

### 3. 生产者（Publish 逻辑）

- 在 `ServiceContext` 中按 `Topics` 列表创建 `map[string]*kq.Pusher`，每个 topic 一个 Pusher
- `Publish` 根据请求中的 `topic` 从 map 中查找对应 Pusher，调用 `pusher.Push(ctx, string(value))`
- `PublishWithTrace` 同理，调用 `pusher.PushWithKey(ctx, key, value)`
- 未配置的 topic 返回错误 `kafka topic %s not configured`

### 4. 消费者（StreamHandler）

- 实现 go-queue 的 `Consume(ctx context.Context, key, value string) error` 接口
- 消费到消息后，构造 `streamevent.ReceiveKafkaMessageReq`，调用 gRPC
- 使用 `threading.TaskRunner` 异步处理，避免阻塞消费

### 5. main 入口

- 与 bridgemqtt 一致：`flag` 加载配置 → `svc.NewServiceContext` → `zrpc.MustNewServer` 注册 gRPC
- 额外使用 `service.NewServiceGroup()` 管理 Kafka 消费队列生命周期
- 注册 Nacos 服务

## 依赖关系

| 依赖 | 说明 |
|------|------|
| `github.com/zeromicro/go-queue/kq` | Kafka 生产者/消费者 |
| `facade/streamevent` | 下游 gRPC 接口 |
| `socketapp/socketpush` | SocketIO 推送（可选） |
| `common/Interceptor` | gRPC 拦截器 |
| `common/nacosx` | Nacos 服务注册 |
| `common/tool` | UUID 生成等工具 |

## 与 bridgemqtt 的差异

| 维度 | bridgemqtt | bridgekafka |
|------|-----------|-------------|
| 协议 | MQTT | Kafka |
| 客户端库 | common/mqttx（paho） | go-queue/kq |
| 消息模型 | topic + payload | topic + key + value |
| 消费方式 | mqttx.ConsumeHandler | kq.Consume 接口 |
| 多集群 | 通过 Broker 数组 | 通过 Brokers 配置 |

## 风险与对策

| 风险 | 对策 |
|------|------|
| go-queue Pusher 不支持单实例动态 topic | 已通过多 Pusher map 方案解决：配置 Topics 列表，每个 topic 创建独立 Pusher |
| Kafka 消费者组 rebalance 导致重复消费 | 下游 streamevent 需幂等处理 |
| 大消息序列化超限 | 参照 bridgemqtt 配置 MaxCallSendMsgSize |
