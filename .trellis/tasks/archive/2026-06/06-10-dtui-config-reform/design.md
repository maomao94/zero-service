# Technical Design: dtui 配置统一与 Tab 重构

## 概述

本次重构涉及以下核心变更：
1. Tab 名称重命名（发布 → 部署）
2. 按键逻辑调整（编排页 `c` 键改为编辑 compose 文件）
3. 配置结构重命名（DeployTarget → DeployTarget）

## 变更范围

### 1. 消息类型重命名 (messages.go)

```go
// 变更前
type DeployTarget struct { ... }

// 变更后
type DeployTarget struct { ... } // 保持不变，语义一致
```

### 2. Tab 名称更新 (views/layout.go)

```go
// 变更前
tabs := []string{"容器", "镜像", "编排", "发布", "设置"}

// 变更后
tabs := []string{"容器", "镜像", "编排", "部署", "设置"}
```

### 3. 按键逻辑调整 (keys.go)

**变更前**：
```go
case "c":
    if m.mode == ComposeView || m.mode == DeployView || m.mode == SettingsView {
        _, cmd := m.openConfigCmd()
        return m, cmd
    }
```

**变更后**：
```go
case "c":
    if m.mode == SettingsView {
        _, cmd := m.openConfigCmd()
        return m, cmd
    }
    if m.mode == ComposeView {
        return m.openComposeFileCmd()
    }
```

### 4. 新增编辑 compose 文件命令 (commands.go)

```go
func (m Model) openComposeFileCmd() (tea.Model, tea.Cmd) {
    if len(m.composeSvcs) == 0 || m.cursor >= len(m.composeSvcs) {
        return m, nil
    }
    svc := m.composeSvcs[m.cursor]
    composePath := filepath.Join(svc.Path, "docker-compose.yml")
    
    // 检查文件是否存在
    if _, err := os.Stat(composePath); os.IsNotExist(err) {
        m.status = viewsRenderDim("文件不存在: " + composePath + "，按 i 初始化")
        return m, nil
    }
    
    // 复用 openConfigCmd 的编辑器逻辑
    editor := os.Getenv("EDITOR")
    if editor == "" {
        for _, e := range []string{"vim", "nano", "vi"} {
            if p, err := exec.LookPath(e); err == nil {
                editor = p
                break
            }
        }
    }
    if editor == "" {
        return m, func() tea.Msg {
            return ActionMsg{Err: fmt.Errorf("未找到编辑器，请设置 EDITOR 环境变量")}
        }
    }
    
    m.logCmd(fmt.Sprintf("$ %s %s", filepath.Base(editor), composePath))
    return m, tea.ExecProcess(&exec.Cmd{
        Path:   editor,
        Args:   []string{editor, composePath},
        Stdin:  os.Stdin,
        Stdout: os.Stdout,
        Stderr: os.Stderr,
    }, func(err error) tea.Msg {
        if err != nil {
            return ActionMsg{Err: fmt.Errorf("编辑器错误: %w", err)}
        }
        return ActionMsg{Text: "compose 文件已保存，按 r 刷新"}
    })
}
```

### 5. 底部帮助栏更新 (views/layout.go)

```go
case 2: // compose
    extra = fmt.Sprintf("%s/%s 启动  %s 初始化  %s 编辑 compose",
        styles.HelpKey.Render("s"), styles.HelpKey.Render("u"), 
        styles.HelpKey.Render("i"), styles.HelpKey.Render("c"))
case 3: // deploy
    extra = fmt.Sprintf("%s 部署",
        styles.HelpKey.Render("d"))
```

### 6. 配置结构更新 (config/config.go)

```go
// 变更前
type Config struct {
    ComposeDirs   []ComposeDir   `json:"compose_dirs"`
    DeployTargets []DeployTarget `json:"deploy_targets"`
}

// 变更后 - 保持 JSON 字段名不变以兼容旧配置
type Config struct {
    ComposeDirs   []ComposeDir   `json:"compose_dirs"`
    DeployTargets []DeployTarget `json:"deploy_targets"`
}
```

### 7. 默认配置更新 (config/config.go)

```go
func InitDefault(path string) {
    // ...
    cfg := Config{
        ComposeDirs: []ComposeDir{
            {Name: "测试项目", Path: filepath.Join(home, ".dtui/compose")},
        },
        DeployTargets: []DeployTarget{
            {Name: "测试部署", Container: "dtui-hello", HtmlPath: "/usr/share/nginx/html", BackupDir: "/tmp/dtui-backups"},
        },
    }
    // ...
}
```

## 数据流

```
用户按键 → keys.go handleKey → 视图模式判断 → 执行对应命令
    ↓
ComposeView + "c" → openComposeFileCmd() → 编辑器打开 compose 文件
DeployView + "d"  → executeAction(ActionDeploy) → 部署流程
SettingsView + "c" → openConfigCmd() → 编辑器打开 config.json
```

## 兼容性

- 配置文件 JSON 字段名保持不变，向后兼容
- Tab 枚举值不变（ContainersView=0, ImagesView=1, ComposeView=2, DeployView=3, SettingsView=4）
- 只有显示名称变更，不影响内部逻辑

## 风险点

1. 编辑器打开 compose 文件时，如果文件不存在需要提示用户先初始化
2. 部署页移除 `c` 键后，用户可能习惯性按 `c` 编辑配置，需要在帮助栏明确提示
