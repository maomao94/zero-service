# Implementation Plan: dtui 备份路径重构与操作历史

## 执行清单

### Step 1: 备份路径重构
- [ ] `docker/runner.go`: 新增 `RunDockerCpToHost` 方法
- [ ] `config/config.go`: 新增 `HistoryPath()` 和历史读写方法
- [ ] `config/config.go`: 新增 `CleanOldBackups` 方法
- [ ] `commands.go`: 修改 `runDeployFlowCmd` 使用主机备份
- [ ] `config/config.go`: 更新默认 `BackupDir` 为空字符串

### Step 2: 操作历史数据结构
- [ ] `messages.go`: 新增 `HistoryEntry` 结构体
- [ ] `messages.go`: 新增 `HistoryLoadedMsg`
- [ ] `config/config.go`: 实现历史持久化

### Step 3: 操作历史面板
- [ ] `views/history.go`: 新建 `HistoryPanelModel`
- [ ] `views/history.go`: 实现时间轴渲染
- [ ] `styles/styles.go`: 新增历史相关样式

### Step 4: 主布局重构
- [ ] `model.go`: 移除 `cmdLog` 和 `logCmd`
- [ ] `model.go`: 新增 `history` 和 `historyPanel`
- [ ] `model.go`: 简化 `buildBodyLines` 移除右侧日志
- [ ] `model.go`: 更新 `logPanelWidth` / `listWidth` 逻辑

### Step 5: 按键与命令
- [ ] `keys.go`: 新增 `H` 键打开历史面板
- [ ] `commands.go`: 新增操作记录函数
- [ ] `commands.go`: 修改各操作命令记录历史

### Step 6: 帮助栏与验证
- [ ] `views/layout.go`: 更新帮助栏
- [ ] `go build` 编译验证
- [ ] `go test` 测试验证

## 验证命令

```bash
cd cli/dtui && go build -o bin/dtui .
go test ./cli/dtui/...
# 手动验证：
# 1. 部署后检查 ~/.dtui/backups/{container}/{timestamp}/ 是否有备份
# 2. 按 H 查看操作历史面板
# 3. 操作历史持久化：重启后历史仍在
```
