# dtui 布局自适应重构与模块修复

## Goal

重构 dtui TUI 的布局系统，实现真正的自适应布局；修复设置、编排、发布三大模块的功能问题，使其可用。

## 当前问题分析

### P1: 布局系统问题 (High)

**现状**: 布局计算散落在 `model.go` 的 `listWidth()`、`bodyHeight()`、`availableListHeight()` 方法中，使用硬编码的魔法数字：
- `headerH := 6` (model.go:79)
- `footerH := 1` (model.go:83)
- `m.bodyHeight() - 4` (model.go:93)
- `m.width - 2` (model.go:71)

各视图函数（`RenderContainers`、`RenderImages` 等）自行计算列宽，模式不统一。

**目标**: 集中布局计算，业务代码只接收计算好的尺寸。

### P2: 设置模块 'c' 键问题 (High)

**现状**: `openConfigCmd()` (commands.go:445) 直接打开外部编辑器编辑 JSON 配置文件。
**问题**: 
1. 如果 EDITOR 未设置且 vim/nano/vi 都找不到，会报错
2. 用户期望能在 TUI 内直接编辑单个配置项
3. 设计文档要求提供"表单编辑"和"JSON 编辑器"两种模式

**目标**: 'c' 键提供选择：表单编辑（修改当前选中项）或 JSON 编辑器（高级模式）。

### P3: 编排模块功能问题 (Medium)

**现状**: 
- 按 `s` 或 `u` 直接执行 `docker compose up`，没有确认细节
- 按 `c` 打开外部编辑器编辑 compose 文件
- 按 `i` 初始化 compose 文件（如果不存在）
- 服务列表只显示服务名，缺少状态信息

**目标**: 
- 显示服务运行状态（running/stopped/exited）
- compose up 前显示将要执行的命令
- 提供 compose down 选项

### P4: 发布模块功能问题 (Medium)

**现状**: 
- 按 `d` 打开 ExecPanel 输入路径，然后执行部署
- 部署流程是：备份 → 解压 → 清空 → 复制
- 没有进度显示，只有最终成功/失败

**目标**: 
- 显示部署进度（备份中/解压中/部署中）
- 支持选择本地文件（filepicker）
- 部署前确认目标容器和路径

### P5: 视图渲染问题 (Medium)

**现状**: 
- `RenderContainers` 在列表下方显示详情，占用列表空间
- `RenderImages` 同样在列表下方显示详情
- 小终端下列宽可能为 0 或负数

**目标**: 
- 详情区独立于列表区，不挤占列表空间
- 列宽有最小值保护
- 小终端下列宽自动调整

## Requirements

### R1: 布局引擎重构

1. 创建 `layout` 包，定义 `Metrics` 结构体：
   - HeaderH, FooterH, StatusH, BodyH
   - ListW, DetailW (双栏模式)
   - PanelW, PanelH (面板模式)
2. `Calculate(width, height) Metrics` 函数集中计算
3. 消除 `model.go` 中的 `listWidth()`、`bodyHeight()`、`availableListHeight()` 方法
4. 视图函数接收 `Metrics` 参数，不再自行计算

### R2: 设置模块修复

1. 'c' 键弹出选择面板：表单编辑 / JSON 编辑器
2. 表单编辑复用现有 `FormPanel`，预填当前选中项的值
3. JSON 编辑器保持现有行为
4. 编辑完成后自动刷新配置

### R3: 编排模块改进

1. 服务列表显示运行状态（通过 `docker compose ps`）
2. `s`/`u` 键执行前显示确认对话框，包含命令预览
3. 新增 `d` 键支持 `docker compose down`
4. 服务详情区显示 compose 文件路径和服务命令

### R4: 发布模块改进

1. 部署流程增加状态提示（备份中/解压中/部署中）
2. 支持 filepicker 选择本地文件
3. 部署前确认面板显示：目标容器、HTML 路径、备份目录
4. 部署完成后显示备份路径

### R5: 视图渲染优化

1. 容器/镜像列表和详情分离渲染，详情区固定在右侧或底部
2. 列宽计算添加最小值保护（每列至少 6 字符）
3. 小终端（<80 列）自动切换为单栏模式
4. 状态栏显示当前视图名称和选中项索引

## Acceptance Criteria

- [ ] `go build ./cli/dtui` 编译成功
- [ ] `go vet ./cli/dtui/...` 无错误
- [ ] 终端 80x24 下所有视图正常显示，无溢出
- [ ] 终端 120x40 下双栏布局正常
- [ ] 设置页按 `c` 弹出选择面板，可选择表单编辑或 JSON 编辑器
- [ ] 编排页服务列表显示运行状态
- [ ] 编排页 `s`/`u` 键执行前有确认对话框
- [ ] 发布页 `d` 键显示确认面板
- [ ] 所有表格列宽不为 0 或负数

## Out of Scope

- Kubernetes 支持
- 远程 Docker 支持
- 完整 UI 重写（只优化关键体验）
- 新增功能（只修 bug 和改善现有功能）
