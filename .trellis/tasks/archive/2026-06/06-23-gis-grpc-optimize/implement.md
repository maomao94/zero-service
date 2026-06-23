# implement.md — 执行计划

## 执行顺序

### 1. 抽取公共方法到 helper

**步骤 1.1：** CreateFence / UpdateFence 公共 H3+geohash 计算 → `helper.go`
- 提取 `computeFenceCells(polygon orb.Polygon, h3Resolution, geohashPrecision int) (h3Cells, h3CellStrings, geohashes []string, err error)`
- create 和 update 调用此方法
- 涉及文件：`createfencelogic.go`, `updatefencelogic.go`, `helper.go`

**步骤 1.2：** computeGeohashCells / GenerateFenceCells 公共核心 → `helper.go`
- 提取 `scanGeohashCells(orb.Polygon, precision int, ...) (map[string]struct{}, error)` 含 logger 和邻居参数
- 两个调用方各自适配
- 涉及文件：`generatefencecellslogic.go`, `helper.go`

**步骤 1.3：** 移除 CreateFence / UpdateFence 冗余多边形校验
- 删掉 `ValidatePoints(in.Points...)` 和 `len(in.Points) < 3`
- 涉及文件：`createfencelogic.go`, `updatefencelogic.go`

### 2. 修复行为不一致

**步骤 2.1：** 修复 H3 resolution=0 语义
- `createfencelogic.go` 和 `updatefencelogic.go` 中 `resolution <= 0` → `resolution == 0` 去掉默认逻辑
- 涉及文件：`createfencelogic.go`, `updatefencelogic.go`

**步骤 2.2：** BatchTransformCoord 增加 source_type/target_type 校验
- 复用 `transformcoordlogic.go` 中的 `validateReq` 或提取公共校验
- 涉及文件：`batchtransformcoordlogic.go`, `transformcoordlogic.go`

**步骤 2.3：** PointInFences 去掉下标 fallback
- fence_id 为空时跳过此 fence，不放入结果
- 涉及文件：`pointinfenceslogic.go`

### 3. 修复错误处理

**步骤 3.1：** GenerateFenceCells GEOS 错误处理
- 收集错误，最终返回 error 而非 continue 静默跳过
- 涉及文件：`generatefencecellslogic.go`

**步骤 3.2：** fencestore.go JSON 反序列化错误处理
- `_ = json.Unmarshal` → 记录日志或返回 error
- 涉及文件：`model/fencestore.go`

### 4. 性能优化

**步骤 4.1：** GenerateFenceCells 邻居扩展两阶段
- 先扫完全部命中格子，再统一扩展邻居
- 涉及文件：`generatefencecellslogic.go`

## 验证

- `go build ./app/gis/...`
- `go vet ./app/gis/...`
- `go test ./app/gis/...`
