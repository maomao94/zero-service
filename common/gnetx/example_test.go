package gnetx_test

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"testing"
	"time"

	"github.com/panjf2000/gnet/v2"

	"zero-service/common/gnetx"
)

// 这个示例展示 gnetx 的典型用法：长度前缀分帧 + 请求-响应（tid 响应式）。
// 完整可运行版本见 TestExampleMinimalProtocol。

// === 协议消息 ===
type exampleReq struct{ Serial int }

func (m *exampleReq) TID() string    { return strconv.Itoa(m.Serial) }
func (m *exampleReq) MessageID() int { return 1 }

type exampleResp struct{ RespSerial int }

func (m *exampleResp) ResponseTID() string { return strconv.Itoa(m.RespSerial) }

// === 自定义序列化：payload 格式 "REQ:<serial>" / "RESP:<serial>" ===
type exampleSerializer struct{}

func (exampleSerializer) Decode(raw []byte, _ *gnetx.Session) (any, error) {
	s := string(raw)
	switch {
	case len(s) > 4 && s[:4] == "REQ:":
		serial, _ := strconv.Atoi(s[4:])
		return &exampleReq{Serial: serial}, nil
	case len(s) > 5 && s[:5] == "RESP:":
		serial, _ := strconv.Atoi(s[5:])
		return &exampleResp{RespSerial: serial}, nil
	}
	return nil, errors.New("unknown")
}

func (exampleSerializer) Encode(msg any, _ *gnetx.Session) ([]byte, error) {
	switch m := msg.(type) {
	case *exampleReq:
		return []byte("REQ:" + strconv.Itoa(m.Serial)), nil
	case *exampleResp:
		return []byte("RESP:" + strconv.Itoa(m.RespSerial)), nil
	}
	return nil, errors.New("unknown")
}

func newExampleCodec() gnetx.Codec {
	return gnetx.NewLengthPrefixCodec(4, binary.BigEndian, exampleSerializer{}, gnetx.WithMaxFrameSize(1<<20))
}

// frameWrite 用长度前缀封帧写入 net.Conn（模拟原始 TCP client）。
func frameWrite(c net.Conn, payload string) error {
	out := make([]byte, 4+len(payload))
	binary.BigEndian.PutUint32(out, uint32(len(payload)))
	copy(out[4:], payload)
	_, err := c.Write(out)
	return err
}

func frameRead(c net.Conn) (string, error) {
	hdr := make([]byte, 4)
	if _, err := io.ReadFull(c, hdr); err != nil {
		return "", err
	}
	n := binary.BigEndian.Uint32(hdr)
	body := make([]byte, n)
	if _, err := io.ReadFull(c, body); err != nil {
		return "", err
	}
	return string(body), nil
}

func freePortExample() int {
	l, _ := net.Listen("tcp", "127.0.0.1:0")
	port := l.Addr().(*net.TCPAddr).Port
	_ = l.Close()
	return port
}

// TestExampleMinimalProtocol 演示：server 收 REQ 回 RESP，原始 TCP client 发请求收响应。
// 这是 gnetx 最小可用形态：Codec + Handler + Server。
func TestExampleMinimalProtocol(t *testing.T) {
	port := freePortExample()

	// 1) 构造 server：handler 收到 exampleReq 返回 exampleResp
	handler := gnetx.HandlerFunc(func(ctx context.Context, s *gnetx.Session, msg any) (any, error) {
		if req, ok := msg.(*exampleReq); ok {
			return &exampleResp{RespSerial: req.Serial}, nil
		}
		return nil, nil
	})

	srv, err := gnetx.NewServer(
		gnetx.WithAddr(fmt.Sprintf("127.0.0.1:%d", port)),
		gnetx.WithCodec(newExampleCodec()),
		gnetx.WithHandler(handler),
		gnetx.WithMaxFrameLength(1<<20),
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

	// 2) 原始 TCP client 模拟：发 REQ:1，收 RESP:1
	conn, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	if err := frameWrite(conn, "REQ:1"); err != nil {
		t.Fatalf("write: %v", err)
	}
	resp, err := frameRead(conn)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if resp != "RESP:1" {
		t.Fatalf("resp = %q, want RESP:1", resp)
	}
}

// 让 gnet 在示例文件中可被引用（gnet.Conn 在 exampleSerializer 签名中作为参数类型注解）。
var _ gnet.Conn = (gnet.Conn)(nil)
