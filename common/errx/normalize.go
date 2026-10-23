package errx

import (
	"errors"

	"zero-service/third_party/extproto"

	"github.com/zeromicro/go-zero/core/logx"
	xerrors "github.com/zeromicro/x/errors"
)

// SuccessCode 是统一成功码，所有成功响应都使用它。
const SuccessCode = int(extproto.Code__0_OK)

// IsSuccess 判断业务码是否表示成功。
func IsSuccess(code int) bool {
	return code == SuccessCode
}

// PublicError 将任意错误转换为可对外返回的六位码与文案。
// 优先级：CodeMsg → gRPC ErrorInfo.reason（仅 extproto 中已登记的码）→ gRPC Code 兜底。
// 基础设施类错误会折叠为“系统繁忙”，真实原因由调用方写日志。
func PublicError(err error) (int, string) {
	if err == nil {
		return SuccessCode, ""
	}

	var codeMsg *xerrors.CodeMsg
	if errors.As(err, &codeMsg) && codeMsg != nil {
		code := codeMsg.Code
		if code == SuccessCode {
			// 错误对象不允许携带成功码，避免把失败当成功。
			code = int(extproto.Code__1_00_UNKNOWN)
		}
		return FoldPublic(code), PublicMessage(code, codeMsg.Msg)
	}

	code, msg := CodeFromStatus(err)
	if code == 0 {
		code = int(extproto.Code__1_00_UNKNOWN)
	}
	if msg == "" {
		msg = ErrorMessage(code)
	}
	return FoldPublic(code), PublicMessage(code, msg)
}

// NormalizeError 是统一错误出口：先归一化，再校验错误码是否合法。
//
// 合法判据只有一条：code 存在于 extproto 枚举。不存在时降级为 100999 并记录日志，
// 把开发者误用暴露在服务端，而不是泄漏给客户端。
func NormalizeError(err error) (int, string) {
	if err == nil {
		return SuccessCode, ""
	}

	code, msg := PublicError(err)
	if IsDefinedCode(code) {
		return FoldPublic(code), PublicMessage(code, msg)
	}

	logx.Errorf("非法错误码 %d（不在 extproto 枚举内），已降级为 %d：%s",
		code, int(extproto.Code__1_00_UNKNOWN), msg)
	return int(extproto.Code__1_00_UNKNOWN), SystemBusyMessage
}
