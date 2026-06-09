# bridgekafka 实施计划

## 前置条件

- [x] 已分析 bridgemqtt 完整代码结构
- [x] 已掌握 go-queue (kq) 生产者/消费者用法
- [x] 已确认 streamevent.proto 已有 ReceiveKafkaMessage RPC 定义

## 实施步骤

### 第 1 步：创建 bridgekafka.proto 并生成代码

- [ ] 创建 `app/bridgekafka/bridgekafka.proto`，定义 BridgeKafka 服务
- [ ] 创建 `app/bridgekafka/gen.sh`，编写 proto 编译脚本（参照 bridgemqtt/gen.sh）
- [ ] 执行编译生成 pb 代码到 `app/bridgekafka/bridgekafka/`

验证：编译无错误，生成 `.pb.go` 文件

### 第 2 步：实现配置层

- [ ] 创建 `internal/config/config.go`
  - `Config` 结构体：嵌入 `zrpc.RpcServerConf`
  - `KafkaPushConfig`：Brokers + Topic
  - `KafkaConsumeConfig`：使用 `kq.KqConf`
  - NacosConfig、StreamEventConf、SocketPushConf
- [ ] 创建 `etc/bridgekafka.yaml` 配置模板

验证：配置能被 `conf.MustLoad` 正确解析

### 第 3 步：实现服务上下文

- [ ] 创建 `internal/svc/servicecontext.go`
  - 初始化 `kq.Pusher`（生产者）
  - 初始化 `streamevent.StreamEventClient` gRPC 客户端
  - 可选初始化 `socketpush.SocketPushClient`

验证：ServiceContext 初始化无 panic

### 第 4 步：实现 gRPC 业务逻辑

- [ ] 创建 `internal/logic/publishlogic.go`
  - 调用 `svcCtx.KafkaPusher.Push(ctx, string(value))`
- [ ] 创建 `internal/logic/publishwithtracelogic.go`
  - 包装 traceId，调用 `svcCtx.KafkaPusher.PushWithKey(ctx, key, value)`

验证：逻辑层代码编译通过

### 第 5 步：实现 gRPC Server

- [ ] 创建 `internal/server/bridgekafkaserver.go`（goctl 生成的骨架）
  - 实现 `Publish` 和 `PublishWithTrace` 方法

验证：实现 BridgeKafkaServer 接口

### 第 6 步：实现 Kafka 消费转发 Handler

- [ ] 创建 `internal/handler/kafkastreamhandler.go`
  - 实现 `Consume(ctx, key, value string) error`
  - 调用 `streameventCli.ReceiveKafkaMessage` 转发
  - 可选调用 `socketPushCli.BroadcastRoom` 广播
  - 使用 `threading.TaskRunner` 异步处理

验证：handler 编译通过，接口签名正确

### 第 7 步：实现 main 入口

- [ ] 创建 `bridgekafka.go` main 函数
  - 加载配置 → 创建 ServiceContext → 注册 gRPC Server
  - 使用 `service.NewServiceGroup()` 管理 Kafka 消费队列
  - 注册 Nacos 服务
  - 添加 gRPC 拦截器

验证：`go build` 编译通过

### 第 8 步：部署与环境配置

- [ ] 创建 `Dockerfile`（参照 bridgemqtt）
- [ ] 创建 `deploy.sh`
- [ ] 创建 `env/test.env`

## 验证命令

```bash
# 编译
cd app/bridgekafka && go build .

# 启动测试（需要 Kafka 实例）
go run bridgekafka.go -f etc/bridgekafka.yaml

# gRPC 调用测试（需要 grpcurl 或客户端）
grpcurl -plaintext -d '{"topic":"test","key":"k1","value":"dGVzdA=="}' localhost:25006 bridgekafka.BridgeKafka.Publish
```

## 回滚策略

- 每一步完成后确认编译通过再继续
- 如果 go-queue 集成有问题，可回退到 segmentio/kafka-go 直接使用
- 如果 proto 生成有问题，检查 goctl 版本和 proto 语法
