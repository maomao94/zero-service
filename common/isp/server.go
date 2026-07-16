package isp

import (
	"context"
	"time"

	"zero-service/common/gnetx"

	"github.com/panjf2000/gnet/v2"
	"github.com/zeromicro/go-zero/core/logx"
)

// ServerHandler registers ISP protocol handlers for a server endpoint.
type ServerHandler func(*ServerRouter)

// ServerRouter registers ISP protocol command handlers for a server endpoint.
type ServerRouter struct{ router *gnetx.Router }

// Handle registers a server-side handler for one ISP message id.
func (r *ServerRouter) Handle(messageID int, fn IspHandler) {
	serverHandleAsync(r.router, messageID, fn)
}

// Fallback registers the server-side handler for unmatched ISP messages.
func (r *ServerRouter) Fallback(fn IspHandler) {
	serverFallbackAsync(r.router, fn)
}

// Server is an ISP TCP server backed by gnetx.
type Server struct {
	srv     *gnetx.Server
	manager *gnetx.SessionManager
}

// NewServer creates an ISP TCP server and registers protocol command handlers.
func NewServer(conf ServerConfig, register ServerHandler) (*Server, error) {
	conf.ApplyDefaults()
	codec := NewCodec(conf.RootName, conf.MaxFrameLength, conf.DebugLog)
	codec = &rootNameCodec{inner: codec, rootName: conf.RootName}
	router := gnetx.NewRouter()
	if register != nil {
		register(&ServerRouter{router: router})
	}

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
	return &Server{srv: srv, manager: srv.Manager()}, nil
}

// Start starts the underlying TCP server.
func (s *Server) Start() { s.srv.Start() }

// Stop stops the underlying TCP server.
func (s *Server) Stop() { s.srv.Stop() }

// Manager exposes active TCP sessions for management RPCs.
func (s *Server) Manager() *gnetx.SessionManager { return s.manager }

type rootNameCodec struct {
	inner    gnetx.Codec
	rootName string
}

func (c *rootNameCodec) Decode(gc gnet.Conn, conn gnetx.Conn) (any, error) {
	msg, err := c.inner.Decode(gc, conn)
	if err != nil {
		return nil, err
	}
	m, ok := msg.(*Message)
	if !ok {
		return msg, nil
	}
	if err := ValidateRootName(c.rootName, m.RootName); err != nil {
		logx.Errorf("[isp] RootName 校验失败, 远端=%s: %v, 断开连接", conn.RemoteAddr(), err)
		return nil, err
	}
	return msg, nil
}

func (c *rootNameCodec) Encode(ctx context.Context, msg any, conn gnetx.Conn) ([]byte, error) {
	return c.inner.Encode(ctx, msg, conn)
}
