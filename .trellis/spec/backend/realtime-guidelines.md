# 实时事件规范

## 适用范围

修改 `common/socketiox`、`socketapp/socketgtw`、`socketapp/socketpush`、`app/bridgekafka`、`facade/streamevent` 或对外事件/topic/payload 时读取。

## 外部事件契约

- Socket.IO event 常量、room 规则和 JSON tag 是外部契约；修改前搜索 gateway、push、StreamEvent、前端/调用方和测试。
- `common/socketiox` 只拥有连接、session、room 和 emit 机制；业务 event/payload 在对应服务或 facade 定义。
- payload 当前是 object、string 还是 bytes 必须保持；不要为了内部便利改变序列化层级或字段名。
- StreamEvent proto 是跨服务事件门面契约，服务间通过生成 client 调用，不导入彼此 `internal/`。

依据：`common/socketiox/server.go`、`socketapp/socketgtw`、`facade/streamevent/streamevent.proto`。

## 连接与房间

- `SessionID` 表示 Socket.IO 连接，用户/设备/room 是业务身份；绑定和授权分别处理，不能把可猜 room 名当权限。
- container 的 map 在锁内读写；广播先在锁内复制当前 client/session 快照，再锁外执行 emit，避免持锁网络 I/O。
- 断开清理 room 和身份绑定必须幂等，并只清理当前 session，防止旧连接删除新绑定。
- gateway 上行事件统一转发到 StreamEvent；新增事件要同步注册、payload、转发和消费者。

依据：`common/socketiox/container.go`、`common/socketiox/server.go`、`socketapp/socketgtw/internal`。

## 推送与交付语义

- `socketpush.BroadcastRoom` 对当前 socketgtw client 快照异步 fan-out，并立即返回；它是 best-effort，不提供远端确认、可靠重试、顺序或 Exactly Once。
- 单个 gateway 失败不应阻止其他 gateway 的 fan-out；错误通过日志/指标观察，除非契约明确增加聚合回执。
- Kafka producer 返回的是 broker publish 结果；有 key 与无 key 影响分区选择，必须按调用契约传递，不能声称消费者已处理。
- MQTT/StreamEvent/Socket.IO 多跳链路分别保留 trace 与业务 correlation ID，不用一个字段替代所有身份。

依据：`socketapp/socketpush/internal/logic`、`app/bridgekafka/internal`、`common/mqttx`。

## 反模式

- 重命名事件或 JSON 字段却只改一个服务。
- 持 container 锁进行网络 emit。
- 广播 RPC 返回 nil 就报告所有客户端收到消息。
- 未授权地让任意连接加入业务 room。
- 从 client API 推断消息可靠、顺序、故障转移或 Exactly Once。

## 验证

- 契约测试断言 event string、JSON shape、room/session 绑定和上行/下行路由。
- 广播测试覆盖零/多 gateway、单点失败、异步立即返回和当前快照语义。
- Kafka/MQTT 测试区分 publish error、未知 response、重复消息和消费者业务失败；并发容器运行 race test。
