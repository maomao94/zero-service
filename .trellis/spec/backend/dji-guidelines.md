# DJI Cloud 与 DRC 规范

## 适用范围

修改 `common/djisdk`、`app/djicloud`、DJI Cloud MQTT topic/method/payload、设备在线状态、OSD/State/Event 持久化、DRC 或飞行区同步时读取。

## SDK 与协议所有权

- `common/djisdk` 的协议包拥有 topic 构造、method、typed payload 和结果解析；`app/djicloud` Logic/hook 不手工拼 topic 或通用 map。
- client option 只写构造配置；`NewClient` 负责连接、handler、reply router 和运行态初始化。
- `SendCommand` 表达需要设备响应的 request/reply；fire-and-forget publish 是另一种语义，不能因返回 error 形式相似而互换。
- 请求应答使用 TID/BID 等协议关联标识，先注册后发送，区分 broker publish、设备回复和 reply result。

依据：`common/djisdk/client.go`、`common/djisdk/option.go`、`common/djisdk/handler.go`、`common/djisdk` 协议目录与测试。

## 上行处理与数据所有权

- `app/djicloud` 的 event、telemetry、DRC、request hooks 按 DJI 上行类型分工；新增 method 放到对应注册边界，不建立第二套总分发器。
- 设备/拓扑/最新状态属于快照，使用查询后更新或现有 `FirstOrCreate` + `Assign` 模式；事件/告警历史使用追加 `Create`，不得 Upsert 覆盖历史。
- 在线语义由当前设备/topology/heartbeat 处理链路拥有；单条旧消息或迟到 callback 不能覆盖更新 session 的在线状态。
- Store/model 决定事务和更新字段。不要从 hook 整行 `Save`，也不要以方言专属 Upsert SQL 替换已经测试的写入模式。

依据：`app/djicloud/internal/hooks`、`app/djicloud/model/gormmodel` 及其测试。

## DRC 并发模型

- Manager map 由 manager 锁保护，session 可变字段由 session 锁保护；统一按 manager -> session 的顺序取得，不允许反向。
- 取得 session 后的网络写入只持 session 所需锁，避免长期占用 manager 锁阻塞其他设备。
- 每次新会话分配新的 session ID/version；旧 read/heartbeat/cleanup goroutine 在修改状态前验证自己仍是当前 session。
- `heartbeatCancel` 属于对应 session，只能由该 session 的替换/关闭路径调用和清除。
- 清理采用锁内标记/快照、锁外关闭与回调，避免持锁 I/O 和 callback 重入死锁。

依据：`common/djisdk/drc.go`、`common/djisdk/protocol_drc_test.go`。

## 飞行区与外部资源

- DJI 自定义飞行区 payload、坐标和文件字段复用现有 protocol/model helper；GIS 规则见 [gis-guidelines.md](./gis-guidelines.md)。
- OSS 上传、设备同步和数据库记录是多个阶段。失败时保留可重试状态，删除按现有 device/file ID 所有权执行，不把部分成功报告为全部完成。
- 外部设备返回原始结构在 SDK 边界转换，业务 model 不直接绑定所有上游字段。

## 反模式

- 在 hook 中硬编码 topic/method 或用 `map[string]any` 绕过 typed payload。
- 把事件历史做 Upsert，或把设备快照每次 Insert 成历史。
- manager/session 锁顺序不一致，或持锁执行 callback/MQTT/数据库。
- 旧 session goroutine 关闭新 session 的 heartbeat。
- publish 成功即更新为设备命令成功。

## 验证

- SDK 测试覆盖 topic、handler 路由、快速回复、设备错误、超时和 fire-and-forget 差异。
- Model/hook 测试分别断言快照更新、事件追加、在线状态和重复/迟到消息。
- DRC 修改运行 `go test -race ./common/djisdk -count=10`，覆盖替换、心跳、清理和回调重入。
