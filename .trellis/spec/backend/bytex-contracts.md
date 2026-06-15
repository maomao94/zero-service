# bytex 字节/寄存器工具包合约

> `common/bytex/` 是 Modbus 寄存器和字节操作的唯一工具包。详细函数说明和用法示例见 [`common/bytex/README.md`](../../../common/bytex/README.md)。

## When to read

- 处理 Modbus 寄存器读写（保持寄存器、输入寄存器、线圈、离散输入）。
- 字节流与数值类型之间的转换。
- gRPC proto `uint32`/`int32` 与 Modbus `uint16`/`int16` 之间的类型对齐。
- 带范围校验的数值转换。

## 核心合约

### 1. 字节序（Endianness）

Modbus 协议使用 **BigEndian（大端序）**：高字节在前，低字节在后。

```
字节流: [0x12, 0x34] → 16 位值: 0x1234 = 4660
```

`bytex` 所有函数默认 BigEndian。如需 LittleEndian，自行在调用方处理。

### 2. 有符号 vs 无符号

16 位寄存器的底层二进制固定，解读方式不同：

| 类型 | 范围 | `0xFF00` 解读为 |
|------|------|-----------------|
| `uint16` | 0 ~ 65535 | 65280 |
| `int16` | -32768 ~ 32767 | -256 |

`BinaryValues` 同时提供两种解读，调用方按业务语义选择。

### 3. 范围校验

gRPC → Modbus 写入前**必须校验**范围，否则静默截断导致数据错误：

```go
// 无符号：校验 [0, 65535]
uint16s, errIdx, err := bytex.Uint32SliceToUint16SliceValidate(uint32s)

// 有符号：校验 [-32768, 32767]
int16s, errIdx, err := bytex.Int32SliceToInt16SliceValidate(int32s)
```

`errIdx` 是第一个出错值的索引（0-based），无错误时为 -1。

### 4. 线圈/离散输入 Bit 打包

线圈和离散输入是 1-bit 开关量，按 LSB-first 打包在字节中：

```
字节 0x5A = 01011010 (二进制)
→ bool 列表: [false, true, false, true, true, false, true, false]
```

读取 N 个线圈返回 `ceil(N / 8)` 字节。

## 泛型工具

```go
// Integer 数值类型约束
type Integer interface {
    ~int16 | ~uint16 | ~int32 | ~uint32
}

// ConvertSlice 泛型切片转换
func ConvertSlice[From Integer, To Integer](values []From, convert func(From) To) []To
```

用于替代重复的 `XxxSliceToYyySlice` 函数模式。

## gRPC 错误包装约定

Logic 层从 `bytex` 校验函数获取 error 后，**必须包装为 ext 错误**再返回 gRPC：

```go
uint16s, errIdx, err := bytex.Uint32SliceToUint16SliceValidate(in.Values)
if err != nil {
    return nil, tool.NewErrorByPbCode(extproto.Code__1_01_PARAM,
        "第 %d 个值超过 16 位寄存器的最大值 (65535)", errIdx+1)
}
```

Wrong:

```go
// 直接返回 bytex 原始 error，gRPC 客户端无法识别项目错误码
return nil, err
```

Correct:

```go
// 包装为 ext 错误，gRPC 客户端可识别并展示
return nil, tool.NewErrorByPbCode(extproto.Code__1_01_PARAM, "第 %d 个值超出范围", errIdx+1)
```

## Validation & Error Matrix

| 条件 | 正确行为 |
|------|----------|
| uint32 值 > 65535 | `tool.NewErrorByPbCode(extproto.Code__1_01_PARAM, ...)` |
| int32 值超出 [-32768, 32767] | `tool.NewErrorByPbCode(extproto.Code__1_01_PARAM, ...)` |
| uint32 值在范围内 | 返回 `uint16`/`int16`，`errIdx = -1` |
| 批量校验中途出错 | 返回 `nil, errIdx, err`，`errIdx` 指向第一个出错值 |

## Tests Required

- `bytex.ConvertSlice` 单测：断言各类型组合的转换正确性。
- `bytex.Uint32SliceToUint16SliceValidate` 单测：边界值（0, 65535, 65536）和批量校验。
- `bytex.Int32SliceToInt16SliceValidate` 单测：边界值（-32768, 32767, -32769）和批量校验。
- `bytex.BytesToBinaryValues` 单测：BigEndian 解析、奇数字节长度。
- `bytex.BytesToBools`/`BoolsToBytes` 单测：bit 打包/解包。
