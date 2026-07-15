package isp

import (
	"context"
	"time"

	"zero-service/app/ispserver/internal/config"
	"zero-service/common/gnetx"
	"zero-service/common/isp"

	"github.com/panjf2000/gnet/v2"
	"github.com/zeromicro/go-zero/core/logx"
)

// Server 为 ISP 协议 TCP 服务端，基于 gnetx 实现多连接管理。
// 对标 Java com.allcore.sip.transport.endpoint.SipEndpoint。
type Server struct {
	srv     *gnetx.Server
	manager *gnetx.SessionManager
	conf    config.IspConf
}

// NewServer 构造 ISP TCP 服务端。
func NewServer(conf config.IspConf) (*Server, error) {
	codec := isp.NewCodec(conf.RootName, conf.MaxFrameLength, conf.DebugLog)
	codec = &rootNameCodec{inner: codec, rootName: conf.RootName}
	router := NewRouter(conf)

	srv, err := gnetx.NewServer(
		gnetx.WithAddr(conf.ListenAddr),
		gnetx.WithCodec(codec),
		gnetx.WithServerHandler(router),
		gnetx.WithMaxFrameLength(conf.MaxFrameLength),
		gnetx.WithIdleTimeout(time.Duration(conf.IdleTimeoutSeconds)*time.Second),
		gnetx.WithMulticore(true),
	)
	if err != nil {
		return nil, err
	}

	return &Server{srv: srv, manager: srv.Manager(), conf: conf}, nil
}

// Start 启动 ISP TCP 服务端，实现 go-zero service.Service 接口。
func (s *Server) Start() { s.srv.Start() }

// Stop 关闭 ISP TCP 服务端，实现 go-zero service.Service 接口。
func (s *Server) Stop() { s.srv.Stop() }

// Manager 返回 SessionManager，用于查询/管理已连接会话。
func (s *Server) Manager() *gnetx.SessionManager { return s.manager }

// rootNameCodec 包装 ISP Codec，在解码后校验 RootName 是否与预期一致。
// 不一致时返回错误，触发 gnetx handleDecodeError → DecodeErrorClose 断开连接。
type rootNameCodec struct {
	inner    gnetx.Codec
	rootName string
}

func (c *rootNameCodec) Decode(gc gnet.Conn, conn gnetx.Conn) (any, error) {
	msg, err := c.inner.Decode(gc, conn)
	if err != nil {
		return nil, err
	}
	m, ok := msg.(*isp.Message)
	if !ok {
		return msg, nil
	}
	if err := isp.ValidateRootName(c.rootName, m.RootName); err != nil {
		logx.Errorf("[ispserver] RootName 校验失败, 远端=%s: %v, 断开连接", conn.RemoteAddr(), err)
		return nil, err
	}
	return msg, nil
}

func (c *rootNameCodec) Encode(ctx context.Context, msg any, conn gnetx.Conn) ([]byte, error) {
	return c.inner.Encode(ctx, msg, conn)
}
