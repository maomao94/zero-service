# Implementation Plan: dtui 面板架构重构

## 执行清单

### Phase 1: 定义接口和管理器
- [ ] `internal/tui/panel.go`: 创建 Panel 接口定义
- [ ] `internal/tui/panel_manager.go`: 创建 PanelManager 实现
- [ ] `internal/tui/model.go`: 添加 panels 字段到 Model

### Phase 2: 迁移 ExecPanel
- [ ] `internal/tui/panels/exec.go`: 创建 ExecPanel 实现
- [ ] `internal/tui/keys.go`: 更新 exec 相关按键处理
- [ ] `internal/tui/model.go`: 移除 execInput/execOutput 字段

### Phase 3: 迁移 LogPanel
- [ ] `internal/tui/panels/log.go`: 创建 LogPanel 实现
- [ ] `internal/tui/keys.go`: 更新 log 相关按键处理
- [ ] `internal/tui/update.go`: 更新日志消息处理
- [ ] `internal/tui/model.go`: 移除 logPanel 字段

### Phase 4: 迁移 StatsPanel
- [ ] `internal/tui/panels/stats.go`: 创建 StatsPanel 实现
- [ ] `internal/tui/keys.go`: 更新 stats 相关按键处理
- [ ] `internal/tui/update.go`: 更新 stats 消息处理
- [ ] `internal/tui/model.go`: 移除 statsCh/statsErrCh/statsPanel 字段

### Phase 5: 迁移 InspectPanel
- [ ] `internal/tui/panels/inspect.go`: 创建 InspectPanel 实现
- [ ] `internal/tui/keys.go`: 更新 inspect 相关按键处理
- [ ] `internal/tui/model.go`: 移除 inspectPanel 字段

### Phase 6: 迁移 ImageHistoryPanel
- [ ] `internal/tui/panels/image_history.go`: 创建 ImageHistoryPanel 实现
- [ ] `internal/tui/keys.go`: 更新 image history 相关按键处理
- [ ] `internal/tui/model.go`: 移除 imageHistoryPanel 字段

### Phase 7: 迁移 HistoryPanel
- [ ] `internal/tui/panels/history.go`: 创建 HistoryPanel 实现
- [ ] `internal/tui/keys.go`: 更新 history 相关按键处理
- [ ] `internal/tui/model.go`: 移除 historyPanel 字段

### Phase 8: 清理和验证
- [ ] `internal/tui/model.go`: 清理 Model，移除所有面板特有状态
- [ ] `internal/tui/update.go`: 简化 Update 逻辑
- [ ] `internal/tui/keys.go`: 简化按键处理逻辑
- [ ] `go build` 编译验证
- [ ] `go test` 测试验证

## 验证命令

```bash
cd cli/dtui && go build -o bin/dtui .
go test ./cli/dtui/...
# 手动验证：
# 1. 所有面板切换正常
# 2. ESC 键正确关闭面板
# 3. 日志流式更新正常
# 4. Stats 实时更新正常
# 5. Exec 面板功能正常
# 6. 布局在不同终端尺寸下稳定
```

## 相关文件

- `cli/dtui/internal/tui/model.go` - 核心模型
- `cli/dtui/internal/tui/update.go` - 消息处理
- `cli/dtui/internal/tui/keys.go` - 按键处理
- `cli/dtui/internal/tui/views/` - 面板视图
