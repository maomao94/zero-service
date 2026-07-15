package tool

import "strings"

// CountSignificantDigits 统计数值字符串的有效数字位数。
// 规则：去掉符号、前导零、小数点后统计剩余数字个数。
// "51.88" -> 4, "0.001234" -> 4, "100" -> 3, "0" -> 0.
func CountSignificantDigits(s string) int {
	s = strings.TrimSpace(s)
	s = strings.TrimLeft(s, "+-")
	if idx := strings.IndexAny(s, "eE"); idx >= 0 {
		s = s[:idx]
	}
	s = strings.TrimLeft(s, "0")
	if i := strings.IndexByte(s, '.'); i >= 0 {
		s = s[:i] + s[i+1:]
	}
	s = strings.TrimLeft(s, "0")
	return len(s)
}
