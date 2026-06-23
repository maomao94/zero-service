# gisx-整体优化：代码审查、文档补全、GEOS C API 指南

## Goal

对 `common/gisx/` 包进行全面代码审计和交付级优化：修复已知缺陷、补全单测、统一接口设计、完善文档，确保代码零 bug、可交付、可阅读。

## Requirements

### R1: 修复 4 个已知低优问题

1. **validationError 未导出** — 导出为 `ValidationError`，调用方可 `errors.As`
2. **Centroid 空几何错误信息不准** — 区分"输入 nil"和"运算结果空"，新增专用错误
3. **IsEmpty 对 nil 返回 (true,nil)** — 统一为 `(false, ErrNil)`，与 IsSimple/IsClosed/IsRing/HasZ 一致
4. **FenceInfo.Points []orb.Point 无法表达洞** — 改为 `orb.Polygon`，同步更新 FenceStore 接口及实现

### R2: 修复代码逻辑 bug

5. **OrbRingToH3LatLng 修改调用方数组** — `ring = append(ring, ring[0])` 可能覆盖，改为先 copy
6. **safeRunErr 死代码** — 删除未使用的函数
7. **nil 错误信息不一致** — `oneBool`/`predicateTwo` 统一使用 `errNil`

### R3: 补全单测覆盖率

当前 ~70.9% → 目标 ≥95%，共补 ~43 个函数/方法测试 + nil/error 边界测试

### R4: 文档补全

doc.go 更新、过期引用修复

## Acceptance Criteria

- [ ] 4 个低优问题 + 3 个逻辑 bug 全部修复
- [ ] `go test ./common/gisx/...` 全绿，覆盖率 ≥95%
- [ ] `go vet ./common/gisx/...` 无警告
- [ ] FenceStore 所有调用方编译通过
- [ ] 文档与实际代码一致

## Notes

- `FenceInfo.Points → orb.Polygon` 是 breaking change，需同步更新 `app/gis/model/fencestore.go`
- `orb.Polygon` 语义：polygon[0]=外环，polygon[1:]=洞
