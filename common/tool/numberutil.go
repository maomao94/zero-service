package tool

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/duke-git/lancet/v2/random"
)

// RandomDigits 生成指定位数的随机数字字符串（首位不为0）。
// n 必须在 1-9 之间（int 最大 10 位，9 位安全）。
func RandomDigits(n int) (string, error) {
	if n <= 0 || n > 9 {
		return "", fmt.Errorf("位数必须在1-9之间")
	}
	return strconv.Itoa(random.RandNumberOfLength(n)), nil
}

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
