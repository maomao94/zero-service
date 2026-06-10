# Implementation Plan: dtui TUI 面板架构重构

## 执行清单

### Step 1: 修复面板引用错误
- [ ] `model.go`: 修复 `renderChromePanel()` 中 `PanelImageHistory` 使用正确的面板引用

### Step 2: 统一面板切换逻辑
- [ ] `model.go`: 新增 `openPanel()` 和 `closePanel()` 方法
- [ ] `keys.go`: 更新 ESC 键处理，调用 `closePanel()`
- [ ] `keys.go`: 更新所有面板打开逻辑，调用 `openPanel()`

### Step 3: 清理面板状态
- [ ] `model.go`: 新增 `cleanupPanel()` 方法
- [ ] `update.go`: 更新 `syncPanelSizes()` 逻辑
- [ ] `update.go`: 更新 `TickMsg` 处理逻辑

### Step 4: 布局系统重构
- [ ] `model.go`: 新增 `Layout` 结构体和 `calculateLayout()` 方法
- [ ] `model.go`: 更新 `renderMain()` 使用统一布局计算
- [ ] `model.go`: 更新 `renderActivePanel()` 使用统一布局

### Step 5: 日志面板改进
- [ ] `model.go`: 新增 `initLogPanel()` 方法
- [ ] `model.go`: 新增 `streamLogs()` 方法
- [ ] `update.go`: 更新日志流处理逻辑

### Step 6: 渲染逻辑优化
- [ ] `model.go`: 重构 `View()` 方法，统一渲染入口
- [ ] `model.go`: 重构 `renderActivePanel()` 方法

### Step 7: 验证
- [ ] `go build` 编译验证
- [ ] `go test` 测试验证
- [ ] 手动验证所有面板切换

## 验证命令

```bash
cd cli/dtui && go build -o bin/dtui .
go test ./cli/dtui/...
# 手动验证：
# 1. 按 l 打开日志面板，验证日志流式更新
# 2. 按 i 打开详情面板，验证切换正常
# 3. 按 x 打开 stats 面板，验证切换正常
# 4. 按 ESC 关闭面板，验证状态清理
# 5. 按 H 打开操作历史，验证切换正常
# 6. 验证不同终端尺寸下的布局稳定性
```

## 相关文件

- `cli/dtui/internal/tui/model.go` - 核心模型和渲染
- `cli/dtui/internal/tui/update.go` - 消息处理
- `cli/dtui/internal/tui/keys.go` - 按键处理
- `cli/dtui/internal/tui/views/layout.go` - 布局渲染
