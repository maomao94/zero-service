# implement: djisdk 代码审阅优化

## Execution Order

按依赖关系排序，每一步独立可验证。

### Step 0: Config struct + MustNewClient 签名变更

- [ ] 新增 `Config` struct（内嵌 `mqttx.MqttConfig` + `PendingTTL` + `ReplyOptions` + `Drc *DrcConfig`）
- [ ] `WithPendingTTL`/`WithReplyOptions`/`WithDrcConfig` 改为未导出（`withPendingTTL`/`withReplyOptions`/`withDrcConfig`）
- [ ] `MustNewClient(config Config, opts ...ClientOption)` 签名字面量变更
- [ ] `MustNewClient` 内部：Config → with* option → applyOptions
- [ ] 更新 `servicecontext.go`：`MustNewClient(djisdk.Config{...}, handlerOpts...)`
- **验证**: `go build ./common/djisdk/... ./app/djicloud/...`

### Step 1: 删除 internal/drc 死代码

- [ ] 删除 `app/djicloud/internal/drc/` 整个目录（manager.go、state.go、manager_test.go）
- [ ] 确认 `app/djicloud/internal/drc/` 无任何生产代码 import（已 grep 验证，仅 self-test 引用）
- **验证**: `go build ./app/djicloud/...`

### Step 2: DrcConfig 默认值保护

- [ ] 在 `common/djisdk/drc.go` 新增 `DefaultDrcConfig()` 函数
- [ ] 在 `newDrcManager` 开头对零值字段填充默认值
- **验证**: `go build ./common/djisdk/... && go vet ./common/djisdk/...`

### Step 3: 创建 handlers struct（Client 聚合重构）

- [ ] 在 `common/djisdk/` 定义未导出 `handlers` struct（17 个回调 + onlineChecker）
- [ ] `clientOptions` 的 `onXxx` 字段替换为 `handlers` 内嵌
- [ ] `Client` struct 同理
- [ ] 所有 `WithXxx` option 函数改为 `options.handlers.onXxx = handler`
- [ ] `buildClient` 中 `c.handlers = opt.handlers`
- [ ] 全局 `c.onXxx` 引用 → `c.handlers.onXxx`
- **验证**: `go build ./common/djisdk/... ./app/djicloud/...`

### Step 4: 拆分 option.go 和 handler.go

- [ ] **创建 `option.go`**：`Config`、`ClientOption`、`ReplyOptions`、`DefaultReplyOptions`、`clientOptions`、`handlers`、所有 `WithXxx`/`with*` 函数、`defaultClientOptions`、`applyOptions`
- [ ] **创建 `handler.go`**：`HandleEvents`/`HandleOsd`/`HandleState`/`HandleStatus`/`HandleRequests`/`HandleDrcUp`、`tryDispatch*`、`reply*`、`extractDeviceSn`、`logFields`、`SubscribeAll`
- [ ] `client.go` 保留：`Client` struct、`MustNewClient`/`NewClient`/`buildClient`、`replyRouters`、router 工厂、`SendCommand`/`SendCommandFireAndForget`、`SetProperty`、全部命令方法、DRC Manager API、`Close`
- **验证**: `go build ./common/djisdk/... ./app/djicloud/...`

### Step 5: 更新 spec

- [ ] 更新 `.trellis/spec/backend/djisdk-guidelines.md` 文件组织表 + Config 文档

### Step 6: 全面验证

- [ ] `go build ./...` 全量编译
- [ ] `go vet ./common/djisdk/... ./app/djicloud/...`
- [ ] `go test ./common/djisdk/... ./app/djicloud/...` 全量测试

## Validation Commands

```bash
# 逐步验证
go build ./common/djisdk/... ./app/djicloud/...         # Step 0
go build ./app/djicloud/...                            # Step 1
go build ./common/djisdk/... && go vet ./common/djisdk/...   # Step 2
go build ./common/djisdk/... ./app/djicloud/...         # Step 3
go build ./common/djisdk/... ./app/djicloud/...         # Step 4
go build ./...                                           # Step 6
go vet ./common/djisdk/... ./app/djicloud/...            # Step 6
go test ./common/djisdk/... ./app/djicloud/...           # Step 6
```

## Risk Points

- **Step 0**: `MustNewClient` 签名变更是 **breaking change**——唯一调用方 `servicecontext.go` 需同步修改
- ~~**Step 1**: 死代码删除零风险~~
- **Step 2**: `DrcConfig` 默认值 `<=0` 判断不会覆盖已设置的值，安全
- **Step 3**: 全局 `c.onXxx` → `c.handlers.onXxx` 替换涉及 ~30 处引用，需逐处检查
- **Step 4**: 确保新增文件 package 声明为 `package djisdk`

## Rollback Points

- Step 1 后：`git restore app/djicloud/internal/drc/`
- Step 2-3 后：`git restore common/djisdk/`
- Step 4 后：删除 `option.go` `handler.go`，`git checkout common/djisdk/client.go`（还原到拆分前内容）
- 全程：`git stash` 即可全量回滚
