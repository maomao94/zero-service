# grpc coordinate order swap

## Goal

将 `proto` 及 gRPC 层的坐标点字段统一为 **经度在前，纬度在后**，与 GeoJSON/WKT/PostGIS 等业界标准保持一致。

## Requirements

1. 修改 `app/gis/gis.proto` 的 `Point` message，让 `lon` 在 `lat` 前面表示
2. **field number 不能变**（`lon=2, lat=1`），保持 wire format 向后兼容，避免已序列化数据不需要反序列化失败
3. proto 注释同步更新
4. 检查所有 gRPC 调用方和 logic 层，确认硬编码的 `lat/lon` 顺序没有隐含依赖

## Non-Requirements

- 不修改 H3 或 geohash 底层库本身（它们内部仍按 `lat, lon` 传入）
- 不修改已有的持久化数据（wire format 不变所以不需要迁移）
- 不修改 GeoJSON 或其他外部格式转换时的逻辑

## Acceptance Criteria

- [ ] `gis.proto` 中 `Point message` 字段声明顺序改为 `lon=2, lat=1`，且 wire format 不变
- [ ] `go generate` / protoc 重新生成后代码编译通过
- [ ] 已存在的单元测试全部通过
- [ ] 所有在 gRPC 层 `Point` 的构造/解析处顺序语义确认无误

## Notes

- wire format 绑定 field number，只要 number 不变（`lat=1, lon=2`），proto source 顺序调整就是源码级约定，不影响序列化
- Go 生成的 struct 字段顺序由 field number 决定（先 1 后 2），所以 `Lat` 仍会在 `Lon` 前面，但这只是 Go struct 内部布局，不对外暴露
