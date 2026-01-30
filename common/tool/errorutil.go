package tool

import (
	"fmt"
	"zero-service/third_party/extproto"

	gkiterrors "github.com/songzhibin97/gkit/errors"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func NewErrorByPbCode(code extproto.Code, args ...interface{}) error {
	errorName, httpCode := getErrorInfoByPbCode(code)
	message := errorName
	if len(args) > 0 {
		hasFormat := false
		for i := 0; i < len(message); i++ {
			if message[i] == '%' && i+1 < len(message) {
				hasFormat = true
				break
			}
		}
		if hasFormat {
			message = fmt.Sprintf(message, args...)
		} else {
			switch v := args[0].(type) {
			case string:
				message = v
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
