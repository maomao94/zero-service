# uix — Chat-like CLI/TUI Shell

> **⚠️ 实验性代码**。`cli/uix/`、`cli/dtui/` 及其子模块由 AI 自动生成，未经人工审查，存在状态机边界问题、测试盲区和架构缺陷。**生产环境不可用**。非必要不要使用。正式使用前必须逐文件 human review 并重写关键逻辑。

## 1. Scope / Trigger

- `cli/uix/**` — Chat-like glue shell，owns prompt / command palette / timeline / modal overlays / status bar / module entry-exit
- `cli/dtui/**` — uix host，目前是测试验证载体
- 所有实现 `uix.Module` 的 Bubble Tea module

Business CLIs 是 host/module，不得拥有全局 prompt 路由。

## 2. Core Signatures

```go
// Shell construction
app := uix.NewShell("dtui > ")
app.RegisterModule(test.New())
app.RegisterCommand(uix.Command{Name: "doctor", Run: func(app *uix.Shell) tea.Cmd { ... }})
app.Run()

// Module contract
type Module interface {
    Name(), Description() string
    Aliases() []string
    Init() tea.Cmd
    Update(msg tea.Msg) (tea.Model, tea.Cmd)
    View() string
    SetSize(width, height int)
    Bindings() []HelpBinding
    IsRoot() bool
}

// Command contract
type Command struct {
    Name, Description string
    Aliases []string
    Run     func(*Shell) tea.Cmd
}

// Runner contract
type Runner interface { Run(input string, history []Message) tea.Cmd }
```

Shell messages: `ShowModalMsg`, `ConfirmMsg`, `FileSelectedMsg`, `AppendMessageMsg`, `StatusMsg`（`cli/uix/app.go`）。

Message roles: `RoleUser`, `RoleAssistant`, `RoleSystem`, `RoleTool`, `RoleModule`。

## 3. Layout & Prompt Mode

```
┌──────────────────────────────┐
│ Timeline or active Module    │
├──────────────────────────────┤
│ Dropdown / FilePicker        │  inline overlay
├──────────────────────────────┤
│ StatusBar                    │  mode + help
├──────────────────────────────┤
│ CmdBar prompt                │  always focused
└──────────────────────────────┘
```

| Prefix | Mode | Behavior |
|--------|------|----------|
| none | chat | `enter` → `RoleUser` + `Runner.Run` |
| `/` | command | palette → 选择或执行命令 |
| `@` | reference | 占位，暂无可信 provider |
| `#` | resource | file picker |
| `!` | shell | **默认禁用**，仅追加 disabled-shell 消息 |

Module active 时正常按键路由到 module，仅 `/`、`@`、`#` 走 shell 路由。

## 4. Contracts

- `IsRoot() == true` + `esc` → shell exit module + timeline event
- `IsRoot() == false` + `esc` → forward to module
- Modal full-screen overlay：`enter`/`esc`/`left`/`right`/`h`/`l` 由 modal 处理，module 不接收
- Module 返回非 `Module` 的 model → 保持前一个 active module
- `status/help` 在 `routeToActive`、`handleEscape`（non-root）、`EnterModule` 后自动刷新
- Module subview 应使用 `IsRoot() == false`，通过 `Bindings()` 返回当前模式帮助键
- `withTime` 字段控制控制命令的 typeId（NA=不带时标 / TA=带时标），不通过算术计算 typeId（某些区间有跳跃）

## 5. Design Decisions

| # | Decision | Rationale |
|---|----------|-----------|
| D1 | Chat-like shell 先于业务 module | `uix` 独立验证后方可迁移 Docker 等模块 |
| D2 | Prompt prefix 是 shell 级概念 | Module 不得重复实现 `/`、`@`、`#`、`!` |
| D3 | `!` 默认禁用 | shell 执行需另行设计权限/流控/审计 |
| D4 | Inline overlay 优先于全屏 | Dropdown/file picker 内联保留上下文 |
| D5 | 默认不启用 mouse capture | 保持终端原生选择功能 |
| D6 | 业务 client 延迟初始化 | `ensureClient()` 模式，启动不依赖外部 daemon |
| D7 | Subview 用 `IsRoot()` toggle | 非 root 时 `esc` 先退 subview 再退 module |

## 6. Bubble Tea Gotchas

- `textinput.New()` 默认 unfocused，需立即 `Focus()`
- `filepicker.Init()` 必调，否则不加载目录
- `bubbles/table` columns 必须先于 `SetRows()`
- `SetSize()` 必须 `guard width/height <= 0`
- 可见文本截断用 `theme.Truncate`，禁止 `s[:n]`
- ntcharts 组件在 `SetData()`/`SetSize()`/style 变更后必须 `Draw()`
- ntcharts 用 v1 path `github.com/NimbleMarkets/ntcharts`，v2 不兼容

## 7. Verification

```bash
go test ./cli/uix/... ./cli/dtui/...
go build -o /dev/null ./cli/dtui
go vet ./cli/uix/... ./cli/dtui/...
```

- `CmdBar.Prefix()` 正确识别 `/`、`@`、`#`、`!`、自由文本
- `!` 输入不执行 shell 命令
- `dtui` 可在无 Docker 环境编译启动
