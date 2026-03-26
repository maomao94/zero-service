package ctxprop

import (
	"context"
	"strings"

	"zero-service/common/ctxdata"

	"github.com/duke-git/lancet/v2/cryptor"
	"google.golang.org/grpc/metadata"
)

// base64Prefix 是非 ASCII 字符 base64 编码后的前缀标记
const base64Prefix = "b64:"

// hasNotPrintable 检查字符串是否包含非 ASCII 可打印字符（0x20-0x7E）
func hasNotPrintable(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] < 0x20 || s[i] > 0x7E {
			return true
		}
	}
	return false
}

// InjectToGrpcMD 从 context values 提取所有字段，注入到 outgoing gRPC metadata。
// 用于 gRPC 客户端拦截器：将上下文字段传播到下游 RPC 服务。
// 只有 string 类型的非空值才会被注入。
// gRPC metadata 只支持 ASCII 可打印字符，非 ASCII 字符会进行 base64 编码并添加前缀。
func InjectToGrpcMD(ctx context.Context) context.Context {
	md, _ := metadata.FromOutgoingContext(ctx)
	md = md.Copy()
	for _, f := range ctxdata.PropFields {
		v := ctx.Value(f.CtxKey)
		if v == nil {
			continue
		}
		str, ok := v.(string)
		if !ok || str == "" {
			continue
		}
		// gRPC metadata 只支持 ASCII 可打印字符，非 ASCII 字符进行 base64 编码并添加前缀标记
		if hasNotPrintable(str) {
			str = base64Prefix + cryptor.Base64StdEncode(str)
		}
		md.Set(f.GrpcHeader, str)
	}
	return metadata.NewOutgoingContext(ctx, md)
}

// ExtractFromGrpcMD 从 incoming gRPC metadata 提取所有字段，注入到 context values。
// 用于 gRPC 服务端拦截器：将 metadata 中的字段恢复到 context 供业务层使用。
// 只有有效的非空字符串值才会被注入。
// 只有带有 b64: 前缀的值才会进行 base64 解码。
func ExtractFromGrpcMD(ctx context.Context) context.Context {
	md, _ := metadata.FromIncomingContext(ctx)
	for _, f := range ctxdata.PropFields {
		if v := md.Get(f.GrpcHeader); len(v) > 0 && v[0] != "" {
			val := v[0]
			// 只有带有 b64: 前缀的值才进行 base64 解码
			if strings.HasPrefix(val, base64Prefix) {
				encoded := val[len(base64Prefix):]
				val = cryptor.Base64StdDecode(encoded)
			}
			ctx = context.WithValue(ctx, f.CtxKey, val)
		}
	}
	return ctx
}
