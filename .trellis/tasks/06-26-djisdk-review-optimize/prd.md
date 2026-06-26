# djisdk 代码审阅优化

## Goal

清理 djisdk 包的死代码与重复代码、优化 Client struct 结构减少冗余、为 DrcConfig 添加零值保护，提升代码可维护性与健壮性。

## Confirmed Facts（审阅确认）

1. **`app/djicloud/internal/drc/` 是死代码**：`manager.go` + `state.go` + `manager_test.go` 与 `common/djisdk/drc.go` 的 `drcManager` 功能 100% 重复（session map、Enable/Disable/OnDeviceHeartbeat/GetNextSeq/cleanLoop/heartbeatLoop）。`NewManager` 仅在自身 `manager_test.go` 中调用，生产代码零引用。业务全部走 `djisdk.Client.EnableDrc/DisableDrc/DrcNextSeq/DrcStatus`。

2. **`Client` struct 与 `clientOptions` 字段重复**：17 个回调 handler + `pendingTTL` + `replyOptions` + `onlineChecker` 在两个 struct 中各复制一份，`buildClient` 手工逐字段映射。新增 handler 需改 4 处（spec djisdk-guidelines.md 也承认此痛点）。

3. **`DrcConfig` 零值有 panic 风险**：`newDrcManager` 内 `time.NewTicker(cfg.HeartbeatInterval)`，若 `HeartbeatInterval=0` 则 `time.NewTicker(0)` panic。`HeartbeatTimeout=0` 导致 IsAlive 判定异常。虽然 `config.DrcConfig` 有 go-zero `default=2s/300s` 标签，但 `djisdk.DrcConfig` 是公开类型，外部直接构造零值即可触发。

4. `common/djisdk/client.go` 1660 行，职责混杂（option 定义 + Client 构造 + handler 分发 + 命令发送），可选择性拆文件。

## Requirements

### R1: 删除死代码包
- 删除 `app/djicloud/internal/drc/` 整个目录（manager.go、state.go、manager_test.go）
- `go build ./...` + `go test ./...` 保持通过

### R2: Client struct 聚合 handlers
- 将 17 个 `onXxx` 回调字段从 `Client` struct 和 `clientOptions` 中提取到独立的 handlers 结构体
- `buildClient` 改为一次赋值而非逐字段映射
- 新增 handler 的改动点从 4 处减少到 2 处（handlers struct 加字段 + With* option 函数）
- 外部 API（`WithXxx` option 函数签名、`Client` 公开方法签名）保持不变

### R3: DrcConfig 默认值保护
- 为 `djisdk.DrcConfig` 提供 `DefaultDrcConfig()` 函数返回合理默认值
- `newDrcManager` / `WithDrcConfig` option 对零值字段自动填充默认值
- 防止 `time.NewTicker(0)` panic 和 IsAlive 异常判定

### R4: client.go 文件拆分
- 将 `client.go` 按职责拆分为 `option.go` + `handler.go`
- 同步更新 `djisdk-guidelines.md` spec 中的文件组织表

### R5: Config struct 统一初始化配置
- 新增 `Config` struct，内嵌 `mqttx.MqttConfig`，收束 `PendingTTL`、`ReplyOptions`、`DrcConfig`
- `MustNewClient(config Config, opts ...ClientOption)` 替代 `MustNewClient(mqttx.MqttConfig, opts...)`
- `WithPendingTTL`、`WithReplyOptions`、`WithDrcConfig` 降为未导出（Config 已提供等价字段，`NewClient` 仍可通过 opts 传递——转为内部使用）
- `NewClient(mqttClient, opts...)` 保持不变
- 调用方 `servicecontext.go` 从三行 option 变为一个 `Config` 字面量

## Acceptance Criteria

- [ ] `app/djicloud/internal/drc/` 目录已删除，`go build ./...` 通过
- [ ] 所有 Client struct 字段通过 handlers 聚合，不再与 clientOptions 逐字段重复
- [ ] `buildClient` 中 handler 赋值从逐字段映射改为一次性赋值
- [ ] `DrcConfig` 零值不再导致 panic（HeartbeatInterval/HeartbeatTimeout 为 0 时自动用默认值）
- [ ] 现有外部 API 完全兼容（`WithXxx` option 函数、`Client` 方法无 breaking change）
- [ ] `go build ./common/djisdk/... ./app/djicloud/...` 通过
- [ ] `go vet ./common/djisdk/... ./app/djicloud/...` 通过
- [ ] `go test ./common/djisdk/... ./app/djicloud/...` 通过
- [ ] `MustNewClient` 签名变更为 `MustNewClient(config Config, opts ...ClientOption)`
- [ ] 调用方 `servicecontext.go` 使用 `Config` 字面量替代分散的 `WithPendingTTL`/`WithReplyOptions`/`WithDrcConfig` option
- [ ] spec `djisdk-guidelines.md` 文件组织表与实际一致

## Out of Scope

- 不改动 `NewClient(mqttClient, opts...)` 签名（保留已有 MQTT 客户端复用路径）
- 不改动 `ReplyOptions`/`DrcConfig` 等公开配置 struct 的字段
- 不改动 `protocol.go`/`method.go`/`topic.go`/`error*.go`/`protocol_drc.go` 等文件

## Open Questions

1. ~~R2 的 handlers 聚合方式~~ → 已决定：未导出 `handlers` struct，按值持有
2. ~~R4 client.go 是否拆分~~ → 已决定：拆 option.go（全部 WithXxx + clientOptions + applyOptions） + handler.go（全部 Handle*/tryDispatch*/reply*），命令方法留 client.go
