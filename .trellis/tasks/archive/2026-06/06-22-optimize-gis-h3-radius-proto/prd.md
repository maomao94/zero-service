# 优化 GIS H3 与半径命中接口

## Goal

优化 GIS proto 契约，明确半径命中结果、多精度编码和 H3 GridDisk 周边格子接口，为后续实现阶段提供稳定输入输出定义。

## Requirements

- `PointsWithinRadius` 返回命中点下标和中心点到命中点的米级距离，不返回点坐标，避免响应数据量偏大。
- 保留现有单精度 `EncodeGeoHash` 和 `EncodeH3` 接口，新增多精度编码接口，命名需符合 Go/proto 风格且与现有接口保持一致。
- 新增 H3 官方语义一致的 `GridDisk` 接口，用于按 H3 origin index 返回周围指定圈数内的 H3 cells。
- 如保留经纬度便利入口，使用独立 `GridDiskByPoint` RPC，避免单个 request 同时包含 `h3_index` 和 `point` 两种主输入。
- `GridDisk` 返回 H3 cell 和圈数语义，字段命名避免让调用方误解为米级距离。
- 本阶段只编写 `app/gis/gis.proto`，不生成代码、不实现 logic；后续编码前由用户确认 proto。

## Acceptance Criteria

- [x] `PointsWithinRadiusRes` 使用精简命中结构表达 `index` 和 `distance_meters`。
- [x] proto 包含 `EncodeGeoHashMulti` 和 `EncodeH3Multi` 多精度编码接口。
- [x] proto 包含 `GridDisk` RPC，使用 `h3_index + k` 获取 origin 周围 cells。
- [x] proto 包含 `GridDiskByPoint` RPC，使用 `point + resolution + k` 获取 origin 周围 cells。
- [x] `GridDisk` 响应字段使用 `ring` 表达 H3 圈数，不使用易误解的地理距离字段名。
- [x] 只修改了 proto、生成代码和 logic 层，未改动无关业务上下文。

## Notes

- Keep `prd.md` focused on requirements, constraints, and acceptance criteria.
- Lightweight tasks can remain PRD-only.
- For complex tasks, add `design.md` for technical design and `implement.md` for execution planning before `task.py start`.
