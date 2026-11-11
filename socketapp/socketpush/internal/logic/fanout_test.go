package logic

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
	"zero-service/socketapp/socketgtw/socketgtw"

	"github.com/zeromicro/go-zero/core/logx"
)

// fakeGtwClient 仅用于验证扇出调度逻辑，call 不会调用其嵌入的 gRPC 方法。
type fakeGtwClient struct {
	socketgtw.SocketGtwClient
	fail  bool
	block bool
}

func newFakeClients() map[string]socketgtw.SocketGtwClient {
	return map[string]socketgtw.SocketGtwClient{
		"ok-1": &fakeGtwClient{},
		"ok-2": &fakeGtwClient{},
		"bad":  &fakeGtwClient{fail: true},
		"slow": &fakeGtwClient{block: true},
	}
}

func fakeCall(called *atomic.Int32) func(context.Context, socketgtw.SocketGtwClient) error {
	return func(ctx context.Context, cli socketgtw.SocketGtwClient) error {
		if called != nil {
			called.Add(1)
		}
		fake, ok := cli.(*fakeGtwClient)
		if !ok {
			return nil
		}
		switch {
		case fake.block:
			<-ctx.Done()
			return ctx.Err()
		case fake.fail:
			return errors.New("gateway unreachable")
		default:
			return nil
		}
	}
}

func TestFanoutPushNoClients(t *testing.T) {
	logger := logx.WithContext(context.Background())
	if err := fanoutPush(context.Background(), logger, nil, 0, fakeCall(nil)); err != nil {
		t.Fatalf("expect nil error, got %v", err)
	}
}

func TestFanoutPushCollectsFailedAndTimedOutGateways(t *testing.T) {
	logger := logx.WithContext(context.Background())
	var called atomic.Int32
	err := fanoutPush(context.Background(), logger, newFakeClients(), 50*time.Millisecond, fakeCall(&called))
	if err == nil {
		t.Fatal("expect error, got nil")
	}

	var fanoutErr *PushFanoutError
	if !errors.As(err, &fanoutErr) {
		t.Fatalf("expect *PushFanoutError, got %T: %v", err, err)
	}
	if fanoutErr.Total != 4 {
		t.Fatalf("expect total 4, got %d", fanoutErr.Total)
	}
	if len(fanoutErr.Failed) != 2 {
		t.Fatalf("expect 2 failed gateways, got %v", fanoutErr.Failed)
	}
	// 失败节点按名排序，便于日志和断言稳定。
	if fanoutErr.Failed[0] != "bad" || fanoutErr.Failed[1] != "slow" {
		t.Fatalf("unexpected failed gateways: %v", fanoutErr.Failed)
	}
	if called.Load() != 4 {
		t.Fatalf("expect all 4 gateways invoked, got %d", called.Load())
	}
}

func TestFanoutPushSuccess(t *testing.T) {
	logger := logx.WithContext(context.Background())
	clients := map[string]socketgtw.SocketGtwClient{
		"a": &fakeGtwClient{},
		"b": &fakeGtwClient{},
	}
	if err := fanoutPush(context.Background(), logger, clients, 0, fakeCall(nil)); err != nil {
		t.Fatalf("expect nil error, got %v", err)
	}
}
