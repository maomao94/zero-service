package isp

import (
	"context"

	"zero-service/common/gnetx"
)

// IspHandler 服务端 ISP 消息处理函数签名。
// 返回响应消息或 error；ServerWrap 负责 SendSeq 填充及 error→251-3 转换。
type IspHandler func(ctx context.Context, conn gnetx.Conn, req *Message) (*Message, error)

// ServerWrap 包装服务端 IspHandler，统一处理:
//  1. 消息类型断言（*Message）
//  2. 入站日志（LogInbound）
//  3. 业务处理
//  4. error 自动转为 251-3 通用应答（Code=500）
//  5. SendSeq 填充（conn.NextSendSeq）
func ServerWrap(fn IspHandler) gnetx.Handler {
	return gnetx.HandlerFunc(func(ctx context.Context, conn gnetx.Conn, msg any) (any, error) {
		m, ok := msg.(*Message)
		if !ok {
			return nil, ErrError
		}
		LogInbound(ctx, m)
		resp, err := fn(ctx, conn, m)
		if err != nil {
			resp = NewResponse(m, SessionSourceServer, ResponseCode(err), CommandGenericResponseWithoutItems, nil)
		}
		if resp != nil {
			resp.SendSeq = conn.NextSendSeq()
		}
		return resp, nil
	})
}
