# Technical Design: dtui TUI 面板架构重构

## 概述

重构 TUI 面板系统，解决渲染切换问题，提供最佳实践。

## 1. 面板生命周期管理

### 1.1 Panel 接口定义

```go
type Panel interface {
    Open(width, height int)
    Close()
    Render() string
    HandleKey(key string) tea.Cmd
    SetSize(width, height int)
}
```

### 1.2 面板状态管理

```go
type PanelState int

const (
    PanelStateClosed PanelState = iota
    PanelStateOpen
    PanelStateLoading
)
```

### 1.3 Model 中的面板管理

```go
type Model struct {
    // 当前活动面板
    activePanel PanelType
    panelState  PanelState
    
    // 面板实例
    panels map[PanelType]Panel
}
```

## 2. 修复面板引用错误

### 2.1 renderChromePanel 修复

**变更前**：
```go
case PanelImageHistory:
    if m.historyPanel != nil {
        content = m.historyPanel.Render()
    }
```

**变更后**：
```go
case PanelImageHistory:
    if m.imageHistoryPanel != nil {
        content = m.imageHistoryPanel.Render()
    }
```

## 3. 面板切换逻辑

### 3.1 统一的面板切换方法

```go
func (m *Model) openPanel(panel PanelType) tea.Cmd {
    // 关闭当前面板
    if m.activePanel != PanelNone {
        m.closePanel()
    }
    
    // 打开新面板
    m.activePanel = panel
    m.panelState = PanelStateOpen
    
    // 初始化面板
    switch panel {
    case PanelLogs:
        return m.initLogPanel()
    case PanelInspect:
        return m.initInspectPanel()
    // ...
    }
    return nil
}

func (m *Model) closePanel() {
    if m.activePanel != PanelNone {
        // 清理面板状态
        if p, ok := m.panels[m.activePanel]; ok {
            p.Close()
        }
        m.activePanel = PanelNone
        m.panelState = PanelStateClosed
    }
}
```

### 3.2 ESC 键处理

```go
case "esc":
    m.closePanel()
    return m, tea.Batch(m.loadContainersCmd(), m.loadImagesCmd())
```

## 4. 日志面板改进

### 4.1 日志流管理

```go
func (m *Model) initLogPanel() tea.Cmd {
    c := m.selectedContainer()
    if c == nil {
        return nil
    }
    
    m.logPanel = views.NewLogPanelModel(m.width, m.height)
    return m.loadLogsCmd(c.ID)
}

func (m *Model) streamLogs() tea.Cmd {
    if m.activePanel != PanelLogs {
        return nil
    }
    
    c := m.selectedContainer()
    if c == nil {
        return nil
    }
    
    return m.streamLogsCmd(c.ID)
}
```

### 4.2 TickMsg 处理

```go
case TickMsg:
    if m.activePanel == PanelLogs {
        return m, m.streamLogs()
    }
    if m.activePanel == PanelStats && m.statsCh != nil {
        return m, m.readStatsEntryCmd()
    }
    return m, nil
```

## 5. 布局系统重构

### 5.1 统一布局计算

```go
type Layout struct {
    HeaderHeight  int
    TabHeight     int
    BodyHeight    int
    FooterHeight  int
    StatusHeight  int
    FilterHeight  int
}

func (m Model) calculateLayout() Layout {
    h := Layout{
        HeaderHeight: 1,
        TabHeight:    1,
        FooterHeight: 1,
        StatusHeight: 1,
    }
    
    if m.listFilterMode {
        h.FilterHeight = 1
    }
    
    h.BodyHeight = m.height - h.HeaderHeight - h.TabHeight - 
                   h.FooterHeight - h.StatusHeight - h.FilterHeight - 4
    
    if h.BodyHeight < 1 {
        h.BodyHeight = 1
    }
    
    return h
}
```

### 5.2 面板尺寸计算

```go
func (m Model) panelDimensions() (width, height int) {
    layout := m.calculateLayout()
    return m.width, layout.BodyHeight
}
```

## 6. 渲染逻辑优化

### 6.1 统一渲染入口

```go
func (m Model) View() string {
    // Docker 未连接
    if !m.dockerOK && m.status != "" {
        return m.renderDockerError()
    }
    
    // 确认对话框
    if m.pending != ActionNone {
        return m.renderConfirmDialog()
    }
    
    // 活动面板
    if m.activePanel != PanelNone {
        return m.renderActivePanel()
    }
    
    // 主布局
    return m.renderMain()
}
```

### 6.2 面板渲染

```go
func (m Model) renderActivePanel() string {
    panel, ok := m.panels[m.activePanel]
    if !ok {
        return m.renderMain()
    }
    
    header := views.RenderHeader() + "\n"
    content := panel.Render()
    footer := views.RenderFooter(m.busy, int(m.mode))
    status := views.RenderStatus(m.status, 0, 0, "")
    
    return lipgloss.JoinVertical(lipgloss.Left,
        lipgloss.NewStyle().Height(2).MaxHeight(2).Render(header),
        lipgloss.NewStyle().Height(m.height-4).MaxHeight(m.height-4).Render(content),
        lipgloss.NewStyle().Height(1).MaxHeight(1).Render(footer),
        lipgloss.NewStyle().Height(1).MaxHeight(1).Render(status),
    )
}
```

## 7. 状态清理

### 7.1 面板关闭时清理

```go
func (m *Model) cleanupPanel(panel PanelType) {
    switch panel {
    case PanelLogs:
        m.logPanel = nil
    case PanelStats:
        m.statsCh = nil
        m.statsErrCh = nil
        m.statsPanel = nil
    case PanelInspect:
        m.inspectPanel = nil
    case PanelImageHistory:
        m.imageHistoryPanel = nil
    case PanelHistory:
        m.historyPanel = nil
    case PanelExec:
        m.execInput = ""
        m.execOutput = ""
    }
}
```

## 8. 兼容性

- 保持现有按键映射不变
- 保持现有面板类型不变
- 渐进式重构，避免破坏性变更

## 9. 风险点

1. 面板生命周期管理需要仔细测试
2. 日志流的启停需要正确处理
3. 布局计算需要在不同终端尺寸下验证
