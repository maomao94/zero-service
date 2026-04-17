// Package uilang 统一 zh/en 会话与中断 UI 语言归一化 (与静态前端 i18n 一致).
package uilang

import "strings"

// Normalize 将任意写法收束为 zh 或 en；无法识别时返回空 (表示不覆盖/不设置).
func Normalize(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	low := strings.ToLower(s)
	if strings.HasPrefix(low, "en") {
		return "en"
	}
	if strings.HasPrefix(low, "zh") {
		return "zh"
	}
	return ""
}
