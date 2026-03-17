package bytex

import (
	"fmt"
)

// 字节 → uint16（核心） → int16（有符号）
type BinaryValues struct {
	Hex    []string `json:"hex"`    // 16位十六进制
	Uint16 []uint16 `json:"uint16"` // 核心：无符号16位（唯一真值）
	Int16  []int16  `json:"int16"`  // 有符号16位（由uint16转换）
	Bytes  []byte   `json:"bytes"`  // 原始字节（源头）
	Binary []string `json:"binary"` // 16位二进制
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
	intVals := make([]int16, len(values))
	for i, v := range values {
		intVals[i] = Uint16ToInt16(v)
	}
	return intVals
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
	res := make([]uint32, len(values))
	for i, v := range values {
		res[i] = Uint16ToUint32(v)
	}
	return res
}

func Uint16SliceToInt32Slice(values []uint16) []int32 {
	res := make([]int32, len(values))
	for i, v := range values {
		res[i] = Uint16ToInt32(v)
	}
	return res
}

func Int16SliceToInt32Slice(values []int16) []int32 {
	res := make([]int32, len(values))
	for i, v := range values {
		res[i] = int32(v)
	}
	return res
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
	res := make([]uint16, len(values))
	for i, v := range values {
		res[i] = Uint32ToUint16(v)
	}
	return res
}

func Int32SliceToInt16Slice(values []int32) []int16 {
	res := make([]int16, len(values))
	for i, v := range values {
		res[i] = Int32ToInt16(v)
	}
	return res
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
	n := (len(bools) + 7) / 8
	data := make([]byte, n)
	for i, b := range bools {
		if b {
			data[i/8] |= 1 << (i % 8)
		}
	}
	return data
}
