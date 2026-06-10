# Implementation Plan

## Pre-flight Checklist

- [ ] 确认 `dt.StatsEntry` 有时间戳字段
- [ ] 确认现有测试覆盖范围
- [ ] 创建 git 分支 `fix/dtui-panic-usability`

## Execution Order

### Phase 1: 修复 Panic 和窗口大小 (Critical)

**1.1 ImageTable columns 初始化**
- 文件: `cli/dtui/internal/tui/pages/images/table.go`
- 修改: `NewImageTable()` 添加 `table.WithColumns(imageColumns(80))`
- 验证: `go vet ./cli/dtui/internal/tui/pages/images/...`

**1.2 ContainerTable columns 初始化**
- 文件: `cli/dtui/internal/tui/pages/containers/table.go`
- 修改: `NewContainerTable()` 处理 width=0 的情况，使用默认宽度 80
- 修改: `page.go` 的 `SetSize()` 同步更新 table 尺寸
- 验证: `go vet ./cli/dtui/internal/tui/pages/containers/...`

**1.3 ComposeTable columns 初始化**
- 文件: `cli/dtui/internal/tui/pages/compose/table.go`
- 修改: `NewComposeTable()` 处理 width=0 的情况，使用默认宽度 80
- 验证: `go vet ./cli/dtui/internal/tui/pages/compose/...`

**1.4 DeployTable columns 初始化**
- 文件: `cli/dtui/internal/tui/pages/deploy/table.go`
- 修改: `NewDeployTable()` 处理 width=0 的情况，使用默认宽度 80
- 验证: `go vet ./cli/dtui/internal/tui/pages/deploy/...`

**1.5 ContainerPage 窗口大小处理**
- 文件: `cli/dtui/internal/tui/pages/containers/page.go`
- 修改: `SetSize()` 同步更新 table 尺寸，处理 width=0 的情况
- 验证: `go vet ./cli/dtui/internal/tui/pages/containers/...`

**1.6 ImagePage 窗口大小处理**
- 文件: `cli/dtui/internal/tui/pages/images/page.go`
- 修改: `SetSize()` 处理 width=0 的情况，使用默认宽度
- 验证: `go vet ./cli/dtui/internal/tui/pages/images/...`

**1.7 全局验证**
- 验证: `go build ./cli/dtui`
- 验证: `go run ./cli/dtui` 启动不 panic
- 验证: 切换 tab 不 panic
- 验证: 小终端 (80x24) 下表格不溢出

### Phase 2: Stats 历史改进

**2.1 添加 StatsEntry 时间戳**
- 文件: `cli/dtui/internal/docker/stats.go`
- 现状: `StatsEntry` 没有 `Timestamp` 字段
- 修改: 添加 `Timestamp time.Time` 字段，在 `parseStats()` 中赋值 `time.Now()`

**2.2 修改 renderHistory**
- 文件: `cli/dtui/internal/tui/pages/containers/stats.go`
- 修改: 倒序遍历 + 时间戳显示
- 格式: `HH:MM:SS  CPU x.x%  MEM xxx  NET ↑xxx`

**2.3 验证**
- 验证: `go vet ./cli/dtui/...`
- 验证: 手动测试 Stats 面板

### Phase 3: 设置编辑改进

**3.1 添加编辑模式选择**
- 文件: `cli/dtui/internal/tui/pages/settings/page.go`
- 修改: `openConfigEditor()` 弹出选择面板
- 新增: `editModePanel` 组件

**3.2 实现表单编辑模式**
- 文件: `cli/dtui/internal/tui/pages/settings/form.go`
- 修改: 支持预填现有值
- 新增: `editFieldsForSection()` 函数

**3.3 验证**
- 验证: `go vet ./cli/dtui/...`
- 验证: 手动测试 `c` 键编辑

### Phase 4: UI 列宽优化

**4.1 优化 imageColumns**
- 文件: `cli/dtui/internal/tui/pages/images/table.go`
- 修改: 最小宽度保护，避免负数或零

**4.2 优化其他 table 列宽**
- 文件: 各 table.go
- 修改: 同样添加最小宽度保护

**4.3 验证**
- 验证: 小终端测试 (80x24)
- 验证: 大终端测试 (200x50)

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

- Phase 1 完成后: 如果 panic 修复但其他功能有问题，可以只回滚 Phase 2-4
- Phase 2 完成后: 如果 Stats 改进有问题，可以只回滚 Phase 2
- 每个 Phase 独立，可以单独回滚

## Risk Mitigation

| 风险 | 缓解措施 |
|------|---------|
| 列宽计算错误 | 添加最小宽度保护 |
| StatsEntry 无时间戳 | 检查后再修改 |
| 表单编辑复杂 | 复用现有 FormPanel |
| 小终端兼容 | 测试 80x24 终端 |
