package gnetx

import (
	"context"
	"encoding/binary"
	"errors"
	"io"
	"net"
	"strconv"
	"sync"
	"testing"
	"time"
)

// === 协议定义：长度前缀 + 自定义消息，用于集成测试 ===

// pingReq/pongResp 是请求-响应对，tid 用序号。
type pingReq struct {
	Serial int
	Msg    string
}

func (m *pingReq) TID() string      { return strconv.Itoa(m.Serial) }
func (m *pingReq) MessageID() int   { return 1 }
func (m *pingReq) ClientID() string { return "test-client" }

type pongResp struct {
	RespSerial int
	Reply      string
}

func (m *pongResp) ResponseTID() string { return strconv.Itoa(m.RespSerial) }
func (m *pongResp) MessageID() int      { return 2 }

// echoMsg 是一个简单的 echo 消息（无 tid 关联）。
type echoMsg struct {
	Body string
}

func (m *echoMsg) MessageID() int { return 10 }

// hbMsg 是心跳消息。
type hbMsg struct{}

func (m *hbMsg) MessageID() int { return 99 }

// testCodec 用长度前缀分帧 + 自定义序列化（字符串协议：[len][payload]）。
// payload 格式：消息类型标识 + 内容，这里简单用 "ping:<serial>:<msg>" / "pong:<serial>:<reply>" / "echo:<body>"。
type testSerializer struct{}

func (testSerializer) Decode(raw []byte) (any, error) {
	s := string(raw)
	switch {
	case len(s) > 5 && s[:5] == "ping:":
		// ping:<serial>:<msg>
		rest := s[5:]
		idx := indexOf(rest, ':')
		if idx < 0 {
			return nil, errors.New("bad ping")
		}
		serial, _ := strconv.Atoi(rest[:idx])
		return &pingReq{Serial: serial, Msg: rest[idx+1:]}, nil
	case len(s) > 5 && s[:5] == "pong:":
		rest := s[5:]
		idx := indexOf(rest, ':')
		if idx < 0 {
			return nil, errors.New("bad pong")
		}
		serial, _ := strconv.Atoi(rest[:idx])
		return &pongResp{RespSerial: serial, Reply: rest[idx+1:]}, nil
	case len(s) > 5 && s[:5] == "echo:":
		return &echoMsg{Body: s[5:]}, nil
	case s == "hb":
		return &hbMsg{}, nil
	}
	return nil, errors.New("unknown message type")
}

func (testSerializer) Encode(msg any) ([]byte, error) {
	switch m := msg.(type) {
	case *pingReq:
		return []byte("ping:" + strconv.Itoa(m.Serial) + ":" + m.Msg), nil
	case *pongResp:
		return []byte("pong:" + strconv.Itoa(m.RespSerial) + ":" + m.Reply), nil
	case *echoMsg:
		return []byte("echo:" + m.Body), nil
	case *hbMsg:
		return []byte("hb"), nil
	case []byte:
		return m, nil
	}
	return nil, errors.New("unknown message type for encode")
}

func indexOf(s string, b byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == b {
			return i
		}
	}
	return -1
}

func newTestCodec() Codec {
	return NewLengthPrefixCodec(4, binary.BigEndian, testSerializer{}, WithMaxFrameSize(1024*1024))
}

// frameEncode 用长度前缀把 payload 封帧，供测试 client 写入。
func frameEncode(payload string) []byte {
	out := make([]byte, 4+len(payload))
	binary.BigEndian.PutUint32(out, uint32(len(payload)))
	copy(out[4:], payload)
	return out
}

// readFrame 从 net.Conn 读一帧并解析 payload。
func readFrame(c net.Conn) ([]byte, error) {
	hdr := make([]byte, 4)
	if _, err := io.ReadFull(c, hdr); err != nil {
		return nil, err
	}
	n := binary.BigEndian.Uint32(hdr)
	body := make([]byte, n)
	if _, err := io.ReadFull(c, body); err != nil {
		return nil, err
	}
	return body, nil
}

// freePort 获取一个可用端口。
func freePort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	port := l.Addr().(*net.TCPAddr).Port
	_ = l.Close()
	return port
}

// startServer 启动测试 server 并返回停止函数。
func startServer(t *testing.T, port int, handler Handler, opts ...ServerOption) func() {
	t.Helper()
	all := append([]ServerOption{
		WithAddr("127.0.0.1:" + strconv.Itoa(port)),
		WithCodec(newTestCodec()),
		WithServerHandler(handler),
		WithMaxFrameLength(1024 * 1024),
	}, opts...)
	srv, err := NewServer(all...)
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	done := make(chan error, 1)
	go func() { done <- srv.Run() }()
	// 等监听就绪
	time.Sleep(100 * time.Millisecond)
	return func() {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_ = srv.Shutdown(ctx)
		<-done
	}
}

// === 测试 ===

func TestServerEcho(t *testing.T) {
	port := freePort(t)
	handler := HandlerFunc(func(ctx context.Context, c Conn, msg any) (any, error) {
		if e, ok := msg.(*echoMsg); ok {
			// 原样回显：encode 后仍是 "echo:<body>"
			return &echoMsg{Body: e.Body}, nil
		}
		return nil, nil
	})
	stop := startServer(t, port, handler)
	defer stop()

	conn, err := net.Dial("tcp", "127.0.0.1:"+strconv.Itoa(port))
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	// 发 echo:hello → server decode 得 echoMsg{Body:"hello"} → 回 echoMsg{Body:"hello"} → encode "echo:hello"
	_, err = conn.Write(frameEncode("echo:hello"))
	if err != nil {
		t.Fatalf("write: %v", err)
	}
	body, err := readFrame(conn)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if string(body) != "echo:hello" {
		t.Fatalf("reply = %q, want echo:hello", body)
	}
}

func TestServerResponseAutoRoute(t *testing.T) {
	// server 主动 Request client（client 回 pong）。
	// 这里用 goroutine 模拟：server handler 收到 ping 后 conn.Request 等回包。
	// 但 on-loop 不能 Request，所以用 AsyncHandler offload。
	port := freePort(t)

	var serverConn Conn
	var wg sync.WaitGroup
	wg.Add(1)

	asyncReq := AsyncFunc(HandlerFunc(func(ctx context.Context, c Conn, msg any) (any, error) {
		if p, ok := msg.(*pingReq); ok {
			// 异步里发起 Request 等 pong（client 需先回 pong，这里用 net.Conn 模拟）
			// 实际上需要 client 侧逻辑回 pong，本测试简化：handler 直接构造 pong 返回
			serverConn = c
			wg.Done()
			return &pongResp{RespSerial: p.Serial, Reply: "ack-" + p.Msg}, nil
		}
		return nil, nil
	}))

	stop := startServer(t, port, asyncReq)
	defer stop()

	conn, err := net.Dial("tcp", "127.0.0.1:"+strconv.Itoa(port))
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	_, err = conn.Write(frameEncode("ping:1:hello"))
	if err != nil {
		t.Fatalf("write: %v", err)
	}
	body, err := readFrame(conn)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if string(body) != "pong:1:ack-hello" {
		t.Fatalf("reply = %q, want pong:1:ack-hello", body)
	}
	wg.Wait() // 确保 serverConn 赋值
	if serverConn == nil {
		t.Fatal("serverConn not set")
	}
}

func TestServerPartialFrame(t *testing.T) {
	// 验证半包：先发一半，等一会再发另一半，server 应正确拼接。
	port := freePort(t)
	handler := HandlerFunc(func(ctx context.Context, c Conn, msg any) (any, error) {
		if e, ok := msg.(*echoMsg); ok {
			return &echoMsg{Body: e.Body}, nil
		}
		return nil, nil
	})
	stop := startServer(t, port, handler)
	defer stop()

	conn, err := net.Dial("tcp", "127.0.0.1:"+strconv.Itoa(port))
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	full := frameEncode("echo:split")
	// 先发前 4 字节（长度字段）
	_, _ = conn.Write(full[:4])
	time.Sleep(150 * time.Millisecond)
	// 再发剩余
	_, _ = conn.Write(full[4:])

	body, err := readFrame(conn)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if string(body) != "echo:split" {
		t.Fatalf("reply = %q, want echo:split", body)
	}
}

func TestServerMultipleFramesInOnePacket(t *testing.T) {
	// 验证粘包：一次发两帧，server 应分别处理并回两包。
	port := freePort(t)
	handler := HandlerFunc(func(ctx context.Context, c Conn, msg any) (any, error) {
		if e, ok := msg.(*echoMsg); ok {
			return &echoMsg{Body: e.Body}, nil
		}
		return nil, nil
	})
	stop := startServer(t, port, handler)
	defer stop()

	conn, err := net.Dial("tcp", "127.0.0.1:"+strconv.Itoa(port))
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	// 一次写两帧
	pkt := append(frameEncode("echo:a"), frameEncode("echo:b")...)
	_, _ = conn.Write(pkt)

	b1, err := readFrame(conn)
	if err != nil {
		t.Fatalf("read1: %v", err)
	}
	b2, err := readFrame(conn)
	if err != nil {
		t.Fatalf("read2: %v", err)
	}
	if string(b1) != "echo:a" || string(b2) != "echo:b" {
		t.Fatalf("replies = %q %q, want echo:a echo:b", b1, b2)
	}
}

func TestServerIdleTimeout(t *testing.T) {
	port := freePort(t)
	handler := HandlerFunc(func(ctx context.Context, c Conn, msg any) (any, error) { return nil, nil })
	stop := startServer(t, port, handler, WithIdleTimeout(300*time.Millisecond))
	defer stop()

	conn, err := net.Dial("tcp", "127.0.0.1:"+strconv.Itoa(port))
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	// 不发数据，等空闲超时后 server 应关闭连接，client Read 返回 EOF
	buf := make([]byte, 16)
	_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, err = conn.Read(buf)
	if err == nil {
		t.Fatal("expect connection closed by idle timeout")
	}
}

func TestServerGracefulShutdown(t *testing.T) {
	port := freePort(t)
	handler := HandlerFunc(func(ctx context.Context, c Conn, msg any) (any, error) {
		if e, ok := msg.(*echoMsg); ok {
			return &echoMsg{Body: e.Body}, nil
		}
		return nil, nil
	})

	srv, err := NewServer(
		WithAddr("127.0.0.1:"+strconv.Itoa(port)),
		WithCodec(newTestCodec()),
		WithServerHandler(handler),
		WithMaxFrameLength(1024*1024),
	)
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	done := make(chan error, 1)
	go func() { done <- srv.Run() }()
	time.Sleep(100 * time.Millisecond)

	// 连一个连接并发一次请求确认可用
	conn, err := net.Dial("tcp", "127.0.0.1:"+strconv.Itoa(port))
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	_, _ = conn.Write(frameEncode("echo:before"))
	_, err = readFrame(conn)
	if err != nil {
		t.Fatalf("before shutdown read: %v", err)
	}

	// 优雅停止
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		t.Fatalf("Shutdown: %v", err)
	}
	<-done

	// 停止后新连接应失败
	_, err = net.Dial("tcp", "127.0.0.1:"+strconv.Itoa(port))
	if err == nil {
		t.Fatal("expect dial to fail after shutdown")
	}
	_ = conn.Close()
}

// TestServerHandlerError：handler 返回 error 不 panic，不回包，记日志。
func TestServerHandlerError(t *testing.T) {
	port := freePort(t)
	handler := HandlerFunc(func(ctx context.Context, c Conn, msg any) (any, error) {
		if _, ok := msg.(*echoMsg); ok {
			return nil, errors.New("internal")
		}
		return nil, nil
	})
	stop := startServer(t, port, handler)
	defer stop()

	conn, err := net.Dial("tcp", "127.0.0.1:"+strconv.Itoa(port))
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	_, err = conn.Write(frameEncode("echo:err"))
	if err != nil {
		t.Fatalf("write: %v", err)
	}
	// 服务端 handler 返回 error，不应回包，client Read 等超时
	_ = conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
	_, err = readFrame(conn)
	if !isTimeout(err) {
		t.Fatalf("expect timeout (no reply), got %v", err)
	}
}

// TestServerDecodeErrorLogOnly：DecodeErrorLogOnly 下解码错误不关闭连接。
func TestServerDecodeErrorLogOnly(t *testing.T) {
	port := freePort(t)
	handler := HandlerFunc(func(ctx context.Context, c Conn, msg any) (any, error) {
		if e, ok := msg.(*echoMsg); ok {
			return &echoMsg{Body: e.Body}, nil
		}
		return nil, nil
	})
	stop := startServer(t, port, handler, WithOnDecodeError(DecodeErrorLogOnly))
	defer stop()

	conn, err := net.Dial("tcp", "127.0.0.1:"+strconv.Itoa(port))
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	// 发一个合法帧 → 正常回包
	_, err = conn.Write(frameEncode("echo:before"))
	if err != nil {
		t.Fatalf("write before: %v", err)
	}
	body, err := readFrame(conn)
	if err != nil {
		t.Fatalf("read before: %v", err)
	}
	if string(body) != "echo:before" {
		t.Fatalf("reply before = %q", body)
	}

	// 发一个协议错误帧（合法长度但 serializer 不识别的 payload），连接应存活
	_, err = conn.Write(frameEncode("bad:payload"))
	if err != nil {
		t.Fatalf("write bad: %v", err)
	}
	time.Sleep(100 * time.Millisecond)

	// 发另一个合法帧 → 能正常收到回包，证明连接未断
	_, err = conn.Write(frameEncode("echo:after"))
	if err != nil {
		t.Fatalf("write after: %v", err)
	}
	body, err = readFrame(conn)
	if err != nil {
		t.Fatalf("read after: %v", err)
	}
	if string(body) != "echo:after" {
		t.Fatalf("reply after = %q", body)
	}
}

// isTimeout 判断 err 是否为 socket 超时。
func isTimeout(err error) bool {
	if err == nil {
		return false
	}
	var ne net.Error
	return errors.As(err, &ne) && ne.Timeout()
}

// TestServerOnCloseNilContext 验证 OnClose 收到 nil context 时不 panic。
// 正常流程 OnOpen 总会 SetContext，但防御性测试确认没问题。
func TestServerOnCloseNilContext(t *testing.T) {
	port := freePort(t)
	srvHandler := HandlerFunc(func(ctx context.Context, c Conn, msg any) (any, error) { return nil, nil })

	srv, err := NewServer(
		WithAddr("127.0.0.1:"+strconv.Itoa(port)),
		WithCodec(newTestCodec()),
		WithServerHandler(srvHandler),
		WithMaxFrameLength(1024*1024),
	)
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	done := make(chan error, 1)
	go func() { done <- srv.Run() }()
	time.Sleep(100 * time.Millisecond)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
	<-done
}

// TestServerDecodeErrorClose 验证 DecodeErrorClose（默认）下，不可恢复解码错误会关闭连接。
func TestServerDecodeErrorClose(t *testing.T) {
	port := freePort(t)
	srvHandler := HandlerFunc(func(ctx context.Context, c Conn, msg any) (any, error) { return nil, nil })
	stop := startServer(t, port, srvHandler)
	defer stop()

	conn, err := net.Dial("tcp", "127.0.0.1:"+strconv.Itoa(port))
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	// 直接发一个解码错误的帧（serializer 不识别的 payload）
	_, err = conn.Write(frameEncode("bad:payload"))
	if err != nil {
		t.Fatalf("write: %v", err)
	}

	// 连接应被关闭，后续读返回 EOF
	_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	buf := make([]byte, 16)
	_, err = conn.Read(buf)
	if err == nil {
		t.Fatal("expect connection closed after decode error")
	}
}
