package nacosx

import (
	"context"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/nacos-group/nacos-sdk-go/v2/clients/naming_client"
	"github.com/nacos-group/nacos-sdk-go/v2/model"
	"github.com/nacos-group/nacos-sdk-go/v2/vo"
)

type fakeNamingClient struct {
	naming_client.INamingClient

	mu           sync.Mutex
	subscribes   int
	unsubscribes int
	failTimes    int
}

func (f *fakeNamingClient) Subscribe(param *vo.SubscribeParam) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.subscribes++
	if f.subscribes <= f.failTimes {
		return errFakeSubscribe
	}
	return nil
}

func (f *fakeNamingClient) Unsubscribe(*vo.SubscribeParam) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.unsubscribes++
	return nil
}

type fakeError string

func (e fakeError) Error() string { return string(e) }

var errFakeSubscribe error = fakeError("fake subscribe failure")

func TestExtractHealthyGRPCInstancesHandlesEmptyInput(t *testing.T) {
	for name, instances := range map[string][]model.Instance{
		"nil":   nil,
		"empty": {},
	} {
		t.Run(name, func(t *testing.T) {
			if addrs := extractHealthyGRPCInstances(instances); len(addrs) != 0 {
				t.Fatalf("unexpected addresses: %v", addrs)
			}
		})
	}
}

func TestExtractHealthyInstancesReflectsOnlyAddresses(t *testing.T) {
	base := model.Instance{Ip: "10.0.0.1", Port: 25001, Healthy: true, Enable: true}
	changed := base
	changed.InstanceId = "another-instance"
	changed.Weight = 10
	changed.Metadata = map[string]string{"gRPC_port": "25001", "deployMode": "cluster"}

	if got, want := extractHealthyGRPCInstances([]model.Instance{changed}), extractHealthyGRPCInstances([]model.Instance{base}); !slices.Equal(got, want) {
		t.Fatalf("weight/metadata-only change should not alter addresses: got %v, want %v", got, want)
	}

	unhealthy := base
	unhealthy.Healthy = false
	if addrs := extractHealthyGRPCInstances([]model.Instance{unhealthy}); len(addrs) != 0 {
		t.Fatalf("unhealthy instance should be filtered out, got %v", addrs)
	}
}

func TestNormalizeAddrs(t *testing.T) {
	got := normalizeAddrs([]string{"b:1", "a:1", "b:1", "", " "})
	want := []string{"a:1", "b:1"}
	if len(got) != len(want) {
		t.Fatalf("normalizeAddrs = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("normalizeAddrs = %v, want %v", got, want)
		}
	}
}

func TestDeliverAddrsDeduplicatesAndReportsEmpty(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pipe := make(chan []string)
	var mu sync.Mutex
	var calls [][]string
	done := make(chan struct{}, 1)
	go deliverAddrs(ctx, pipe, func(addrs []string) {
		mu.Lock()
		calls = append(calls, addrs)
		mu.Unlock()
		select {
		case done <- struct{}{}:
		default:
		}
	})

	wait := func() {
		t.Helper()
		select {
		case <-done:
		case <-time.After(time.Second):
			t.Fatal("onChange was not called in time")
		}
	}

	pipe <- []string{"b:1", "a:1", "b:1"}
	wait()
	pipe <- []string{"a:1", "b:1"} // 集合未变化，不应重复回调
	select {
	case <-done:
		t.Fatal("unchanged address set should not trigger onChange")
	case <-time.After(50 * time.Millisecond):
	}
	pipe <- nil // 实例全部下线
	wait()

	mu.Lock()
	defer mu.Unlock()
	if len(calls) != 2 {
		t.Fatalf("unexpected callbacks: %v", calls)
	}
	if len(calls[0]) != 2 || calls[0][0] != "a:1" || calls[0][1] != "b:1" {
		t.Fatalf("unexpected first callback: %v", calls[0])
	}
	if len(calls[1]) != 0 {
		t.Fatalf("empty instance list should be reported as empty, got %v", calls[1])
	}
	if calls[1] == nil {
		t.Fatal("empty instance list should be a non-nil empty slice")
	}
}

func TestSubscribeNacosRetriesAndUnsubscribes(t *testing.T) {
	restore := withSubscribeRetry(5*time.Millisecond, 20*time.Millisecond)
	defer restore()

	client := &fakeNamingClient{failTimes: 2}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	subscribeNacos(ctx, client, target{Service: "socket-gtw"}, func([]model.Instance, error) {})

	client.mu.Lock()
	defer client.mu.Unlock()
	if client.subscribes != 3 {
		t.Fatalf("subscribes = %d, want 3", client.subscribes)
	}
	// 每次重试前都要摘掉上一次失败注册的回调，避免同一次推送触发多次回调。
	if client.unsubscribes != 2 {
		t.Fatalf("unsubscribes = %d, want 2", client.unsubscribes)
	}
}

func TestSubscribeNacosStopsOnContextCancel(t *testing.T) {
	restore := withSubscribeRetry(5*time.Millisecond, 20*time.Millisecond)
	defer restore()

	client := &fakeNamingClient{failTimes: 1 << 30}
	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		subscribeNacos(ctx, client, target{Service: "socket-gtw"}, func([]model.Instance, error) {})
		close(done)
	}()

	time.Sleep(20 * time.Millisecond)
	cancel()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("subscribeNacos did not stop after context cancellation")
	}
}

func withSubscribeRetry(min, max time.Duration) func() {
	oldMin, oldMax := subscribeRetryMin, subscribeRetryMax
	subscribeRetryMin, subscribeRetryMax = min, max
	return func() {
		subscribeRetryMin, subscribeRetryMax = oldMin, oldMax
	}
}
