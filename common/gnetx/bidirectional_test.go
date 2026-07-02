package gnetx

import (
	"context"
	"strconv"
	"sync"
	"testing"
	"time"
)

// TestServerInitiatedRequest 验证双向报文的 headline 能力：
// server 端通过已连接会话的 Session.Request 主动向 client 发请求，client 回包，
// server 的 Request 按 tid 匹配到回包并返回。
//
// 流程：
//  1. gnetx Client 连接 server，client handler 收到 pingReq 时回 pongResp。
//  2. server 端业务 goroutine 从 Manager 取到该会话，调用 Request 发 pingReq 等 pongResp。
//  3. client 收到 pingReq → 回 pongResp → server OnTraffic 识别为 Response → resolveResponse 命中在途 → Request 返回。
func TestServerInitiatedRequest(t *testing.T) {
	port := freePort(t)

	// server handler：普通消息不处理（本测试由 server 主动发起请求）。
	srvReady := make(chan *Session, 1)
	srvHandler := HandlerFunc(func(ctx context.Context, s *Session, msg any) (any, error) {
		return nil, nil
	})
	srv, err := NewServer(
		WithAddr("127.0.0.1:"+strconv.Itoa(port)),
		WithCodec(newTestCodec()),
		WithHandler(srvHandler),
		WithMaxFrameLength(1<<20),
		WithSessionListener(&captureListener{ch: srvReady}),
	)
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	done := make(chan error, 1)
	go func() { done <- srv.Run() }()
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_ = srv.Shutdown(ctx)
		<-done
	}()
	time.Sleep(100 * time.Millisecond)

	// client handler：收到 pingReq 回 pongResp（同 serial）。
	cli, err := NewClient("tcp", "127.0.0.1:"+strconv.Itoa(port),
		WithClientCodec(newTestCodec()),
		WithClientHandler(HandlerFunc(func(ctx context.Context, s *Session, msg any) (any, error) {
			if p, ok := msg.(*pingReq); ok {
				return &pongResp{RespSerial: p.Serial, Reply: "client-ack-" + p.Msg}, nil
			}
			return nil, nil
		})),
		WithClientMaxFrameLength(1<<20),
	)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer cli.Close()

	if cli.Session() == nil {
		t.Fatal("client Session() nil after connect")
	}

	// 等 server 端会话建立
	var srvSess *Session
	select {
	case srvSess = <-srvReady:
	case <-time.After(2 * time.Second):
		t.Fatal("server session not created")
	}

	// server 端主动发起 Request（off-loop：测试 goroutine）
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	resp, err := srvSess.Request(ctx, &pingReq{Serial: 100, Msg: "hi"}, 5*time.Second)
	if err != nil {
		t.Fatalf("server Request: %v", err)
	}
	pong, ok := resp.(*pongResp)
	if !ok {
		t.Fatalf("resp type = %T, want *pongResp", resp)
	}
	if pong.RespSerial != 100 || pong.Reply != "client-ack-hi" {
		t.Fatalf("pong = %+v, want {100 client-ack-hi}", pong)
	}
}

// captureListener 捕获 server 端新建的会话，供测试拿到 Session 主动发起请求。
type captureListener struct {
	noopSessionListener
	once sync.Once
	ch   chan *Session
}

func (l *captureListener) OnCreated(s *Session) {
	l.once.Do(func() { l.ch <- s })
}
