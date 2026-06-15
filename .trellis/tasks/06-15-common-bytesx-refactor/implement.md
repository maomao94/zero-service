# Implement: common bytex 重构&统一优化

## Execution Plan

### Step 1: bytex 新增验证函数
- [ ] 在 `common/bytex/bytex.go` 末尾添加：
  - `Uint32ToUint16WithValidate(v uint32) (uint16, error)`
  - `Uint32SliceToUint16SliceWithValidate(values []uint32) ([]uint16, int, error)`
  - `Uint32ToInt16WithValidate(v uint32) (int16, error)`
  - `Uint32SliceToInt16SliceWithValidate(values []uint32) ([]int16, int, error)`
- [ ] 从 `.trellis/spec/` 读取编码规范确认命名风格
- [ ] 运行 `go vet ./common/bytex/...` 验证

### Step 2: bytex README
- [ ] 在 `common/bytex/README.md` 记录所有导出函数的说明和示例
- [ ] 按功能分组：byte↔uint16、uint16↔int16、uint16↔uint32/int32、BinaryValues/BitValues、bool/bit、验证函数

### Step 3: 删除 tool/util.go 重复代码
- [ ] 删除 `common/tool/util.go` 中 300-468 行的 `BinaryValues`/`BitValues` struct 和 13 个函数
- [ ] 确保文件连接处无空行断层
- [ ] 运行 `go build ./common/tool/...` 验证

### Step 4: 删除 tool/idutil.go 死代码
- [ ] 删除 `common/tool/idutil.go:52-59` 的 `func (u *IdUtil) SimpleUUID()`
- [ ] 运行 `go vet ./common/tool/...` 验证

### Step 5: 重构 bridgemodbus 校验逻辑
- [ ] `writemultipleregisterslogic.go`: 替换手动 range 校验为 `bytex.Uint32SliceToUint16SliceWithValidate`
- [ ] `readwritemultipleregisterslogic.go`: 同上
- [ ] `writesingleregisterlogic.go`: 替换为 `bytex.Uint32ToUint16WithValidate`
- [ ] `maskwriteregisterlogic.go`: 替换两个掩码校验
- [ ] `batchconvertdecimaltoregisterlogic.go`: 替换有符号/无符号校验逻辑
- [ ] 运行 `go vet ./app/bridgemodbus/...` 和 `go build ./app/bridgemodbus/...` 验证

### Step 6: 质量验证
- [ ] `go vet ./...`
- [ ] `go build ./...`

### Step 7: 小包合并评估（可选）
- [ ] 评估 `carbonx/` 合并可行性
- [ ] 评估 `copierx/` 合并可行性
- [ ] 评估 `executorx/` 合并可行性
- [ ] 如需合并则执行，否则在 PRD 中标记
