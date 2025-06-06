package util

import (
	"fmt"
	"github.com/wendy512/go-iecp5/asdu"
	"strings"
)

func QdsContainsAny(qds asdu.QualityDescriptor, flags ...asdu.QualityDescriptor) bool {
	for _, flag := range flags {
		if (qds & flag) != 0 {
			return true
		}
	}
	return false
}

func QdsContainsAll(qds asdu.QualityDescriptor, flags ...asdu.QualityDescriptor) bool {
	for _, flag := range flags {
		if (qds & flag) == flag {
			return false
		}
	}
	return true
}

func QdsIsGood(qds asdu.QualityDescriptor) bool {
	return qds == asdu.QDSGood
}

func QdsIsOverflow(qds asdu.QualityDescriptor) bool {
	return (qds & asdu.QDSOverflow) != 0
}

func QdsIsBlocked(qds asdu.QualityDescriptor) bool {
	return (qds & asdu.QDSBlocked) != 0
}

func QdsIsSubstituted(qds asdu.QualityDescriptor) bool {
	return (qds & asdu.QDSSubstituted) != 0
}

func QdsIsNotTopical(qds asdu.QualityDescriptor) bool {
	return (qds & asdu.QDSNotTopical) != 0
}

func QdsIsInvalid(qds asdu.QualityDescriptor) bool {
	return (qds & asdu.QDSInvalid) != 0
}

func QdsString(qds asdu.QualityDescriptor) string {
	// 首先获取所有有效标志
	var flags []string

	if qds&asdu.QDSOverflow != 0 {
		flags = append(flags, "Overflow")
	}
	if qds&asdu.QDSBlocked != 0 {
		flags = append(flags, "Blocked")
	}
	if qds&asdu.QDSSubstituted != 0 {
		flags = append(flags, "Substituted")
	}
	if qds&asdu.QDSNotTopical != 0 {
		flags = append(flags, "NotTopical")
	}
	if qds&asdu.QDSInvalid != 0 {
		flags = append(flags, "Invalid")
	}

	// 获取完整的8位二进制表示
	binaryStr := fmt.Sprintf("%08b", uint8(qds))

	// 处理特殊状态
	switch {
	case qds == asdu.QDSGood:
		// 完全为0的情况
		return fmt.Sprintf("QDS(%s)[%s]", binaryStr, "QDSGood")
	case len(flags) == 0:
		// 没有标准标志位，但有保留位
		return fmt.Sprintf("QDS(%s)[ReservedBits]", binaryStr)
	//case len(flags) == 1:
	//	// 单个标志位，保留二进制信息
	//	return fmt.Sprintf("%s(%s)", flags[0], binaryStr)
	default:
		// 多个标志位
		return fmt.Sprintf("QDS(%s)[%s]", binaryStr, strings.Join(flags, "|"))
	}
}
