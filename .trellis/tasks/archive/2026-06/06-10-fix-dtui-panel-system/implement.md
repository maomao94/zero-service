# 执行计划

## 前置检查

- [ ] `go build ./cli/dtui/` 当前编译通过
- [ ] 确认 `docker.StreamLogs` 方法签名可用

## 步骤

### 步骤 1: Model 结构体清理

**文件**: `internal/tui/model.go`

- [ ] 移除 Model 中的旧字段: `panel`, `logPanel`, `inspectPanel`, `statsPanel`, `imageHistoryPanel`, `execInput`, `execOutput`, `detailFilter`, `statsCh`, `statsErrCh`, `history`, `historyPanel`
- [ ] 移除 `renderLegacyPanel()` 方法
- [ ] 修改 `View()`: 移除 `m.panel` 分支，只保留 `m.panels.active` 分支
- [ ] 修改 `handleActionMsg`: `m.panel == PanelExec` → `m.panels.Active() == PanelExec`，委派 ExecResultMsg 到 PanelManager
- [ ] 确认 `PanelManager` 已暴露 `Active()` 方法（当前已存在）

**验证**: `go build ./cli/dtui/` 编译错误归零（可能有尚未修完的引用）

### 步骤 2: handlePanelKey 重构

**文件**: `internal/tui/keys.go`

- [ ] 将 `handlePanelKey` 简化为: Esc 关闭面板，其余全部委派 `m.panels.HandleKey(key)`
- [ ] 移除 `m.panel` / `m.logPanel` / `m.inspectPanel` / `m.statsPanel` / `m.imageHistoryPanel` / `m.execInput` / `m.historyPanel` 的所有直接引用
- [ ] `/` 键搜索切换移到 LogPanelImpl.HandleKey
- [ ] `r` 键刷新移到 LogPanelImpl.HandleKey（通过回调或 tea.Cmd）
- [ ] `enter` 键在 Exec 面板的处理移到 ExecPanelImpl.HandleKey

### 步骤 3: ExecPanelImpl 完善

**文件**: `internal/tui/panels_exec.go`

- [ ] `HandleKey("enter")` 中确保 `p.onExec` 返回 `ExecResultMsg`
- [ ] `HandleMsg(ExecResultMsg)` 中正确处理错误和输出回显
- [ ] 确认 `runExecCmd` 返回 `ExecResultMsg`（当前已返回正确类型）

### 步骤 4: LogPanelImpl 日志流式化

**文件**: `internal/tui/panels_log.go`, `internal/tui/commands.go`, `internal/tui/update.go`, `internal/tui/messages.go`

- [ ] 新增 `LogStreamReadyMsg` 消息类型（如果尚未存在）
- [ ] `LogPanelImpl` 新增 `logCh <-chan string`, `errCh <-chan error` 字段
- [ ] `LogPanelImpl.Open()` 返回 `beginStreamLogsCmd` 启动流式日志
- [ ] `LogPanelImpl.HandleMsg(LogStreamReadyMsg)` 保存 channels
- [ ] `LogPanelImpl.HandleMsg(TickMsg)` 非阻塞 drain channels，追加到 model
- [ ] `commands.go`: 新增 `beginStreamLogsCmd` 调用 `m.client.StreamLogs`
- [ ] `commands.go`: `streamLogsCmd` 移除或保留为备用（不再被 TickMsg 触发）
- [ ] `update.go`: TickMsg 处理中，`m.panels.active == PanelLogs` 时委派到 PanelManager
- [ ] `LogPanelImpl.HandleKey("r")` 实现刷新（重新加载批量日志）

### 步骤 5: 编译与修复

- [ ] `go build ./cli/dtui/` — 修完所有编译错误
- [ ] 检查所有 `m.panel` / `m.logPanel` / `m.execInput` 等旧字段的残留引用并修复

### 步骤 6: LSP 诊断

- [ ] `lsp_diagnostics` on `cli/dtui/internal/tui/` 全部文件

### 步骤 7: 手动验证（终态）

- [ ] 编译通过
- [ ] 无 LSP error
