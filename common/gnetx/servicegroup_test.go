package gnetx

import (
	"context"
	"net"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/zeromicro/go-zero/core/service"
)

// TestServerWithServiceGroup 验证 Server 实现 service.Service 接口，可接入 service.NewServiceGroup()。
// 用 service.Group 管理 Server 生命周期（Start 阻塞，proc 信号或 Stop 触发停止）。
func TestServerWithServiceGroup(t *testing.T) {
	port := freePort(t)

	srv, err := NewServer(
		WithAddr("127.0.0.1:"+strconv.Itoa(port)),
		WithCodec(newTestCodec()),
		WithServerHandler(HandlerFunc(func(ctx context.Context, c Conn, msg any) (any, error) {
			if e, ok := msg.(*echoMsg); ok {
				return &echoMsg{Body: e.Body}, nil
			}
			return nil, nil
		})),
		WithMaxFrameLength(1024*1024),
	)
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	sg := service.NewServiceGroup()
	sg.Add(srv)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		sg.Start() // 阻塞
	}()

	// 等监听就绪
	time.Sleep(150 * time.Millisecond)

	// 验证 server 可用
	conn, err := net.Dial("tcp", "127.0.0.1:"+strconv.Itoa(port))
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()
	_, _ = conn.Write(frameEncode("echo:hi"))
	body, err := readFrame(conn)
	if err != nil || string(body) != "echo:hi" {
		t.Fatalf("echo: err=%v body=%q", err, body)
	}

	// 停止 service group
	sg.Stop()
	wg.Wait()

	// 停止后新连接应失败
	_, err = net.Dial("tcp", "127.0.0.1:"+strconv.Itoa(port))
	if err == nil {
		t.Fatal("expect dial to fail after service group stop")
	}
}
