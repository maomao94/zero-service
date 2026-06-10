# Implementation Plan

## Pre-flight Checklist

- [ ] 确认 `dt.ComposeStatus` 函数存在或可实现
- [ ] 确认 `config.UpdateComposeDir` 和 `config.UpdateDeployTarget` 函数存在或可实现
- [ ] 确认 `charmbracelet/bubbles/filepicker` 依赖可用

## Execution Order

### Phase 1: 布局引擎 (Critical)

**1.1 创建 layout 包**
- 文件: `cli/dtui/internal/tui/layout/layout.go`
- 内容: `Metrics` 结构体、`Calculate()` 函数、`ProportionalColumns()` 函数
- 验证: `go vet ./cli/dtui/internal/tui/layout/...`

**1.2 重构 Model 结构体**
- 文件: `cli/dtui/internal/tui/model.go`
- 修改: 添加 `metrics layout.Metrics` 字段
- 修改: 删除 `listWidth()`、`bodyHeight()`、`availableListHeight()` 方法
- 修改: 所有使用这些方法的地方改为使用 `m.metrics`
- 验证: `go vet ./cli/dtui/...`

**1.3 重构 Update 处理 WindowSizeMsg**
- 文件: `cli/dtui/internal/tui/update.go`
- 修改: 处理 `tea.WindowSizeMsg` 时调用 `layout.Calculate()`
- 验证: `go vet ./cli/dtui/...`

**1.4 重构视图函数签名**
- 文件: `cli/dtui/internal/tui/views/containers.go`
- 修改: `RenderContainers`、`RenderImages`、`RenderCompose`、`RenderDeploy`、`RenderSettings` 签名改为接收 `layout.Metrics`
- 修改: 内部使用 `m.ContentPad` 替代硬编码
- 验证: `go vet ./cli/dtui/...`

**1.5 更新 model.go 中的调用**
- 文件: `cli/dtui/internal/tui/model.go`
- 修改: `renderBody()` 调用视图函数时传入 `m.metrics`
- 验证: `go vet ./cli/dtui/...`

**1.6 全局验证**
- 验证: `go build ./cli/dtui`
- 验证: `go run ./cli/dtui` 启动不 panic
- 验证: 终端 80x24 下正常显示

### Phase 2: 设置模块修复

**2.1 新增 config 更新函数**
- 文件: `cli/dtui/internal/config/config.go`
- 新增: `UpdateComposeDir()`、`UpdateDeployTarget()`
- 验证: `go vet ./cli/dtui/internal/config/...`

**2.2 修改 'c' 键行为**
- 文件: `cli/dtui/internal/tui/keys.go`
- 修改: 'c' 键弹出选择面板
- 验证: `go vet ./cli/dtui/...`

**2.3 实现选择面板**
- 文件: `cli/dtui/internal/tui/views/select.go` (新建)
- 内容: `RenderSelect()` 函数，显示选项列表
- 验证: `go vet ./cli/dtui/...`

**2.4 实现表单编辑**
- 文件: `cli/dtui/internal/tui/commands.go`
- 修改: `settingsEditCmd()` 函数，预填当前值
- 验证: `go vet ./cli/dtui/...`

**2.5 全局验证**
- 验证: `go build ./cli/dtui`
- 验证: 设置页按 `c` 弹出选择面板

### Phase 3: 编排模块改进

**3.1 新增 ComposeStatus 函数**
- 文件: `cli/dtui/internal/docker/compose.go`
- 新增: `ComposeStatus()` 函数，调用 `docker compose ps`
- 验证: `go vet ./cli/dtui/internal/docker/...`

**3.2 修改 ComposeService 结构体**
- 文件: `cli/dtui/internal/tui/views/containers.go`
- 修改: `ComposeService` 添加 `Status` 字段
- 修改: `RenderCompose` 显示状态
- 验证: `go vet ./cli/dtui/...`

**3.3 修改 loadComposeCmd**
- 文件: `cli/dtui/internal/tui/commands.go`
- 修改: `loadComposeCmd()` 调用 `ComposeStatus()` 获取状态
- 验证: `go vet ./cli/dtui/...`

**3.4 修改确认对话框**
- 文件: `cli/dtui/internal/tui/views/confirm.go`
- 修改: `RenderConfirm()` 显示命令预览
- 验证: `go vet ./cli/dtui/...`

**3.5 新增 compose down**
- 文件: `cli/dtui/internal/tui/keys.go`
- 新增: 'd' 键支持 `ActionComposeDown`
- 文件: `cli/dtui/internal/tui/commands.go`
- 新增: `composeDownCmd()` 函数
- 验证: `go vet ./cli/dtui/...`

**3.6 全局验证**
- 验证: `go build ./cli/dtui`
- 验证: 编排页显示服务状态

### Phase 4: 发布模块改进

**4.1 修改确认对话框**
- 文件: `cli/dtui/internal/tui/views/confirm.go`
- 修改: 部署确认显示目标容器、路径、备份目录
- 验证: `go vet ./cli/dtui/...`

**4.2 修改部署流程**
- 文件: `cli/dtui/internal/tui/commands.go`
- 修改: `runDeployFlowCmd()` 发送进度消息
- 验证: `go vet ./cli/dtui/...`

**4.3 处理进度消息**
- 文件: `cli/dtui/internal/tui/update.go`
- 新增: 处理 `DeployProgressMsg`，更新状态栏
- 验证: `go vet ./cli/dtui/...`

**4.4 全局验证**
- 验证: `go build ./cli/dtui`
- 验证: 发布页显示确认面板

### Phase 5: 视图渲染优化

**5.1 列宽最小值保护**
- 文件: `cli/dtui/internal/tui/layout/layout.go`
- 修改: `ProportionalColumns()` 添加最小宽度保护
- 验证: `go vet ./cli/dtui/internal/tui/layout/...`

**5.2 小终端适配**
- 文件: `cli/dtui/internal/tui/layout/layout.go`
- 修改: `Calculate()` 在 width < 80 时切换为单栏模式
- 验证: `go vet ./cli/dtui/internal/tui/layout/...`

**5.3 全局验证**
- 验证: `go build ./cli/dtui`
- 验证: 终端 80x24 下正常显示
- 验证: 终端 120x40 下双栏布局正常

## Validation Commands

```bash
# 编译
go build ./cli/dtui

# 静态检查
go vet ./cli/dtui/...

# 启动测试
go run ./cli/dtui

# 切换 tab，检查是否 panic
# 按 s 切换到镜像 tab
# 按 c 测试设置编辑
# 按 H 测试历史面板
```

## Rollback Points

- Phase 1 完成后: 布局引擎重构完成，可以独立回滚 Phase 2-5
- Phase 2 完成后: 设置模块修复完成，可以独立回滚 Phase 3-5
- 每个 Phase 独立，可以单独回滚

## Risk Mitigation

| 风险 | 缓解措施 |
|------|---------|
| 布局引擎重构影响所有视图 | 分步重构，每步验证 |
| 设置模块修改影响现有功能 | 保持外部编辑器选项 |
| 编排状态查询增加启动时间 | 异步查询，不阻塞主线程 |
| filepicker 依赖增加包体积 | 作为可选功能，不强制使用 |
