package gnetx

import (
	"context"
	"errors"
	"strconv"
	"testing"
	"time"
)

func TestDialerRequest(t *testing.T) {
	port := freePort(t)

	srvHandler := HandlerFunc(func(ctx context.Context, c Conn, msg any) (any, error) {
		if p, ok := msg.(*pingReq); ok {
			return &pongResp{RespSerial: p.Serial, Reply: "dialer-ack-" + p.Msg}, nil
		}
		return nil, nil
	})
	stop := startServer(t, port, srvHandler)
	defer stop()

	dialer := NewDialer(WithClientCodec(newTestCodec()))
	defer dialer.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	resp, err := dialer.Request(ctx, "tcp", "127.0.0.1:"+strconv.Itoa(port), &pingReq{Serial: 99, Msg: "test"})
	if err != nil {
		t.Fatalf("Request: %v", err)
	}
	pong, ok := resp.(*pongResp)
	if !ok {
		t.Fatalf("resp type = %T, want *pongResp", resp)
	}
	if pong.RespSerial != 99 || pong.Reply != "dialer-ack-test" {
		t.Fatalf("pong = %+v, want {99 dialer-ack-test}", pong)
	}
}

func TestDialerRequestTimeout(t *testing.T) {
	port := freePort(t)

	// server handler that does NOT reply (returns nil)
	srvHandler := HandlerFunc(func(ctx context.Context, c Conn, msg any) (any, error) {
		return nil, nil
	})
	stop := startServer(t, port, srvHandler)
	defer stop()

	dialer := NewDialer(WithClientCodec(newTestCodec()))
	defer dialer.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	_, err := dialer.Request(ctx, "tcp", "127.0.0.1:"+strconv.Itoa(port), &pingReq{Serial: 1, Msg: "no-reply"})
	if err == nil {
		t.Fatal("expect timeout error, got nil")
	}
}

func TestDialerDial(t *testing.T) {
	port := freePort(t)

	srvHandler := HandlerFunc(func(ctx context.Context, c Conn, msg any) (any, error) {
		if e, ok := msg.(*echoMsg); ok {
			return &echoMsg{Body: e.Body}, nil
		}
		return nil, nil
	})
	stop := startServer(t, port, srvHandler)
	defer stop()

	dialer := NewDialer(WithClientCodec(newTestCodec()))
	defer dialer.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	conn, err := dialer.Dial(ctx, "tcp", "127.0.0.1:"+strconv.Itoa(port))
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	defer conn.Close()

	if conn.ID() == "" {
		t.Fatal("Dial returned conn with empty ID")
	}
	if conn.RemoteAddr() == nil {
		t.Fatal("Dial returned conn with nil RemoteAddr")
	}

	if err := conn.WriteAsync(ctx, &echoMsg{Body: "via-dialer"}); err != nil {
		t.Fatalf("Send: %v", err)
	}
}

func TestDialerClose(t *testing.T) {
	port := freePort(t)
	stop := startServer(t, port, noopServerHandler())
	defer stop()

	dialer := NewDialer(WithClientCodec(newTestCodec()))
	dialer.Close()

	ctx := context.Background()
	_, err := dialer.Dial(ctx, "tcp", "127.0.0.1:"+strconv.Itoa(port))
	if !errors.Is(err, ErrSessionClosed) {
		t.Fatalf("Dial after close: want ErrSessionClosed, got %v", err)
	}

	_, err = dialer.Request(ctx, "tcp", "127.0.0.1:"+strconv.Itoa(port), &pingReq{Serial: 1})
	if !errors.Is(err, ErrSessionClosed) {
		t.Fatalf("Request after close: want ErrSessionClosed, got %v", err)
	}
}

func TestDialerRequestConnectError(t *testing.T) {
	dialer := NewDialer(WithClientCodec(newTestCodec()))
	defer dialer.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	_, err := dialer.Request(ctx, "tcp", "127.0.0.1:1", &pingReq{Serial: 1})
	if err == nil {
		t.Fatal("expect dial error connecting to closed port, got nil")
	}
}

func TestDialerCodecRequired(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expect panic for NewDialer without codec")
		}
	}()
	_ = NewDialer()
}
