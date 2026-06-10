# dtui Panel 系统 v3 架构完成 — 技术设计

## 现状问题

dtui 的 panel 管理存在两条独立路径：

```
旧路径: Model.m.panel (PanelType) → handlePanelKey 直接分支 switch m.panel
         → 访问 m.logPanel / m.execInput / m.inspectPanel 等旧字段

新路径: Model.m.panels (*PanelManager) → openPanel() → Panel 接口实现
         → renderPanelWithManager() → pm.Render()
```

两条路径各自工作，但 **按键处理走旧路径，渲染走新路径**，导致状态分离。

## 架构目标

完全收敛到 PanelManager + Panel 接口：

```
Model.m.panels (*PanelManager) — 唯一面板入口
    ├── Open(panelType, factory)  → Panel.Open()
    ├── Close()
    ├── Render()                  → Panel.Render()
    ├── HandleKey(key)            → Panel.HandleKey(key)
    ├── HandleMsg(msg)            → Panel.HandleMsg(msg)
    └── SetSize(w, h)             → Panel.SetSize(w, h)
```

## 变更清单

### 1. Model 结构体清理 (model.go)

**移除字段：**
- `panel PanelType` — 旧 panel 枚举
- `logPanel *views.LogPanelModel` — 旧日志面板
- `inspectPanel *views.InspectPanelModel`
- `statsPanel *views.StatsPanelModel`
- `imageHistoryPanel *views.ImageHistoryPanelModel`
- `execInput string` — 旧 exec 输入
- `execOutput string` — 旧 exec 输出
- `detailFilter string` — 旧详情过滤（未使用）
- `statsCh <-chan dt.StatsEntry` — 移到 StatsPanelImpl
- `statsErrCh <-chan error` — 移到 StatsPanelImpl
- `history []views.HistoryEntry` — 旧历史数据（已由 HistoryPanelImpl 管理）
- `historyPanel *views.HistoryPanelModel` — 旧历史面板引用

**保留字段：**
- `panels *PanelManager` — 唯一面板入口

### 2. handlePanelKey 重构 (keys.go)

当前 `handlePanelKey` 为每个 panel 类型硬编码分支，直接访问 Model 字段。

**改为：** 将非通用按键（非 Esc）直接委派给 `PanelManager`：

```go
func (m Model) handlePanelKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
    key := msg.String()
    switch key {
    case "esc":
        m.closePanel()
        return m, tea.Batch(m.loadContainersCmd(), m.loadImagesCmd())
    default:
        return m, m.panels.HandleKey(key)
    }
}
```

所有面板特有逻辑移到各自 `PanelImpl.HandleKey()` 中。

### 3. ExecPanelImpl 修复 (panels_exec.go)

当前问题：按键写入 `m.execInput`，渲染读取 `p.input`。

**修复：** ExecPanelImpl 自管理输入状态，通过 `HandleMsg(ExecResultMsg)` 接收执行结果。

ExecPanelImpl 已经是正确的实现，问题在于 `handlePanelKey` 绕过了它。

额外修复：`r` 键刷新日志的逻辑移到 `LogPanelImpl.HandleKey()`。

### 4. 日志流式化 (panels_log.go + commands.go)

当前 `streamLogsCmd` 只是重新调用 `FetchLogs`，返回 `LogLoadedMsg{Lines, Reset: true}`，导致每次全量替换。

**改为：** 使用 `docker.StreamLogs` 的 channel 接口。

在 `LogPanelImpl.Open()` 中启动流式通道，TickMsg 到来时从 channel 读取增量行。

```go
// LogPanelImpl 新增字段
type LogPanelImpl struct {
    // ... existing fields
    logCh  <-chan string  // StreamLogs 返回的 channel
    errCh  <-chan error
}

func (p *LogPanelImpl) Open(width, height int) tea.Cmd {
    // ... existing init
    // 启动流式日志
    p.logCh, p.errCh = p.container.StreamLogs(...)
    return nil
}

// HandleMsg 处理 TickMsg: 非阻塞读取 logCh，追加新行
```

或者更简单的方案：保留 `streamLogsCmd` 但改为使用 `StreamLogs` 并在 TickMsg 处理中读取 channel。

**简化方案（推荐）：** 不引入 channel 到 PanelImpl。改为让 `streamLogsCmd` 真正流式化：使用 `Follow: true` 参数，通过 goroutine 持续读取并发送 `LogStreamMsg{Line, Done: false}`。

实际上，docker SDK 的 ContainerLogs with Follow 会一直阻塞读取。需要在 goroutine 中读取并通过 channel 发回。

让我重新考虑：`StreamLogs` 已经实现了 channel 接口。在 TickMsg 中非阻塞读取即可。

```go
// commands.go
func (m Model) beginStreamLogsCmd(containerID string) tea.Cmd {
    return func() tea.Msg {
        logCh, errCh := m.client.StreamLogs(containerID, dt.LogOptions{Tail: "0", Follow: true})
        return LogStreamReadyMsg{Ch: logCh, ErrCh: errCh}
    }
}

// LogPanelImpl 收 StreamLogs channel
type LogPanelImpl struct {
    // ...
    logCh  <-chan string
    errCh  <-chan error  
}

func (p *LogPanelImpl) HandleMsg(msg tea.Msg) (Panel, tea.Cmd) {
    switch m := msg.(type) {
    case LogLoadedMsg:
        // initial batch load
        p.model.SetLines(m.Lines, true)
    case LogStreamReadyMsg:
        p.logCh = m.Ch
        p.errCh = m.ErrCh
    case TickMsg:
        // non-blocking drain
        for {
            select {
            case line, ok := <-p.logCh:
                if ok {
                    p.model.AppendLine(line)
                }
            case err, ok := <-p.errCh:
                // handle error
            default:
                return p, nil
            }
        }
    }
}
```

对应的 Update 中 TickMsg 处理简化为委派：

```go
case TickMsg:
    if m.panels.active == PanelLogs || m.panels.active == PanelStats {
        m.panels.HandleMsg(msg)
    }
    return m, m.autoTickCmd()
```

### 5. View 方法清理 (model.go)

- 移除 `renderLegacyPanel()` 
- `View()` 直接在 `m.panels.active != PanelNone` 时走 `renderPanelWithManager()`
- 确认弹窗 `m.pending` 优先级高于面板

### 6. openPanel 清理 (model.go)

`openPanel` 已在用 PanelManager，无需改动。但需要为 ExecPanel 确保 `runExecCmd` 回调正确处理 `ExecResultMsg` 回显。

当前 `runExecCmd` 返回 `ExecResultMsg`，panels_exec.go 的 `HandleMsg` 可以处理。但 `handleActionMsg` 中对 `m.panel == PanelExec` 的处理需要改为委派 PanelManager：

```go
// update.go handleActionMsg
if m.panels.active == PanelExec {
    m.panels.HandleMsg(ExecResultMsg{Output: msg.Text, Err: msg.Err})
}
```

### 7. handleActionMsg (update.go)

当前 `handleActionMsg` 处理 `m.panel == PanelExec` 时写 `m.execOutput`。改为：

```go
func (m *Model) handleActionMsg(msg ActionMsg) (tea.Model, tea.Cmd) {
    m.busy = false
    if msg.Err != nil {
        m.status = styles.Error.Render(msg.Err.Error())
    } else {
        m.status = styles.Success.Render("执行完成")
    }
    // 委派到 PanelManager
    if m.panels.Active() == PanelExec {
        m.panels.HandleMsg(ExecResultMsg{Output: msg.Text, Err: msg.Err})
    }
    return m, tea.Batch(m.loadContainersCmd(), m.loadImagesCmd())
}
```

## 数据流

```
按键 (KeyMsg)
  → Update() 
    → m.panels.active != PanelNone 
      → handlePanelKey()
        → Esc → closePanel()
        → 其他 → m.panels.HandleKey(key)
          → PanelImpl.HandleKey(key) — 面板自包含逻辑

异步消息 (LoadedMsg, ActionMsg, TickMsg...)
  → Update()
    → 根据 msg 类型判断
      → 委派 m.panels.HandleMsg(msg)
        → PanelImpl.HandleMsg(msg)
```

## 不变更范围

- `docker/` 包 — API 已完备，只需正确调用
- `views/` 子面板模型 — API 稳定
- `styles/` — 不涉及
- Compose/Deploy 功能 — 不涉及
- 配置系统 — 不涉及
