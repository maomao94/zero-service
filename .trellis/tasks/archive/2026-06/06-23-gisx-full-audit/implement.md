# implement.md — gisx 整体优化实施清单

## Phase 1: 已知问题修复

### 1a. 导出 ValidationError
- [x] `validate.go`: `validationError` → `ValidationError`，字段 `msg` → `Msg`

### 1b. 修复 Centroid 错误信息
- [x] `geos/errors.go`: 新增 `ErrEmptyResult`
- [x] `geos/overlay.go`: `Centroid` nil 检查前置，结果空返回 `ErrEmptyResult`

### 1c. 统一 IsEmpty nil 行为
- [x] `geos/introspect.go`: `IsEmpty(nil)` 返回 `(false, errNil)`

### 1d. FenceInfo.Points → orb.Polygon
- [x] `store.go`: `FenceInfo.Points` 类型改 `orb.Polygon`
- [x] `store.go`: `CreateFence`, `LoadFencePolygon`, `UpdateFence` 签名改
- [x] `app/gis/model/fencestore.go`: 适配接口变更
- [x] `app/gis/internal/logic/listfenceslogic.go`: 适配字段类型变更

## Phase 2: 逻辑 bug 修复

### 2a. OrbRingToH3LatLng copy-before-append
- [x] `gisx.go`: 不修改入参 ring，先分配新 slice

### 2b. 删除 safeRunErr 死代码
- [x] `geos/context.go`: 确认 `safeRunErr` 被 `strtree.go` 使用（非死代码），保留

### 2c. 统一 nil 错误信息
- [x] `geos/predicate.go`: `predicateTwo` 用 `errNil` 替代 `fmt.Errorf`
- [x] `geos/relation.go`: `Relate`, `RelatePattern`, `HausdorffDistance`, `NearestPoints` 用 `errNil`

## Phase 3: 单测补全

### 3a. store_test.go (新增文件)
- [x] `TestNoopFenceStore_CreateFence`
- [x] `TestNoopFenceStore_LoadFencePolygon`
- [x] `TestNoopFenceStore_FindNearbyFenceIds`
- [x] `TestNoopFenceStore_FindFenceIdsByCellIds`
- [x] `TestNoopFenceStore_UpdateFence`
- [x] `TestNoopFenceStore_RemoveFence`
- [x] `TestNoopFenceStore_ListFences`
- [x] `TestNoopFenceStore_GetFence`
- [x] `TestErrFenceStoreNotImplemented`

### 3b. geos/ 补缺测试
- [x] `TestConstructEmpty` — 4 个 NewEmpty* 函数
- [x] `TestBuildArea`
- [x] `TestLineMerge`
- [x] `TestNode`
- [x] `TestDistance`
- [x] `TestSimplifyTopology`
- [x] `TestOffsetCurve`
- [x] `TestEndStartPoint`
- [x] `TestMinimumClearance`
- [x] `TestPrecision`
- [x] `TestRelatePattern`
- [x] `TestNearestPoints`
- [x] `TestPreparedContainsGeom`
- [x] `TestPreparedDisjoint`
- [x] `TestExtractMultiSafe`
- [x] `TestCrosses` 断言补全

### 3c. orbconv/ 补缺测试
- [x] `TestGeomToPoint`
- [x] `TestGeomToLineString`
- [x] `TestGeomToMultiPoint`
- [x] `TestGeomToMultiLineString`
- [x] `TestLineStringToGeom`
- [x] `TestMultiPointToGeom`
- [x] `TestMultiLineStringToGeom`
- [x] `TestConversionNilInputs` — 所有 GeomTo* nil 输入

### 3d. nil/error 边界测试
- [x] `TestNilInputs` — 全包 exported 函数 nil 输入验证
- [x] `TestErrorsSentinel` — 验证 ErrNil/ErrClosed/ErrEmptyRing 等哨兵错误

## Phase 4: 文档 & 最终验证

### 4a. doc.go 更新
- [x] 更新 validate.go 注释（ValidationError 引用）
- [x] 更新 store.go 注释（FenceInfo.Points 类型变更）
- [x] 更新 doc.go 包级文档

### 4b. 最终验证
- [x] `go test ./common/gisx/... -v -count=1`
- [x] `go test ./common/gisx/... -cover`
- [x] `go vet ./common/gisx/...`
- [x] 全项目编译验证（含 FenceStore 调用方）

## Phase 5: 第二轮审查修复

### 5a. 文档同步
- [x] `geos/doc.go`: 修正函数计数（10→12, 8→7, 12→11, 6→5, 66→96）
- [x] `gisx/doc.go`: 子包函数数 66→96
- [x] `geos/doc.go`: 修正错误处理文档（panic 恢复 vs 哨兵错误前缀区分）
- [x] `geos/introspect.go`: 修正废弃注释
- [x] `geos/overlay.go`: 修正废弃注释

### 5b. Bug 修复
- [x] `geos/construct.go`: `NewMultiPolygon` 加入 `ErrEmptyOuterRing` 守卫
- [x] `geos/extract.go`: nil 返回统一为 `errNil`（ExtractCoords/ExtractMulti/ExtractMultiSafe/ExtractPoints/ExtractPolygonOrMultiCoords）
- [x] `geos/introspect.go`: 移除 `IsEmpty` 冗余 nil 检查（oneAttr 已处理）

### 5c. 代码质量
- [x] `oneBool/oneFloat/oneInt` 合并为泛型 `oneAttr[T]`（context.go）
- [x] 测试 nil 断言强化（t.Logf → t.Error）
- [x] `TestNilInputs` 补充 relation.go 函数 nil 测试
- [x] `orbconv_test.go` 补充不支持类型 + 错误路径测试（~15 用例）

### 5d. 验证
- [x] `go test ./common/gisx/... -count=1 -cover` — 全部通过 (gisx:95.2%, geos:89.6%, orbconv:79.7%)
- [x] `go vet ./common/gisx/...` — 无警告
- [x] `go build ./common/gisx/... ./app/gis/...` — 编译通过
