# Technical Design: dtui 备份路径重构与操作历史

## 概述

两个核心变更：
1. 部署备份从容器内改为主机 `.dtui/backups/`
2. 右侧命令日志 panel 改为操作历史时间轴

## 1. 备份路径重构

### 备份逻辑变更 (commands.go)

**变更前**：在容器内执行备份
```go
backupPath := t.BackupDir + "/" + ts
dt.RunDockerExec(t.Container, "mkdir", "-p", backupPath)
dt.RunDockerExec(t.Container, "cp", "-r", t.HtmlPath+"/.", backupPath+"/")
```

**变更后**：在主机执行 docker cp
```go
backupDir := t.BackupDir
if backupDir == "" {
    home, _ := os.UserHomeDir()
    backupDir = filepath.Join(home, ".dtui", "backups", t.Container)
}
backupPath := filepath.Join(backupDir, ts)
os.MkdirAll(backupPath, 0755)
// docker cp container:/path ./local-path
dt.RunDockerCpToHost(t.Container, t.HtmlPath, backupPath)
```

### 新增 docker 命令 (docker/runner.go)

```go
func RunDockerCpToHost(container, containerPath, hostPath string) (string, error) {
    out, err := exec.Command("docker", "cp", container+":"+containerPath+"/.", hostPath+"/").CombinedOutput()
    return string(out), err
}
```

### 备份清理逻辑

```go
func cleanOldBackups(backupDir string, keep int) {
    entries, _ := os.ReadDir(backupDir)
    if len(entries) <= keep { return }
    // 按时间排序，删除最旧的
    sort.Slice(entries, func(i, j int) bool {
        ti, _ := entries[i].Info()
        tj, _ := entries[j].Info()
        return ti.ModTime().Before(tj.ModTime())
    })
    for i := 0; i < len(entries)-keep; i++ {
        os.RemoveAll(filepath.Join(backupDir, entries[i].Name()))
    }
}
```

## 2. 操作历史

### 数据结构 (messages.go)

```go
type HistoryEntry struct {
    Time    time.Time `json:"time"`
    Action  string    `json:"action"`
    Target  string    `json:"target"`
    Detail  string    `json:"detail"`
    Success bool      `json:"success"`
    Error   string    `json:"error,omitempty"`
}

type HistoryLoadedMsg struct {
    Entries []HistoryEntry
    Err     error
}
```

### 持久化 (config/config.go)

```go
func HistoryPath() string {
    home, _ := os.UserHomeDir()
    return filepath.Join(home, ".dtui", "history.json")
}

func LoadHistory(path string) []HistoryEntry { ... }
func SaveHistory(path string, entries []HistoryEntry) { ... }
```

### 操作历史面板 (views/history.go)

新增 `HistoryPanelModel`，显示时间轴格式：
- 按时间倒序
- 颜色区分成功/失败
- 支持滚动和搜索

### Model 变更 (model.go)

- 移除 `cmdLog []string`
- 移除 `logCmd()` 方法
- 新增 `historyPanel *views.HistoryPanelModel`
- 新增 `history []HistoryEntry`

## 3. 主布局变更

**变更前**：列表 + 右侧日志 panel
```
header
[列表] │ [命令日志]
footer
status
```

**变更后**：列表占满宽度
```
header
[列表]
footer
status
```

`buildBodyLines` 简化为只渲染列表，不再有右侧 panel。

## 4. 按键映射

- `H`：打开操作历史面板
- `H` 在历史面板内：关闭

## 5. 兼容性

- `BackupDir` 配置字段保留，为空时使用默认路径
- 旧配置文件无需迁移
- `cmdLog` 完全移除，不影响其他功能
