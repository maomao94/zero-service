# dtui 全面架构重构

## Goal

基于 Lazydocker/LazyGit/k9s 的成熟架构模式，用 BubbleTea + Bubbles + LipGloss 生态全面重构 dtui，最大化复用成熟组件，建立可扩展的现代化 TUI 架构。

dtui 区别于 Lazydocker 的核心价值：Compose 服务选择性启动 + 前端 HTML 部署。

## Confirmed Facts

- 现有 dtui 约 3500 行代码，5 个视图 + 7 个 Panel
- 技术栈：Go + Cobra + BubbleTea + LipGloss + Bubbles + Docker SDK
- 保留：`internal/docker/*`、`internal/config/*`、`internal/tui/styles/*`、`internal/logger/*`、`internal/cli/root.go`、`main.go`
- 完全重构：`internal/tui/*` 下除 `styles/` 外的所有业务代码

## Requirements

### R1: 架构抽象层

- **Keybinding Registry**: 声明式按键绑定，global/context-specific keys，统一快捷键展示
- **Context/Page 抽象**: 每个视图是独立 Context，自有 Model/Update/View，ContextManager 管理切换
- **Action 分发系统**: 统一操作执行管道，替代 switch-case
- **Event Bus**: 组件间松耦合通信

### R2: 全面采用 Bubbles 生态

| 组件 | 用途 |
|------|------|
| `bubbles/table` | 容器/镜像/服务列表，内置排序、选中行、列宽 |
| `bubbles/viewport` | 日志、详情、Stats 面板滚动 |
| `bubbles/textinput` | 搜索过滤、Exec 输入、表单输入 |
| `bubbles/help` | 统一快捷键提示 |
| `bubbles/spinner` | 加载状态指示 |
| `bubbles/filepicker` | 部署路径选择、镜像保存路径选择 |

### R3: 增强组件

- `bubblezone`: 鼠标点击（列表选中、Tab 切换、按钮）
- `lipgloss`: 布局和样式（已有，保持）

### R4: Lazydocker 双栏布局

- 左侧：列表（bubbles/table），支持搜索过滤
- 右侧：实时详情面板，随选中项自动切换
- 右侧支持 `enter` 展开全屏
- Tab 切换 5 个视图

### R5: 功能完整性（5 视图 + 7 Panel）

- 容器：列表/启停/重启/日志/Shell/详情/Stats
- 镜像：列表/保存/删除/标签/清理/历史层
- Compose：服务列表/选择性启动/初始化/编辑
- 前端发布：文件夹+zip 双模式/确认/备份
- 设置：编排目录+发布目标+发布包 CRUD
- 操作历史：全类型记录

### R6: UX 增强

- Lazydocker 风格双栏布局
- bubbles/help 统一快捷键展示
- bubbles/filepicker 可视化文件选择
- bubblezone 鼠标支持
- Tokyo Night 主题

## Acceptance Criteria

- [ ] 5 个主视图功能完整，行为与重构前一致
- [ ] 7 个 Panel 功能完整
- [ ] 所有快捷键正常工作
- [ ] 鼠标操作正常
- [ ] `go build ./cli/dtui/...` 通过
- [ ] `go vet ./cli/dtui/...` 无问题
- [ ] `go test ./cli/dtui/...` 通过
- [ ] 新增架构层测试（keybinding registry、context manager）
- [ ] 单文件不超过 400 行
- [ ] 80x24 终端不崩溃

## Constraints

- 保留 `internal/docker/*`、`internal/config/*`、`internal/tui/styles/*`、`internal/logger/*`、`internal/cli/root.go`、`main.go`
- 重构范围：`internal/tui/*` 下除 `styles/` 外的所有文件
- Go 版本：与项目一致
- **注释要求**：关键结构体、接口、方法必须有中文注释说明用途和设计意图，方便学习
- **不兼容旧代码**：当新项目写，不保留向后兼容，旧文件可直接删除

## Out of Scope

- 远程 Docker/SSH、K8s/Swarm、国际化、插件系统、网络/Volume 管理
- `harmonica` 动画、`bubbles/textarea`

## Task Structure

```
dtui-refactor-v2 (parent) — 架构共享层
├── Container View + panels (logs/stats/inspect/exec)
├── Image View + panels (history/save/tag/prune)
├── Compose View
├── Deploy View
└── Settings View + History Panel
```

父任务先完成共享层，子任务可并行实现。
