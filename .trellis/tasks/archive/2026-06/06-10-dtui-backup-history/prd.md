# PRD: dtui 备份路径重构与操作历史

## 背景

1. **备份路径**：当前部署时备份在容器内执行（`docker exec` + `cp`），路径由配置 `BackupDir` 指定，默认 `/tmp/dtui-backups`。用户希望备份到主机的 `~/.dtui/backups/` 下。
2. **命令日志**：当前右侧面板 `cmdLog []string` 是简单的字符串列表，没有时间戳、没有分类，显示混乱。用户希望砍掉这个 panel，改为操作历史时间轴，并持久化到 log 文件。

## 需求

### 1. 备份路径改到主机 `.dtui/` 下

**目标**：
- 默认备份路径：`~/.dtui/backups/{container-name}/{timestamp}/`
- 备份在主机上执行（`docker cp`），不再在容器内执行
- `BackupDir` 配置变为可选：为空时使用默认路径，非空时使用自定义路径

**备份文件名规则**：
- 目录结构：`~/.dtui/backups/{container-name}/20260610-143022/`
- 保留最近 10 个备份，超出自动清理

### 2. 命令日志 → 操作历史

**目标**：
- 砍掉右侧 `cmdLog` panel
- 新增操作历史面板（按键 `H` 或从设置页进入）
- 操作历史持久化到 `~/.dtui/history.json`

**操作历史数据结构**：
```go
type HistoryEntry struct {
    Time    time.Time `json:"time"`
    Action  string    `json:"action"`  // deploy, compose-up, exec, etc.
    Target  string    `json:"target"`  // container name, service name
    Detail  string    `json:"detail"`  // zip path, command, etc.
    Success bool      `json:"success"`
    Error   string    `json:"error,omitempty"`
}
```

**显示格式**（时间轴）：
```
── 操作历史 ──

  14:30:22  部署  nginx-container  backup.zip  ✓
  14:28:15  编排启动  myproject/web  ✓
  14:25:00  执行命令  nginx-container  ls /etc/nginx  ✗ 权限拒绝
```

### 3. 主布局调整

- 移除右侧命令日志 panel
- 主列表占满宽度
- 操作历史作为独立 panel（按键 `H` 进入）

## 验收标准

1. 部署备份到 `~/.dtui/backups/{container}/{timestamp}/`
2. 备份使用 `docker cp` 在主机执行
3. 超过 10 个备份自动清理最旧的
4. 右侧命令日志 panel 移除
5. 按 `H` 进入操作历史面板
6. 操作历史持久化到 `~/.dtui/history.json`
7. 操作历史显示时间、动作、目标、结果
