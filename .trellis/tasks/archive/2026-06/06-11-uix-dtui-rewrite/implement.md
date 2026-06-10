# Implementation Plan: uix + dtui

## 总体策略

自底向上构建：先 uix 骨架（零外部依赖），再逐个添加 Docker 模块。每个 Phase 独立可编译、可运行、可验证。

Phase 之间是严格依赖关系：Phase N+1 依赖 Phase N 的产物。同一 Phase 内步骤可部分并行。

---

## Phase 1: uix 最小可运行骨架

**目标**: `cli/uix/` 作为一个独立 Go 包存在，编译通过，能启动一个显示 "Hello uix" 的空白 TUI。

### 1.1 创建目录结构和 go.mod 集成
- 创建 `cli/uix/` 目录
- 验证 `cli/uix/` 下的代码属于 module `zero-service`（不需要独立 go.mod）
- 文件: 无需新建文件，准备目录即可

### 1.2 创建 theme 包
- 文件: `cli/uix/theme/theme.go` — Tokyo Night 颜色常量
- 文件: `cli/uix/theme/styles.go` — `WidthStyle`、`Truncate`、`Border` 工厂函数
- 验证: `go vet ./cli/uix/theme/...`

### 1.3 创建 Plugin 接口和 Registry
- 文件: `cli/uix/plugin.go` — `Plugin` 接口、`HelpBinding`、`PluginContext`
- 文件: `cli/uix/registry.go` — `PluginRegistry`（Register/Resolve/Search/List）
- 验证: `go vet ./cli/uix/...`

### 1.4 创建 FrameworkApp
- 文件: `cli/uix/app.go` — `FrameworkApp` 结构体、`NewApp`、`Init`、`Update`、`View`、`Run`
- 实现三段式布局: CmdBar(1行) + Body(剩余) + StatusBar(1行)
- 实现 WindowSizeMsg 处理
- 实现全局按键: Ctrl+C 退出
- 验证: 用 `_ "zero-service/cli/uix"` 空白导入编译通过

### 1.5 创建 CmdBar 组件
- 文件: `cli/uix/components/cmdbar.go`
- 封装 `bubbles/textinput`，prompt 可配置
- 支持: Enter(Ctrl+J)/Ctrl+C 识别
- 命令历史暂不实现（Phase 3）
- 验证: 组件可导入编译

### 1.6 创建 StatusBar 组件
- 文件: `cli/uix/components/statusbar.go`
- 左侧显示模块名，右侧显示帮助文本
- 验证: 组件可导入编译

### 1.7 Phase 1 验证
```bash
go build ./cli/uix/...
go vet ./cli/uix/...
# 编写一个临时 main.go 验证 TUI 可启动:
# go run ./cli/uix/_example/
```

---

## Phase 2: uix 命令系统

**目标**: `/` 命令面板可用，能切换不同 Plugin。

### 2.1 创建 Palette 组件
- 文件: `cli/uix/components/palette.go`
- 基于 `bubbles/textinput` 的模糊搜索 overlay
- `lipgloss.Place` 居中渲染
- 参考: bubbletea-commandpalette 的 open/close 控制模式
- ↑↓ 选择，Enter 确认，Esc 关闭
- 验证: 手动测试命令面板打开/关闭/选择

### 2.2 CmdBar 命令模式
- 修改: `cli/uix/components/cmdbar.go`
- 首字符 `/` 时激活命令模式
- Tab 触发自动补全（基于注册的命令名）
- Enter 时返回命令字符串
- 验证: 输入 `/` → Palette 弹出 → 选择 → 切换 Plugin

### 2.3 FrameworkApp 集成命令系统
- 修改: `cli/uix/app.go`
- `handleKey`: `/` → 激活命令模式/打开 Palette
- Palette 选择 → `registry.Resolve()` → 切换 active Plugin
- Plugin 切换: 调用旧 Plugin 的 `OnDeactivate()`，新 Plugin 的 `OnActivate()`
- 验证: 注册 2 个 dummy Plugin，用 `/` 切换

### 2.4 命令历史
- 修改: `cli/uix/components/cmdbar.go`
- ↑↓ 浏览历史命令（已执行过的命令）
- 验证: 输入 `/images` Enter，再输入 `/` ↑ 恢复 `/images`

### 2.5 Phase 2 验证
```bash
go build ./cli/uix/...
go vet ./cli/uix/...
# 运行空白应用，测试:
# - / → palette 打开，显示注册的 plugin 列表
# - 上下选择 + Enter → 切换到对应 plugin
# - ↑↓ 命令历史正常
```

---

## Phase 3: uix 弹窗系统

**目标**: Modal 组件可用，能在 Plugin 中弹出确认框。

### 3.1 创建 Modal 组件
- 文件: `cli/uix/components/modal.go`
- 标题 + 消息 + 按钮（可配置 1-N 个）
- `lipgloss.Place` 居中
- Enter 确认默认按钮，←→ 切换按钮，Esc 关闭
- 验证: 手动测试 Modal 打开/交互/关闭

### 3.2 NavStack
- 文件: `cli/uix/navigation.go`
- 泛型栈：Push/Pop/Current/Depth
- 验证: 编译通过

### 3.3 FrameworkApp 集成 Modal
- 修改: `cli/uix/app.go`
- Modal 打开时，消息不路由到 Plugin（阻断交互）
- Esc 关闭 Modal
- Plugin 可通过返回值请求打开 Modal
- 验证: Plugin 触发 Modal → 确认/取消 → 回调执行

### 3.4 Phase 3 验证
```bash
go build ./cli/uix/...
go vet ./cli/uix/...
```

---

## Phase 4: dtui 入口 + 容器模块

**目标**: dtui 可启动，容器模块可用。

### 4.1 dtui 入口
- 文件: `cli/dtui/main.go`（重写）
- Docker client 初始化
- 创建 FrameworkApp，注册所有 Plugin
- 默认激活 containers plugin
- 验证: `go run ./cli/dtui` 启动成功，显示 CLI 输入栏

### 4.2 容器模块 Plugin
- 文件: `cli/dtui/plugins/containers/plugin.go`
- 实现 uix.Plugin 接口
- `Init`: 异步加载容器列表（`tea.Cmd`）
- `View`: 左 table + 右 detail（lipgloss.JoinHorizontal）
- `Update`: ↑↓ 移动选择、Enter 打开详情
- 快捷键: s 启动/停止, R 重启, x 删除, r 刷新
- 复用: `cli/dtui/internal/docker/` 包（不修改）

### 4.3 容器列表表格
- 文件: `cli/dtui/plugins/containers/table.go`
- 封装 `bubbles/table`
- 列: Name(30%), Image(25%), Status(15%), Ports(20%), Created(10%)
- 宽度自适应（基于 Plugin 传入的 width）
- 状态着色: running=绿, exited=红, 其他=黄

### 4.4 容器详情面板
- 文件: `cli/dtui/plugins/containers/detail.go`
- 封装 `bubbles/viewport`
- 显示: ID, Image, Status, Ports, Command, Mounts, Network, Env
- 选中容器变化时更新内容

### 4.5 容器操作
- 文件: `cli/dtui/plugins/containers/actions.go`
- 启动/停止/重启/删除 → 异步 `tea.Cmd`
- 操作完成 → 刷新列表
- 删除前弹出确认 Modal
- 验证: 实际操作 Docker 容器

### 4.6 Phase 4 验证
```bash
go build ./cli/dtui && go vet ./cli/dtui/...
go run ./cli/dtui
# 手动验证:
# - 容器列表正确显示
# - ↑↓ 移动选择，详情面板同步更新
# - s 启动/停止容器
# - x 删除容器（弹出确认 Modal）
```

---

## Phase 5: 镜像模块

**目标**: `/images` 可用。

### 5.1 镜像 Plugin
- 文件: `cli/dtui/plugins/images/plugin.go`
- 表格: Repository, Tag, ID, Size, Created
- 操作: 删除(x)、保存为 tar(s)、清理悬空(p)、历史(i)

### 5.2 镜像操作
- 保存: 弹出 Modal + filepicker 选择目标目录
- 删除: 确认 Modal
- 清理: 确认 Modal + 显示回收空间
- 验证: `go run ./cli/dtui` → `/images` → 操作镜像

---

## Phase 6: 编排模块

**目标**: `/compose` 可用。

### 6.1 编排 Plugin
- 文件: `cli/dtui/plugins/compose/plugin.go`
- 按项目分组显示服务列表
- 显示运行状态（通过 `docker compose ps`）
- 操作: up(s)、down(d)、编辑 compose 文件(c)

### 6.2 确认对话框
- up/down 前显示 Modal，预览命令
- 复用: `cli/dtui/internal/docker/compose.go`
- 验证: `go run ./cli/dtui` → `/compose` → up/down

---

## Phase 7: 部署 + 配置模块

**目标**: `/deploy` 和 `/config` 可用。

### 7.1 部署 Plugin
- 文件: `cli/dtui/plugins/deploy/plugin.go`
- 部署目标列表
- 部署流程: 选择文件 → 确认 Modal（显示目标/路径/备份）→ 执行
- 进度 Modal 显示: 备份中 → 解压中 → 部署中 → 完成

### 7.2 配置 Plugin
- 文件: `cli/dtui/plugins/settings/plugin.go`
- 编排目录/部署目标的增删改
- 表单编辑: Modal + textinput 字段
- 外部编辑器: `tea.ExecProcess`
- 复用: `cli/dtui/internal/config/config.go`

### 7.3 Phase 6+7 验证
```bash
go build ./cli/dtui && go vet ./cli/dtui/...
go run ./cli/dtui
```

---

## Phase 8: 清理 + 全局验证

**目标**: 删除旧代码，最终质量检查。

### 8.1 删除旧 TUI 代码
```bash
rm -rf cli/dtui/internal/tui/
```

### 8.2 更新 cli/dtui/internal/cli/root.go
- 不再调用旧的 `tui.Run`，改为创建 uix FrameworkApp

### 8.3 全局验证
```bash
go build ./cli/dtui && go build ./cli/uix/...
go vet ./cli/dtui/... && go vet ./cli/uix/...
go run ./cli/dtui
# 验证:
# - 80x24 终端正常显示
# - 120x40 终端双栏正常
# - / → 命令面板显示全部 5 个模块
# - 所有模块切换正常
# - 容器操作正常
# - 镜像操作正常
```

---

## 验证清单

每个 Phase 结束后执行：

```bash
# 编译
go build ./cli/uix/... && go build ./cli/dtui

# 静态分析
go vet ./cli/uix/... && go vet ./cli/dtui/...

# LSP diagnostics
# 对每个变更文件运行 lsp_diagnostics
```

Phase 4-7 额外验证：
```bash
# 启动 TUI（需 Docker daemon 运行）
go run ./cli/dtui
```

---

## 风险缓解

| 风险 | 缓解 |
|------|------|
| bubbles/table API 变更 | 使用 go.mod 中已固定的 v1.0.0 |
| Docker SDK 兼容性 | 复用现有 `internal/docker/` 包，已验证可用 |
| 命令面板性能 | Palette 只搜索已注册的 Plugin（<20 个），无性能问题 |
| 旧代码干扰 | Phase 8 统一删除，Phase 1-7 新旧并存 |

## 回滚点

- Phase 1 完成后: uix 骨架独立，不影响旧 dtui
- Phase 4 完成后: 容器模块可独立使用
- 任何 Phase 失败: 回滚到上一 Phase，不丢失进度
