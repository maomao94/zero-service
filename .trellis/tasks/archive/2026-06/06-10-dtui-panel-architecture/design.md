# Technical Design: dtui 面板架构重构

## 概述

重构面板系统，引入统一的 Panel 接口和 PanelManager，实现面板状态自包含。

## 1. Panel 接口定义

```go
// Panel 定义面板的统一接口
type Panel interface {
    // Open 面板打开时调用，返回需要执行的命令
    Open(width, height int) tea.Cmd
    
    // Close 面板关闭时调用，清理资源
    Close()
    
    // Render 渲染面板内容
    Render() string
    
    // HandleKey 处理按键事件
    HandleKey(key string) tea.Cmd
    
    // HandleMsg 处理异步消息
    HandleMsg(msg tea.Msg) (Panel, tea.Cmd)
    
    // SetSize 更新面板尺寸
    SetSize(width, height int)
    
    // Help 返回当前面板的帮助文本
    Help() string
}
```

## 2. PanelManager 设计

```go
// PanelManager 管理面板的生命周期
type PanelManager struct {
    active PanelType
    panel  Panel
    width  int
    height int
}

func NewPanelManager(width, height int) *PanelManager {
    return &PanelManager{
        active: PanelNone,
        width:  width,
        height: height,
    }
}

// Open 打开指定类型的面板
func (pm *PanelManager) Open(panelType PanelType, factory func() Panel) tea.Cmd {
    if pm.active != PanelNone {
        pm.Close()
    }
    
    pm.active = panelType
    pm.panel = factory()
    return pm.panel.Open(pm.width, pm.height)
}

// Close 关闭当前面板
func (pm *PanelManager) Close() {
    if pm.panel != nil {
        pm.panel.Close()
    }
    pm.active = PanelNone
    pm.panel = nil
}

// Render 渲染当前面板
func (pm *PanelManager) Render() string {
    if pm.panel == nil {
        return ""
    }
    return pm.panel.Render()
}

// HandleKey 处理按键
func (pm *PanelManager) HandleKey(key string) tea.Cmd {
    if pm.panel == nil {
        return nil
    }
    return pm.panel.HandleKey(key)
}

// HandleMsg 处理消息
func (pm *PanelManager) HandleMsg(msg tea.Msg) (Panel, tea.Cmd) {
    if pm.panel == nil {
        return nil, nil
    }
    return pm.panel.HandleMsg(msg)
}

// SetSize 更新尺寸
func (pm *PanelManager) SetSize(width, height int) {
    pm.width = width
    pm.height = height
    if pm.panel != nil {
        pm.panel.SetSize(width, height)
    }
}

// Help 获取当前面板帮助
func (pm *PanelManager) Help() string {
    if pm.panel == nil {
        return ""
    }
    return pm.panel.Help()
}
```

## 3. 面板实现示例

### 3.1 ExecPanel

```go
type ExecPanel struct {
    container *dt.Container
    input     string
    output    string
    width     int
    height    int
}

func NewExecPanel(container *dt.Container) *ExecPanel {
    return &ExecPanel{container: container}
}

func (p *ExecPanel) Open(width, height int) tea.Cmd {
    p.width = width
    p.height = height
    return nil
}

func (p *ExecPanel) Close() {
    p.input = ""
    p.output = ""
}

func (p *ExecPanel) Render() string {
    var b strings.Builder
    b.WriteString(styles.PanelTitle.Render("── 容器执行: " + p.container.Name + " ──") + "\n")
    b.WriteString(views.RenderDim("  输入命令后按 Enter 执行，Esc 返回") + "\n\n")
    b.WriteString(fmt.Sprintf("  %s %s %s█",
        styles.DetailField.Render("$ "),
        styles.DetailValue.Render(p.input),
        styles.ListArrow.Render("")))
    
    if p.output != "" {
        b.WriteString("\n\n" + styles.ListDimText.Render("  ── 输出 ──") + "\n")
        for _, line := range splitTrimRight(p.output, "\n") {
            b.WriteString("  " + styles.DetailValue.Render(line) + "\n")
        }
    }
    
    b.WriteString("\n" + views.RenderDim("  Enter 执行  Esc 返回"))
    return b.String()
}

func (p *ExecPanel) HandleKey(key string) tea.Cmd {
    switch key {
    case "enter":
        if p.input != "" {
            cmd := p.input
            p.input = ""
            return p.runExecCmd(cmd)
        }
    case "backspace":
        if len(p.input) > 0 {
            p.input = p.input[:len(p.input)-1]
        }
    default:
        if len(key) == 1 {
            p.input += key
        }
    }
    return nil
}

func (p *ExecPanel) HandleMsg(msg tea.Msg) (Panel, tea.Cmd) {
    if actionMsg, ok := msg.(ActionMsg); ok {
        p.output = actionMsg.Text
    }
    return p, nil
}

func (p *ExecPanel) SetSize(width, height int) {
    p.width = width
    p.height = height
}

func (p *ExecPanel) Help() string {
    return fmt.Sprintf("%s 执行  %s 返回",
        styles.HelpKey.Render("enter"),
        styles.HelpKey.Render("esc"))
}
```

### 3.2 LogPanel

```go
type LogPanel struct {
    container *dt.Container
    model     *views.LogPanelModel
    width     int
    height    int
}

func NewLogPanel(container *dt.Container) *LogPanel {
    return &LogPanel{container: container}
}

func (p *LogPanel) Open(width, height int) tea.Cmd {
    p.width = width
    p.height = height
    p.model = views.NewLogPanelModel(width, height)
    return p.loadLogs()
}

func (p *LogPanel) Close() {
    p.model = nil
}

func (p *LogPanel) Render() string {
    if p.model == nil {
        return views.RenderDim("加载中...")
    }
    return p.model.Render(p.container.Name)
}

func (p *LogPanel) HandleKey(key string) tea.Cmd {
    if p.model != nil {
        return p.model.HandleKey(key)
    }
    return nil
}

func (p *LogPanel) HandleMsg(msg tea.Msg) (Panel, tea.Cmd) {
    switch m := msg.(type) {
    case LogLoadedMsg:
        if p.model != nil {
            p.model.SetLines(m.Lines, m.Reset)
        }
    case LogStreamMsg:
        if p.model != nil && !m.Done {
            p.model.AppendLine(m.Line)
        }
    }
    return p, nil
}

func (p *LogPanel) SetSize(width, height int) {
    p.width = width
    p.height = height
    if p.model != nil {
        p.model.SetSize(width, height)
    }
}

func (p *LogPanel) Help() string {
    return fmt.Sprintf("%s 搜索  %s 刷新  %s/%s 滚动",
        styles.HelpKey.Render("/"),
        styles.HelpKey.Render("r"),
        styles.HelpKey.Render("↑↓"),
        styles.HelpKey.Render("jk"))
}
```

## 4. Model 重构

```go
type Model struct {
    client   *dt.Client
    mode     ViewMode
    width    int
    height   int
    dockerOK bool
    busy     bool
    status   string
    
    // 游标
    cursor       int
    scrollOffset int
    
    // 业务数据
    containers    []dt.Container
    images        []dt.Image
    composeSvcs   []ComposeService
    deployTargets []DeployTarget
    
    // 面板管理
    panels *PanelManager
    
    // 确认对话框
    pending PendingAction
    
    // 列表过滤
    listFilter     string
    listFilterMode bool
    
    // 配置
    configPath string
}
```

## 5. Update 重构

```go
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.WindowSizeMsg:
        m.width = msg.Width
        m.height = msg.Height
        m.panels.SetSize(msg.Width, msg.Height)
        return m, nil
        
    case tea.KeyMsg:
        if m.pending != ActionNone {
            return m.handleConfirm(msg)
        }
        return m.handleKey(msg)
        
    default:
        // 委托给面板管理器处理
        if panel, cmd := m.panels.HandleMsg(msg); panel != nil {
            return m, cmd
        }
        return m, nil
    }
}
```

## 6. View 重构

```go
func (m Model) View() string {
    if !m.dockerOK && m.status != "" {
        return m.renderDockerError()
    }
    
    if m.pending != ActionNone {
        return m.renderConfirmDialog()
    }
    
    if m.panels.active != PanelNone {
        return m.renderPanel()
    }
    
    return m.renderMain()
}

func (m Model) renderPanel() string {
    header := views.RenderHeader() + "\n"
    content := m.panels.Render()
    footer := views.RenderFooter(m.busy, int(m.mode), m.panels.Help())
    status := views.RenderStatus(m.status, 0, 0, "")
    
    return lipgloss.JoinVertical(lipgloss.Left,
        lipgloss.NewStyle().Height(2).MaxHeight(2).Render(header),
        lipgloss.NewStyle().Height(m.height-4).MaxHeight(m.height-4).Render(content),
        lipgloss.NewStyle().Height(1).MaxHeight(1).Render(footer),
        lipgloss.NewStyle().Height(1).MaxHeight(1).Render(status),
    )
}
```

## 7. 迁移策略

### Phase 1: 定义接口和管理器
- 创建 Panel 接口
- 创建 PanelManager
- 保持现有代码不变

### Phase 2: 逐个迁移面板
- 先迁移简单的面板（如 HistoryPanel）
- 再迁移复杂的面板（如 LogPanel, StatsPanel）
- 最后迁移 ExecPanel

### Phase 3: 清理 Model
- 移除面板特有的状态
- 简化 Update 和 View

## 8. 兼容性

- 保持现有按键映射不变
- 保持现有功能不变
- 渐进式重构，每个阶段都可以独立测试

## 9. 风险点

1. 面板接口设计需要考虑周全
2. 消息处理需要正确委托给面板
3. 面板间的交互需要仔细设计
