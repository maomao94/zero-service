// Package errx 是项目唯一的错误码转换层：把任意错误归一化成 extproto 六位码。
//
// 它不依赖 HTTP、不依赖网关，服务端、facade、SDK 都可以直接复用。
// HTTP 信封写出属于 common/gtwx，不属于这里。
package errx

import (
	"errors"
	"strconv"
	"strings"

	"zero-service/third_party/extproto"

	"github.com/zeromicro/go-zero/core/logx"
	xerrors "github.com/zeromicro/x/errors"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// SystemBusyMessage 是对外折叠后的统一文案。
const SystemBusyMessage = "系统繁忙"

// SourceOf 返回六位错误码的来源位 A；非六位码返回 -1。
func SourceOf(code int) int {
	if code < 100000 || code > 999999 {
		return -1
	}
	return code / 100000
}

// IsDefinedCode 判断 code 是否为 extproto 中已定义的错误码。
//
// 这是唯一的码表判据：存在即合法，不存在即降级为 100999。
// 判据与来源位无关——不推断是平台码、设备码还是云端码，只看是否登记在册。
func IsDefinedCode(code int) bool {
	_, ok := extproto.Code_name[int32(code)]
	return ok
}

// CodeFromStatus 将错误转换为六位错误码与可展示消息。
// 优先读取 google.rpc.ErrorInfo.reason（仅接受 extproto 中已定义的码），
// 取不到时按 gRPC Code 兜底；未定义的原始码不进入信封，按内部错误折叠。
func CodeFromStatus(err error) (int, string) {
	if err == nil {
		return 0, ""
	}

	var codeMsg *xerrors.CodeMsg
	if errors.As(err, &codeMsg) && codeMsg != nil {
		return codeMsg.Code, codeMsg.Msg
	}

	st, ok := status.FromError(err)
	if !ok {
		// 既不是 CodeMsg，也不是 gRPC status：说明业务侧没有做错误转换。
		// 不能把内部 error 文本回显给客户端，折叠为系统繁忙，原始值写日志。
		logx.Errorf("未转换的普通错误，已折叠为系统繁忙：%v", err)
		return int(extproto.Code__1_00_INTERNAL), ""
	}
	if st.Code() == codes.OK {
		return 0, ""
	}

	// 数字型 reason，但不在 extproto 枚举内：无法识别的原始码，禁止进入信封。
	// 标记后统一按内部错误折叠，避免未定义码被当成平台业务码透传。
	// 判据只看枚举是否存在，与来源位无关：将来纳入 extproto 的云端码同样会放行。
	undefinedCode := false
	for _, detail := range st.Details() {
		info, ok := detail.(*errdetails.ErrorInfo)
		if !ok {
			continue
		}
		code, convErr := strconv.Atoi(strings.TrimSpace(info.GetReason()))
		if convErr != nil {
			continue
		}
		if !IsDefinedCode(code) {
			logx.Errorf("未登记的错误码 %s（domain=%s），已折叠为系统繁忙", info.GetReason(), info.GetDomain())
			undefinedCode = true
			continue
		}
		msg := st.Message()
		if msg == "" {
			msg = ErrorMessage(code)
		}
		return code, msg
	}
	if undefinedCode {
		return int(extproto.Code__1_00_INTERNAL), ""
	}

	code := codeFromGRPCStatus(st.Code())
	msg := st.Message()
	if msg == "" {
		msg = ErrorMessage(code)
	}
	return code, msg
}

func codeFromGRPCStatus(code codes.Code) int {
	switch code {
	case codes.InvalidArgument, codes.OutOfRange:
		return int(extproto.Code__1_00_PARAM)
	case codes.NotFound:
		return int(extproto.Code__1_00_RECORD_NOT_EXIST)
	case codes.AlreadyExists:
		return int(extproto.Code__1_00_RECORD_ALREADY_EXIST)
	case codes.Aborted:
		return int(extproto.Code__1_00_DATA_CONFLICT)
	case codes.Unauthenticated:
		return int(extproto.Code__1_00_UNAUTHORIZED)
	case codes.PermissionDenied:
		return int(extproto.Code__1_00_FORBIDDEN)
	case codes.FailedPrecondition:
		return int(extproto.Code__1_00_BIZ_STATE)
	case codes.ResourceExhausted:
		return int(extproto.Code__1_00_RATE_LIMITED)
	case codes.Unavailable:
		return int(extproto.Code__1_00_RPC)
	case codes.DeadlineExceeded:
		return int(extproto.Code__1_00_TIMEOUT)
	case codes.Internal, codes.Unknown, codes.DataLoss:
		return int(extproto.Code__1_00_INTERNAL)
	default:
		return int(extproto.Code__1_00_UNKNOWN)
	}
}

// FoldPublic 将基础设施类错误折叠为对外的“系统繁忙”，真实原因只进日志。
func FoldPublic(code int) int {
	switch code {
	case int(extproto.Code__1_00_INTERNAL),
		int(extproto.Code__1_00_DB),
		int(extproto.Code__1_00_CACHE),
		int(extproto.Code__1_00_CACHE_MISS),
		int(extproto.Code__1_00_MQ),
		int(extproto.Code__1_00_RPC),
		int(extproto.Code__1_00_THIRD_PARTY):
		return int(extproto.Code__1_00_UNKNOWN)
	default:
		return code
	}
}

// PublicMessage 在错误被折叠时返回统一的对外消息。
func PublicMessage(code int, msg string) string {
	if FoldPublic(code) != code {
		return SystemBusyMessage
	}
	return msg
}

// ErrorMessage 读取 extproto 中定义的错误名称。
func ErrorMessage(code int) string {
	enumCode := extproto.Code(code)
	desc := enumCode.Descriptor()
	if desc == nil {
		return "请求失败"
	}
	value := desc.Values().ByNumber(protoreflect.EnumNumber(code))
	if value == nil {
		return "请求失败"
	}
	opts := value.Options()
	if opts == nil {
		return "请求失败"
	}
	name, ok := proto.GetExtension(opts, extproto.E_Name).(string)
	if !ok || name == "" {
		return "请求失败"
	}
	return name
}
