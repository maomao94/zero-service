# dtui 全模块重构：面板布局、部署流程、操作可靠性

## Goal

修复 dtui TUI 的面板布局问题、重构部署流程支持文件夹/zip 双模式、补全操作历史记录，确保所有模块可靠运行。

## Requirements

### R1: 面板布局一致性

所有子面板（日志、历史、exec、stats、inspect、镜像历史）在任意终端尺寸下填满分配区域，无底层内容透出。各面板 `visibleLines()` 计算统一，小终端（80x24）不崩溃。

### R2: 部署流程重构

- 支持输入文件夹路径：直接 `docker cp` 文件夹内容到容器目标目录
- 支持输入 zip 路径：用 Go 标准库 `archive/zip` 解压后 `docker cp`
- `docker cp` 改用 Docker SDK `CopyToContainer` API（替代 `exec.Command`）
- 部署前显示确认对话框，列出操作步骤
- 部署过程有步骤进度反馈（备份 → 解压 → 清空 → 部署）
- 部署失败保留备份路径，方便回滚

### R3: 操作历史完善

所有用户操作记录到 `~/.dtui/history.json`：容器启停/重启、compose up、exec 命令、镜像保存/删除/打标签/清理、部署。历史面板展示所有类型，中文标签正确，最新操作在上方。

### R4: 按键简化

移除 `1-9` 数字键快速跳转。容器/镜像通常超过 10 个，`1-9` 只覆盖前 9 个，实用性低。用户定位用 `/` 搜索过滤（已有）足够。释放数字键减少认知负担。

### R5: 整体可靠性

- 确认弹窗居中显示
- footer/status 始终在底部
- Tab 切换后游标正确重置
- 面板关闭后 busy 状态清理
- Docker daemon 断连有明确提示

## Acceptance Criteria

- [ ] 所有面板在 80x24 终端下填满，无底层内容透出
- [ ] 日志面板最后一条不被 footer 遮挡
- [ ] 部署支持输入文件夹路径，内容正确解压到容器目录
- [ ] 部署支持输入 zip 路径，用 Go 标准库解压
- [ ] docker cp 使用 SDK CopyToContainer
- [ ] 所有操作（启停/compose/exec/镜像操作/部署）记录历史
- [ ] 历史面板展示所有操作类型，中文标签
- [ ] build + vet + test 全通过
- [ ] `1-9` 数字键跳转已移除

## Constraints

- 遵循 `.trellis/spec/backend/dtui-conventions.md`
- Panel 接口不变
- 消息类型不变
- 保持 `tea.WithAltScreen()` 模式

## Modules to Change

| 模块 | 文件 | 说明 |
|------|------|------|
| PanelManager | `panel.go` | Render padding 统一 |
| 日志面板 | `panels_log.go` + `views/detail_logs.go` | visibleLines 修正 |
| 历史面板 | `views/history.go` | 高度填充 + 滚动 |
| exec 面板 | `panels_exec.go` | 高度填充 |
| stats/inspect/镜像历史 | `views/detail_stats.go`, `views/detail_logs.go` | 高度填充 |
| 部署流程 | `commands.go` + `docker/compose.go` | zip/folder + SDK cp |
| 历史记录 | `commands.go` + `views/history.go` | 全操作记录 + 标签 |
| 按键 | `keys.go` | 移除 1-9 选行 |
