# 执行计划 - 迁移废弃 API (SA1019, 18处)

## 执行顺序（按文件）

### 1. cli/uix/components/filepicker.go（3处）
1. filepicker.go:34 `fp.Height = 8` → `fp.SetHeight(8)`
2. filepicker.go:105 `fp.fp.Height = height - 4` → `fp.fp.SetHeight(height - 4)`
3. filepicker.go:110 `Height()` 返回值改为用包装层 `height` 字段：
   - `NewFilePicker` 内 `fp.SetHeight(8)` 后同步 `fp.height = 8`
   - `Height()` 返回 `fp.height + 4`（原 `fp.fp.Height + 4`）

### 2. cli/uix/components/logviewer.go（2处）
- :119 `lv.viewport.LineUp(1)` → `lv.viewport.ScrollUp(1)`
- :124 `lv.viewport.LineDown(1)` → `lv.viewport.ScrollDown(1)`

### 3. cli/uix/timeline.go（2处）
- :88 `t.viewport.LineUp(1)` → `t.viewport.ScrollUp(1)`
- :89 `t.viewport.LineDown(1)` → `t.viewport.ScrollDown(1)`

### 4. common/einox/agent（5处）→ 删除废弃 API
- agent_option.go:31 `middlewares []adk.AgentMiddleware` 字段
- agent_option.go:85-93 `WithMiddleware`/`WithMiddlewares` 函数及注释
- agent.go:120-122 `agentCfg.Middlewares` 赋值块
- factory.go:67-69 `deepCfg.Middlewares` 赋值块
- factory.go:203 `cc.middlewares` 拷贝行
- factory.go:187 `needsWorkflowCoordinator` 中 `len(cfg.middlewares) > 0` 判断

### 5. common/mcpx/client.go（6处）→ 删除透传
- :213-217 `CreateMessageHandler` 透传块
- :261-265 `LoggingMessageHandler` 透传块

## 验证门（每步后）

```bash
go build ./...
```

## 完成门

```bash
go build ./...
go vet ./...
staticcheck -checks=SA1019 ./...   # 必须为 0
go test ./cli/uix/... ./common/einox/agent/... ./common/mcpx/...  # 相关包测试
```

## 回归确认

- cli/uix 组件滚动/高度行为与迁移前一致（若测试覆盖则跑测试）
- common/einox/agent 对外 Option API：删除的是无调用方的 `WithMiddleware`/`WithMiddlewares`，`WithHandler(s)` 保留
- common/mcpx `WithOptions` 透传减少两个废弃 handler，编译通过即可

## 不做的事

- 不修改 omitempty 相关 JSON tag（用户已确认保留现状）
- 不处理 `go fix` 已迁移的 x/net/context（已完成）
- 不重跑 `go fix ./...`
