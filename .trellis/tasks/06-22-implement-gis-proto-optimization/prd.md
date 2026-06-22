# 实现 GIS proto 优化：代码层同步

## Goal

将上一轮 `gis.proto` 契约优化的所有变更同步到 Go 逻辑层（logic）、辅助函数和生成代码，确保编译通过、语义正确。

## Requirements

- 解码接口补回精度/分辨率字段：`DecodeGeoHashRes.precision`、`DecodeH3Res.resolution`
- `Fence` message 字段名 `id` → `fence_id` 同步到所有引用点（`pointinfencelogic`、`pointinfenceslogic`）
- `FenceDetail` 的 `h3_resolution`/`geohash_precision` 类型 `int32` → `uint32` 同步到 `fenceInfoToDetail`
- `GenFenceCellsReq`、`GenFenceH3CellsReq` 移除 `fence_id` 字段，纯计算 RPC 不再支持按 ID 从 store 加载围栏
- 执行 `gen.sh` 重新生成 pb/grpc 代码，确保通过 `go build`
- 不改动单元测试范围（可修改测试代码适配字段名变化）

## Acceptance Criteria

- [ ] `DecodeGeoHash` 返回的响应包含 `precision`（等于输入 geohash 长度）
- [ ] `DecodeH3` 返回的响应包含 `resolution`（等于输入 H3 index 的 resolution）
- [ ] `PointInFence` / `PointInFences` 引用 `Fence.FenceId` 而非 `Fence.Id`
- [ ] `fenceInfoToDetail` 不再使用 `int32()` 强转精度/分辨率字段
- [ ] `GenerateFenceCells` / `GenerateFenceH3Cells` 不再引用 `in.FenceId`
- [ ] `gen.sh` 执行后 `gisserver.go` 及 pb 文件与最新 proto 一致
- [ ] `go build ./app/gis/...` 编译通过

## Notes

- 本阶段只改 logic 层、helper 函数和生成代码（`gen.sh`），不涉及 model/store 层
- `go.sum` / `go.mod` 不需要变更
