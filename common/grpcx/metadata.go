package grpcx

import (
	"context"
	"strings"

	"zero-service/common/authctx"

	"github.com/duke-git/lancet/v2/cryptor"
	"google.golang.org/grpc/metadata"
)

const (
	HeaderUserId        = "x-user-id"
	HeaderUserName      = "x-user-name"
	HeaderDeptCode      = "x-dept-code"
	HeaderAuthorization = "authorization"
	HeaderAuthType      = "x-auth-type"
	base64Prefix        = "b64:"
)

type metadataField struct {
	contextKey string
	grpcKey    string
}

// metadataFields is ordered to preserve the existing wire propagation behavior.
var metadataFields = []metadataField{
	{authctx.CtxAuthorizationKey, HeaderAuthorization},
	{authctx.CtxUserIdKey, HeaderUserId},
	{authctx.CtxUserNameKey, HeaderUserName},
	{authctx.CtxDeptCodeKey, HeaderDeptCode},
	{authctx.CtxAuthTypeKey, HeaderAuthType},
}

func hasNotPrintable(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] < 0x20 || s[i] > 0x7E {
			return true
		}
	}
	return false
}

// InjectToGrpcMD injects non-empty string context values into outgoing metadata.
func InjectToGrpcMD(ctx context.Context) context.Context {
	md, _ := metadata.FromOutgoingContext(ctx)
	md = md.Copy()
	for _, f := range metadataFields {
		str := authctx.GetByKey(ctx, f.contextKey)
		if str == "" {
			continue
		}
		if hasNotPrintable(str) {
			str = base64Prefix + cryptor.Base64StdEncode(str)
		}
		md.Set(f.grpcKey, str)
	}
	return metadata.NewOutgoingContext(ctx, md)
}

// ExtractFromGrpcMD restores the first non-empty incoming metadata value to context.
func ExtractFromGrpcMD(ctx context.Context) context.Context {
	md, _ := metadata.FromIncomingContext(ctx)
	for _, f := range metadataFields {
		if values := md.Get(f.grpcKey); len(values) > 0 && values[0] != "" {
			val := values[0]
			if strings.HasPrefix(val, base64Prefix) {
				val = cryptor.Base64StdDecode(val[len(base64Prefix):])
			}
			ctx = authctx.WithKey(ctx, f.contextKey, val)
		}
	}
	return ctx
}
