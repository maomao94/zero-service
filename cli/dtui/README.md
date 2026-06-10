# dtui

`dtui` 是一个 Go + Cobra + Bubble Tea 终端 UI 工具，用于管理本机 Docker 容器和镜像。
进入 TUI 后用键盘选择目标并执行操作，减少手工输入 `docker` 命令的负担。

## 快速开始

```bash
# 直接进入 TUI
go run ./cli/dtui

# 查看帮助
go run ./cli/dtui --help

# 编译
cd cli/dtui && ./build.sh
```

## 快捷键

| 键 | 作用 |
|---|---|
| `tab` | 切换容器 / 镜像视图 |
| `↑` `↓` 或 `j` `k` | 上下移动选择 |
| `/` | 搜索过滤 |
| `r` | 刷新当前视图 |
| `s` | 启动/停止容器（容器视图）/ 保存镜像（镜像视图） |
| `R` | 重启容器 |
| `l` | 查看容器日志（按键后需 Enter 确认） |
| `e` | 进入容器 shell（按键后需 Enter 确认） |
| `p` | 清理悬空镜像（按键后需 Enter 确认） |
| `q` 或 `ctrl+c` | 退出 |

> 终端默认支持鼠标选中文本复制（容器ID、镜像名等），无需额外操作。

所有操作（s/R/l/e/p）按下后都需要按 **Enter 确认** 才会执行，按 **Esc 取消**。防止误操作。

## 界面颜色

- 绿色 ▶ 标记当前选中行
- 容器状态：绿色 = 运行中，红色 = 已退出，黄色 = 其他
- 标题栏、标签栏、快捷键栏均有颜色区分

## 目录结构

```text
cli/dtui/
  main.go              # Go 入口: 执行 Cobra root command
  build.sh             # 编译脚本
  internal/
    cli/
      root.go          # Cobra root command: 启动 TUI 或打印 help
    docker/
      runner.go        # 调用本机 docker 命令 (exec.Command 参数数组)
      container.go     # 容器列表查询解析
      image.go         # 镜像列表查询解析
    tui/
      model.go         # Bubble Tea TUI: Model/Init/Update/View/按键处理
```

## Cobra 学习要点

### Root command (`internal/cli/root.go`)

```go
cmd := &cobra.Command{
    Use:           "dtui",      // 命令名
    Short:         "Docker 宿主机管理 TUI",
    Long:          "...",          // 完整帮助
    Version:       "0.1.0",        // --version 支持
    Args:          cobra.NoArgs,   // 位置参数校验: 不接受额外参数
    SilenceUsage:  true,           // 错误时不重复打印 help
    SilenceErrors: true,           // 错误由 main.go 统一处理
    RunE: func(cmd *cobra.Command, args []string) error {
        return tui.Run()           // 启动 Bubble Tea TUI
    },
}
```

关键概念：

- **Use**: Cobra 根据 Use 生成 usage 行，并自动处理子命令匹配。
- **Long**: 显示在 `--help` 的完整帮助中。
- **Version**: Cobra 自动添加 `--version` flag。
- **RunE**: 返回 error 的执行函数；Cobra 会在 RunE 返回 error 时打印错误。
- **SilenceUsage**: 设为 true 避免 RunE 返回 error 后 Cobra 又打印整段 help。
- **SilenceErrors**: 设为 true 让 `main.go` 用 `os.Exit(1)` 统一退出。

### 注册子命令

如果没有子命令，像 dtui 这样只需 root command 即可。有子命令时：

```go
cmd.AddCommand(NewPsCommand())
```

Cobra 自动根据子命令的 Use 字段做路由匹配。

## Bubble Tea 学习要点

### Model-Update-View 架构

`internal/tui/model.go` 实现了 Bubble Tea 的核心三方法：

```go
type Model struct {
    runner     dtui.Runner
    mode       viewMode       // 容器/镜像视图
    containers []dtui.Container
    images     []dtui.Image
    // ...
}
```

**Init()**: 启动时自动调用一次，通常返回 `tea.Batch(...)` 启动异步加载。

```go
func (m Model) Init() tea.Cmd {
    return tea.Batch(m.loadContainersCmd(), m.loadImagesCmd())
}
```

**Update(msg)**: 每次按键/异步操作完成/窗口大小变化时都会触发。根据 msg 类型分发到不同的处理逻辑。

```go
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        // 处理按键
    case loadedMsg:
        // 处理 Docker 数据加载结果
    }
}
```

**View()**: 渲染终端界面字符串，每次 Update 后自动调用。

```go
func (m Model) View() string {
    // 返回当前界面的字符串表示
}
```

### tea.Cmd（异步操作）

Docker 查询被包装为 `tea.Cmd`：

```go
func (m Model) loadContainersCmd() tea.Cmd {
    return func() tea.Msg {
        containers, err := m.runner.ListContainers("")
        return loadedMsg{view: containersView, containers: containers, err: err}
    }
}
```

Bubble Tea runtime 会在 goroutine 中执行这个函数，完成后把结果以 `tea.Msg` 的形式送回 `Update`。

### Cobra 如何启动 TUI

在 `RunE` 中调用 `tea.NewProgram(model).Run()`：

```go
RunE: func(cmd *cobra.Command, args []string) error {
    return tui.Run()
}
```

`tui.Run()` 创建 `tea.NewProgram` 并调用 `.Run()`；Cobra 负责命令行参数解析，Bubble Tea 负责终端交互。

### Docker 调用边界

`internal/docker/runner.go` 使用 `exec.Command("docker", args...)` 参数数组调用，不拼接 shell 字符串。

## 如何新增功能

1. 在 `internal/tui/model.go` 的 `handleKey` 中添加新按键分支。
2. 实现对应的 `tea.Cmd` 包装 Docker 操作。
3. 确保 Docker 操作完成后返回 `actionMsg`，Update 会收到并刷新列表。

## 限制

- 只管理本机 Docker，不包含 Kubernetes、SSH 或远程 Docker。
- Docker daemon 必须正在运行才能执行实际操作。
- `help` 和 `build` 不需要 Docker daemon。
