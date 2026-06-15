# common bytex 重构&统一优化

## Goal

将 `common/bytex/` 完善为核心字节工具包，消除 `tool/util.go` 中的重复代码，优化 `bridgemodbus` 中的重复校验逻辑，同时评估 `common/` 下其他小包的合并可行性。

## Requirements

### 1. bytex 完善
- `common/bytex/` 缺少 README，需补充函数说明和用法示例
- 添加带校验的 `uint32 → uint16` 转换函数（替换 bridgemodbus 中重复的 `v > 65535` 校验）
- 考虑添加 LittleEndian 支持（可选）

### 2. 消除 tool/util.go 中的字节函数重复
- `tool/util.go:300-468` 完整复制了 `bytex/bytex.go` 的 13 个函数 + 2 个 struct，需删除
- 确认无外部调用者引用 `tool.BytesToUint16Slice` 等（研究已确认无调用）

### 3. 消除 tool/idutil.go 中的死代码
- `func (u *IdUtil) SimpleUUID()` 方法从未被调用（所有调用者都使用 `tool.SimpleUUID()` 独立函数）
- 删除该方法

### 4. bridgemodbus 校验逻辑优化
- `WriteMultipleRegistersLogic`、`ReadWriteMultipleRegistersLogic`、`MaskWriteRegisterLogic`、`WriteSingleRegisterLogic` 中重复的 `v > 65535` 校验统一到 bytex
- `BatchConvertDecimalToRegisterLogic` 中的范围校验统一到 bytex

### 5. （可选）评估小包合并
- `carbonx/`（单个 init）、`copierx/`（单个 var）、`executorx/`（3 个函数）
- 如价值高则执行合并，否则记录待办

## Constraints

- 不改动 protobuf 定义和 gRPC 接口
- 不改动 `bytex/` 现有函数的签名（向后兼容）
- `tool/util.go` 删除重复代码后不影响其他包编译

## Acceptance Criteria

- [ ] `common/bytex/README.md` 包含所有导出函数的说明和示例
- [ ] `bytex` 新增 `Uint32SliceToUint16SliceWithValidate`（或类似）函数用于带范围校验的转换
- [ ] `tool/util.go` 不再包含 `BinaryValues`、`BitValues` 和任何字节转换函数
- [ ] `tool/idutil.go` 不再包含 `func (u *IdUtil) SimpleUUID()`
- [ ] bridgemodbus 中 `v > 65535` 的校验通过 bytex 统一完成
- [ ] `go vet ./...` 和 `go build ./...` 全部通过
- [ ] （可选）carbonx/copierx/executorx 评估并决定是否合并
