# uix: 通用 CLI/TUI 应用框架 + dtui Docker 管理工具

## 核心理念

仿照 opencode 的交互模式：一个始终可见的 CLI 输入栏（类似终端 prompt），输入 `/` 弹出命令面板切换模块，下方内容区渲染当前模块的 UI。框架本身零业务逻辑——它就是终端里的"前端后台管理骨架"，可以往上面粘任意模块。

## 为什么需要 uix

1. 当前 dtui 是一次性拼凑的 Docker TUI，代码耦合严重，改动即 bug
2. 项目未来会有多个 CLI 工具：Modbus 测试器、MQTT 调试器、Kafka 发送器、LLM 聊天终端等——每个都从零写 TUI 不可持续
3. Bubble Tea 官方提供了优秀的组件库（Bubbles、Lip Gloss），但缺少一个将组件拼成完整应用的框架层
4. uix 就是这一层：插件注册、命令路由、布局骨架、弹窗系统、主题——全是通用能力，不和任何业务绑定

## uix 框架需求

### F1: 应用骨架（FrameworkApp）

一个实现 `tea.Model` 的结构体，提供三段式布局：

```
┌──────────────────────────────────────────┐
│  dtui > /containers                      │  ← 命令行输入栏（始终可见）
├──────────────────────────────────────────┤
│                                          │
│  [当前模块的 View() 输出]                 │  ← 内容区（由当前 Plugin 渲染）
│                                          │
├──────────────────────────────────────────┤
│  /containers /images ...  |  Ctrl+C 退出 │  ← 状态/帮助栏
└──────────────────────────────────────────┘
```

**行为**:
- WindowSizeMsg 触发自适应布局，内容区填满剩余高度
- 全局按键（Ctrl+C 退出、Esc 返回）在框架层处理
- 模块级按键委托给当前 Plugin
- 弹窗/面板以 overlay 形式覆盖在内容区上方（lipgloss.Place 居中）

### F2: Plugin 接口

每个业务模块实现的标准契约。接口设计参考了 Bubble Tea 官方 examples 中的 model composition 模式——每个 Plugin 就是一个独立的 `tea.Model`，外加注册元数据。

```go
type Plugin interface {
    Name() string           // 命令名，如 "containers"
    Description() string    // 描述，显示在命令面板中
    Aliases() []string      // 别名，如 ["c", "cnt"]

    Init(ctx PluginContext) tea.Cmd
    Update(msg tea.Msg) (tea.Model, tea.Cmd)
    View() string
    SetSize(width, height int)

    // 声明本模块的快捷键，框架会自动渲染到帮助栏
    Bindings() []HelpBinding

    // 生命周期回调
    OnActivate() tea.Cmd
    OnDeactivate()
}
```

**参考来源**: Bubble Tea composable-views example，shelfctl 的 hub/browse/edit 多视图架构

### F3: PluginRegistry（模块注册中心）

- `Register(p Plugin)`: 注册模块，自动建立 name 和 aliases 的索引
- `Resolve(input string) Plugin`: 根据用户输入（完整名或别名）查找模块
- `Search(query string) []Plugin`: 模糊搜索，用于命令面板的实时过滤
- `List() []Plugin`: 按注册顺序列出所有模块

### F4: 命令输入栏（CmdBar）

封装 `bubbles/textinput`，行为参考 opencode 的 CLI prompt：

- 正常模式：显示 `prompt > `，接受自由文本（为未来的聊天/命令功能预留）
- 命令模式：输入以 `/` 开头时激活，自动弹出命令面板
- Enter: 命令模式下解析并执行命令（如 `/containers` 切换到容器模块），否则委托给当前 Plugin
- ↑↓: 浏览命令历史
- Tab: 自动补全命令名（循环匹配）
- Ctrl+C: 退出应用
- Esc: 清除输入 / 关闭命令面板

**参考来源**: Bubble Tea 官方 textinput example，bubbletea-commandpalette 的 open/close 控制模式

### F5: 命令面板（Palette）

模糊搜索 overlay，类似 VS Code Ctrl+P。输入 `/` 时出现在内容区上方。

- 实时模糊过滤已注册命令（搜索范围：Name + Description + Aliases）
- ↑↓ 选择，Enter 确认，Esc 关闭
- 搜索结果上限 N 条（无滚动偏移——输入更精确的查询来缩小范围）
- 使用 `lipgloss.Place` 居中渲染为带边框的浮层

**参考来源**: [bubbletea-commandpalette](https://github.com/blackwell-systems/bubbletea-commandpalette) 的设计模式

### F6: 弹窗系统（Modal）

通用弹窗组件，用于确认对话框、表单等阻断式交互。

- 弹窗出现时覆盖内容区，框架暂停向 Plugin 路由消息
- Esc 关闭弹窗，Enter 确认（可配置按钮）
- 使用 `lipgloss.Place` 居中渲染

### F7: 主题系统（Theme）

Tokyo Night 配色方案，提供颜色常量和常用样式工厂：

- 颜色令牌：Bg/Fg/Accent/Green/Red/Yellow/Dim/Border/Selected
- 工厂函数：`WidthStyle(w)`、`Truncate(s, maxW)`、`Border(title)`

**参考来源**: Lip Gloss 官方 style 文档中的 Border/Width/MaxWidth 模式

## dtui Docker 管理工具需求

基于 uix 框架构建 `cli/dtui/`，提供 5 个业务模块。

### D1: 容器模块（/containers, /c）

- 表格列表：Name、Image、Status、Ports、Created
- 右侧详情面板：选中容器的完整信息（ID、环境变量、挂载、网络）
- 操作：启动(s)、停止(s)、重启(R)、删除(x)、查看日志(l)、Stats 监控(p)、Inspect(i)、Exec(e)
- 使用 `bubbles/table` + `bubbles/viewport` 构建双栏布局（参考 Bubble Tea 官方 table example）

### D2: 镜像模块（/images, /i）

- 表格列表：Repository、Tag、ID、Size、Created
- 操作：删除(x)、保存为 tar、清理悬空镜像、查看历史
- 保存镜像操作使用 filepicker 选择目标目录

### D3: 编排模块（/compose, /co）

- 按项目分组的服务列表，显示运行状态
- 操作：up(s)、down(d)、重启、编辑 compose 文件
- up/down 前弹出确认 Modal，显示将要执行的命令
- 不支持 docker compose 的 SDK 操作仍走 exec.Command

### D4: 部署模块（/deploy, /d）

- 部署目标列表：Name、Container、HTML 路径、备份目录
- 部署流程：选择本地文件 → 确认目标 → 备份 → 解压 → 复制到容器
- 通过 Modal 显示部署进度（备份中 → 解压中 → 部署中 → 完成）

### D5: 配置模块（/config, /cfg）

- 编排目录的增删改
- 部署目标的增删改
- 表单编辑（使用 Modal + textinput）和外部编辑器（vim/nano）双模式
- 编辑后自动刷新配置

## 代码质量

- `cli/uix/` 零依赖 Docker SDK，独立可编译
- `cli/dtui/` 复用现有 `cli/dtui/internal/docker/` 包（Docker SDK 封装层不重写）
- `cli/dtui/` 复用现有 `cli/dtui/internal/config/` 包（配置管理不重写）
- 所有新代码通过 `go build` 和 `go vet`
- 遵循 Bubble Tea 官方 model composition 模式，不引入自定义抽象层（不用 Context 接口、不用 ContextManager）

## Acceptance Criteria

- [ ] `cli/uix` 包独立编译，`go vet` 无警告
- [ ] 启动 dtui 后显示 CLI 输入栏 + 容器列表
- [ ] 输入 `/` → 弹出命令面板，显示全部 5 个模块
- [ ] 输入 `/i` + Tab → 自动补全为 `/images`，Enter → 切换到镜像模块
- [ ] 容器模块：表格显示正常、↑↓ 移动选择、详情面板同步更新
- [ ] 镜像模块：操作（删除/保存）正常
- [ ] 编排模块：服务状态显示正常，up/down 前有确认 Modal
- [ ] 部署模块：部署流程有进度 Modal
- [ ] 配置模块：表单编辑和外部编辑器均可用
- [ ] 80x24 终端下正常显示，120x40 终端下双栏布局正常
- [ ] 旧 `cli/dtui/internal/tui/` 目录可整体删除（被新代码替代）

## Out of Scope

- Modbus/MQTT/Kafka/LLM 模块（仅预留 Plugin 接口扩展能力）
- Kubernetes 支持
- 远程 Docker 连接
- 鼠标交互
- 国际化
- 动画过渡（harmonica）
