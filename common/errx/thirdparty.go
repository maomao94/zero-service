package errx

import (
	"strings"

	"zero-service/third_party/extproto"
)

// 第三方接口错误跨语言文案模板。Go 用 fmt.Sprintf，Java 用 String.format，
// 只使用 %s，避免 %d/%v 等语言相关占位符。
//
// 构造入口直接复用 NewErrorByPbCode 的 args 格式化能力，不另造格式化函数。
const (
	ThirdPartyAPIErrorTemplate          = "第三方接口错误，vendor=%s，code=%s，message=%s"
	ThirdPartyAPIErrorTemplateNoMessage = "第三方接口错误，vendor=%s，code=%s"
)

const (
	thirdPartyAPIPrefix     = "第三方接口错误，vendor="
	thirdPartyAPICodeSep    = "，code="
	thirdPartyAPIMessageSep = "，message="

	thirdPartyAPIMaxMsg    = 200
	thirdPartyAPIMaxVendor = 32
	thirdPartyAPIMaxCode   = 64
	thirdPartyAPIUnknown   = "unknown"
)

// NewThirdPartyAPIError 构造“第三方接口明确返回错误码”的平台错误。
//
// 使用 extproto.Code__1_00_THIRD_PARTY_API（100603），该码不参与网关折叠，
// 因此 msg 会原样返回给调用方。vendor 是厂商标识（如 dji），code 是厂商原始
// 错误码（如 312014），message 是厂商原始描述。
//
// 文案由 NewErrorByPbCode 的 args 直接格式化，模板中的 %s 按顺序替换：
//
//	errx.NewThirdPartyAPIError("dji", "312014", "设备升级中，请勿重复操作")
//	// 第三方接口错误，vendor=dji，code=312014，message=设备升级中，请勿重复操作
func NewThirdPartyAPIError(vendor, code, message string) error {
	vendor = sanitizeToken(vendor, thirdPartyAPIMaxVendor)
	code = sanitizeToken(code, thirdPartyAPIMaxCode)
	message = sanitizeMessage(message, thirdPartyAPIMaxMsg)

	if message == "" {
		return NewErrorByPbCode(
			extproto.Code__1_00_THIRD_PARTY_API,
			ThirdPartyAPIErrorTemplateNoMessage,
			vendor, code,
		)
	}
	return NewErrorByPbCode(
		extproto.Code__1_00_THIRD_PARTY_API,
		ThirdPartyAPIErrorTemplate,
		vendor, code, message,
	)
}

// ParseThirdPartyAPIError 从 %s 模板文案中解析厂商标识、厂商错误码和原始描述。
// 仅识别 NewThirdPartyAPIError 生成的文案，解析失败时 ok 为 false。
// 业务侧可以按同样规则自行解析：以 “，” 分隔字段，再按 “=” 取值。
func ParseThirdPartyAPIError(msg string) (vendor, code, message string, ok bool) {
	rest, found := strings.CutPrefix(msg, thirdPartyAPIPrefix)
	if !found {
		return "", "", "", false
	}

	vendor, rest, found = strings.Cut(rest, thirdPartyAPICodeSep)
	if !found || vendor == "" {
		return "", "", "", false
	}

	code, message, found = strings.Cut(rest, thirdPartyAPIMessageSep)
	if code == "" {
		return "", "", "", false
	}
	if !found {
		message = ""
	}
	return vendor, code, message, true
}

func sanitizeToken(value string, maxRunes int) string {
	value = strings.Join(strings.Fields(value), "_")
	value = strings.NewReplacer("，", "_", ",", "_", "=", "_").Replace(value)
	if value == "" {
		return thirdPartyAPIUnknown
	}
	return truncateRunes(value, maxRunes)
}

func sanitizeMessage(value string, maxRunes int) string {
	value = strings.Join(strings.Fields(value), " ")
	return truncateRunes(value, maxRunes)
}

func truncateRunes(value string, maxRunes int) string {
	runes := []rune(value)
	if len(runes) <= maxRunes {
		return value
	}
	return string(runes[:maxRunes]) + "…"
}
