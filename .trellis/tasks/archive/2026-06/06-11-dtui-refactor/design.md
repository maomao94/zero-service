# dtui 重构技术设计

## 1. 面板高度统一策略

**问题**：`PanelManager.Render()` 做 padding，但各面板内部 `visibleLines()` 也做高度裁剪，双重作用导致不一致。

**方案**：统一由 `PanelManager.Render()` 负责填充，各面板 `Render()` 不做高度 padding，只负责内容渲染。面板内部的 `visibleLines()` 仅用于滚动计算（决定渲染多少条目），不用于最终输出裁剪。

**变更**：
- `PanelManager.Render()`: 保持现有 padding 逻辑（`height-4`）
- 各面板 `Render()`: 移除末尾 padding 代码，只输出内容
- 各面板 `visibleLines()`: 保留，用于滚动条目数计算

## 2. 部署流程重构

**现状流程**：
```
输入 zip 路径 → 备份容器目录 → unzip 解压 → rm -rf 容器目录 → docker cp 到容器
```

**新流程**：
```
输入路径 → 判断是文件夹还是 zip → 确认弹窗 → 备份 → 解压(仅zip) → 清空容器目录 → docker cp → 记录历史
```

**关键变更**：

### 2.1 路径类型检测
```go
func pathType(path string) string {
    info, err := os.Stat(path)
    if err != nil { return "invalid" }
    if info.IsDir() { return "folder" }
    if strings.HasSuffix(strings.ToLower(path), ".zip") { return "zip" }
    return "unknown"
}
```

### 2.2 zip 解压用 Go 标准库
```go
func unzipToDir(zipPath, destDir string) error {
    r, err := zip.OpenReader(zipPath)
    if err != nil { return err }
    defer r.Close()
    for _, f := range r.File {
        // 解压每个文件到 destDir
    }
    return nil
}
```

### 2.3 docker cp 用 SDK
```go
func (c *Client) CopyToContainer(containerID, dstPath, srcPath string) error {
    // 打包 srcPath 为 tar
    // 调用 c.cli.CopyToContainer(ctx, containerID, dstPath, tarReader, types.CopyToContainerOptions{})
}
```

### 2.4 部署确认弹窗
复用现有 `RenderConfirm`，增加操作步骤说明：
```
── 确认部署 ──
目标: lalserver
类型: 文件夹
路径: /path/to/dist
步骤: 备份 → 清空 → 部署

Enter 确认  Esc 取消
```

## 3. 历史记录

在 `commands.go` 的每个操作函数中添加 `config.RecordHistory` 调用。action 类型统一命名：
- `start`, `stop`, `restart` — 容器操作
- `compose-up` — 编排启动
- `exec` — 命令执行
- `save`, `delete`, `tag`, `prune` — 镜像操作
- `deploy` — 部署

`views/history.go` 的 action 标签映射补全所有类型。

## 4. 数据流

```
用户按键 → keys.go handleKey → executeAction → commands.go 操作函数
  → Docker SDK / exec.Command → ActionMsg → Update → 状态更新 + RecordHistory
  → View 重渲染
```

部署特殊流：
```
用户按 d → openPanel(PanelExec) → 用户输入路径 → ExecLineMsg
  → deployZipCmd → pathType 检测 → requireConfirm(ActionDeploy)
  → 用户确认 → runDeployFlowCmd → 步骤执行 → ActionMsg
```
