# 实现 GIS proto 优化：代码层同步

## 设计

### 1. 解码接口精度/分辨率返回

| 接口 | 逻辑文件 | 变更 |
|------|---------|------|
| DecodeGeoHash | `decodegeohashlogic.go` | 在返回体中增加 `Precision: uint32(len(in.Geohash))` |
| DecodeH3 | `decodeh3logic.go` | 调用 `cell.Resolution()` 获取 H3 resolution 返回 |

`DecodeGeoHash` 的精度可以直接从入参字符串长度推断，无需额外计算。
`DecodeH3` 的 resolution 从 `h3.Cell.Resolution()` 获取，h3-go v4 API 返回 `int`，proto 期望 `uint32`，直接转型即可。

### 2. Fence 字段名变更 `id` → `fence_id`

`Fence` message 的 `Id` 字段在生成的 Go 代码中变为 `FenceId`：

- `pointinfencelogic.go`: `in.Fence.Id` → `in.Fence.FenceId`，两处引用
- `pointinfenceslogic.go`: `fence.Id` → `fence.FenceId`，四处引用

### 3. FenceDetail 类型变更 `int32` → `uint32`

`fenceInfoToDetail` 函数中移除 `int32()` 强转：

```go
// Before
H3Resolution:     int32(f.H3Resolution),
GeohashPrecision: int32(f.GeohashPrecision),
// After
H3Resolution:     f.H3Resolution,
GeohashPrecision: f.GeohashPrecision,
```

`gisx.FenceInfo` 的字段类型是 `uint32`，pb `FenceDetail` 现在也是 `uint32`，直接赋值即可。

### 4. 纯计算 RPC 移除 fence_id

`GenerateFenceCells` 和 `GenerateFenceH3Cells` 的 proto 已移除 `fence_id`，logic 层同时删除从 store 加载围栏的 else-if 分支：

- 移除 `} else if in.FenceId != "" { ... }` 整段
- 调整错误提示信息：只剩 `len(in.Points) > 0` 一条路径，points 为空时直接报参数缺失
- 这两个 RPC 现在是纯计算，不再有 store 依赖

### 5. 代码生成

```bash
cd app/gis && bash gen.sh
```

生成后 `gisserver.go`、`gis.pb.go`、`gis_grpc.pb.go` 全部随 proto 更新。goctl 生成的 `gisserver.go` 可能覆盖我们已编辑的逻辑引用深度，但 goctl 目前只管理 Server 注册，不管理 logic 本身，所以 logic 改动不受影响。

### 数据流

```
proto field change
  → gen.sh regenerate pb/grpc
  → Go struct field name auto-updates (Fence.Id → Fence.FenceId, etc.)
  → logic files reference new field names
  → go build validates all references
```
