package tool

import (
	"fmt"
	"zero-service/third_party/extproto"

	gkiterrors "github.com/songzhibin97/gkit/errors"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func NewErrorByPbCode(code extproto.Code, args ...interface{}) error {
	errorName, httpCode := getErrorInfoByPbCode(code)
	message := errorName
	if len(args) > 0 {
		if hasFormatPlaceholder(message) {
			message = fmt.Sprintf(message, args...)
		} else {
			switch v := args[0].(type) {
			case string:
				if len(args) > 1 && hasFormatPlaceholder(v) {
					message = fmt.Sprintf(v, args[1:]...)
				} else {
					message = v
				}
			case fmt.Stringer:
				message = v.String()
			default:
				message = fmt.Sprintf("%v", v)
			}
		}
	}
	reason := fmt.Sprintf("%d", code)
	switch httpCode {
	case 400:
		return gkiterrors.BadRequest(reason, message)
	case 401:
		return gkiterrors.Unauthorized(reason, message)
	case 403:
		return gkiterrors.Forbidden(reason, message)
	case 404:
		return gkiterrors.NotFound(reason, message)
	case 409:
		return gkiterrors.Conflict(reason, message)
	case 499:
		return gkiterrors.ClientClosed(reason, message)
	case 500:
		return gkiterrors.InternalServer(reason, message)
	case 503:
		return gkiterrors.ServiceUnavailable(reason, message)
	case 504:
		return gkiterrors.GatewayTimeout(reason, message)
	default:
		return gkiterrors.InternalServer(reason, message)
	}
}

func hasFormatPlaceholder(message string) bool {
	for i := 0; i < len(message); i++ {
		if message[i] == '%' && i+1 < len(message) {
			return true
		}
	}
	return false
}

// NewErrorByPbCodeWrap wraps a cause error with a protobuf error code.
// Implements Go 1.20 multi-error Unwrap, so both the structured code error and
// the original cause are traversable via errors.Is/As and gkiterrors.Reason.
// GRPCStatus returns the structured error so status.FromError resolves correctly.
func NewErrorByPbCodeWrap(code extproto.Code, cause error, args ...interface{}) error {
	if cause == nil {
		return NewErrorByPbCode(code, args...)
	}
	return &withCause{
		structured: NewErrorByPbCode(code, args...),
		cause:      cause,
	}
}

type withCause struct {
	structured error
	cause      error
}

func (w *withCause) Error() string {
	return fmt.Sprintf("%s: %v", w.structured.Error(), w.cause)
}

func (w *withCause) Unwrap() []error {
	return []error{w.structured, w.cause}
}

// GRPCStatus 满足 gRPC status.FromError 接口，使 HTTP 网关能正确映射状态码。
func (w *withCause) GRPCStatus() *status.Status {
	return status.Convert(w.structured)
}

func getErrorInfoByPbCode(code extproto.Code) (string, int) {
	errorName := "error"
	httpCode := 400
	if enumDesc := code.Descriptor(); enumDesc != nil {
		if enumValue := enumDesc.Values().ByNumber(protoreflect.EnumNumber(code)); enumValue != nil {
			if options := enumValue.Options(); options != nil {
				if name := proto.GetExtension(options, extproto.E_Name); name != nil {
					if nameStr, ok := name.(string); ok {
						errorName = nameStr
					}
				}
				if codeVal := proto.GetExtension(options, extproto.E_HttpCode); codeVal != nil {
					if codeInt, ok := codeVal.(int32); ok {
						httpCode = int(codeInt)
					}
				}
			}
		}
	}
	return errorName, httpCode
}

func IsErrorByPbCode(err error, code extproto.Code) bool {
	expectedReason := fmt.Sprintf("%d", code)
	grpcReason := gkiterrors.Reason(err)
	if grpcReason == expectedReason {
		return true
	}
	return false
}
