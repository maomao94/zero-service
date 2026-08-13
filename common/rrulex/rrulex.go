// Package rrulex 在官方 github.com/teambition/rrule-go 基础上补充
// 完整 RRULE Set 的解析校验、迭代起点平移与批量推进封装。
// 单次 after 查询不在此封装：调用方直接使用 ParseSet + 官方 set.After 的原生风格。
package rrulex

import (
	"errors"
	"strings"

	"github.com/teambition/rrule-go"
)

// ErrUnsupportedDescription 表示 RRULE 合法，但包含无法准确转换为业务文案的组合。
var ErrUnsupportedDescription = errors.New("[rrulex] unsupported rrule description")

// ParseSet 解析完整 RRULE Set：非空字符串必须同时包含显式 DTSTART 与 RRULE。
// 解析前统一 CRLF 换行，返回官方 *rrule.Set。
func ParseSet(value string) (*rrule.Set, error) {
	value = strings.ReplaceAll(strings.TrimSpace(value), "\r\n", "\n")
	set, err := rrule.StrToRRuleSet(value)
	if err != nil {
		return nil, err
	}
	if set.GetDTStart().IsZero() {
		return nil, errors.New("RRULE Set requires DTSTART")
	}
	if set.GetRRule() == nil {
		return nil, errors.New("RRULE Set requires RRULE")
	}
	return set, nil
}

// Validate 校验 RRULE Set 配置是否包含显式 DTSTART 和 RRULE。
// 空字符串表示一次性任务，是合法配置。
func Validate(value string) error {
	if value == "" {
		return nil
	}
	_, err := ParseSet(value)
	return err
}
