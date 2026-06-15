package bytex

import (
	"fmt"
)

// Integer 数值类型约束，用于泛型切片转换
type Integer interface {
	~int16 | ~uint16 | ~int32 | ~uint32
}

// ConvertSlice 泛型切片转换，将 []From 转换为 []To
func ConvertSlice[From Integer, To Integer](values []From, convert func(From) To) []To {
	result := make([]To, len(values))
	for i, v := range values {
		result[i] = convert(v)
	}
	return result
}

// 字节 → uint16（核心） → int16（有符号）
type BinaryValues struct {
	Hex    []string `json:"hex"`    // 16位十六进制
	Uint16 []uint16 `json:"uint16"` // 核心：无符号16位（唯一真值）
	Int16  []int16  `json:"int16"`  // 有符号16位（由uint16转换）
	Bytes  []byte   `json:"bytes"`  // 原始字节（源头）
	Binary []string `json:"binary"` // 16位二进制
}

type BitValues struct {
	Bytes  []byte   `json:"bytes"`  // 原始字节数组
	Bools  []bool   `json:"bools"`  // 每个元素对应一个线圈/bit
	Binary []string `json:"binary"` // 可读二进制字符串，每个 byte
}

// ------------------------------
// 字节 → uint16
// ------------------------------
func BytesToUint16Slice(data []byte) []uint16 {
	n := (len(data) + 1) / 2
	result := make([]uint16, 0, n)

	for i := 0; i < n; i++ {
		idx := i * 2
		if idx+1 < len(data) {
			// 常规组合：高8位 + 低8位
			result = append(result, uint16(data[idx])<<8|uint16(data[idx+1]))
		} else {
			// 奇数长度：最后一个字节补 0
			result = append(result, uint16(data[idx])<<8)
		}
	}
	return result
}

// ------------------------------
// uint16 → 字节
// ------------------------------
func Uint16SliceToBytes(values []uint16) []byte {
	bytes := make([]byte, len(values)*2)
	for i, v := range values {
		bytes[2*i] = byte(v >> 8)
		bytes[2*i+1] = byte(v & 0xFF)
	}
	return bytes
}

// ------------------------------
// uint16 ↔ int16（有符号负数转换）
// ------------------------------
func Uint16ToInt16(u uint16) int16 {
	return int16(u)
}

func Uint16SliceToInt16Slice(values []uint16) []int16 {
	return ConvertSlice(values, Uint16ToInt16)
}

// ------------------------------
// uint16 → uint32 / int32
// 给 grpc 对接用，不污染核心结构
// ------------------------------
func Uint16ToUint32(u uint16) uint32 {
	return uint32(u)
}

func Uint16ToInt32(u uint16) int32 {
	return int32(int16(u))
}

func Uint16SliceToUint32Slice(values []uint16) []uint32 {
	return ConvertSlice(values, Uint16ToUint32)
}

func Uint16SliceToInt32Slice(values []uint16) []int32 {
	return ConvertSlice(values, Uint16ToInt32)
}

func Int16SliceToInt32Slice(values []int16) []int32 {
	return ConvertSlice(values, func(v int16) int32 { return int32(v) })
}

// ------------------------------
// uint32 / int32 → uint16 / int16
// 从 gRPC 类型转换回核心类型
// ------------------------------
func Uint32ToUint16(u uint32) uint16 {
	return uint16(u)
}

func Int32ToInt16(i int32) int16 {
	return int16(i)
}

func Uint32SliceToUint16Slice(values []uint32) []uint16 {
	return ConvertSlice(values, Uint32ToUint16)
}

func Int32SliceToInt16Slice(values []int32) []int16 {
	return ConvertSlice(values, Int32ToInt16)
}

// ------------------------------
// 字节 → 完整 BinaryValues
// ------------------------------
func BytesToBinaryValues(data []byte) *BinaryValues {
	uint16Vals := BytesToUint16Slice(data)
	int16Vals := Uint16SliceToInt16Slice(uint16Vals)
	n := len(uint16Vals)

	hexVals := make([]string, n)
	binVals := make([]string, n)

	for i := range uint16Vals {
		val := uint16Vals[i]
		hexVals[i] = fmt.Sprintf("0x%04X", val)
		binVals[i] = fmt.Sprintf("%016b", val)
	}

	if len(hexVals) != n || len(binVals) != n {
		panic("BinaryValues: length mismatch")
	}

	return &BinaryValues{
		Hex:    hexVals,
		Uint16: uint16Vals,
		Int16:  int16Vals,
		Bytes:  data,
		Binary: binVals,
	}
}

// ------------------------------
// uint16 数组 → BinaryValues
// ------------------------------
func Uint16SliceToBinaryValues(values []uint16) *BinaryValues {
	int16Vals := Uint16SliceToInt16Slice(values)
	n := len(values)

	hexVals := make([]string, n)
	binVals := make([]string, n)

	for i := range values {
		val := values[i]
		hexVals[i] = fmt.Sprintf("0x%04X", val)
		binVals[i] = fmt.Sprintf("%016b", val)
	}

	if len(hexVals) != n || len(binVals) != n {
		panic("BinaryValues: length mismatch")
	}
	return &BinaryValues{
		Hex:    hexVals,
		Uint16: values,
		Int16:  int16Vals,
		Bytes:  Uint16SliceToBytes(values),
		Binary: binVals,
	}
}

// ------------------------------------------------------
// 字节 ↔ 布尔位
// ------------------------------------------------------
func BytesToBools(data []byte, quantity int) []bool {
	bools := make([]bool, quantity)
	for i := 0; i < quantity; i++ {
		byteIndex := i / 8
		bitIndex := i % 8
		bools[i] = (data[byteIndex] & (1 << bitIndex)) != 0
	}
	return bools
}

// ------------------------------------------------------
// 布尔位 ↔ 字节
// ------------------------------------------------------
func BoolsToBytes(bools []bool) []byte {
	// 将 repeated bool 转换为 bytes
	// 每字节包含 8 个线圈的状态，按位存储
	// (count + N - 1) / N
	n := (len(bools) + 7) / 8
	data := make([]byte, n)
	for i, b := range bools {
		if b {
			data[i/8] |= 1 << (i % 8)
		}
	}
	return data
}

func BytesToBitValues(data []byte, quantity int) *BitValues {
	bools := BytesToBools(data, quantity)
	n := len(data)
	binaryStr := make([]string, n)
	for i, b := range data {
		binaryStr[i] = fmt.Sprintf("%08b", b) // 按 byte 打印 LSB-first
	}
	return &BitValues{
		Bytes:  data,
		Bools:  bools,
		Binary: binaryStr,
	}
}

func BoolsToBitValues(bools []bool) *BitValues {
	bytes := BoolsToBytes(bools)
	return BytesToBitValues(bytes, len(bools))
}

// ------------------------------------------------------
// 带校验的 uint32 → uint16 / int16 转换
// ------------------------------------------------------

// Uint32ToUint16Validate 将 uint32 转换为 uint16，超出 [0, 65535] 范围返回 error。
func Uint32ToUint16Validate(v uint32) (uint16, error) {
	if v > 65535 {
		return 0, fmt.Errorf("value %d exceeds uint16 range [0, 65535]", v)
	}
	return uint16(v), nil
}

// Uint32SliceToUint16SliceValidate 批量将 uint32 转换为 uint16。
// 超出范围时返回第一个出错值的索引（0-based）和 error。
func Uint32SliceToUint16SliceValidate(values []uint32) ([]uint16, int, error) {
	result := make([]uint16, len(values))
	for i, v := range values {
		if v > 65535 {
			return nil, i, fmt.Errorf("index %d: value %d exceeds uint16 range [0, 65535]", i, v)
		}
		result[i] = uint16(v)
	}
	return result, -1, nil
}

// Uint32ToInt16Validate 将 uint32 转换为有符号 int16，超出 [-32768, 32767] 范围返回 error。
func Uint32ToInt16Validate(v uint32) (int16, error) {
	// 有符号范围：0 ~ 32767 直接转，32768 ~ 65535 映射为 -32768 ~ -1
	if v > 65535 {
		return 0, fmt.Errorf("value %d exceeds int16 range [-32768, 32767]", v)
	}
	return int16(v), nil
}

// Uint32SliceToInt16SliceValidate 批量将 uint32 转换为有符号 int16。
// 超出范围时返回第一个出错值的索引（0-based）和 error。
func Uint32SliceToInt16SliceValidate(values []uint32) ([]int16, int, error) {
	result := make([]int16, len(values))
	for i, v := range values {
		if v > 65535 {
			return nil, i, fmt.Errorf("index %d: value %d exceeds int16 range [-32768, 32767]", i, v)
		}
		result[i] = int16(v)
	}
	return result, -1, nil
}

// ------------------------------------------------------
// 带校验的 int32 → int16 转换（有符号范围）
// ------------------------------------------------------

// Int32ToInt16Validate 将 int32 转换为 int16，超出 [-32768, 32767] 范围返回 error。
func Int32ToInt16Validate(v int32) (int16, error) {
	if v > 32767 || v < -32768 {
		return 0, fmt.Errorf("value %d exceeds int16 range [-32768, 32767]", v)
	}
	return int16(v), nil
}

// Int32SliceToInt16SliceValidate 批量将 int32 转换为 int16。
// 超出范围时返回第一个出错值的索引（0-based）和 error。
func Int32SliceToInt16SliceValidate(values []int32) ([]int16, int, error) {
	result := make([]int16, len(values))
	for i, v := range values {
		if v > 32767 || v < -32768 {
			return nil, i, fmt.Errorf("index %d: value %d exceeds int16 range [-32768, 32767]", i, v)
		}
		result[i] = int16(v)
	}
	return result, -1, nil
}
