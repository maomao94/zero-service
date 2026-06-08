# 修复 IEC 104 文档与代码不一致问题

## Goal

修复 `docs/iec104-protocol.md` 和 `docs/iec104.md` 中与实际代码不一致的问题，确保文档准确反映当前实现。

## Requirements

### P0 - 高优先级

1. **响应类型文档错误**
   - 位置：`iec104-protocol.md` Section 7.1, 7.13
   - 问题：文档说"所有控制命令返回空的 SendCommandRes"，但 proto 定义了带 `value` 字段的响应
   - 修复：更新文档，说明响应包含从站回显值

2. **缺少 `enable_raw_insert` 字段**
   - 位置：`iec104-protocol.md` Section 1.4
   - 问题：`device_point_mapping` 表定义缺少该字段
   - 修复：添加字段说明

### P1 - 中优先级

3. **Proto 与 SQL Schema 不同步**
   - Proto `PbDevicePointMapping` 缺少 `description` 和 `enableRawInsert` 字段
   - 决策点：是否更新 proto 或在文档中说明差异

4. **端口类型不一致**
   - Proto 中 `port` 字段混用 `int32` 和 `uint32`
   - 建议统一为 `uint32`

### P2 - 低优先级

5. **新增响应字段未文档化**
   - `ClearPointMappingCacheRes.clearedCount`
   - `ClearPointMappingCacheReq.keyInfos`

## Acceptance Criteria

- [ ] `iec104-protocol.md` Section 7.1, 7.13 正确描述响应类型
- [ ] `device_point_mapping` 表定义包含 `enable_raw_insert` 字段
- [ ] Proto 字段与文档一致（或文档明确说明差异）
- [ ] 端口类型统一
- [ ] 所有新增字段有文档说明

## Reference Files

- `docs/iec104-protocol.md`
- `docs/iec104.md`
- `app/ieccaller/ieccaller.proto`
- `common/iec104/types/types.go`

## Notes

- 这是一个文档修复任务，主要修改 markdown 文件
- 涉及 proto 文件修改时需要重新生成代码
