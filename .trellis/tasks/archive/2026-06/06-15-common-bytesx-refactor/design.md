# Design: common bytex 重构&统一优化

## 1. bytex 新增函数设计

### 1.1 带校验的 uint32 → uint16 转换

```go
// Uint32ToUint16WithValidate 将 uint32 转换为 uint16，超出范围返回 error
func Uint32ToUint16(v uint32) (uint16, error)

// Uint32SliceToUint16SliceWithValidate 批量转换并校验范围 (0-65535)
// 超出范围时返回第一个出错值的索引和错误
func Uint32SliceToUint16SliceWithValidate(values []uint32) ([]uint16, int, error)

// Uint32ToInt16WithValidate 将 uint32 转换为有符号 int16 (-32768 ~ 32767)
func Uint32ToInt16(v uint32) (int16, error)

// Uint32SliceToInt16SliceWithValidate 批量转换并校验有符号范围
func Uint32SliceToInt16SliceWithValidate(values []uint32) ([]int16, int, error)
```

这些函数替代 bridgemodbus 中重复的 `v > 65535` 手工校验。

### 1.2 LittleEndian 支持（可选）

如需要，在 bytex 中新增：

```go
// BytesToUint16SliceLE LittleEndian 版本的字节转 uint16
func BytesToUint16SliceLE(data []byte) []uint16

// Uint16SliceToBytesLE LittleEndian 版本的 uint16 转字节
func Uint16SliceToBytesLE(values []uint16) []byte
```

## 2. 删除 tool/util.go 重复代码

`tool/util.go:300-468` 包含 `BinaryValues`/`BitValues` struct 和 13 个 bytes 转换函数，与 `bytex/bytex.go` 完全重复。

**删除策略**：直接删除这些行。研究已确认无外部调用者使用 `tool.BytesToUint16Slice` 等。

注意：`util.go` 在 `300行前` 还有 2 个函数 (`EstimateTokens`、`SimpleUUID`)，在 `468行后` 还有 token estimation 相关函数。删除后合并文件，确保中间无空行断层。

## 3. 删除 tool/idutil.go 死代码

删除 `tool/idutil.go:52-59` 中的 `func (u *IdUtil) SimpleUUID()`。

注意保留 `IdUtil` struct 定义和其他方法（如 `GenUUID`），只删除该单个方法。

## 4. bridgemodbus 重构

### 4.1 替换重复校验

| 文件 | 当前写法 | 替换为 |
|------|----------|--------|
| `writemultipleregisterslogic.go:42-51` | 手动 for range + if v > 65535 | `bytex.Uint32SliceToUint16SliceWithValidate` |
| `readwritemultipleregisterslogic.go:38-47` | 同上 | 同上 |
| `writesingleregisterlogic.go:38-42` | 单值校验 | `bytex.Uint32ToUint16WithValidate` |
| `maskwriteregisterlogic.go:37-43` | 双值校验 | 同上 ×2 |
| `batchconvertdecimaltoregisterlogic.go:32-56` | 有符号/无符号校验 + 转换 | `bytex.Uint32SliceToUint16SliceWithValidate` / `bytex.Uint32SliceToInt16SliceWithValidate` |

### 4.2 日志调整

部分文件在校验通过后直接使用 `binaryValues` 构造日志，重构后需保留日志行为。

## 5. 小包合并评估（可选）

- `carbonx/`（16 行）：单 `init()` 设置 carbon 时区。可合并到首个 import carbon 的包，或放入 `tool/`
- `copierx/`（56 行）：导出 `copier.Option{}` 变量。可合并到 `tool/`
- `executorx/`（44 行）：`ChunkMessagesPusher` 封装。可合并到 `tool/`
