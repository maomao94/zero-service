# app/gis gRPC 业务逻辑优化

## Goal

对 `app/gis/` 目录下的 gRPC 业务逻辑执行 10 项代码优化，消除重复代码、统一校验行为、修复静默吞错误问题。

## 优化项

1. [x] 提取 CreateFence / UpdateFence 公共计算逻辑到 helper
2. [x] 抽取 computeGeohashCells / GenerateFenceCells 公共核心算法
3. [x] 移除 CreateFence / UpdateFence 冗余的多边形校验
4. [x] 修复 H3 resolution=0 语义歧义
5. [x] 统一 BatchTransformCoord / TransformCoord 的校验行为
6. [x] 修复 PointInFences 数组下标作为 fence_id fallback
7. [x] GenerateFenceCells GEOS 错误静默吞掉问题
8. [x] GenerateFenceCells 邻居扩展两阶段优化
9. [x] fencestore.go JSON 反序列化错误处理
10. [x] RoutePoints 球面距离 2-opt 说明

## Acceptance Criteria

- [ ] 所有提取的公共方法位于 `helper.go` 中
- [ ] 优化后业务逻辑正确性不变（编译通过）
- [ ] 无废弃/冗余代码残留
