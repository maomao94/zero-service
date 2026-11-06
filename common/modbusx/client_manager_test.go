package modbusx

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestClientManagerConcurrentReuseAndCodeIsolation(t *testing.T) {
	m := NewClientManager()
	defer m.Close()
	load := func(context.Context) (*ModbusClientConf, error) {
		return &ModbusClientConf{Address: "127.0.0.1:1", Slave: 1}, nil
	}
	clients := make(chan *ModbusClient, 32)
	var wg sync.WaitGroup
	for range 32 {
		wg.Go(func() {
			cli, err := m.GetClient(context.Background(), "device", load)
			if err != nil {
				t.Errorf("Get: %v", err)
			}
			clients <- cli
		})
	}
	wg.Wait()
	close(clients)
	var first *ModbusClient
	for cli := range clients {
		if first == nil {
			first = cli
		}
		if cli != first {
			t.Fatal("same code created multiple clients")
		}
	}
	other, err := m.GetClient(context.Background(), "other", load)
	if err != nil || other == first {
		t.Fatalf("code isolation: %v", err)
	}
	defaultClient, err := m.GetClient(context.Background(), "", load)
	if err != nil || defaultClient == first {
		t.Fatalf("default isolation: %v", err)
	}
}

func TestClientManagerCleanStaleRemovesUpdatedConfig(t *testing.T) {
	m := NewClientManager()
	defer m.Close()

	var current atomic.Pointer[ModbusClientConf]
	current.Store(&ModbusClientConf{Address: "127.0.0.1:1", Slave: 1})
	load := func(context.Context) (*ModbusClientConf, error) {
		conf := *current.Load()
		return &conf, nil
	}

	first, err := m.GetClient(context.Background(), "device", load)
	if err != nil {
		t.Fatal(err)
	}
	again, err := m.GetClient(context.Background(), "device", load)
	if err != nil || again != first {
		t.Fatalf("cache hit: %v", err)
	}

	// 配置未变化时保留客户端，不重建连接。
	m.cleanStale()
	same, err := m.GetClient(context.Background(), "device", load)
	if err != nil || same != first {
		t.Fatalf("unchanged config removed client: %v", err)
	}

	// 配置变化后由后台清理移除，下一次取用重建。
	current.Store(&ModbusClientConf{Address: "127.0.0.1:2", Slave: 2})
	m.cleanStale()
	next, err := m.GetClient(context.Background(), "device", load)
	if err != nil || next == first {
		t.Fatalf("updated config did not rebuild: %v", err)
	}
	if _, err := first.WriteSingleRegister(context.Background(), 0, 42); !errors.Is(err, ErrClientInvalidated) {
		t.Fatalf("updated config left old reference usable: %v", err)
	}
	if next.invalidated.Load() {
		t.Fatal("rebuilt client is invalidated")
	}
}

func TestClientManagerCleanStaleRemovesUnavailableConfig(t *testing.T) {
	m := NewClientManager()
	defer m.Close()

	var unavailable atomic.Bool
	load := func(context.Context) (*ModbusClientConf, error) {
		if unavailable.Load() {
			return nil, ErrConfigUnavailable
		}
		return &ModbusClientConf{Address: "127.0.0.1:1"}, nil
	}
	first, err := m.GetClient(context.Background(), "device", load)
	if err != nil {
		t.Fatal(err)
	}
	unavailable.Store(true)
	m.cleanStale()
	if _, err := first.WriteSingleRegister(context.Background(), 0, 42); !errors.Is(err, ErrClientInvalidated) {
		t.Fatalf("unavailable config left old reference usable: %v", err)
	}
	if _, err := m.GetClient(context.Background(), "device", load); err == nil {
		t.Fatal("unavailable config returned a client")
	}
}

func TestClientManagerCleanStaleKeepsClientOnLoadError(t *testing.T) {
	m := NewClientManager()
	defer m.Close()

	var failed atomic.Bool
	load := func(context.Context) (*ModbusClientConf, error) {
		if failed.Load() {
			return nil, errors.New("database unavailable")
		}
		return &ModbusClientConf{Address: "127.0.0.1:1"}, nil
	}
	first, err := m.GetClient(context.Background(), "device", load)
	if err != nil {
		t.Fatal(err)
	}
	failed.Store(true)
	m.cleanStale()

	// 查询失败不清客户端，避免数据库故障放大为设备通信中断。
	got, err := m.GetClient(context.Background(), "device", load)
	if err != nil || got != first {
		t.Fatalf("load error removed client: %v", err)
	}
}

func TestClientManagerCleanStaleDoesNotWaitForDeviceIO(t *testing.T) {
	for _, action := range []string{"unavailable", "updated"} {
		t.Run(action, func(t *testing.T) {
			m := NewClientManager()
			defer m.Close()
			var changed atomic.Bool
			load := func(context.Context) (*ModbusClientConf, error) {
				address := "127.0.0.1:1"
				if changed.Load() {
					if action == "unavailable" {
						return nil, ErrConfigUnavailable
					}
					address = "127.0.0.1:2"
				}
				return &ModbusClientConf{Address: address}, nil
			}
			clients := make([]*ModbusClient, 0, 2)
			for _, code := range []string{"a", "b"} {
				cli, err := m.GetClient(context.Background(), code, load)
				if err != nil {
					t.Fatal(err)
				}
				clients = append(clients, cli)
			}
			// 模拟两个设备均有在途 I/O，关闭连接必须等待发送锁。
			for _, cli := range clients {
				cli.mu.Lock()
			}
			// 即使断言失败也先释放发送锁，再执行管理器关闭。
			defer func() {
				for _, cli := range clients {
					cli.mu.Unlock()
				}
			}()
			changed.Store(true)
			cleaned := make(chan struct{})
			go func() { m.cleanStale(); close(cleaned) }()
			select {
			case <-cleaned:
			case <-time.After(time.Second):
				t.Fatal("configuration cleanup blocked by device I/O")
			}
			for _, cli := range clients {
				if !cli.invalidated.Load() {
					t.Fatal("cleanup left an old client usable")
				}
			}
			m.mu.Lock()
			remaining := len(m.clients)
			m.mu.Unlock()
			if remaining != 0 {
				t.Fatalf("cached clients = %d, want 0", remaining)
			}
			// 缓存已经清空，退出仍须等待被后台淘汰的连接关闭。
			closed := make(chan struct{})
			go func() { m.Close(); close(closed) }()
			select {
			case <-closed:
				t.Fatal("manager closed before retired connections finished")
			case <-time.After(20 * time.Millisecond):
			}
			// 释放发送锁后清空切片，避免 defer 重复解锁。
			for _, cli := range clients {
				cli.mu.Unlock()
			}
			clients = nil
			select {
			case <-closed:
			case <-time.After(time.Second):
				t.Fatal("manager did not finish closing retired connections")
			}
		})
	}
}

// RemoveClient 必须等旧配置加载发布后再删除，不能让旧结果回填。
func TestClientManagerRemoveDuringLoad(t *testing.T) {
	m := NewClientManager()
	defer m.Close()
	loaded, resume := make(chan struct{}), make(chan struct{})
	var unavailable atomic.Bool
	result := make(chan error, 1)
	go func() {
		_, err := m.GetClient(context.Background(), "device", func(context.Context) (*ModbusClientConf, error) {
			conf := &ModbusClientConf{Address: "127.0.0.1:1"}
			close(loaded)
			<-resume
			return conf, nil
		})
		result <- err
	}()
	<-loaded
	unavailable.Store(true)
	removed := make(chan bool, 1)
	go func() { removed <- m.RemoveClient("device") }()
	select {
	case <-removed:
		close(resume)
		<-result
		t.Fatal("remove returned before in-flight load completed")
	case <-time.After(20 * time.Millisecond):
	}
	close(resume)
	if !<-removed {
		t.Fatal("loaded client was not removed")
	}
	if err := <-result; err != nil && !errors.Is(err, ErrConfigChanged) {
		t.Fatalf("in-flight load: %v", err)
	}
	cli, err := m.GetClient(context.Background(), "device", func(context.Context) (*ModbusClientConf, error) {
		if unavailable.Load() {
			return nil, ErrConfigUnavailable
		}
		return &ModbusClientConf{Address: "127.0.0.1:2"}, nil
	})
	if cli != nil || !errors.Is(err, ErrConfigUnavailable) {
		t.Fatalf("deleted configuration returned stale client: %v", err)
	}
}

func TestClientManagerSlowLoadDoesNotBlockOtherCodes(t *testing.T) {
	m := NewClientManager()
	defer m.Close()
	load := func(context.Context) (*ModbusClientConf, error) {
		return &ModbusClientConf{Address: "127.0.0.1:1"}, nil
	}
	for _, code := range []string{"cached", ""} {
		if _, err := m.GetClient(context.Background(), code, load); err != nil {
			t.Fatal(err)
		}
	}
	entered, release, done := make(chan struct{}), make(chan struct{}), make(chan struct{})
	go func() {
		defer close(done)
		_, _ = m.GetClient(context.Background(), "slow", func(ctx context.Context) (*ModbusClientConf, error) {
			close(entered)
			<-release
			return load(ctx)
		})
	}()
	<-entered
	defer func() { close(release); <-done }()
	for _, code := range []string{"cached", "", "new"} {
		result := make(chan error, 1)
		go func() {
			_, err := m.GetClient(context.Background(), code, load)
			result <- err
		}()
		select {
		case err := <-result:
			if err != nil {
				t.Fatal(err)
			}
		case <-time.After(time.Second):
			t.Fatalf("code %q blocked by another load", code)
		}
	}
}

func TestClientManagerCloseDuringLoad(t *testing.T) {
	m := NewClientManager()
	entered, release := make(chan struct{}), make(chan struct{})
	done := make(chan error, 1)
	go func() {
		_, err := m.GetClient(context.Background(), "device", func(context.Context) (*ModbusClientConf, error) {
			close(entered)
			<-release
			return &ModbusClientConf{Address: "127.0.0.1:1"}, nil
		})
		done <- err
	}()
	<-entered
	m.Close()
	close(release)
	if err := <-done; !errors.Is(err, ErrManagerClosed) {
		t.Fatalf("load after close = %v", err)
	}
}

func TestClientManagerFailureAndClose(t *testing.T) {
	m := NewClientManager()
	want := errors.New("configuration missing")
	load := func(context.Context) (*ModbusClientConf, error) { return nil, want }
	if _, err := m.GetClient(context.Background(), "missing", load); !errors.Is(err, want) {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := m.GetClient(ctx, "", load); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	m.Close()
	m.Close()
	if _, err := m.GetClient(context.Background(), "", load); !errors.Is(err, ErrManagerClosed) {
		t.Fatal(err)
	}
}

func TestClientManagerInvalidatesOldReferences(t *testing.T) {
	for _, action := range []string{"remove", "close"} {
		t.Run(action, func(t *testing.T) {
			m := NewClientManager()
			defer m.Close()
			var sends atomic.Int32
			address := startFakeModbusServer(t, func(req []byte) []byte {
				sends.Add(1)
				return req // 写单寄存器响应回显请求。
			})
			load := func(context.Context) (*ModbusClientConf, error) {
				return &ModbusClientConf{Address: address, Timeout: 1000}, nil
			}
			old, err := m.GetClient(context.Background(), "device", load)
			if err != nil {
				t.Fatal(err)
			}
			if action == "remove" {
				if !m.RemoveClient("device") {
					t.Fatal("cached client not removed")
				}
			} else {
				m.Close()
			}
			if _, err := old.WriteSingleRegister(context.Background(), 0, 42); !errors.Is(err, ErrClientInvalidated) {
				t.Fatalf("old reference write = %v", err)
			}
			if sends.Load() != 0 {
				t.Fatal("invalidated client sent a write to device")
			}
			if action == "remove" {
				next, err := m.GetClient(context.Background(), "device", load)
				if err != nil || next == old {
					t.Fatalf("rebuild: %v", err)
				}
				if _, err := next.WriteSingleRegister(context.Background(), 0, 43); err != nil {
					t.Fatalf("new client write: %v", err)
				}
				if sends.Load() != 1 {
					t.Fatalf("device writes = %d, want 1", sends.Load())
				}
			}
		})
	}
}

func TestClientManagerInvalidatesBeforeWaitingForClientLock(t *testing.T) {
	for _, action := range []string{"remove", "clean", "close"} {
		t.Run(action, func(t *testing.T) {
			m := NewClientManager()
			defer m.Close()
			cli, err := m.GetClient(context.Background(), "device", func(context.Context) (*ModbusClientConf, error) {
				return &ModbusClientConf{Address: "127.0.0.1:1"}, nil
			})
			if err != nil {
				t.Fatal(err)
			}
			entry := m.clients["device"]
			// 固定占住发送锁，确保写请求和连接关闭都必须等待。
			cli.mu.Lock()
			writeDone := make(chan error, 1)
			go func() {
				_, err := cli.WriteSingleRegister(context.Background(), 0, 42)
				writeDone <- err
			}()
			removed := make(chan struct{})
			go func() {
				defer close(removed)
				switch action {
				case "remove":
					m.RemoveClient("device")
				case "clean":
					m.removeIfCurrent("device", entry, "test")
				case "close":
					m.Close()
				}
			}()
			deadline := time.NewTimer(time.Second)
			ticker := time.NewTicker(time.Millisecond)
			defer deadline.Stop()
			defer ticker.Stop()
			for !cli.invalidated.Load() {
				select {
				case <-ticker.C:
				case <-deadline.C:
					cli.mu.Unlock()
					<-removed
					<-writeDone
					t.Fatal("invalidation waited for the client lock")
				}
			}
			cli.mu.Unlock()
			if err := <-writeDone; !errors.Is(err, ErrClientInvalidated) {
				t.Fatalf("queued write = %v", err)
			}
			<-removed
		})
	}
}
