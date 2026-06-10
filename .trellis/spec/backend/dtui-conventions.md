# DTUI — Docker SDK 交互规范

> ⚠️ **TUI 架构已迁移到 uix 框架**。Panel 接口、PanelManager、旧 Model 结构已废弃。
> TUI 开发请阅读 [uix-framework.md](./uix-framework.md)。
> 本文件仅保留 Docker SDK 交互契约，TUI 相关内容仅作历史参考。

## 架构（当前：uix 插件体系）

dtui 已从单体 Model 重构为基于 uix 框架的插件体系：

| 包 | 职责 |
| --- | --- |
| `cli/uix/` | TUI 框架核心（FrameworkApp、Plugin 接口、组件、主题） |
| `cli/dtui/main.go` | 入口：创建 app、注册插件、启动 |
| `cli/dtui/plugins/` | 插件实现（containers、images、compose、deploy、settings） |
| `cli/dtui/internal/docker/` | Docker SDK 封装 |
| `cli/dtui/internal/config/` | 配置文件管理 |

### 启动流程

```go
app := uix.NewApp("dtui > ")
app.Register(containers.New(client))
app.Register(images.New(client))
app.Register(deploy.New(client, cfg))
app.SetHome(func() string { return homeScreen.View() })
app.Run()
```

---

---

## Scenario: Docker SDK 交互

### 1. Scope / Trigger

将所有 Docker 操作从 `exec.Command("docker", ...)` 迁移到 `github.com/docker/docker/client` SDK，获得结构化返回值和类型安全。

### 2. Signatures

```go
// Client 封装
type Client struct {
    cli     *client.Client
    ctx     context.Context
    timeout time.Duration  // 默认 10s
}

func NewClient() (*Client, error)       // FromEnv + API 版本协商
func (c *Client) Ping() error            // 3s 超时健康检查
func (c *Client) Close() error           // 释放连接
func (c *Client) RawClient() *client.Client  // 暴露原始 SDK（慎用）
func (c *Client) withTimeout() (context.Context, context.CancelFunc)
```

```go
// 容器操作
func (c *Client) ListContainers(filter string) ([]Container, error)
func (c *Client) StartContainer(id string) error
func (c *Client) StopContainer(id string) error
func (c *Client) RestartContainer(id string) error
func (c *Client) RemoveContainer(id string, force bool) error

// 镜像操作
func (c *Client) ListImages(filter string) ([]Image, error)
func (c *Client) SaveImage(ref, outputPath string) error
func (c *Client) RemoveImage(ref string, force bool) error
func (c *Client) TagImage(source, target string) error
func (c *Client) PruneImages() (reclaimed uint64, err error)
func (c *Client) ImageHistory(ref string) ([]ImageHistoryEntry, error)

// 日志
func (c *Client) FetchLogs(id string, opts LogOptions) ([]string, error)
func (c *Client) StreamLogs(id string, opts LogOptions) (<-chan string, <-chan error)

// 详情 & Stats
func (c *Client) InspectContainer(id string) (*ContainerDetail, error)
func (c *Client) StreamStats(id string) (<-chan StatsEntry, <-chan error)
```

### 3. Contracts

**Container 结构体（从 SDK `types.Container` 精简）：**
```go
type Container struct {
    ID      string
    Image   string
    Command string
    Created string   // 2006-01-02 15:04:05 格式
    Status  string
    Ports   string   // 逗号分隔 "8080:80/tcp, 443:443/tcp"
    Name    string   // 已去掉 / 前缀
    State   string   // running / exited / paused / created
}
func (c Container) Running() bool  // strings.EqualFold(c.State, "running")
```

**LogOptions：**
```go
type LogOptions struct {
    Tail       string // "200" or "all"
    Since      string // "10m", "1h"
    Follow     bool
    Timestamps bool
}
```

**StatsEntry：**
```go
type StatsEntry struct {
    CPUPercent float64  // 0-100 * numCPUs
    MemUsage   uint64   // bytes
    MemLimit   uint64   // bytes
    MemPercent float64  // 0-100
    NetRx      uint64   // bytes
    NetTx      uint64   // bytes
    BlockRead  uint64   // bytes
    BlockWrite uint64   // bytes
    PIDs       uint64
}
```

**ContainerDetail（inspect 结果）：**
```go
type ContainerDetail struct {
    ID, Name, Image, Platform, Created string
    State    ContainerState{Status, Running, Paused, StartedAt, ExitCode int, Error string}
    Mounts   []MountInfo{Type, Source, Destination, Mode string}
    Network  []NetworkInfo{Name, IPAddress, Gateway, MacAddress string}
    Env      []string
    Cmd, Entrypoint []string
    WorkingDir, RestartPolicy string
}
```

### 4. Validation & Error Matrix

| 条件 | 错误 |
| --- | --- |
| Docker daemon 未运行 | `Ping()` 返回 error → Cobra 入口捕获并退出 |
| 容器不存在 | SDK 方法返回 `Error response from daemon: No such container` |
| 日志流连接断开 | `scanner.Err() != nil && err != io.EOF` → 推送到 `errCh` |
| Stats 流 EOF | `decoder.Decode` 返回 `io.EOF` → 关闭 statsCh，正常结束 |
| CPU 分母为零 | `cpuDelta` 或 `systemDelta` ≤ 0 → `CPUPercent` = 0 |
| `OnlineCPUs` 为 0 | 回退到 `len(PercpuUsage)`，仍为 0 则设为 1 |

### 5. Good/Base/Bad Cases

**Good — SDK 直接获取结构化数据：**
```go
containers, err := c.cli.ContainerList(ctx, container.ListOptions{All: true})
// 直接读取 ctr.ID, ctr.Image, ctr.State, ctr.Ports...
```

**Base — compose/exec 保留 exec.Command（SDK 不支持）：**
```go
func RunComposeUp(composeFile, serviceName string) (string, error) {
    cmd := exec.Command("docker", "compose", "-f", composeFile, "up", "-d", serviceName)
    out, err := cmd.CombinedOutput()
    return string(out), err
}
```

**Bad — 混用 SDK 和 exec.Command 做同一操作：**
```go
// 不要这样做
containers, _ := c.cli.ContainerList(...)
// 又用 exec.Command 获取同样的信息
out, _ := exec.Command("docker", "ps").Output()
```

### 6. Tests Required

- `TestNewClient`: 验证 FromEnv 配置和 API 版本协商
- `TestPing`: mock Docker daemon 连接成功/失败
- `TestListContainers`: 验证结构化返回，测试空列表、filter 匹配
- `TestFetchLogs`: 验证 8-byte header 解析、空日志、多行日志
- `TestStreamStats`: 验证 CPU 百分比计算、边界情况（零值 denominator）
- `TestInspectContainer`: 验证 ContainerDetail 所有字段映射

### 7. Wrong vs Correct

#### Wrong — 用字符串解析 Docker 输出
```go
out, _ := exec.Command("docker", "ps", "--format", "{{.ID}}|{{.Image}}").Output()
parts := strings.Split(strings.TrimSpace(string(out)), "|")
```

#### Correct — 用 SDK 获取结构化数据
```go
containers, err := c.cli.ContainerList(ctx, container.ListOptions{All: true})
for _, ctr := range containers {
    fmt.Println(ctr.ID, ctr.Image)
}
```

---

## Scenario: Docker 日志流解析

### 1. Scope / Trigger

Docker API 的 `ContainerLogs` 返回两种格式：**多路复用流**（非 TTY 容器，8 字节帧头）和**原始文本流**（TTY 容器，直接可读）。必须用 `stdcopy.StdCopy` 统一处理。

### 2. Signatures

```go
import "github.com/docker/docker/pkg/stdcopy"

// FetchLogs 使用 stdcopy 正确解复用
func (c *Client) FetchLogs(id string, opts LogOptions) ([]string, error) {
    reader, _ := c.cli.ContainerLogs(ctx, id, container.LogsOptions{...})
    defer reader.Close()

    var buf bytes.Buffer
    stdcopy.StdCopy(&buf, &buf, reader) // stdout+stderr 合并

    scanner := bufio.NewScanner(&buf)
    for scanner.Scan() { ... }
}
```

### 3. Contracts

**stdcopy.StdCopy 自动识别格式：**
- 多路复用流：解析 8 字节 header `[type(1)][padding(3)][size(4)][payload]`，stdout→outWriter, stderr→errWriter
- TTY 流：直接透传原始文本

**LogOptions（不变）：**
```go
type LogOptions struct {
    Tail       string
    Since      string
    Follow     bool
    Timestamps bool
}
```

### 4. Validation & Error Matrix

| 条件 | 行为 |
| --- | --- |
| TTY 容器 | `stdcopy` 直接透传原始文本 |
| 非 TTY 容器 | `stdcopy` 解析 8 字节帧头，合并 stdout+stderr |
| 超时 | `context.WithTimeout` 取消，返回已读行 |
| scanner 错误 | 检查非 nil → 包装返回 |

### 5. Good/Base/Bad Cases

**Good — FetchLogs 用 stdcopy 批量读取：**
```go
// Follow=false: stdcopy 读取全部 → buffer → scanner
var buf bytes.Buffer
stdcopy.StdCopy(&buf, &buf, reader)
scanner := bufio.NewScanner(&buf)
```

**Base — StreamLogs 用 stdcopy + io.Pipe 逐行推送（Follow=true 需要）：**
```go
// Follow=true: stdcopy 在后台持续解复用，scanner 从 pipe 逐行读取
pipeReader, pipeWriter := io.Pipe()
go func() {
    _, err := stdcopy.StdCopy(pipeWriter, pipeWriter, reader)
    _ = pipeWriter.CloseWithError(err)
}()
scanner := bufio.NewScanner(pipeReader)
```

**Bad — StreamLogs 用 stdcopy 写入 bytes.Buffer：**
```go
// Follow=true 时 StdCopy 不返回，scanner 永远读不到 buffer
var buf bytes.Buffer
stdcopy.StdCopy(&buf, &buf, reader)
```

### 6. Tests Required

- `TestFetchLogs`: 对已有容器验证日志获取，至少返回非空行
- `TestStreamLogs`: 验证 Follow=true 时能在 3s 内收到新行

### 7. Wrong vs Correct

#### Wrong — StreamLogs with bytes.Buffer blocks on Follow=true
```go
var buf bytes.Buffer
stdcopy.StdCopy(&buf, &buf, reader)
scanner := bufio.NewScanner(&buf) // Follow=true 时不会执行到这里
```

#### Correct — stdcopy + io.Pipe 兼容多路复用和 TTY
```go
pipeReader, pipeWriter := io.Pipe()
go func() { _, err := stdcopy.StdCopy(pipeWriter, pipeWriter, reader); _ = pipeWriter.CloseWithError(err) }()
scanner := bufio.NewScanner(pipeReader)
```

---

## Scenario: Docker 日志流式推送 (Follow=true)

### 1. Scope / Trigger

`StreamLogs` 方法用于 `Follow: true` 场景，需要从 Docker daemon 持续接收日志行并通过 channel 推送给 TUI 层。`stdcopy.StdCopy` 可以用于 Follow 模式，但必须写入 `io.Pipe`，由 scanner 同时从 pipe 读取；不能写入 `bytes.Buffer` 后再扫描。

### 2. Signatures

```go
// StreamLogs 流式获取容器日志（逐行推送，不阻塞）
func (c *Client) StreamLogs(id string, opts LogOptions) (<-chan string, <-chan error)

```

### 3. Contracts

**解复用规则：**
- 使用 `stdcopy.StdCopy(pipeWriter, pipeWriter, reader)` 解析 Docker 非 TTY 多路复用帧头
- TTY 原始文本由 `stdcopy` 透传
- scanner 只负责按换行切分已经解复用后的文本

**StreamLogs channel 契约：**
- logCh: 缓冲 200，goroutine 逐行写入，关闭时 close(logCh)
- errCh: 缓冲 1，连接/解析错误写入
- goroutine 使用 `c.ctx`（WithCancel），Close() 可取消

### 4. Validation & Error Matrix

| 条件 | 行为 |
| --- | --- |
| Follow=true + TTY 容器 | `stdcopy` 透传，scanner 按行推送 |
| Follow=true + 非 TTY 容器 | `stdcopy` 解复用 8 字节帧头，scanner 按行推送 |
| scanner.Err() != nil | 推送到 errCh |
| ctx 取消 (Close) | goroutine 退出，close(logCh) |
| 容器停止 | reader EOF → scanner 结束 → close(logCh) |

### 5. Good/Base/Bad Cases

**Good — stdcopy + io.Pipe + channel：**
```go
pipeReader, pipeWriter := io.Pipe()
go func() { _, err := stdcopy.StdCopy(pipeWriter, pipeWriter, reader); _ = pipeWriter.CloseWithError(err) }()
scanner := bufio.NewScanner(pipeReader)
for scanner.Scan() {
    select {
    case logCh <- scanner.Text():
    case <-ctx.Done(): return
    }
}
```

**Bad — stdcopy 写入 bytes.Buffer 再扫描：**
```go
var buf bytes.Buffer
stdcopy.StdCopy(&buf, &buf, reader)
scanner := bufio.NewScanner(&buf)
```

### 6. Tests Required

- `TestStreamLogs_Follow`: 启动容器，写入日志，验证 3s 内收到新行
- `TestStreamLogs_TTY`: 原始文本（无帧头）正确按行分割
- `TestStreamLogs_Multiplexed`: 带 8 字节帧头的数据由 stdcopy 正确剥离

### 7. Wrong vs Correct

#### Wrong — bytes.Buffer waits for EOF
```go
var buf bytes.Buffer
stdcopy.StdCopy(&buf, &buf, reader)
scanner := bufio.NewScanner(&buf)
```

#### Correct — io.Pipe allows scanner to consume during Follow=true
```go
pipeReader, pipeWriter := io.Pipe()
go func() { _, err := stdcopy.StdCopy(pipeWriter, pipeWriter, reader); _ = pipeWriter.CloseWithError(err) }()
scanner := bufio.NewScanner(pipeReader)
for scanner.Scan() { logCh <- scanner.Text() }
```

---

## Scenario: Stats CPU 百分比计算

### 1. Scope / Trigger

从 Docker stats API 的 `container.StatsResponse` 计算实时 CPU 使用率。

### 2. Signatures

```go
func (c *Client) StreamStats(id string) (<-chan StatsEntry, <-chan error)
func parseStats(v *container.StatsResponse, prevCPU *uint64, prevSystem *uint64) StatsEntry
```

### 3. Contracts

**CPU 百分比公式：**
```
CPU% = (cpuDelta / systemDelta) × numCPUs × 100
```

其中：
- `cpuDelta = v.CPUStats.CPUUsage.TotalUsage - prevCPU`
- `systemDelta = v.CPUStats.SystemUsage - prevSystem`
- `numCPUs = v.CPUStats.OnlineCPUs`（回退链：OnlineCPUs → len(PercpuUsage) → 1）

**前置条件：** `systemDelta > 0 && cpuDelta > 0`，否则 `CPUPercent = 0`。

### 4. Validation & Error Matrix

| 条件 | 行为 |
| --- | --- |
| `systemDelta <= 0` | `CPUPercent = 0`（首次调用或时间倒退） |
| `cpuDelta <= 0` | `CPUPercent = 0` |
| `OnlineCPUs == 0` | 回退 `len(PercpuUsage)` |
| 两次回退后仍为 0 | 设为 1（安全默认值） |
| `MemLimit == 0` | `MemPercent = 0`（无 cgroup 限制） |

### 5. Good/Base/Bad Cases

**Good:**
```go
cpuDelta := float64(v.CPUStats.CPUUsage.TotalUsage - *prevCPU)
systemDelta := float64(v.CPUStats.SystemUsage - *prevSystem)
*prevCPU = v.CPUStats.CPUUsage.TotalUsage
*prevSystem = v.CPUStats.SystemUsage
if systemDelta > 0 && cpuDelta > 0 {
    numCPUs := ...
    entry.CPUPercent = (cpuDelta / systemDelta) * numCPUs * 100
}
```

**Bad — 不保存上一帧数据：**
```go
// 每次都是绝对值，无法计算增量百分比
entry.CPUPercent = float64(v.CPUStats.CPUUsage.TotalUsage) / float64(v.CPUStats.SystemUsage) * 100
```

### 6. Tests Required

- `TestParseStats_Normal`: 两帧正常数据，验证 CPU 百分比
- `TestParseStats_ZeroDenominator`: systemDelta=0, cpuDelta=0
- `TestParseStats_NoOnlineCPUs`: OnlineCPUs=0 时的回退
- `TestParseStats_NoMemoryLimit`: MemLimit=0 时 MemPercent=0

### 7. Wrong vs Correct

#### Wrong — 忽略分母保护
```go
entry.CPUPercent = (cpuDelta / systemDelta) * numCPUs * 100  // 可能 panic 或 Inf
```

#### Correct — 条件保护
```go
if systemDelta > 0 && cpuDelta > 0 {
    entry.CPUPercent = (cpuDelta / systemDelta) * numCPUs * 100
}
```

---

## Bubble Tea 模式

### Model 结构

```go
type Model struct {
    client   *dt.Client
    mode     ViewMode
    width    int
    height   int
    pending  PendingAction
    busy     bool
    status   string
    dockerOK bool

    cursor       int
    scrollOffset int

    containers    []dt.Container
    images        []dt.Image
    composeSvcs   []ComposeService
    deployTargets []DeployTarget

    // 面板管理 — 统一生命周期
    panels *PanelManager

    listFilter     string
    listFilterMode bool
    configPath     string
}
```

**关键变更 (v3)：** 移除所有子面板字段（`logPanel`, `inspectPanel`, `statsPanel`, `imageHistoryPanel`, `execInput`, `execOutput`, `statsCh`, `statsErrCh`），统一由 `PanelManager` 管理面板生命周期。新增面板只需实现 `Panel` 接口。

---

## Scenario: Panel 接口 + PanelManager 架构 (v3)

### 1. Scope / Trigger

面板状态（日志流、stats 通道、exec 输入输出）之前散落在 Model 中，新增面板需要修改 Model 结构体、`openPanel`、`closePanel`、`cleanupPanel`、`syncPanelSizes` 等多处。重构为 Panel 接口 + PanelManager 统一管理。

### 2. Signatures

```go
// Panel 是所有子面板必须实现的接口
type Panel interface {
    Open(width, height int) tea.Cmd
    Close()
    Render() string
    HandleKey(key string) tea.Cmd
    HandleMsg(msg tea.Msg) (Panel, tea.Cmd)
    SetSize(width, height int)
    Help() string
}

// PanelManager 管理面板生命周期
type PanelManager struct {
    active PanelType
    panel  Panel
    width  int
    height int
}
func NewPanelManager(width, height int) *PanelManager
func (pm *PanelManager) Open(panelType PanelType, factory func() Panel) tea.Cmd
func (pm *PanelManager) Close()
func (pm *PanelManager) Render() string
func (pm *PanelManager) HandleKey(key string) tea.Cmd
func (pm *PanelManager) HandleMsg(msg tea.Msg) tea.Cmd
func (pm *PanelManager) SetSize(width, height int)
func (pm *PanelManager) Help() string
```

### 3. Contracts

**面板实现约定：**
- `Open` — 初始化面板状态，返回加载命令（nil 表示无需异步加载）
- `Close` — 清理面板资源（关闭通道、释放模型）
- `Render` — 返回面板内容字符串，使用当前 width/height
- `HandleKey` — 处理按键，返回可能触发的命令
- `HandleMsg` — 处理异步消息（如 LogLoadedMsg、StatsMsg），可返回新 Panel 实例
- `SetSize` — 终端尺寸变化时同步
- `Help` — 返回当前面板的帮助文本

**View 集成：**
```go
func (m Model) View() string {
    if m.panels.active != PanelNone {
        header := views.RenderHeader() + "\n"
        content := m.panels.Render()
        footer := views.RenderFooter(m.busy, int(m.mode))
        status := views.RenderStatus(m.status, 0, 0, "")
        return lipgloss.JoinVertical(lipgloss.Left,
            lipgloss.NewStyle().Height(2).MaxHeight(2).Render(header),
            lipgloss.NewStyle().Height(m.height-4).MaxHeight(m.height-4).Render(content),
            lipgloss.NewStyle().Height(1).MaxHeight(1).Render(footer),
            lipgloss.NewStyle().Height(1).MaxHeight(1).Render(status),
        )
    }
    return m.renderMain()
}
```

### 4. Validation & Error Matrix

| 条件 | 行为 |
| --- | --- |
| `selectedContainer() == nil` | openPanel 返回 nil，面板不打开 |
| `factory()` 返回 nil | PanelManager.Open 跳过 Open 调用 |
| 面板内 HandleMsg 返回新 Panel | PanelManager 替换当前面板实例 |
| closePanel() 调用时 | 设置 `busy = false`，防止 footer 残留"执行中..." |

### 5. Good/Base/Bad Cases

**Good — 新增面板只需实现 Panel 接口：**
```go
type ExecPanelImpl struct {
    input  string
    output string
    onExec func(string, string) tea.Cmd
}
func (p *ExecPanelImpl) Open(w, h int) tea.Cmd { return nil }
func (p *ExecPanelImpl) Close() { p.input = ""; p.output = "" }
func (p *ExecPanelImpl) HandleMsg(msg tea.Msg) (Panel, tea.Cmd) { ... }
```

**Bad — 在 Model 中散落面板状态：**
```go
type Model struct {
    execInput    string    // exec 面板特有
    statsCh      <-chan dt.StatsEntry  // stats 面板特有
    logPanel     *views.LogPanelModel  // log 面板特有
}
```

### 6. Tests Required

- `TestPanelOpenClose`: 验证面板打开/关闭生命周期
- `TestPanelHandleMsg`: 验证异步消息正确传递

### 7. Wrong vs Correct

#### Wrong — 面板状态散落，手动清理
```go
func (m *Model) closePanel() {
    m.cleanupPanel(m.panel)  // 需手动维护每种面板的清理逻辑
}
```

#### Correct — Panel 自包含，一行关闭
```go
func (m *Model) closePanel() {
    m.busy = false
    m.panels.Close()
}
```

---

### 消息驱动架构

所有 Docker 操作结果通过自定义消息类型传递，不做阻塞调用：

```go
// 数据加载
type LoadedMsg struct { View ViewMode; Containers []dt.Container; Images []dt.Image; ...; Err error }
// 操作结果
type ActionMsg struct { Text string; Err error }
// 日志加载
type LogLoadedMsg struct { Lines []string; Reset bool }
type LogStreamMsg struct { Line string; Done bool; Err error }
// 实时统计
type StatsMsg struct { Entry dt.StatsEntry; Err error; Done bool }
// 详情
type InspectMsg struct { Detail *dt.ContainerDetail; Err error }
type ImageHistoryMsg struct { Entries []dt.ImageHistoryEntry; Err error }
// 定时器
type TickMsg struct{}
```

### tea.Cmd 工厂模式

```go
func (m Model) loadContainersCmd() tea.Cmd {
    return func() tea.Msg {
        containers, err := m.client.ListContainers("")
        return LoadedMsg{View: ContainersView, Containers: containers, Err: err}
    }
}
```

**Init 并发加载：**
```go
func (m Model) Init() tea.Cmd {
    return tea.Batch(
        m.loadContainersCmd(),
        m.loadImagesCmd(),
        m.loadComposeCmd(),
        m.loadDeployCmd(),
    )
}
```

### 按键委派模式

```go
func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
    // 优先级 1：子面板打开 → 委派子面板
    if m.panel != PanelNone {
        return m.handlePanelKey(msg)
    }
    // 优先级 2：过滤输入模式
    if m.listFilterMode {
        return m.handleFilterKey(msg)
    }
    // 优先级 3：正常主列表按键
    switch msg.String() {
    case "tab": /* 切换视图 */
    case "up", "k": /* 上移游标 */
    case "/": /* 进入过滤模式 */
    // ...
    }
}
```

### 确认弹窗

```go
func (m Model) View() string {
    // 确认弹窗优先级最高（阻断其他交互）
    if m.pending != ActionNone {
        return views.RenderConfirm(string(m.pending), m.width)
    }
    // 子面板
    if m.panel != PanelNone {
        return m.renderPanel()
    }
    // 主布局
    return m.renderMain()
}

func (m Model) handleConfirm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
    // 只响应 Enter(确认执行) / Esc(取消)
}
```

### 子面板模型（自包含组件）

每个子面板是独立的状态机：

```go
// LogPanelModel
type LogPanelModel struct {
    lines      []string
    scrollPos  int
    SearchMode bool
    SearchTerm string
    searchHit  []int
    searchIdx  int
    followTail bool
    width, height int
}
func (lp *LogPanelModel) HandleKey(key string) tea.Cmd   // 返回 tea.Cmd，不返回 interface{}
func (lp *LogPanelModel) Render(containerName string) string
func (lp *LogPanelModel) SetLines(lines []string, reset bool)
func (lp *LogPanelModel) ApplyFilter()

// InspectPanelModel
type InspectPanelModel struct {
    detail    *dt.ContainerDetail
    activeTab int  // 0=Env, 1=Mounts, 2=Network, 3=Command
    scrollPos int
}
func (ip *InspectPanelModel) HandleKey(key string) tea.Cmd
func (ip *InspectPanelModel) Render() string

// StatsPanelModel — 实时刷新
type StatsPanelModel struct { ... }
func (sp *StatsPanelModel) HandleKey(key string) tea.Cmd
func (sp *StatsPanelModel) Render(containerName string) string
func (sp *StatsPanelModel) UpdateStats(entry dt.StatsEntry)

// ImageHistoryPanelModel
type ImageHistoryPanelModel struct {
    entries   []dt.ImageHistoryEntry
    scrollPos int
}
func (ih *ImageHistoryPanelModel) HandleKey(key string) tea.Cmd
func (ih *ImageHistoryPanelModel) Render() string
```

**关键约束：** `HandleKey` 返回 `tea.Cmd`（不是 `interface{}`），确保编译时类型检查。

### 日志面板 vim 键位

| 键 | 操作 |
| --- | --- |
| `↑` / `k` | 上滚 1 行 |
| `↓` / `j` | 下滚 1 行，滚到底自动进入 follow 模式 |
| `PgUp` / `Ctrl+u` | 上翻一页 |
| `PgDn` / `Ctrl+d` | 下翻一页 |
| `g` | 跳到顶部 |
| `G` | 跳到底部 |
| `f` | 切换 follow 模式 |
| `/` | 进入搜索模式 |
| `n` | 跳到下一个匹配 |
| `N` | 跳到上一个匹配 |
| `Esc` | 退出搜索 / 关闭面板 |
| `r` | 刷新日志 |

---

## lipgloss 样式系统

### 设计原则

使用 lipgloss 声明式样式替代原始 ANSI 转义码，解决颜色泄漏（忘记 reset）和换行错位问题。

### 颜色令牌（Tokyo Night 主题）

```go
// internal/tui/styles/theme.go
const (
    ColorFg        = "#c0caf5"
    ColorDim       = "#565f89"
    ColorCyan      = "#7dcfff"
    ColorGreen     = "#9ece6a"
    ColorRed       = "#f7768e"
    ColorYellow    = "#e0af68"
    ColorAccent    = "#7aa2f7"
    ColorBorder    = "#3b4261"
    ColorSelected  = "#364a82"
    ColorTabActive = "#7aa2f7"
    ColorTabText   = "#1a1b26"
    ColorHighlight = "#e0af68"
    ColorLogBg     = "#1a1b26"
    ColorBarFill   = "#7dcfff"
    ColorBarEmpty  = "#3b4261"
)
```

### 工厂函数

```go
// WidthStyle — 运行时创建固定宽度样式
func WidthStyle(w int) lipgloss.Style {
    return lipgloss.NewStyle().Width(w).MaxWidth(w)
}

// TruncateWithStyle — 超长文本截断并加省略号
func TruncateWithStyle(s string, maxWidth int, style lipgloss.Style) string {
    if lipgloss.Width(s) <= maxWidth {
        return style.Render(s)
    }
    // 逐字符削减直到满足宽度
    truncated := s
    for lipgloss.Width(truncated)+2 > maxWidth && len(truncated) > 0 {
        truncated = truncated[:len(truncated)-1]
    }
    return style.Render(truncated + "..")
}
```

### 列表渲染

```go
// 自适应列宽 → 用 WidthStyle 包裹每个字段
styles.WidthStyle(idW).Render(styles.ListDimText.Render(shortID(c.ID)))

// 选中行高亮
styles.ListItemSelected.Width(listW).Render(line)

// 容器状态着色
styles.StatusRunning.Render("▲ Running")
styles.StatusExited.Render("▼ Exited")
```

### 复杂布局使用 `lipgloss.JoinHorizontal`

```go
// Tab 栏拼接
lipgloss.JoinHorizontal(lipgloss.Left, tabParts...)

// 面板边框
lipgloss.NewStyle().
    BorderStyle(lipgloss.RoundedBorder()).
    BorderForeground(ColorBorder).
    Padding(0, 1)
```

---

## Scenario: TUI 布局强制约束

### 1. Scope / Trigger

Bubble Tea `View()` 输出如果短于终端高度，旧画面残留不会清除，导致多个视图叠加。必须用 `lipgloss.NewStyle().Height().MaxHeight()` 强制每段内容填满分配空间。

### 2. Signatures

```go
// renderMain — 使用 JoinVertical + Height/MaxHeight 强制约束
func (m Model) renderMain() string {
    return lipgloss.JoinVertical(lipgloss.Left,
        lipgloss.NewStyle().Height(headerH).MaxHeight(headerH).Render(header),
        lipgloss.NewStyle().Height(bh).MaxHeight(bh).Render(body),
        lipgloss.NewStyle().Height(1).MaxHeight(1).Render(footer),
        lipgloss.NewStyle().Height(1).MaxHeight(1).Render(status),
    )
}

// 全屏面板（log/exec/stats）必须填满终端高度
func (m Model) renderFullPanel(kind string) string {
    return lipgloss.NewStyle().
        Width(m.width).Height(m.height).MaxHeight(m.height).
        Render(content)
}
```

### 3. Contracts

**各段高度计算：**
```go
headerH = 6  // RenderHeader(1) + 2 blank + RenderTabs(1) + 2 blank
footerH = 1
statusH = 1
bodyH = m.height - headerH - footerH - statusH
```

**`Height(h)` 行为：** 内容不足 `h` 行时自动补空行，超过 `h` 行时截断。
**`MaxHeight(h)` 行为：** 硬上限，防止溢出。

### 4. Validation & Error Matrix

| 条件 | 行为 |
| --- | --- |
| `bodyH < 1` | 设为 1（最小高度保护） |
| 内容行数 < 分配高度 | `Height(h)` 补空行，旧内容被覆盖 |
| 内容行数 > 分配高度 | `MaxHeight(h)` 截断，不溢出 |
| `m.height == 0`（WindowSizeMsg 未到） | bodyH=1，等待 resize |

### 5. Good/Base/Bad Cases

**Good — 强制约束：**
```go
lipgloss.NewStyle().Height(bodyH).MaxHeight(bodyH).Render(body)
```

**Bad — 手动计算新行数拼接：**
```go
// 不靠谱：ANSI 码宽度、lipgloss styled 字符串的 \n 后缀都可能导致计数错误
header + body.String() + footer + status
```

### 6. Tests Required

- `TestModelInitAndView`: 所有 5 个 mode 渲染不为空

### 7. Wrong vs Correct

#### Wrong — 手动拼接，依赖隐含的换行数
```go
return header + "\n\n" + body + "\n" + footer + status
```

#### Correct — lipgloss JoinVertical + Height/MaxHeight
```go
return lipgloss.JoinVertical(lipgloss.Left,
    lipgloss.NewStyle().Height(6).MaxHeight(6).Render(header),
    lipgloss.NewStyle().Height(bh).MaxHeight(bh).Render(body),
    lipgloss.NewStyle().Height(1).MaxHeight(1).Render(footer),
    lipgloss.NewStyle().Height(1).MaxHeight(1).Render(status),
)
```

---

## 配置约定

### 使用 `encoding/json` 而非手动解析

```go
func Load(path string) Config {
    cfg := Config{}
    data, _ := os.ReadFile(path)
    json.Unmarshal(data, &cfg)
    return cfg
}

func Save(path string, cfg Config) error {
    data, _ := json.MarshalIndent(cfg, "", "  ")
    return os.WriteFile(path, data, 0644)
}
```

**增删方法：**
```go
func AddComposeDir(path, name, dirPath string) error
func RemoveComposeDir(path string, index int) error
func AddDeployTarget(path, name, container, htmlPath, backupDir string) error
func RemoveDeployTarget(path string, index int) error
```

### 配置结构（含 json tag）

```go
type Config struct {
    ComposeDirs   []ComposeDir   `json:"compose_dirs"`
    DeployTargets []DeployTarget `json:"deploy_targets"`
}
type ComposeDir struct {
    Name string `json:"name"`
    Path string `json:"path"`
}
type DeployTarget struct {
    Name      string `json:"name"`
    Container string `json:"container"`
    HtmlPath  string `json:"html_path"`
    BackupDir string `json:"backup_dir"`
}
```

- 默认路径 `~/.dtui/config.json`
- 首次加载时不存在则自动创建模板（`config.InitDefault()`）
- 模板指向 `~/.dtui/compose/` 和 `dtui-hello` 容器
- `-c/--config` flag 可覆盖
- 设置页支持 `a` 新增、`d` 删除条目

### Don't: 手写 JSON 解析器

```go
// 错误 — 不支持嵌套、不支持写回、UTF-8 多字节字符可能解析错误
func parseComposeDirs(content string) []ComposeDir { ... }
// 正确 — 用 encoding/json
json.Unmarshal(data, &cfg)
```

---

## 常见错误（v2 更新）

### Don't: 在 SDK 可用的场景使用 exec.Command

```go
// 错误 — 放弃结构化返回
out, _ := exec.Command("docker", "ps", "-a", "--format", "{{.ID}}|{{.Image}}").Output()
// 正确 — 用 SDK
containers, err := c.cli.ContainerList(ctx, container.ListOptions{All: true})
```

**例外：** compose up / docker exec 仍可用 `exec.Command`（SDK 不直接支持）。

#### 新增部署辅助函数

```go
// PathType 检测路径类型：folder / zip / invalid / unknown
func PathType(path string) string

// UnzipToDir 用 Go 标准库 archive/zip 解压到目标目录（不依赖外部 unzip）
func UnzipToDir(zipPath, destDir string) error

// CopyToContainer 将本地目录打包为 tar 并通过 SDK 复制到容器内
func (c *Client) CopyToContainer(containerID, dstPath, srcDir string) error
```

### Don't: 日志流忽略 8-byte header

```go
// 错误 — 日志包含二进制乱码
scanner := bufio.NewScanner(reader)
// 正确 — 使用 stdcopy 解复用后再扫描文本行
pipeReader, pipeWriter := io.Pipe()
go func() { _, err := stdcopy.StdCopy(pipeWriter, pipeWriter, reader); _ = pipeWriter.CloseWithError(err) }()
scanner = bufio.NewScanner(pipeReader)
```

### Don't: Stats CPU 计算不保存上一帧

```go
// 错误 — 每次都是绝对值
delta := float64(v.CPUStats.CPUUsage.TotalUsage)
// 正确 — 计算增量
delta := float64(v.CPUStats.CPUUsage.TotalUsage - prevCPU)
```

### Don't: 子面板 HandleKey 返回 interface{}

```go
// 错误 — 丢失编译时类型检查
func (lp *LogPanelModel) HandleKey(key string) interface{} { ... }
// 正确 — 返回 tea.Cmd
func (lp *LogPanelModel) HandleKey(key string) tea.Cmd { ... }
```

### Don't: 使用 lipgloss 时忘记 Width/MaxWidth 约束

```go
// 错误 — 超长文本导致换行错位
lipgloss.NewStyle().Render(veryLongString)
// 正确 — 用 WidthStyle 或 TruncateWithStyle
styles.WidthStyle(maxW).Render(veryLongString)
```

### Don't: 使用 `tea.WithMouseCellMotion()` 并期望文本选中

```go
// 错误 — 启用后终端所有鼠标事件被 TUI 拦截，无法选中文本复制
tea.NewProgram(model, tea.WithMouseCellMotion())
// 正确 — 去掉鼠标模式，键盘导航，终端原生支持文本选中
tea.NewProgram(model, tea.WithAltScreen())
```

**说明：** SGR 鼠标协议是全有或全无的——启用后终端不再处理任何鼠标事件。要支持文本选中，只能去掉鼠标模式，用纯键盘导航（j/k/↑↓//搜索/tab）。

### Don't: 全屏面板不填满终端高度

```go
// 错误 — 面板内容 10 行，终端 40 行，剩余 30 行暴露旧画面
return m.renderExecPanel()
// 正确 — 用 Height/MaxHeight 强制撑满
return lipgloss.NewStyle().Height(m.height).MaxHeight(m.height).Render(content)
```

```go
// 错误 — OnlineCPUs 可能为 0
numCPUs := float64(v.CPUStats.OnlineCPUs)
// 正确 — 加回退链
numCPUs := float64(v.CPUStats.OnlineCPUs)
if numCPUs == 0 { numCPUs = float64(len(v.CPUStats.CPUUsage.PercpuUsage)) }
if numCPUs == 0 { numCPUs = 1 }
```

### Don't: SaveImage 使用默认 10s 超时

```go
// 错误 — 大镜像保存（几百 MB）10s 不够，context deadline exceeded
ctx, cancel := c.withTimeout() // 10s
// 正确 — 使用长超时
ctx, cancel := c.withLongTimeout() // 5min
```

### Don't: Docker client 使用 context.Background()

```go
// 错误 — Close() 无法取消后台 goroutine（stats 流、日志流）
c.ctx = context.Background()
// 正确 — 用 WithCancel，Close() 调用 cancel()
ctx, cancel := context.WithCancel(context.Background())
c.ctx = ctx
c.cancel = cancel
```

### Don't: 在 tui 包下创建子包存放 Panel 实现

```go
// 错误 — 循环依赖
// tui/panels/ 包需要引用 tui 包的类型（LogLoadedMsg, InspectMsg 等）
// tui 包需要引用 tui/panels/ 包的类型（Panel 接口）
package panels

// 正确 — 所有 Panel 实现放在 tui 包内，文件名以 panels_ 前缀区分
// tui/panels_exec.go, tui/panels_log.go, tui/panels_stats.go...
package tui
```

**说明：** Go 不允许循环导入。Panel 实现需要引用 `tui` 包的消息类型和视图类型，PanelManager 需要引用 Panel 实现。放同一包内避免此问题。

### Don't: busy 状态在 closePanel 中未清理

```go
// 错误 — 进入 exec 面板后直接 ESC 退出，footer 显示"执行中..."
func (m *Model) closePanel() {
    m.panels.Close()
}

// 正确 — 关闭面板时重置 busy
func (m *Model) closePanel() {
    m.busy = false
    m.panels.Close()
}
```

### Don't: 新旧两套 Panel 路径并存

```go
// 错误 — Model 同时保留旧字段(m.panel, m.logPanel, m.execInput)和新 PanelManager
// handlePanelKey 用 switch m.panel 直接访问旧字段
// 但 openPanel 通过 PanelManager 创建面板实例
// 结果：按键写入 m.execInput，渲染读取 ExecPanelImpl.input — 状态分离
type Model struct {
    panel     PanelType        // 旧
    execInput string           // 旧
    panels    *PanelManager    // 新
}
```

**正确 — 完全收敛到 PanelManager：**
```go
type Model struct {
    panels *PanelManager  // 唯一面板入口
}
// handlePanelKey 纯委派
func (m Model) handlePanelKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
    if key := msg.String(); key == "esc" {
        m.closePanel()
        return m, tea.Batch(m.loadContainersCmd(), m.loadImagesCmd())
    }
    return m, m.panels.HandleKey(key)
}
```

### Don't: Exec 面板输入/渲染变量分离

```go
// 错误 — ExecPanelImpl 用 p.input 渲染，但 handlePanelKey 写入 m.execInput
// 两个变量各自独立，输入不显示
```

**正确 — ExecPanelImpl 自管理输入，通过 ExecLineMsg 路由业务：**
```go
func (p *ExecPanelImpl) HandleKey(key string) tea.Cmd {
    if key == "enter" && p.input != "" {
        input := p.input
        p.input = ""
        return func() tea.Msg { return ExecLineMsg{Input: input} }
    }
    // backspace / 普通字符直接操作 p.input
}

// Model.Update 根据 mode 路由 ExecLineMsg
case ExecLineMsg:
    if m.mode == DeployView { return m.deployZipCmd(msg.Input) }
    if m.mode == SettingsView { return m.execSettingsCmd(msg.Input) }
    return m, m.runExecLineCmd(msg.Input)
```

### Don't: PanelManager.Render 双重减高度

```go
// 错误 — renderPanelWithManager 已减 4（header+footer+status），PanelManager 又减 4
func (pm *PanelManager) Render() string {
    targetH := pm.height - 4  // pm.height 已经是 m.height - 4
    // 实际渲染高度 = m.height - 8，少 4 行
}

// 正确 — pm.height 已经是内容区高度，直接使用
func (pm *PanelManager) Render() string {
    targetH := pm.height
}
```

**原因：** `syncPanelSizes` 设置 `pm.height = m.height - 4`，`renderPanelWithManager` 的 content 容器也是 `m.height - 4`。PanelManager 不应再减。

### Don't: 面板 Help() 重复 base footer 文本

```go
// 错误 — renderPanelFooter 已有 "Esc 返回"，Help() 又返回 "Esc 返回"
func (p *HistoryPanelImpl) Help() string {
    return styles.HelpKey.Render("Esc") + " 返回"
}
// 结果：footer 显示 "Esc 返回 | Esc 返回"

// 正确 — 面板 Help() 只返回面板特有的快捷键
func (p *HistoryPanelImpl) Help() string {
    return ""  // 历史面板无额外快捷键
}
```

**规则：** `renderPanelFooter` 自动添加 "Esc 返回 | "，面板 `Help()` 只返回 `|` 右侧的特有按键。

### Don't: 确认弹窗不填充终端高度

```go
// 错误 — 确认弹窗只有几行，底层面板/列表内容透出
if m.pending != ActionNone {
    return views.RenderHeader() + "\n\n" + views.RenderConfirm(...)
}

// 正确 — 填充到终端高度
if m.pending != ActionNone {
    full := views.RenderHeader() + "\n\n" + views.RenderConfirm(...)
    lines := strings.Split(full, "\n")
    for len(lines) < m.height { lines = append(lines, "") }
    return strings.Join(lines[:m.height], "\n")
}
```

### Don't: currentItemCount / listStats 遗漏 ViewMode

```go
// 错误 — SettingsView 未处理，default 返回 0，游标不移动
func (m Model) currentItemCount() int {
    switch m.mode {
    case ContainersView: return len(m.containers)
    // ... 其他 mode
    default: return 0  // SettingsView 命中这里
    }
}

// 正确 — 所有 ViewMode 都有对应 case
func (m Model) currentItemCount() int {
    switch m.mode {
    case ContainersView: return len(m.containers)
    // ... 其他 mode
    case SettingsView: return m.settingsToView().TotalItems()
    default: return 0
    }
}
```

**规则：** 新增 ViewMode 时必须同步更新 `currentItemCount()`、`listStats()`、`renderBody()` 三个函数。

---

## 设计决策

### 决策：Docker SDK 替代 exec.Command

**上下文：** 重构前用 `exec.Command("docker", ...)` 调用 CLI，手动解析 `--format` 输出字符串。

**选项：**
1. 保持 exec.Command，手动解析
2. 使用 Docker SDK (`github.com/docker/docker/client`)

**决策：** 选择 Docker SDK。结构化返回消除解析错误，类型安全防止字段错位，容器日志/inspect/stats 等高级功能无需拼接命令行。

**例外保留：** `docker compose up` 和 `docker exec` 仍用 `exec.Command`，因为 Docker SDK 不支持 compose 且 exec 需要交互式终端。

### 决策：子面板自包含模型

**上下文：** v1 中日志面板的状态散落在 Model 的多个字段中（`detailLines`、`detailFilter`、`detailScrollPos`），与主列表状态耦合。

**决策：** 每个面板封装为自己的 `*PanelModel` 结构体，拥有独立的 `HandleKey` 和 `Render`。Model 通过 `m.panel` 枚举值决定委派哪个子面板。

### 决策：Panel 接口 + PanelManager 统一生命周期 (v3)

**上下文：** v2 子面板模型仍然把面板实例指针（`*LogPanelModel`、`*InspectPanelModel` 等）散落在 Model 中，新增面板需修改 Model 结构体、`openPanel`、`closePanel`、`cleanupPanel`、`syncPanelSizes` 等多处。面板特有状态（`execInput`、`statsCh`）污染 Model。

**决策：** 定义 `Panel` 接口（`Open/Close/Render/HandleKey/HandleMsg/SetSize/Help`），由 `PanelManager` 统一管理生命周期。Model 只需一个 `panels *PanelManager` 字段。

**收益：**
- 新增面板只需实现 `Panel` 接口 + 在 `openPanel` 的 switch 中添加一个 factory
- 面板状态完全自包含，不污染 Model
- `closePanel()` 从 N 行 switch 简化为一行 `m.panels.Close()`
- 面板间完全解耦，可独立测试

**Go 包结构决策：** 所有 Panel 实现放在 `tui` 包内（`panels_exec.go`、`panels_log.go` 等），不使用子包。使用子包会导致循环依赖（子包需要引用 `tui` 包的类型）。

### 决策：统一游标替代多 index 字段

**上下文：** v1 有 `containerIndex`、`imageIndex`、`composeIndex`、`deployIndex` 四个游标字段。

**决策：** 使用单一 `cursor int` + `scrollOffset int` 组合，通过 `mode` 区分当前操作的数据集。减少字段数量，降低越界风险。
