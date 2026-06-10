# Technical Design

## Architecture Overview

修复四个独立问题，按优先级排序：

1. **Table panic** - bubbles/table 列初始化防护
2. **设置编辑** - 明确双模式设计
3. **Stats 历史** - 倒序+时间戳
4. **UI 优化** - 列宽计算改进

## D1: Table Panic 修复

### 根因分析

bubbles/table 的 `SetRows()` 会触发 `UpdateViewport()`，进而调用 `renderRow()`。如果此时 columns 为空 (长度 0)，访问 `m.columns[col]` 会 panic。

### 修复策略

所有 table 组件统一采用 **Eager Columns** 模式：

```go
func NewXxxTable() *XxxTable {
    t := &XxxTable{}
    t.model = table.New(
        table.WithColumns(defaultColumns()),  // 必须在这里初始化
        table.WithFocused(true),
        table.WithStyles(tableStyles()),
    )
    return t
}
```

`SetSize()` 更新列宽时，使用最小有效值保护：

```go
func (t *XxxTable) SetSize(width, height int) {
    if width < 1 { width = 1 }
    if height < 1 { height = 1 }
    // ...
}
```

### 影响范围

| 文件 | 修改内容 |
|------|---------|
| `pages/images/table.go` | NewImageTable 添加默认 columns |
| `pages/containers/table.go` | 同上 |
| `pages/compose/table.go` | 同上 |
| `pages/deploy/table.go` | 同上 |
| `pages/settings/table.go` | 已有 columns，无需修改 |

## D2: 设置编辑模式

### 当前行为

- `c` 键调用 `openConfigEditor()` → 外部 EDITOR
- `a` 键打开表单添加
- `d` 键删除选中项

### 改进方案

保持现有行为，但改进交互提示：

1. `c` 键弹出选择：`1. 表单编辑  2. JSON 编辑器`
2. 表单编辑复用现有 `FormPanel`，预填当前值
3. JSON 编辑器保持现有行为
4. 底部状态栏显示当前配置文件路径

### 数据流

```
用户按 c
  → 弹出选择面板
  → 选择"表单编辑" → 复用 FormPanel，onSubmit 调用 config.Save
  → 选择"JSON 编辑器" → 现有行为
  → 完成 → config.Load 刷新
```

## D3: Stats 历史改进

### 当前实现

```go
func renderHistory(history []dt.StatsEntry) string {
    show := history
    if len(history) > 10 {
        show = history[len(history)-10:]
    }
    for _, entry := range show {
        // 从旧到新，无时间戳
    }
}
```

### 改进方案

```go
func renderHistory(history []dt.StatsEntry) string {
    if len(history) == 0 {
        return "  (无历史数据)"
    }
    var b strings.Builder
    // 取最后 N 条，倒序遍历
    show := history
    if len(history) > 10 {
        show = history[len(history)-10:]
    }
    for i := len(show) - 1; i >= 0; i-- {
        entry := show[i]
        b.WriteString(fmt.Sprintf("  %s  CPU %.1f%%  MEM %s  NET ↑%s\n",
            entry.Timestamp.Format("15:04:05"),
            entry.CPUPercent,
            formatBytes(entry.MemUsage),
            formatBytes(entry.NetRx)))
    }
    return b.String()
}
```

### 数据来源

需要检查 `dt.StatsEntry` 是否有时间戳字段。如果没有，需要在流式解析时添加。

## D4: 窗口大小和 UI 列宽优化

### 问题

1. `ContainerPage.SetSize()` 只存储尺寸，不更新 table
2. `ImagePage.SetSize()` 只更新 table 尺寸，但不处理 width=0 的情况
3. 表格列宽计算使用固定百分比，小终端下可能：
   - 列宽为 0 或负数
   - 内容溢出
   - 表格无法渲染

### 改进策略

**ContainerPage 窗口大小处理**:
```go
func (p *ContainerPage) SetSize(w, h int) {
    p.width, p.height = w, h
    // 同步更新 table 尺寸
    if p.table.Model.Width() > 0 {
        p.table = NewContainerTable(w)
    }
}
```

**ImagePage 窗口大小处理**:
```go
func (p *ImagePage) SetSize(width, height int) {
    p.width = width
    p.height = height
    m := layout.Calculate(width, height)
    if m.Mode == layout.ModeDual {
        p.table.SetSize(m.ListW, m.BodyH)
        p.detail.SetSize(m.DetailW, m.BodyH)
        return
    }
    p.table.SetSize(m.ListW, m.BodyH)
    p.detail.SetSize(0, 0)
}
```

**列宽计算优化**:
```go
func imageColumns(width int) []table.Column {
    inner := width - 4
    if inner < 30 { inner = 30 }  // 最小宽度保护
    
    // 动态分配，确保每列至少有最小宽度
    repoW := max(10, inner*30/100)
    tagW := max(6, inner*15/100)
    idW := max(8, inner*15/100)
    sizeW := max(6, inner*15/100)
    createdW := max(10, inner-repoW-tagW-idW-sizeW)
    
    return []table.Column{
        {Title: "Repository", Width: repoW},
        {Title: "Tag", Width: tagW},
        {Title: "ID", Width: idW},
        {Title: "Size", Width: sizeW},
        {Title: "Created", Width: createdW},
    }
}
```

## Trade-offs

| 决策 | 理由 | 风险 |
|------|------|------|
| 保持外部编辑器选项 | 高级用户需要直接编辑 JSON | 无 |
| Stats 只显示最近 10 条 | 避免历史记录过长 | 用户可能想看更多 |
| 不重写整个 UI | 范围可控，风险低 | 无法彻底改善体验 |
