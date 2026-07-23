package isp

import (
	"context"

	"zero-service/common/gnetx"
)

// IspHandler ISP 消息处理函数签名。
// 返回响应消息或 error；方向包装器负责 SendSeq 填充、nil→251-3 成功及 error→251-3 转换。
type IspHandler func(ctx context.Context, conn gnetx.Conn, req *Message) (*Message, error)

func wrapHandler(fn IspHandler, sessionSource byte, after func(conn gnetx.Conn, req *Message)) gnetx.Handler {
	return gnetx.HandlerFunc(func(ctx context.Context, conn gnetx.Conn, msg any) (any, error) {
		m, ok := msg.(*Message)
		if !ok {
			return nil, ErrInternal
		}
		LogInbound(ctx, m)
		resp, err := fn(ctx, conn, m)
		if after != nil {
			after(conn, m)
		}
		if err != nil {
			LogErrorResponse(ctx, m, err, ResponseCode(err))
			resp = NewErrorResponse(m, sessionSource, err)
		} else if resp == nil {
			resp = NewSuccessResponse(m, sessionSource)
		}
		if resp == nil {
			return nil, nil
		}
		resp.SendSeq = conn.NextSendSeq()
		return resp, nil
	})
}

// serverWrap 包装服务端 IspHandler，统一处理:
//  1. 消息类型断言（*Message）
//  2. 入站日志（LogInbound）
//  3. 业务处理
//  4. nil 自动转为 251-3 通用成功应答
//  5. error 自动转为 251-3 通用应答（ResponseCode 映射 Code）
//  6. SendSeq 填充（conn.NextSendSeq）
func serverWrap(fn IspHandler) gnetx.Handler {
	return wrapHandler(fn, SessionSourceServer, nil)
}

func serverHandleAsync(router *gnetx.Router, id int, fn IspHandler) {
	router.Register(id, gnetx.Async(serverWrap(fn)))
}

func serverFallbackAsync(router *gnetx.Router, fn IspHandler) {
	router.Fallback(gnetx.Async(serverWrap(fn)))
}

// clientWrap 包装客户端方向 IspHandler，固定使用 SessionSourceClient，并记录对端 SendSeq。
func clientWrap(fn IspHandler, client *Client) gnetx.Handler {
	return wrapHandler(fn, SessionSourceClient, func(conn gnetx.Conn, req *Message) {
		if client != nil {
			client.trackRecvSeq(req.SendSeq, conn.SessionID())
		}
	})
}

func clientHandleAsync(router *gnetx.Router, id int, client *Client, fn IspHandler) {
	router.Register(id, gnetx.Async(clientWrap(fn, client)))
}

func clientFallbackAsync(router *gnetx.Router, client *Client, fn IspHandler) {
	router.Fallback(gnetx.Async(clientWrap(fn, client)))
}
