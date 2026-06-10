# PRD: dtui 面板架构重构 - 统一生命周期与状态管理

## 背景

当前 dtui 的面板架构存在严重的结构性问题：

### 问题清单

**P1: 状态管理混乱**
- Model 中混杂了全局状态和面板特有状态
- `execInput/execOutput` 是 exec 面板特有，却放在 Model 中
- `statsCh/statsErrCh` 是 stats 面板特有，却放在 Model 中
- 每个面板的指针（logPanel, inspectPanel 等）都散落在 Model 中

**P2: 面板生命周期不清晰**
- 没有统一的 Panel 接口
- 面板的初始化、渲染、清理逻辑分散在多处
- 新增面板需要修改 Model、Update、View、keys 等多个文件

**P3: 状态清理不一致**
- closePanel() 需要手动清理每种面板的状态
- 容易遗漏某些状态的清理（如 busy 状态）
- 错误处理时直接设置 `m.panel = PanelNone`，绕过清理逻辑

**P4: 渲染逻辑分散**
- renderPanel() 根据 panel 类型分发
- renderFullPanel() 处理 log/stats/exec
- renderChromePanel() 处理 inspect/imageHistory
- renderHistoryPanel() 单独处理
- 新增面板需要修改多处渲染逻辑

## 需求

### 1. 定义统一的 Panel 接口

```go
type Panel interface {
    // 生命周期
    Open(width, height int) tea.Cmd
    Close()
    
    // 渲染
    Render() string
    
    // 按键处理
    HandleKey(key string) tea.Cmd
    
    // 消息处理
    HandleMsg(msg tea.Msg) (Panel, tea.Cmd)
    
    // 尺寸更新
    SetSize(width, height int)
}
```

### 2. 面板自包含状态

每个面板管理自己的状态，不再散落在 Model 中：
- ExecPanel 管理 execInput, execOutput
- StatsPanel 管理 statsCh, statsErrCh
- LogPanel 管理日志流状态

### 3. 统一的面板管理器

```go
type PanelManager struct {
    active PanelType
    panel  Panel
    width  int
    height int
}
```

### 4. Model 简化

Model 只保留：
- 全局状态（dockerOK, busy, status）
- 业务数据（containers, images 等）
- PanelManager

## 验收标准

1. 新增面板只需实现 Panel 接口，无需修改 Model
2. 面板状态完全自包含，不污染 Model
3. 面板切换逻辑统一由 PanelManager 管理
4. 所有面板的生命周期一致
5. 编译通过，所有功能正常
