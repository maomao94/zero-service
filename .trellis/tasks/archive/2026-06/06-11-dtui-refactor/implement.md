# dtui 重构执行计划

## Phase 1: 面板高度统一 (1-2h)

### 1.1 审计各面板 visibleLines()
- [ ] `views/detail_logs.go` LogPanelModel.visibleLines() — 确认只用于滚动计算
- [ ] `views/detail_stats.go` ImageHistoryPanelModel.visibleLines() — 同上
- [ ] `views/history.go` HistoryPanelModel.visibleLines() — 同上
- [ ] `views/detail_logs.go` InspectPanelModel — 检查高度使用

### 1.2 确认 PanelManager.Render() padding 足够
- [ ] 验证 `height-4` 计算正确（header 2 + footer 1 + status 1）
- [ ] 小终端测试：80x24

### 1.3 各面板 Render() 输出验证
- [ ] 日志面板：移除多余 padding，依赖 PanelManager
- [ ] 历史面板：确认输出行数 ≤ visibleLines
- [ ] exec 面板：确认输出填充
- [ ] stats 面板：确认输出填充
- [ ] inspect 面板：确认输出填充

**验证**：`go build ./cli/dtui/... && go test ./cli/dtui/...`

## Phase 2: 按键简化 + 历史记录补全 (30min)

### 2.1 移除 1-9 选行
- [ ] `keys.go` 删除 `case "1", "2", ... "9"` 分支

### 2.2 commands.go 添加 RecordHistory
- [ ] `runContainerCmd` — start/stop/restart
- [ ] `saveImageCmd` — save
- [ ] `pruneCmd` — prune
- [ ] `imageRmCmd` — delete
- [ ] `imageTagCmd` — tag
- [ ] `composeUpCmd` — compose-up
- [ ] `runExecLineCmd` — exec

### 2.2 views/history.go 补全标签
- [ ] 添加 start/stop/restart/save/prune/delete/tag 的中文标签

**验证**：`go build ./cli/dtui/... && go test ./cli/dtui/...`

## Phase 3: 部署流程重构 (2-3h)

### 3.1 路径类型检测
- [ ] `docker/compose.go` 添加 `PathType(path) string` 函数
- [ ] 返回 "folder" / "zip" / "invalid"

### 3.2 Go 标准库 zip 解压
- [ ] `docker/compose.go` 添加 `UnzipToDir(zipPath, destDir string) error`
- [ ] 使用 `archive/zip`，处理目录结构、权限

### 3.3 Docker SDK CopyToContainer
- [ ] `docker/compose.go` 添加 `CopyToContainer(containerID, dstPath, srcPath string) error`
- [ ] 打包 srcPath 为 tar → 调用 SDK `CopyToContainer`

### 3.4 部署命令重构
- [ ] `commands.go` 修改 `runDeployFlowCmd` 支持 folder/zip 双模式
- [ ] 部署前确认弹窗显示操作步骤
- [ ] 步骤进度通过 status 栏显示
- [ ] 失败时保留备份路径

### 3.5 deployZipCmd 路由修改
- [ ] `keys.go` 的 `deployZipCmd` 改名为 `deployPathCmd`
- [ ] 调用 `PathType` 检测后走不同分支

**验证**：`go build ./cli/dtui/... && go test ./cli/dtui/...`
**手动测试**：部署文件夹、部署 zip、部署不存在路径

## Phase 4: 集成验证 (30min)

- [ ] `go build ./cli/dtui/...`
- [ ] `go vet ./cli/dtui/...`
- [ ] `go test ./cli/dtui/...`
- [ ] 手动验证：所有面板在 80x24 终端下正常
- [ ] 手动验证：历史面板显示所有操作类型
- [ ] 手动验证：部署文件夹和 zip 都正常

## Rollback

每个 Phase 独立，可单独回滚。Phase 3 改动最大，如出问题可回退到 Phase 2 状态。
