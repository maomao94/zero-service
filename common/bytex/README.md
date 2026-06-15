# bytex

`common/bytex/` 是通用字节/寄存器工具包，提供字节与数值类型之间的转换、带范围校验的转换、以及位（bit/bool）操作。

## 背景知识

### Modbus 寄存器与字节

Modbus 协议中，**一个寄存器（Register）固定占 2 个字节（16 bit）**，传输时采用 **BigEndian（大端序）**：高字节在前，低字节在后。

```
字节流: [0x12, 0x34]
         ─┬─   ─┬─
          │     └── 低字节 (Low Byte)
          └──────── 高字节 (High Byte)

拼接为 16 位值: 0x1234 = 4660 (十进制)
```

读取多个寄存器时，返回的字节流按顺序拼接：`[reg0_hi, reg0_lo, reg1_hi, reg1_lo, ...]`，每 2 个字节对应一个寄存器。

### 无符号 (uint16) vs 有符号 (int16)

16 位寄存器的二进制位数固定为 16 bit，但**解释方式不同**：

| 类型 | 范围 | 位模式示例 `0xFF00` 解释为 |
|------|------|---------------------------|
| `uint16` (无符号) | 0 ~ 65535 | 65280 |
| `int16` (有符号) | -32768 ~ 32767 | -256 |

同一段字节数据，**无符号和有符号只是解读方式不同**，底层二进制完全一致。`BinaryValues` 结构体同时提供两种解读，方便按需使用。

```
字节: [0xFF, 0x00]
uint16: 65280   (0xFF00 = 255×256 + 0)
int16:  -256    (二进制补码: 0xFF00 → 取反+1 → -256)
```

### 线圈与离散输入 (Bit 操作)

线圈（Coil）和离散输入（Discrete Input）是 **1-bit** 的开关量，多个 bit 紧密排列在字节中：

```
字节 0x5A = 01011010 (二进制)
  bit0 = 0  ← LSB (Least Significant Bit)
  bit1 = 1
  bit2 = 0
  bit3 = 1
  bit4 = 1
  bit5 = 0
  bit6 = 1
  bit7 = 0  ← MSB

按 bit 展开为 bool 列表 (LSB-first): [false, true, false, true, true, false, true, false]
```

读取 N 个线圈返回的字节数 = `ceil(N / 8)`，每个字节存储 8 个开关状态。

### gRPC 与 Modbus 的类型差异

gRPC proto 通常使用 `uint32`/`int32`（32 位），Modbus 核心使用 `uint16`/`int16`（16 位）。两者之间需要转换：

```
Modbus (16bit) → gRPC (32bit):  值不变，类型拓宽    (uint16 → uint32)
gRPC (32bit) → Modbus (16bit):  需要范围校验，防止截断 (uint32 → uint16，校验 0~65535)
```

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

## 泛型工具

```go
// Integer 数值类型约束
type Integer interface {
    ~int16 | ~uint16 | ~int32 | ~uint32
}

// ConvertSlice 泛型切片转换，将 []From 转换为 []To
func ConvertSlice[From Integer, To Integer](values []From, convert func(From) To) []To
```

**示例：**

```go
// 将 []uint16 转换为 []uint32
uint32s := bytex.ConvertSlice(uint16s, func(v uint16) uint32 { return uint32(v) })

// 将 []int32 转换为 []int16（带截断）
int16s := bytex.ConvertSlice(int32s, func(v int32) int16 { return int16(v) })
```

## 函数一览

### 字节 ↔ uint16（BigEndian 寄存器解析）

Modbus 读取寄存器返回原始字节流，需要按 BigEndian 拼接为 16 位值。

```go
// 字节 → uint16：每 2 字节拼接一个寄存器值，奇数长度末尾补 0
// [0x12, 0x34, 0x56, 0x78] → [0x1234, 0x5678]
func BytesToUint16Slice(data []byte) []uint16

// uint16 → 字节：按 BigEndian 拆分为字节流
// [0x1234, 0x5678] → [0x12, 0x34, 0x56, 0x78]
func Uint16SliceToBytes(values []uint16) []byte
```

### uint16 ↔ int16（无符号/有符号解读）

同一段 16 位数据，按不同方式解读。不改变底层二进制。

```go
// 无符号 → 有符号：0xFF00 (65280) → -256
func Uint16ToInt16(u uint16) int16
func Uint16SliceToInt16Slice(values []uint16) []int16
```

### uint16 ↔ uint32/int32（gRPC 类型对齐）

gRPC proto 使用 32 位整数，Modbus 寄存器是 16 位。读取后需要拓宽类型以匹配 proto 定义。

```go
// Modbus → gRPC：值不变，类型拓宽
func Uint16ToUint32(u uint16) uint32     // uint16 → uint32
func Uint16ToInt32(u uint16) int32       // uint16 → int16 → int32（保留符号位）
func Uint16SliceToUint32Slice(values []uint16) []uint32
func Uint16SliceToInt32Slice(values []uint16) []int32
func Int16SliceToInt32Slice(values []int16) []int32

// gRPC → Modbus：类型截断（不校验，直接截断高 16 位）
func Uint32ToUint16(u uint32) uint16
func Int32ToInt16(i int32) int16
func Uint32SliceToUint16Slice(values []uint32) []uint16
func Int32SliceToInt16Slice(values []int32) []int16
```

### 带校验的转换（uint32 → uint16）

gRPC 写入 Modbus 前，必须校验值在 16 位范围内，否则会静默截断导致数据错误。

```go
// 单值校验，超出 [0, 65535] 返回 error
func Uint32ToUint16Validate(v uint32) (uint16, error)

// 批量校验，返回第一个出错值的索引（0-based）；无错误时 index = -1
func Uint32SliceToUint16SliceValidate(values []uint32) ([]uint16, int, error)

// 单值校验，超出 [0, 65535] 返回 error（结果为 int16，用于有符号场景）
func Uint32ToInt16Validate(v uint32) (int16, error)

// 批量校验，返回第一个出错值的索引（0-based）；无错误时 index = -1
func Uint32SliceToInt16SliceValidate(values []uint32) ([]int16, int, error)
```

### 带校验的转换（int32 → int16，有符号范围）

当输入是有符号整数（如温度、偏移量）时，校验范围为 [-32768, 32767]。

```go
// 单值校验，超出 [-32768, 32767] 返回 error
func Int32ToInt16Validate(v int32) (int16, error)

// 批量校验，返回第一个出错值的索引（0-based）；无错误时 index = -1
func Int32SliceToInt16SliceValidate(values []int32) ([]int16, int, error)
```

**示例：**

```go
// 写入前校验无符号值
values := []uint32{100, 200, 70000, 300}
uint16s, errIdx, err := bytex.Uint32SliceToUint16SliceValidate(values)
if err != nil {
    // errIdx = 2, err = "index 2: value 70000 exceeds uint16 range [0, 65535]"
    return fmt.Errorf("第 %d 个值超出范围: %w", errIdx+1, err)
}

// 写入前校验有符号值（如温度 -200 ~ 500）
signedVals := []int32{-200, 300, -40000}
int16s, errIdx, err := bytex.Int32SliceToInt16SliceValidate(signedVals)
if err != nil {
    // errIdx = 2, err = "index 2: value -40000 exceeds int16 range [-32768, 32767]"
    return fmt.Errorf("第 %d 个值超出范围: %w", errIdx+1, err)
}
```

### BinaryValues（寄存器多格式展示）

一次读取寄存器后，同时提供无符号、有符号、十六进制、二进制等多种展示格式，方便日志、调试和 API 响应。

```go
// 字节 → BinaryValues：从原始字节解析出所有格式
// [0x12, 0x34, 0xFF, 0x00] → {Hex: ["0x1234", "0xFF00"], Uint16: [4660, 65280], Int16: [4660, -256], ...}
func BytesToBinaryValues(data []byte) *BinaryValues

// uint16 数组 → BinaryValues：从已解析的寄存器值生成展示格式
func Uint16SliceToBinaryValues(values []uint16) *BinaryValues
```

### BitValues（线圈/离散输入位级展示）

线圈和离散输入是 1-bit 开关量，多个 bit 打包在字节中。

```go
// 字节 → bool 列表：按 bit 展开（LSB-first），quantity 指定有效 bit 数
// [0x5A], quantity=8 → [false, true, false, true, true, false, true, false]
func BytesToBools(data []byte, quantity int) []bool

// bool 列表 → 字节：按 bit 打包
// [false, true, false, true, true, false, true, false] → [0x5A]
func BoolsToBytes(bools []bool) []byte

// 字节 → BitValues：同时提供 bools 和 binary 展示
func BytesToBitValues(data []byte, quantity int) *BitValues

// bool 列表 → BitValues
func BoolsToBitValues(bools []bool) *BitValues
```

## 用法示例

### 读取保持寄存器（Function Code 0x03）

```go
import "zero-service/common/bytex"

// 读取 10 个保持寄存器，返回 20 字节
results, _ := mbCli.ReadHoldingRegisters(ctx, 0, 10)

// 解析为多格式展示
bv := bytex.BytesToBinaryValues(results)
fmt.Println(bv.Hex)      // [0x1234 0x5678 ...]
fmt.Println(bv.Uint16)   // [4660 22136 ...]     ← 无符号解读
fmt.Println(bv.Int16)    // [4660 22136 ...]     ← 有符号解读（值相同说明未溢出）
fmt.Println(bv.Binary)   // [0001001000110100 0101011001111000 ...]

// 转为 gRPC 响应类型 (uint32/int32)
resp.UintValues = bytex.Uint16SliceToUint32Slice(bv.Uint16)
resp.IntValues  = bytex.Int16SliceToInt32Slice(bv.Int16)
```

### 写入多个保持寄存器（Function Code 0x10）

```go
// gRPC 请求值为 []uint32，需校验后写入
input := []uint32{100, 200, 300}
regs, errIdx, err := bytex.Uint32SliceToUint16SliceValidate(input)
if err != nil {
    return fmt.Errorf("第 %d 个值超出范围: %w", errIdx+1, err)
}
bytes := bytex.Uint16SliceToBytes(regs)
mbCli.WriteMultipleRegisters(ctx, 0, uint16(len(regs)), bytes)
```

### 读取线圈（Function Code 0x01）

```go
// 读取 8 个线圈，返回 1 字节
coilBytes, _ := mbCli.ReadCoils(ctx, 0, 8)

// 展开为 bool 列表
bools := bytex.BytesToBools(coilBytes, 8)
fmt.Println(bools) // [false true false true true false true false]  ← 对应 0x5A 的各 bit

// 写入线圈前：bool 列表打包为字节
writeBytes := bytex.BoolsToBytes([]bool{true, false, true, true, false, false, false, true})
mbCli.WriteMultipleCoils(ctx, 0, 8, writeBytes)
```

### 有符号寄存器场景（温度、偏移量等）

```go
// 温度值可能为负（如 -200 表示 -20.0°C，精度 0.1）
signedVals := []int32{-200, 300, 150}
int16s, errIdx, err := bytex.Int32SliceToInt16SliceValidate(signedVals)
if err != nil {
    return fmt.Errorf("第 %d 个温度值超出 int16 范围: %w", errIdx+1, err)
}
regs := bytex.ConvertSlice(int16s, func(v int16) uint16 { return uint16(v) })
bytes := bytex.Uint16SliceToBytes(regs)
mbCli.WriteMultipleRegisters(ctx, 100, uint16(len(regs)), bytes)
```
