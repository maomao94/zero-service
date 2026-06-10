# Technical Design

## 架构概览

将布局计算从业务代码中剥离，形成独立的 Layout Engine。同时修复设置、编排、发布模块的功能问题。

## D1: 布局引擎

### 目标

消除 `model.go` 中的 `listWidth()`、`bodyHeight()`、`availableListHeight()` 方法，以及各视图函数中的列宽计算。

### Metrics 结构体

```go
// cli/dtui/internal/tui/layout/layout.go
package layout

type Mode int

const (
    ModeDual   Mode = iota // 双栏：列表 + 详情
    ModeSingle             // 单栏：仅列表
    ModePanel              // 面板模式：全屏面板
)

type Metrics struct {
    // 终端尺寸
    Width, Height int

    // 垂直分区
    HeaderH  int // 标题栏 + Tab 栏 + 搜索栏
    FooterH  int // 快捷键栏
    StatusH  int // 状态栏
    BodyH    int // 内容区高度

    // 水平分区（双栏模式）
    Mode     Mode
    ListW    int // 列表宽度
    DetailW  int // 详情宽度（单栏模式为 0）

    // 面板模式
    PanelW   int // 面板宽度
    PanelH   int // 面板高度

    // 内容区内边距
    ContentPad int // 内容区边框/内边距（通常为 2 或 4）
}
```

### Calculate 函数

```go
func Calculate(width, height int, filterMode bool) Metrics {
    m := Metrics{Width: width, Height: height}

    // 垂直分区
    m.HeaderH = 6 // RenderHeader(1) + 2 blank + RenderTabs(1) + 2 blank
    if filterMode {
        m.HeaderH++
    }
    m.FooterH = 1
    m.StatusH = 1
    m.BodyH = height - m.HeaderH - m.FooterH - m.StatusH
    if m.BodyH < 1 {
        m.BodyH = 1
    }

    // 水平分区
    if width >= 100 {
        m.Mode = ModeDual
        m.ListW = width * 40 / 100
        m.DetailW = width - m.ListW
    } else {
        m.Mode = ModeSingle
        m.ListW = width
        m.DetailW = 0
    }

    // 面板模式
    m.PanelW = width
    m.PanelH = m.BodyH

    // 内容区内边距
    m.ContentPad = 2

    return m
}
```

### 列宽计算辅助函数

```go
// ProportionalColumns 按比例分配列宽，每列有最小宽度保护
func ProportionalColumns(totalWidth int, ratios []int, minWidths []int) []int {
    n := len(ratios)
    widths := make([]int, n)

    // 计算总比例
    totalRatio := 0
    for _, r := range ratios {
        totalRatio += r
    }

    // 按比例分配
    allocated := 0
    for i := 0; i < n-1; i++ {
        w := totalWidth * ratios[i] / totalRatio
        if minWidths != nil && i < len(minWidths) && w < minWidths[i] {
            w = minWidths[i]
        }
        widths[i] = w
        allocated += w
    }
    widths[n-1] = totalWidth - allocated // 最后一列填满剩余

    // 最小宽度保护
    for i := 0; i < n; i++ {
        if widths[i] < 1 {
            widths[i] = 1
        }
    }

    return widths
}
```

### 重构 model.go

删除以下方法：
- `listWidth()` → 使用 `m.metrics.ListW`
- `bodyHeight()` → 使用 `m.metrics.BodyH`
- `availableListHeight()` → 使用 `m.metrics.BodyH - 4`（或在 Metrics 中添加 `ListH`）

Model 结构体添加：
```go
type Model struct {
    // ... existing fields ...
    metrics layout.Metrics
}
```

Update 中处理 WindowSizeMsg 时调用：
```go
case tea.WindowSizeMsg:
    m.width = msg.Width
    m.height = msg.Height
    m.metrics = layout.Calculate(msg.Width, msg.Height, m.listFilterMode)
    m.syncPanelSizes()
```

### 重构视图函数

视图函数签名改为接收 Metrics：
```go
// 之前
func RenderContainers(containers []dt.Container, selected, offset, visible int, listW int) string

// 之后
func RenderContainers(containers []dt.Container, selected, offset int, m layout.Metrics) string
```

视图函数内部使用 `m.ContentPad` 而非硬编码的 `- 4`、`- 6`。

## D2: 设置模块修复

### 当前行为

`openConfigCmd()` (commands.go:445) 直接打开外部编辑器。

### 改进方案

1. 'c' 键弹出选择面板（复用现有 `FormPanel` 或新建 `SelectPanel`）
2. 选项：
   - **表单编辑**: 打开 `FormPanel`，预填当前选中项的值
   - **JSON 编辑器**: 保持现有行为
3. 表单编辑完成后调用 `config.UpdateComposeDir()` 或 `config.UpdateDeployTarget()`
4. JSON 编辑器完成后自动 `config.Load()` 刷新

### 数据流

```
用户按 c
  → 弹出选择面板
  → 选择"表单编辑" → FormPanel(onSubmit=saveConfig)
  → 选择"JSON 编辑器" → tea.ExecProcess(editor)
  → 完成 → m.loadComposeCmd() + m.loadDeployCmd()
```

### 新增 config 函数

```go
// cli/dtui/internal/config/config.go
func UpdateComposeDir(cfgPath string, idx int, name, path string) error
func UpdateDeployTarget(cfgPath string, idx int, name, container, htmlPath, backupDir string) error
```

## D3: 编排模块改进

### 服务状态显示

在 `loadComposeCmd()` 中，对每个服务调用 `docker compose ps` 获取状态：

```go
func (m Model) loadComposeCmd() tea.Cmd {
    return func() tea.Msg {
        // ... existing code ...
        for _, svc := range allServices {
            if !svc.Missing {
                status, _ := dt.ComposeStatus(svc.Path, svc.Name)
                svc.Status = status
            }
        }
        // ...
    }
}
```

### 确认对话框

`s`/`u` 键不再直接执行，而是弹出确认面板：

```go
case "s", "u":
    if m.mode == ComposeView {
        return m.requireConfirm(ActionCompose)
    }
```

确认面板显示：
```
即将执行:
docker compose -f /path/to/docker-compose.yml up -d service-name

确认执行？(Enter/Esc)
```

### compose down

新增 `d` 键支持：

```go
case "d":
    if m.mode == ComposeView {
        return m.requireConfirm(ActionComposeDown)
    }
```

## D4: 发布模块改进

### 部署确认面板

`d` 键弹出确认面板，显示：
```
目标容器: nginx
HTML 路径: /usr/share/nginx/html
备份目录: /tmp/deploy-backups/nginx

确认部署？(Enter/Esc)
```

### 部署进度

在 `runDeployFlowCmd()` 中，发送进度消息：

```go
type DeployProgressMsg struct {
    Stage   string // "备份中", "解压中", "部署中"
    Percent int
}
```

Model 处理进度消息，更新状态栏。

### filepicker 集成

新增 `PanelFilePicker`，复用 `charmbracelet/bubbles/filepicker`：

```go
case PanelFilePicker:
    factory = func() Panel {
        fp := filepicker.New()
        fp.SetHeight(m.metrics.BodyH - 4)
        return &FilePickerPanel{picker: fp}
    }
```

## D5: 视图渲染优化

### 列表/详情分离

容器和镜像视图改为双栏渲染：

```
┌─────────────────────────────────────────────────────────┐
│ 列表区 (40%)                    │ 详情区 (60%)          │
│ ─────────────────────────────── │ ───────────────────── │
│ ▶ 1. container-name            │ ID: abc123            │
│   2. another-container         │ Image: nginx:latest   │
│   3. ...                       │ Status: Up 2 hours    │
│                                │ Ports: 80->8080       │
│                                │ ...                   │
└─────────────────────────────────────────────────────────┘
```

### 列宽最小值保护

在 `ProportionalColumns()` 中，每列最小宽度为 6 字符。

### 小终端适配

`Calculate()` 中：
```go
if width < 80 {
    m.Mode = ModeSingle
    m.ListW = width
    m.DetailW = 0
}
```

单栏模式下，详情区折叠到列表下方（当前行为）。

## Trade-offs

| 决策 | 理由 | 风险 |
|------|------|------|
| 保持外部编辑器选项 | 高级用户需要直接编辑 JSON | 无 |
| 列表/详情双栏仅在 >=80 列时启用 | 小终端空间不足 | 小终端体验略差 |
| 部署进度使用状态栏 | 避免复杂的进度条组件 | 进度不够直观 |
| filepicker 作为可选 | 增加依赖 | 包体积增大 |
