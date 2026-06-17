# go-zero mr 并发工具使用规范

> `mr.Finish` / `mr.FinishVoid` / `mr.MapReduce` 并发任务编排的 canonical source，覆盖签名、选型、错误处理和常见模式。

## When to read

- 并行执行多个独立查询或操作。
- 列表接口需要批量关联数据补充。
- 需要轻量级并发替代 antsx。

## 核心签名

```go
func Finish(fns ...func() error) error
func FinishVoid(fns ...func())
func MapReduce[T, U, V any](generate GenerateFunc[T], mapper MapperFunc[T, U], reducer ReducerFunc[U, V], opts ...Option) (V, error)
func ForEach[T any](generate GenerateFunc[T], mapper ForEachFunc[T], opts ...Option)
```

## 选型

| 场景 | 用哪个 |
| --- | --- |
| 并行执行多个独立操作，任一失败则取消 | `mr.Finish` |
| 并行执行多个独立操作，不需要结果 | `mr.FinishVoid` |
| 数据源 → 并行处理 → 聚合结果 | `mr.MapReduce` |
| 数据源 → 并行处理 → 无输出 | `mr.ForEach` |
| 需要控制并发数 | `mr.WithWorkers(n)` |
| 需要 context 取消 | `mr.WithContext(ctx)` |

## 与 antsx 对比

| 维度 | `mr.Finish` | `antsx.Invoke` |
| --- | --- | --- |
| 依赖 | go-zero 核心包 | common/antsx 扩展包 |
| 适用场景 | 轻量级并行任务 | 需要 Reactor、ReplyPool 等高级特性 |
| 错误处理 | 任一失败返回第一个错误 | `errors.Join` 聚合所有错误 |
| 取消语义 | 自动取消其他任务 | fast-fail + ctx.Done() |
| 结果收集 | 需要外部变量 + mutex | 返回 `[]SettledResult` |

**选择建议**：
- 简单并行查询 → `mr.Finish`
- 需要所有结果（成功/失败） → `antsx.InvokeAllSettled`
- 需要并发控制 → `mr.WithWorkers` 或 `antsx.WithReactor`

## 常见模式

### 模式1：分页 + 并发关联补充（推荐）

```go
// 1. 分页查询主表
pageResult, err := gormx.QueryPage(db.Order("id DESC"), page, pageSize, &devices)
if err != nil {
    return nil, err
}

// 2. for 循环每个设备，并发查询关联数据
list := make([]*DeviceListItem, 0, len(devices))
for i := range devices {
    sn := devices[i].DeviceSn
    gw := devices[i].GatewaySn
    item := &DeviceListItem{Device: toDeviceInfo(&devices[i])}

    var osd *OsdSnapshot
    var state *StateSnapshot
    var topos []Topo
    var task *TaskState

    mr.Finish(
        func() error {
            // 查询失败静默跳过，关联数据非必须
            db.Where("device_sn = ?", sn).First(&osd)
            return nil
        },
        func() error {
            db.Where("device_sn = ?", sn).First(&state)
            return nil
        },
        func() error {
            db.Where("gateway_sn = ? OR sub_device_sn = ?", sn, sn).Find(&topos)
            return nil
        },
        func() error {
            if gw != "" {
                db.Where("gateway_sn = ?", gw).First(&task)
            }
            return nil
        },
    )

    // 组装结果
    if osd != nil { item.Osd = toBrief(osd) }
    if state != nil { item.State = toBrief(state) }
    if len(topos) > 0 { item.Topo = toList(topos) }
    if task != nil { item.TaskState = toInfo(task) }
    list = append(list, item)
}
```

**优点**：代码简洁，不需要 map/mutex，关联查询失败不影响主数据返回。

### 模式2：并行批量查询 + 结果收集

适用于需要批量查询优化的场景（如减少 SQL 次数）：

```go
var (
    osdMap   map[string]*OsdSnapshot
    stateMap map[string]*StateSnapshot
    mu       sync.Mutex
)
if err := mr.Finish(
    func() error {
        items, err := queryOsd(db, deviceSnList)
        if err != nil {
            return err
        }
        mu.Lock()
        osdMap = items
        mu.Unlock()
        return nil
    },
    func() error {
        items, err := queryState(db, deviceSnList)
        if err != nil {
            return err
        }
        mu.Lock()
        stateMap = items
        mu.Unlock()
        return nil
    },
); err != nil {
    return nil, err
}
```

### 模式3：分页优化（count=0 提前返回）

```go
func QueryPage[T any](db *gorm.DB, page, pageSize int, dest *[]T) (*PageResult[T], error) {
    page, pageSize = NormalizePage(page, pageSize)
    var total int64
    if err := db.Count(&total).Error; err != nil {
        return nil, err
    }
    if total == 0 {
        return NewPageResult([]T{}, 0, page, pageSize), nil  // 提前返回，省一次查询
    }
    // ...
}
```

## 错误处理

- `mr.Finish` 任一失败返回第一个错误，其他任务自动取消。
- 需要所有结果时，用 `sync.Mutex` 收集到外部变量。
- 不要在 `mr.Finish` 内部做 panic 恢复，go-zero 已内置。

## 注意事项

- `mr.Finish` 内部的 goroutine 共享外部 context，确保 DB 查询使用 `WithContext`。
- 收集结果时必须加 `sync.Mutex`，因为 mapper 在不同 goroutine 执行。
- `mr.Finish` 第一个任务在当前 goroutine 同步执行，其余并发。
- 空任务列表直接返回，不会 panic。
