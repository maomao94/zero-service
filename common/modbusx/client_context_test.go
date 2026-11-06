package modbusx

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/grid-x/modbus"
)

type blockedTransport struct {
	entered chan struct{}
	release chan struct{}
	sends   atomic.Int32
}

func (t *blockedTransport) Send(context.Context, []byte) ([]byte, error) {
	if t.sends.Add(1) == 1 {
		close(t.entered)
		<-t.release
	}
	return nil, nil
}

func (t *blockedTransport) Close() error { return nil }

func TestClientRejectsCanceledQueuedWrite(t *testing.T) {
	base := &blockedTransport{entered: make(chan struct{}), release: make(chan struct{})}
	// 使用真实 SDK 编码读写，确认取消的写请求不会进入底层 Send。
	h := modbus.NewTCPClientHandler("unused")
	cli := &ModbusClient{client: &clientWithCloser{Client: modbus.NewClient2(h, base), Closer: base}}
	readDone := make(chan struct{})
	go func() { defer close(readDone); _, _ = cli.ReadHoldingRegisters(context.Background(), 0, 1) }()
	<-base.entered
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	writeDone := make(chan error, 1)
	go func() { _, err := cli.WriteSingleRegister(ctx, 1, 42); writeDone <- err }()
	cancel()
	close(base.release)
	<-readDone
	select {
	case err := <-writeDone:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("write = %v", err)
		}
	case <-time.After(time.Second):
		t.Error("canceled write did not return after read completed")
	}
	if got := base.sends.Load(); got != 1 {
		t.Fatalf("sent %d requests, want only read", got)
	}
}
