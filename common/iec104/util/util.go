package util

import (
	"errors"
	"fmt"
	"strings"
	"text/template"
	"zero-service/common/iec104/types"

	"github.com/wendy512/go-iecp5/asdu"
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

func QdpContainsAny(qdp asdu.QualityDescriptorProtection, flags ...asdu.QualityDescriptorProtection) bool {
	for _, flag := range flags {
		if (qdp & flag) != 0 {
			return true
		}
	}
	return false
}

func QdpContainsAll(qdp asdu.QualityDescriptorProtection, flags ...asdu.QualityDescriptorProtection) bool {
	for _, flag := range flags {
		if (qdp & flag) == flag {
			return false
		}
	}
	return true
}

func QdpIsGood(qdp asdu.QualityDescriptorProtection) bool {
	return qdp == asdu.QDPGood
}

func QdpIsElapsedTimeInvalid(qdp asdu.QualityDescriptorProtection) bool {
	return (qdp & asdu.QDPElapsedTimeInvalid) != 0
}

func QdpIsBlocked(qdp asdu.QualityDescriptorProtection) bool {
	return (qdp & asdu.QDPBlocked) != 0
}

func QdpIsSubstituted(qdp asdu.QualityDescriptorProtection) bool {
	return (qdp & asdu.QDPSubstituted) != 0
}

func QdpIsNotTopical(qdp asdu.QualityDescriptorProtection) bool {
	return (qdp & asdu.QDPNotTopical) != 0
}

func QdpIsInvalid(qdp asdu.QualityDescriptorProtection) bool {
	return (qdp & asdu.QDPInvalid) != 0
}

func QdpString(qdp asdu.QualityDescriptorProtection) string {
	// 首先获取所有有效标志
	var flags []string

	if qdp&asdu.QDPElapsedTimeInvalid != 0 {
		flags = append(flags, "ElapsedTimeInvalid")
	}
	if qdp&asdu.QDPBlocked != 0 {
		flags = append(flags, "Blocked")
	}
	if qdp&asdu.QDPSubstituted != 0 {
		flags = append(flags, "Substituted")
	}
	if qdp&asdu.QDPNotTopical != 0 {
		flags = append(flags, "NotTopical")
	}
	if qdp&asdu.QDPInvalid != 0 {
		flags = append(flags, "Invalid")
	}

	// 获取完整的8位二进制表示
	binaryStr := fmt.Sprintf("%08b", uint8(qdp))

	// 处理特殊状态
	switch {
	case qdp == asdu.QDPGood:
		// 完全为0的情况
		return fmt.Sprintf("QDP(%s)[%s]", binaryStr, "QDPGood")
	case len(flags) == 0:
		// 没有标准标志位，但有保留位
		return fmt.Sprintf("QDP(%s)[ReservedBits]", binaryStr)
	//case len(flags) == 1:
	//	// 单个标志位，保留二进制信息
	//	return fmt.Sprintf("%s(%s)", flags[0], binaryStr)
	default:
		// 多个标志位
		return fmt.Sprintf("QDP(%s)[%s]", binaryStr, strings.Join(flags, "|"))
	}
}

func FloatToNormalize(f float64) asdu.Normalize {
	if f >= 1.0 {
		f = 1.0 - 1.0/32768.0 // 最大不能等于1
	} else if f < -1.0 {
		f = -1.0 // 最小值为-1.0
	}
	return asdu.Normalize(int16(f * 32768.0))
}

func NormalizeToFloat(n asdu.Normalize) float32 {
	return float32(n) / 32768.0
}

// GenerateStationId 根据host和port生成stationId
// 替换host中的.为_,然后与port拼接
func GenerateStationId(host string, port interface{}) string {
	safeHost := strings.ReplaceAll(host, ".", "_")
	return fmt.Sprintf("%s_%v", safeHost, port)
}

// generateTopic 根据配置的topic规则和报文值生成最终的topic
func GenerateTopic(topicPattern string, data *types.MsgBody) (string, error) {
	if data == nil {
		return "", errors.New("msg body is nil")
	}

	tmpl, err := template.New("topic").Parse(topicPattern)
	if err != nil {
		return "", errors.New("failed to parse topic template")
	}

	// Execute the template directly with the MsgBody struct
	var result strings.Builder
	err = tmpl.Execute(&result, data)
	if err != nil {
		return "", errors.New("failed to execute topic template")
	}

	topic := result.String()

	if strings.Contains(topic, "{{") && strings.Contains(topic, "}}") {
		return "", errors.New("unresolved placeholders in topic")
	}

	if strings.Contains(topic, "+") || strings.Contains(topic, "#") {
		return "", errors.New("invalid topic pattern")
	}

	// 检查连续的斜杠，例如 "iec//asdu/"，如果发现则返回错误
	if strings.Contains(topic, "//") {
		return "", errors.New("invalid topic: contains consecutive slashes")
	}

	// 检查开头是否有斜杠
	if strings.HasPrefix(topic, "/") {
		return "", errors.New("invalid topic: starts with slash")
	}

	// 检查结尾是否有斜杠
	if strings.HasSuffix(topic, "/") {
		return "", errors.New("invalid topic: ends with slash")
	}

	return topic, nil
}
