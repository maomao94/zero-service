# 修复 dtui Panel 系统架构统一与日志流式/Exec输入 Bug

## Goal

完成 dtui v3 Panel 接口 + PanelManager 架构迁移，修复因新旧两套 panel 系统并存导致的两个关键 bug：容器日志实时刷新失效和 exec 面板输入不显示。

## Requirements

1. **Panel 系统架构统一** — 彻底移除 Model 中遗留的旧 panel 字段（`m.panel`、`m.logPanel`、`m.inspectPanel`、`m.statsPanel`、`m.imageHistoryPanel`、`m.execInput`、`m.execOutput`、`m.statsCh`、`m.statsErrCh`），`handlePanelKey` 全程通过 `PanelManager` 委派按键和消息，不再直接访问旧字段。
2. **日志实时流式刷新** — 利用已存在的 `docker.StreamLogs`（channel 流式接口），替换当前 `streamLogsCmd` 中的 `FetchLogs` 批量调用，实现增量追加而非全量替换。
3. **Exec 面板输入正常化** — 修复按键写入 `m.execInput` 但渲染读取 `ExecPanelImpl.input` 的变量分离问题，统一由 `ExecPanelImpl` 管理输入状态。
4. **布局无残留** — 面板退出后主界面无画面残留，所有子面板渲染填满分配合高度。

## Acceptance Criteria

- [ ] 进入容器日志面板后，日志实时增量刷新（2 秒间隔），新行追加到尾部而非全量替换
- [ ] Exec 面板输入正常：按键输入立即显示在光标位置，Enter 执行命令后显示输出
- [ ] 所有面板（日志/Exec/Inspect/Stats/ImageHistory/History）的按键操作正常
- [ ] Esc 退出面板后主界面正常渲染，无画面残留
- [ ] `lsp_diagnostics` on `cli/dtui/` 无新增 error
- [ ] `go build ./cli/dtui/` 编译通过

## Notes

- 遵循 `.trellis/spec/backend/dtui-conventions.md` 中定义的 v3 Panel 接口 + PanelManager 架构
- 保留 `exec.Command` 调用 docker exec（SDK 不支持交互式 exec）
- 日志流使用 `docker.StreamLogs` 的 channel 接口，TUI 层通过 TickMsg 定时拉取
