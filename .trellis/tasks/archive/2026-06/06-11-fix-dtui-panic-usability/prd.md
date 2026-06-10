# Fix dtui panic and usability

## Goal

修复 dtui TUI 的崩溃问题，改善设置编辑体验，优化 UI 可用性，以及改进 stats 历史展示。

## Requirements

### R1: 修复 Table panic (Critical)

**现象**: bubbles/table 在 `renderRow` 时 index out of range panic。
**根因**: 
1. `ImageTable.NewImageTable()` 创建 table 时没有初始 columns，`SetImages()` 调用 `SetRows()` 时会 panic
2. `ContainerTable` 在 `page.go:91` 创建时 `width` 可能为 0，导致列宽为 0
3. `ComposeTable` 初始化时 width=0, height=0

**修复要求**:
- `NewImageTable()` 必须初始化默认 columns
- `NewContainerTable()` 必须处理 width=0 的情况
- `ComposeTable` 必须在 width=0 时使用默认宽度
- `DeployTable` 必须处理 width=0 的情况
- 所有 table 的 `SetSize()` 必须在 width/height 为 0 时使用默认值

### R2: 设置编辑配置行为 (High)

**现象**: 按 `c` 编辑配置时，使用 `EDITOR` 或 `vi` 打开 JSON 文件。
**问题**: 用户期望 TUI 内表单编辑，而非外部编辑器。

**修复要求**:
- 明确配置编辑有两种模式：TUI 表单编辑 + 外部编辑器
- 表单编辑用于新增/修改单项配置
- 外部编辑器用于高级用户直接编辑 JSON
- 快捷键文档需要说明清楚

### R3: 窗口大小和 UI 可用性优化 (Medium)

**现状**: 用户反馈 UI 很丑，用不了。镜像、容器主要是窗口的大小展示问题。
**范围**:
- `ContainerPage.SetSize()` 必须同步更新 table 尺寸
- `ImagePage.SetSize()` 必须同步更新 table 尺寸
- 表格列宽计算需要更合理，避免内容截断或溢出
- 小终端下列宽不能为 0 或负数
- 颜色对比度需要足够，避免看不清
- 布局在小终端需要降级可用
- 状态栏信息需要清晰

### R4: Stats 历史展示改进 (Medium)

**现状**: Stats 面板的"历史"区域展示最近 10 条记录，从旧到新排列，没有时间戳。
**问题**: 用户期望倒序展示（最新在上），并显示时间戳。

**修复要求**:
- Stats 历史记录倒序渲染（最新在上）
- 每条记录增加时间戳显示
- 格式: `HH:MM:SS  CPU x.x%  MEM xxx  NET ↑xxx`

## Acceptance Criteria

- [ ] `go run ./cli/dtui` 启动后不会 panic
- [ ] 切换到镜像 tab 时，空列表或快速加载都不会崩溃
- [ ] 按 `c` 可以编辑配置，表单编辑和外部编辑器都有入口
- [ ] Stats 面板历史记录显示时间戳，最新在上
- [ ] 表格在小终端不会因为列宽计算错误而 panic
- [ ] `go vet ./cli/dtui/...` 无错误
- [ ] `go build ./cli/dtui` 编译成功

## Out of Scope

- Kubernetes 支持
- 远程 Docker 支持
- 新增功能（只修 bug 和改善现有功能）
- 完整 UI 重写（只优化关键体验）
