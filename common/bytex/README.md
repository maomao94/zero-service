# bytex

`common/bytex/` 是通用字节/寄存器工具包，提供字节与数值类型之间的转换、带范围校验的转换、以及位（bit/bool）操作。

## 核心类型

```go
// BinaryValues 用于展示 Modbus 寄存器的多格式表示
type BinaryValues struct {
    Hex    []string // 16位十六进制，如 "0x1234"
    Uint16 []uint16 // 无符号16位（核心真值）
    Int16  []int16  // 有符号16位（由 uint16 转换）
    Bytes  []byte   // 原始字节（BigEndian）
    Binary []string // 16位二进制，如 "0001001000110100"
}

// BitValues 用于展示线圈/离散输入的位级表示
type BitValues struct {
    Bytes  []byte   // 原始字节数组
    Bools  []bool   // 每个元素对应一个 bit
    Binary []string // 8位二进制字符串，按 byte 打印
}
```

## 函数一览

### 字节 ↔ uint16（BigEndian）

```go
// 字节 → uint16，奇数长度末尾补0
func BytesToUint16Slice(data []byte) []uint16

// uint16 → 字节（BigEndian，高字节在前）
func Uint16SliceToBytes(values []uint16) []byte
```

### uint16 ↔ int16（有符号转换）

```go
func Uint16ToInt16(u uint16) int16
func Uint16SliceToInt16Slice(values []uint16) []int16
```

### uint16 ↔ uint32/int32（gRPC 对齐）

gRPC proto 通常使用 `uint32`/`int32`，Modbus 核心使用 `uint16`/`int16`。以下函数用于两者之间的转换（截断）。

```go
func Uint16ToUint32(u uint16) uint32
func Uint16ToInt32(u uint16) int32
func Uint16SliceToUint32Slice(values []uint16) []uint32
func Uint16SliceToInt32Slice(values []uint16) []int32
func Int16SliceToInt32Slice(values []int16) []int32

func Uint32ToUint16(u uint32) uint16
func Int32ToInt16(i int32) int16
func Uint32SliceToUint16Slice(values []uint32) []uint16
func Int32SliceToInt16Slice(values []int32) []int16
```

### 带校验的转换（uint32 → uint16 / int16）

当 `uint32` 值可能超出 `uint16` 范围时，使用带校验的版本。

```go
// 单值校验，超出 [0, 65535] 返回 error
func Uint32ToUint16Validate(v uint32) (uint16, error)

// 批量校验，返回第一个出错值的索引（0-based）；无错误时 index = -1
func Uint32SliceToUint16SliceValidate(values []uint32) ([]uint16, int, error)

// 单值校验，超出 [-32768, 32767]（即 uint32 > 65535）返回 error
func Uint32ToInt16Validate(v uint32) (int16, error)

// 批量校验，返回第一个出错值的索引（0-based）；无错误时 index = -1
func Uint32SliceToInt16SliceValidate(values []uint32) ([]int16, int, error)
```

**示例：**

```go
values := []uint32{100, 200, 70000, 300}
uint16s, errIdx, err := bytex.Uint32SliceToUint16SliceValidate(values)
if err != nil {
    // errIdx = 2, err = "index 2: value 70000 exceeds uint16 range [0, 65535]"
    return fmt.Errorf("第 %d 个值超出范围: %w", errIdx+1, err)
}
// uint16s = []uint16{100, 200, 70000, 300} 不会执行到这里
```

### BinaryValues（寄存器多格式展示）

```go
// 字节 → BinaryValues（含 hex、uint16、int16、binary 展示）
func BytesToBinaryValues(data []byte) *BinaryValues

// uint16 数组 → BinaryValues（同时反向生成字节）
func Uint16SliceToBinaryValues(values []uint16) *BinaryValues
```

### BitValues（线圈/离散输入位级展示）

```go
// 字节 → bool 列表（按 bit 展开，LSB-first）
func BytesToBools(data []byte, quantity int) []bool

// bool 列表 → 字节（按 bit 打包）
func BoolsToBytes(bools []bool) []byte

// 字节 → BitValues（含 bools 和 binary 展示）
func BytesToBitValues(data []byte, quantity int) *BitValues

// bool 列表 → BitValues
func BoolsToBitValues(bools []bool) *BitValues
```

## 用法示例

```go
import "zero-service/common/bytex"

// 读取 Modbus 保持寄存器后解析
results, _ := mbCli.ReadHoldingRegisters(ctx, 0, 10)
bv := bytex.BytesToBinaryValues(results)
fmt.Println(bv.Hex)      // [0x1234 0x5678 ...]
fmt.Println(bv.Uint16)   // [4660 22136 ...]
fmt.Println(bv.Int16)    // [4660 22136 ...]
fmt.Println(bv.Binary)   // [0001001000110100 0101011001111000 ...]

// 写入前：uint32 → uint16 校验
input := []uint32{100, 200, 300}
regs, errIdx, err := bytex.Uint32SliceToUint16SliceValidate(input)
if err != nil {
    log.Fatalf("第 %d 个值超出范围: %v", errIdx+1, err)
}
bytes := bytex.Uint16SliceToBytes(regs)
mbCli.WriteMultipleRegisters(ctx, 0, uint16(len(regs)), bytes)

// 读取线圈后：字节 → bool 列表
coilBytes, _ := mbCli.ReadCoils(ctx, 0, 8)
bools := bytex.BytesToBools(coilBytes, 8)
fmt.Println(bools) // [true false true ...]
```
