# dtui 面板交互优化技术设计

## 1. 架构概览

本次改造涉及 dtui TUI 的三个层面：
- **配置层** — `internal/config/` 新增 `DeployPackage` 类型 + 表单数据结构
- **面板层** — 新增 `FormPanel` 实现 Panel 接口，改造现有面板的自适应逻辑
- **视图层** — 日志换行、历史展示优化、exec 动态布局

## 2. 配置表单化

### 2.1 新增 FormPanel

```go
type FormPanelImpl struct {
    fields   []FormField
    cursor   int
    width    int
    height   int
    title    string
    onSubmit func(map[string]string)
    onCancel func()
}

type FormField struct {
    Label       string
    Key         string
    Value       string
    Placeholder string
    Default     string
}
```

FormPanel 实现 Panel 接口：
- `Open` — 初始化字段，回填已有值
- `Render` — 渲染表单，每字段一行：`标签: 值█`
- `HandleKey` — Tab 切换字段，字符输入，Enter 提交，Esc 取消
- `HandleMsg` — 不需要

### 2.2 表单触发

设置页按键处理：
- `a` 新增 → 根据当前选中区域（编排目录/前端发布目标/发布包）打开对应 FormPanel
- `e` 编辑 → 回填当前值到 FormPanel
- 提交后调用 `config.AddXxx` 保存

### 2.3 配置结构扩展

```go
type Config struct {
    ComposeDirs    []ComposeDir    `json:"compose_dirs"`
    DeployTargets  []DeployTarget  `json:"deploy_targets"`
    DeployPackages []DeployPackage `json:"deploy_packages"`
}

type DeployPackage struct {
    Name string `json:"name"`
    Path string `json:"path"`
}
```

## 3. 发布包配置

### 3.1 数据流

```
配置文件 → loadDeployCmd → DeployPackage[] → 前端发布视图列表
选中 + d → deployPackageCmd → runDeployFlowCmd(package.Path, target)
```

### 3.2 前端发布视图改造

当前 DeployView 只显示 `DeployTarget`。改造后分两区：
- **发布目标区** — 现有的 DeployTarget 列表
- **发布包区** — DeployPackage 列表，选中后按 `d` 部署到当前选中的目标

或者合并为一个列表，每个条目显示目标+包信息。

### 3.3 按键流程

```
DeployView:
  d → 检查是否有选中的目标和包
    → 有包: 直接用包路径部署
    → 无包: 打开 exec 面板输入路径（保持兼容）
```

## 4. 面板自适应

### 4.1 日志换行

`LogPanelModel.Render()` 中，每行日志根据 `lp.width` 换行：

```go
wrappedLines := WrapLines(line, lp.width-8) // 8 = 行号宽度 + padding
for _, wl := range wrappedLines {
    b.WriteString(lineNum + styles.LogLine.Render(wl) + "\n")
    lineNum = "      " // 续行不显示行号
}
```

`WrapLines` 已在 `views/helpers.go` 中实现。

### 4.2 visibleLines 重算

换行后，visibleLines 需要考虑实际渲染行数而非日志条目数。改为按渲染行数计算滚动。

### 4.3 操作记录优化

当前 `HistoryPanelModel.visibleLines()` 返回 `height-2`。增大信息密度：
- 减少行间距
- 详情区域（选中条目的完整信息）放在底部固定区域
- 列表区域占用更多空间

### 4.4 exec 面板动态布局

```go
func (p *ExecPanelImpl) Render() string {
    inputH := 4 // title + hint + input + blank
    if p.output != "" {
        outputH := min(strings.Count(p.output, "\n")+2, p.height-inputH-2)
        // 输出区域占用剩余空间
    }
    // 输入区域固定，输出区域自适应
}
```

## 5. 文件变更清单

| 文件 | 变更 |
|------|------|
| `config/config.go` | 新增 DeployPackage, AddDeployPackage, RemoveDeployPackage |
| `tui/messages.go` | 新增 DeployPackage 类型, FormMsg |
| `tui/panels_form.go` | 新增 FormPanel 实现 |
| `tui/panels_exec.go` | exec 面板动态布局 |
| `tui/views/detail_logs.go` | 日志换行支持 |
| `tui/views/history.go` | 操作记录展示优化 |
| `tui/views/containers.go` | 前端发布视图增加发布包区 |
| `tui/commands.go` | 新增 deployPackageCmd, loadDeployPackagesCmd |
| `tui/keys.go` | 设置页 a/e 按键改为打开 FormPanel |
| `tui/model.go` | currentItemCount/listStats 补充 |
