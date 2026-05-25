# 跨层思考指南

> 跨层功能先梳理数据流和契约，再实现。多数线上问题发生在接口、生成代码、Logic、Model/SDK、配置和外部系统边界。

## 常见风险

- `.api` / `.proto` 字段变了，但 Logic、Swagger、前端、测试或文档未同步。
- 数据库存储格式和接口返回格式不一致。
- 多个服务各自实现同一协议转换或状态判断。
- 配置结构增加字段，但 yaml、ServiceContext 初始化或默认值未同步。
- 外部系统返回异常时，错误码、日志和用户响应不一致。
- 多个协议系统同时使用 `up/down` 表示方向但参照物不同（如 SocketIO 以服务端为锚点、DJI Cloud API 以设备为锚点），导致跨协议桥接时方向描述冲突。

## 实现前画清数据流

后端常见流转：

```text
.api/.proto 契约
  -> gen.sh 生成 Handler/Server/Types/pb
  -> internal/logic 业务编排
  -> internal/svc ServiceContext 依赖
  -> model / common SDK / client / cache / config
  -> DB / Redis / Kafka / MQTT / OSS / Docker / Eino / DJI Cloud API
```

如果有前端或外部消费者，继续补充：

```text
Swagger / gRPC client / HTTP client
  -> Frontend / external service
  -> 用户可见行为或下游协议
```

## 边界问题

| 边界 | 需要确认 |
| --- | --- |
| `.api` / `.proto` | 字段命名、注释、必填/可选、错误码、兼容性 |
| 生成代码 | 是否执行 `gen.sh`，生成 diff 是否符合预期 |
| Handler/Server → Logic | 参数校验位置、上下文传递、错误返回 |
| Logic → Model/SDK | 数据格式、事务边界、缓存一致性、外部系统失败处理 |
| Config → ServiceContext | yaml 是否同步、默认值是否明确、敏感信息是否脱敏 |
| Backend → Frontend/外部系统 | 序列化格式、时间/枚举/状态码、分页和空值语义 |
| 协议层间方向命名 | up/down 的方向锚点是哪个系统（SocketIO 以服务端、DJI 以设备），跨协议桥接时方向描述是否清晰 |

## 合同定义

每个边界至少明确：

- 输入格式和输出格式。
- 哪一层负责校验。
- 哪些错误会返回给调用方，哪些只记录日志。
- 是否需要更新 `.api`、`.proto`、Swagger、文档、SQL、配置、测试。
- 是否需要兼容已有持久化数据或外部消费者。

## 常见错误

- 只改 Logic，不改契约源文件。
- 改了 `.proto` 但忘记执行 `gen.sh`。
- 在多个 Logic 中散落同一状态机判断。
- 在日志中打印完整请求、密钥、连接串或内网地址。
- 用临时硬编码配置绕过 `internal/config` 和 `ServiceContext`。
- 跨协议描述桥接方向时，只写了 `up`/`down` 但没有指明方向锚点，导致读者误以为两个协议的 up 指向同一端。

## 检查清单

实现前：

- [ ] 已画清完整数据流。
- [ ] 已识别所有层边界和消费者。
- [ ] 已确定契约源文件和生成脚本。
- [ ] 已确认复用位置或新增落点。
- [ ] 如果涉及 `up`/`down` 方向命名，已明确方向锚点是哪个系统，跨协议桥接时已补充完整的流转方向说明。

实现后：

- [ ] `.api` / `.proto` 变更已执行 `gen.sh` 并检查 diff。
- [ ] Logic、model/client/cache/config 已同步。
- [ ] 错误处理、日志和错误码与项目规范一致。
- [ ] 相关测试、构建或手工验证已执行；未执行项已说明原因。
