# dtui 新架构技术设计

## 1. 架构概览

### 1.1 参考来源

| 项目 | 采用的模式 |
|------|-----------|
| Lazydocker | SideListPanel 泛型面板、ContextState Tab 管理、Docker Events 实时刷新、TaskManager 异步任务 |
| LazyGit | Context 驱动解耦、Controller 组合、ContextStack 导航栈、PopupHandler 弹窗、声明式 keybinding |
| k9s | 命令面板、Observer 模式样式刷新、装饰器链 |

### 1.2 核心设计原则

- **Context 驱动**：每个页面/面板是独立 Context，拥有自己的 Model/Update/View
- **声明式按键**：KeyBinding 集中注册，包含描述文本，自动生成 help
- **Bubbles 优先**：table/viewport/textinput/help/spinner/filepicker 替代自研代码
- **单向数据流**：Model → Update(msg) → (Model, Cmd) → View → 用户输入 → msg

### 1.3 Bubble Tea 适配说明

Lazydocker/LazyGit 使用 gocui（多 View 同时渲染），dtui 使用 Bubble Tea（单一 View 字符串）。因此：

- **多面板** → 通过 `lipgloss.JoinHorizontal` 拼接多个组件的 View() 输出
- **多 View 焦点** → 通过 ContextManager 跟踪活跃 Context，路由按键
- **异步任务** → 使用 `tea.Cmd` + `tea.Msg`，而非独立 goroutine + channel
- **鼠标事件** → `bubblezone` 提供区域标记，Bubble Tea 原生支持 MouseMsg

## 2. 目录结构

```
internal/tui/
├── app.go                  # App: 顶层 Model，持有所有 Context + 布局
├── app_update.go           # App.Update(): 全局消息路由
├── app_view.go             # App.View(): 布局拼接
│
├── context/                # Context 系统（借鉴 LazyGit）
│   ├── context.go          # Context 接口 + ContextManager
│   └── tree.go             # ContextTree: 所有 Context 注册
│
├── keybinding/             # 按键系统（借鉴 Lazydocker）
│   ├── binding.go          # KeyBinding 结构体
│   ├── registry.go         # 集中注册所有按键
│   └── help.go             # Help 文本生成
│
├── pages/                  # 5 个主页面（每个是 Context）
│   ├── containers/         # 容器视图
│   │   ├── page.go         # ContainerPage: Model+Update+View
│   │   ├── table.go        # bubbles/table 容器列表
│   │   ├── detail.go       # 右侧详情面板
│   │   └── panels/         # 全屏 Panel（enter 展开）
│   │       ├── log.go      # 日志面板
│   │       ├── stats.go    # Stats 面板
│   │       ├── inspect.go  # 详情面板
│   │       └── exec.go     # Exec 面板
│   ├── images/             # 镜像视图
│   │   ├── page.go
│   │   ├── table.go
│   │   └── panels/
│   │       └── history.go
│   ├── compose/            # 编排视图
│   │   ├── page.go
│   │   └── table.go
│   ├── deploy/             # 发布视图
│   │   ├── page.go
│   │   └── deploy.go
│   └── settings/           # 设置视图
│       ├── page.go
│       └── table.go
│
├── components/             # 可复用组件
│   ├── statusbar.go        # 状态栏组件
│   ├── tabbar.go           # Tab 栏组件
│   └── confirm.go          # 确认弹窗组件
│
├── actions/                # 操作层（无 UI 逻辑）
│   ├── container.go        # 容器操作工厂
│   ├── image.go            # 镜像操作工厂
│   ├── compose.go          # 编排操作工厂
│   ├── deploy.go           # 发布操作工厂
│   └── history.go          # 历史记录
│
└── styles/                 # 保留现有，小幅扩展
    ├── styles.go
    └── theme.go
```

## 3. Context 系统

### 3.1 Context 接口

```go
// context/context.go
type ContextType int

const (
    ContextContainers ContextType = iota
    ContextImages
    ContextCompose
    ContextDeploy
    ContextSettings
    ContextLogPanel
    ContextStatsPanel
    ContextInspectPanel
    ContextExecPanel
    ContextImageHistoryPanel
    ContextHistoryPanel
    ContextFormPanel
    ContextConfirm
    ContextFilePicker
)

type Context interface {
    Type() ContextType
    Init() tea.Cmd
    Update(msg tea.Msg) (Context, tea.Cmd)
    View() string
    Help() []keybinding.Binding    // 当前上下文的按键列表
    SetSize(width, height int)
    Title() string                 // 用于 Tab 显示
}
```

### 3.2 ContextManager（借鉴 LazyGit ContextStack）

```go
// context/context.go
type ContextManager struct {
    stack   []Context          // 导航栈（最后一个=活跃）
    tree    *ContextTree       // 所有 Context 实例
}

func (cm *ContextManager) Push(ctx Context) tea.Cmd  // 压栈进入新 Context
func (cm *ContextManager) Pop() (Context, tea.Cmd)    // 出栈返回上一层
func (cm *ContextManager) Current() Context            // 当前活跃 Context
func (cm *ContextManager) SwitchPage(ct ContextType) tea.Cmd  // 切换主页面

// ContextTree 持有所有 Context 实例
type ContextTree struct {
    Containers      Context
    Images          Context
    Compose         Context
    Deploy          Context
    Settings        Context
    // Panels 按需创建，不常驻
}
```

### 3.3 导航流程

```
主页面切换: Tab 键 → ContextManager.SwitchPage() → 替换栈底
Panel 进入: Enter/l 键 → ContextManager.Push(panel) → 压入 Panel
Panel 退出: Esc 键 → ContextManager.Pop() → 回到主页面
弹窗: 确认/FilePicker → ContextManager.Push(popup) → 临时弹出
```

### 3.4 页面生命周期

```go
// 进入 Context 时
func (cm *ContextManager) Push(ctx Context) tea.Cmd {
    prev := cm.Current()
    prev.OnBlur()                    // 通知旧 Context 失去焦点
    cm.stack = append(cm.stack, ctx)
    return ctx.Init()                // 初始化新 Context
}

// 离开 Context 时
func (cm *ContextManager) Pop() (Context, tea.Cmd) {
    ctx := cm.Current()
    ctx.OnExit()                     // 清理资源（关闭流等）
    cm.stack = cm.stack[:len(cm.stack)-1]
    return cm.Current(), nil
}
```

## 4. 按键系统（借鉴 Lazydocker keybindings.go）

### 4.1 KeyBinding 结构

```go
// keybinding/binding.go
type Scope int
const (
    ScopeGlobal   Scope = iota   // 所有页面可用
    ScopePage                   // 特定页面
    ScopePanel                  // 特定面板
)

type Binding struct {
    Keys        []string        // ["ctrl+c", "q"] 或 ["enter"]
    Description string          // 帮助文本
    Scope       Scope
    Context     ContextType     // 哪个 Context 有效
    Action      string          // 动作标识符
    Enabled     func() bool     // 动态启用条件（nil=始终启用）
}
```

### 4.2 集中注册

```go
// keybinding/registry.go
func AllBindings() []Binding {
    return append(
        globalBindings(),
        containerBindings(),
        imageBindings(),
        composeBindings(),
        deployBindings(),
        settingsBindings(),
        panelBindings(),
    )
}

func containerBindings() []Binding {
    return []Binding{
        {Keys: []string{"s"}, Desc: "启停", Scope: ScopePage, Context: ContextContainers, Action: "toggle"},
        {Keys: []string{"R"}, Desc: "重启", Scope: ScopePage, Context: ContextContainers, Action: "restart"},
        {Keys: []string{"l"}, Desc: "日志", Scope: ScopePage, Context: ContextContainers, Action: "logs"},
        // ...
    }
}
```

### 4.3 Help 自动生成

当前 Context 的按键列表通过 `Context.Help()` 返回，由 `bubbles/help` 组件渲染：

```go
func (p *ContainerPage) Help() []keybinding.Binding {
    return FilterByContext(AllBindings(), ContextContainers)
}
```

## 5. 布局系统

### 5.1 Lazydocker 双栏布局

```
┌──────────────────────────────────────────────────┐
│ dtui | 宿主机 Docker 管理                          │  ← Header
├──────────────────────────────────────────────────┤
│  容器  │  镜像  │  编排  │  前端发布  │  设置      │  ← TabBar (lipgloss)
├──────────────────────┬───────────────────────────┤
│                      │  Stats                    │
│  Container List      │  CPU  ████████░░ 45%      │
│  (bubbles/table)     │  MEM  ████░░░░░░ 2.1G/8G  │
│                      │  NET  ↑1.2M ↓800K         │
│  ▶ nginx-proxy       │                           │
│    redis-cache       │  Info                     │
│    postgres-db       │  Image: nginx:latest      │
│    node-app          │  Created: 2024-01-01      │
│                      │  Ports: 80:8080           │
│                      │                           │
│                      │  Logs (preview)           │
│                      │  10:23:45 GET /index.html │
│                      │  10:23:46 GET /style.css  │
│                      │                           │
├──────────────────────┴───────────────────────────┤
│ tab 切换  r 刷新  ↑↓/jk 选行  H 历史  q 退出   │  │  ← Footer (bubbles/help)
├──────────────────────────────────────────────────┤
│  容器 1/5                                         │  ← StatusBar
└──────────────────────────────────────────────────┘
```

### 5.2 布局计算

```go
func (app *App) calculateLayout() (listW, detailW int) {
    totalW := app.width - 2  // padding
    if totalW < 60 {
        // 小屏幕：单栏模式
        return totalW, 0
    }
    // 正常：40% 列表 + 60% 详情
    listW = totalW * 40 / 100
    detailW = totalW - listW - 1  // 1 for divider
    return
}
```

### 5.3 全屏展开（Enter 键）

选中容器后按 `Enter` → `ContextManager.Push(LogPanel)` → 全屏日志

选中容器后按 `Enter` + Tab → 在 Log/Stats/Inspect/Env 之间切换

## 6. Bubbles 组件适配

### 6.1 bubbles/table

每个列表页使用 `bubbles/table`：

```go
type ContainerTable struct {
    table.Model
    client  *docker.Client
    columns []table.Column
    rows    []table.Row
    filter  string
}
```

- 列定义：Name, Image, Status, Ports, Created
- 排序：点击列头排序（bubbles/table 内置）
- 选中行高亮（bubbles/table 的 `WithFocused`）
- 过滤：通过 `textinput` 修改 `filter`，重新计算 `rows`

### 6.2 bubbles/viewport

日志、Stats、详情面板使用 `bubbles/viewport`：

```go
type LogPanel struct {
    viewport  viewport.Model
    container *docker.Container
    logStream chan string
}
```

- 自动滚动到底部（follow mode）
- 鼠标滚轮支持（viewport 内置）
- 搜索高亮

### 6.3 bubbles/textinput

搜索过滤、Exec 命令输入、表单输入：

```go
type SearchBar struct {
    input textinput.Model
    active bool
}
```

### 6.4 bubbles/help

Footer 自动生成：

```go
func (app *App) renderFooter() string {
    ctx := app.ctx.Current()
    bindings := ctx.Help()
    return app.help.View(bindings)  // bubbles/help 自动布局
}
```

### 6.5 bubbles/spinner

加载状态：

```go
type LoadState struct {
    spinner spinner.Model
    loading bool
    message string
}
```

### 6.6 bubbles/filepicker

部署路径选择、镜像保存路径：

```go
type FilePickPopup struct {
    filepicker filepicker.Model
    onSelect   func(path string)
}
```

### 6.7 bubblezone（鼠标）

```go
// 在 View() 中标记可点击区域
func (p *ContainerPage) View() string {
    return zone.Mark("tab-0", tab0) +
           zone.Mark("tab-1", tab1) +
           // ...
}

// 在 Update() 中处理鼠标点击
case tea.MouseMsg:
    if msg.Action == tea.MouseActionPress {
        if zone.Get(msg).Target == "tab-0" {
            return p, p.ctx.SwitchPage(ContextContainers)
        }
    }
```

## 7. 操作层

### 7.1 Action 工厂函数

所有 Docker 操作抽取为独立 Action，返回 `tea.Cmd`：

```go
// actions/container.go
func StartContainer(client *docker.Client, id string) tea.Cmd {
    return func() tea.Msg {
        err := client.StartContainer(id)
        if err != nil {
            return action.ErrorMsg{Action: "start", Err: err}
        }
        recordHistory("start", id, true)
        return action.DoneMsg{Action: "start"}
    }
}
```

### 7.2 历史记录中间件

不在每个 Action 中重复 `RecordHistory`，而是在 `executeAction` 中统一处理：

```go
func executeAction(client *docker.Client, binding Binding, ctx *ContainerPage) tea.Cmd {
    return func() tea.Msg {
        msg := binding.Action(client, ctx.SelectedID())
        // 统一记录历史
        if done, ok := msg.(action.DoneMsg); ok {
            recordHistory(done.Action, ctx.SelectedID(), true)
        } else if err, ok := msg.(action.ErrorMsg); ok {
            recordHistory(err.Action, ctx.SelectedID(), false)
        }
        return msg
    }
}
```

## 8. 数据流

### 8.1 加载流程

```
App.Init()
  → tea.Batch(
      docker.Ping(),             // 检查 Docker 连接
      loadConfig(),              // 加载配置
    )
  → 收到 PingOk → dockerOK = true
    → tea.Batch(
        loadContainers(),        // 加载各页面数据
        loadImages(),
        loadCompose(),
        loadDeploy(),
      )
    → 收到 LoadedMsg → 更新各 Context 数据
```

### 8.2 操作流程

```
用户按键 → App.Update(KeyMsg)
  → ContextManager 路由到当前 Context
  → Context.Update(KeyMsg)
  → Keybinding 匹配 → executeAction()
  → tea.Cmd 异步执行 Docker 操作
  → 收到 ActionMsg → 刷新列表 + 更新状态栏
  → View() 重渲染
```

### 8.3 实时刷新

利用 Docker Events API（借鉴 Lazydocker）：

```go
func (app *App) watchDockerEvents() tea.Cmd {
    return func() tea.Msg {
        eventsCh, errCh := app.client.Events(ctx, options)
        for {
            select {
            case event := <-eventsCh:
                // 收到任何事件 → 全局刷新
                return events.RefreshMsg{}
            case err := <-errCh:
                return events.ErrorMsg{Err: err}
            }
        }
    }
}
```

## 9. 组件树

```
App (top-level Model)
├── HeaderBar (lipgloss)
├── TabBar (lipgloss)
├── ContextManager
│   ├── ContainerPage (active context)
│   │   ├── ContainerTable (bubbles/table)
│   │   ├── DetailPanel (right side)
│   │   │   ├── StatsWidget (bubbles/viewport)
│   │   │   ├── InfoWidget (bubbles/viewport)
│   │   │   └── LogPreviewWidget (bubbles/viewport)
│   │   └── SearchBar (bubbles/textinput)
│   ├── ImagePage
│   │   ├── ImageTable (bubbles/table)
│   │   └── ...
│   ├── ComposePage
│   ├── DeployPage
│   └── SettingsPage
│   └── Active Panel (when pushed, e.g., LogPanel, StatsPanel)
├── StatusBar
└── HelpBar (bubbles/help)
```

## 10. 消息类型

```go
// 操作结果
type ActionMsg struct {
    Action string
    Text   string
    Err    error
}

// 数据刷新
type RefreshMsg struct {
    Source string  // "containers", "images", "compose", "deploy"
}

// Docker 事件
type DockerEventMsg struct {
    Action string  // "start", "die", "destroy" ...
    ID     string
}

// 页面切换
type SwitchPageMsg struct {
    Target ContextType
}

// 打开面板
type OpenPanelMsg struct {
    Panel PanelType
    ID    string
}

// 面板全屏
type ToggleFullscreenMsg struct{}
```

## 11. 与保留代码的边界

| 保留模块 | 接口 | 说明 |
|---------|------|------|
| `docker.Client` | `ListContainers()`, `StartContainer()`, `StreamLogs()`, etc. | 不变，可能需要新增 `Events()` |
| `config.Config` | `Load()`, `Save()`, `AddComposeDir()`, etc. | 不变 |
| `styles/theme.go` | `ColorBg`, `ColorFg`, ... | 不变 |
| `styles/styles.go` | 各 Style 对象 | 新增 Bubbles 适配 Style |
| `logger` | `Infow()`, `Errorw()` | 不变 |
| `cli/root.go` | `Run(client, configPath)` | 签名不变，内部重构 |

## 12. 文件大小约束

每个文件不超过 400 行。如果超限：

- 主页面 Model 拆分：`page.go` (Model+Update) + `table.go` (Table 组件) + `detail.go` (详情面板)
- Action 文件按域名拆分：`container.go`、`image.go`、`compose.go`、`deploy.go`
- Keybinding 按 Context 拆分注册函数
