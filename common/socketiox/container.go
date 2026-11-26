package socketiox

import (
	"context"
	"fmt"
	"maps"
	"math/rand/v2"
	"strings"
	"sync"

	"zero-service/common/grpcx"
	"zero-service/common/nacosx"
	"zero-service/socketapp/socketgtw/socketgtw"

	"github.com/zeromicro/go-zero/core/discov"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/zrpc"
	"google.golang.org/grpc"
)

// SocketContainer 维护 socket 网关客户端集合，支持 direct / etcd / nacos 三种发现方式。
type SocketContainer struct {
	ClientMap map[string]socketgtw.SocketGtwClient
	lock      sync.RWMutex

	closeOnce sync.Once
	closed    bool
	closers   map[string]func() error
	discovery *nacosx.Discovery
	etcdSub   *discov.Subscriber
}

func MustNewPubContainer(c zrpc.RpcClientConf) *SocketContainer {
	p := &SocketContainer{
		ClientMap: make(map[string]socketgtw.SocketGtwClient),
		closers:   make(map[string]func() error),
	}
	if len(c.Endpoints) != 0 {
		err := p.getConn4Direct(c)
		if err != nil {
			logx.Must(err)
		}
		return p
	}
	if len(c.Etcd.Hosts) != 0 && len(c.Etcd.Key) != 0 {
		err := p.getConn4Etcd(c)
		if err != nil {
			logx.Must(err)
		}
	}
	if len(c.Target) != 0 {
		if strings.HasPrefix(c.Target, "nacos:") {
			err := p.getConn4Nacos(c)
			if err != nil {
				logx.Must(err)
			}
		}
	}
	return p
}

// Close 停止服务发现并释放后台订阅、轮询和已建立的 gRPC 连接，可安全重复调用。
func (p *SocketContainer) Close() {
	p.closeOnce.Do(func() {
		p.lock.Lock()
		p.closed = true
		closers := p.closers
		p.closers = make(map[string]func() error)
		p.lock.Unlock()

		if p.discovery != nil {
			p.discovery.Close()
		}
		if p.etcdSub != nil {
			p.etcdSub.Close()
		}
		for _, closer := range closers {
			_ = closer()
		}
	})
}

func (p *SocketContainer) GetClient(key string) socketgtw.SocketGtwClient {
	p.lock.RLock()
	defer p.lock.RUnlock()
	return p.ClientMap[key]
}

func (p *SocketContainer) GetClients() map[string]socketgtw.SocketGtwClient {
	p.lock.RLock()
	defer p.lock.RUnlock()
	clients := make(map[string]socketgtw.SocketGtwClient, len(p.ClientMap))
	maps.Copy(clients, p.ClientMap)
	return clients
}

const (
	subsetSize = 32
)

type socketClient struct {
	client socketgtw.SocketGtwClient
	close  func() error
}

func newSocketClient(endpoint string, c zrpc.RpcClientConf) (socketClient, error) {
	c.Endpoints = []string{endpoint}
	cli, err := zrpc.NewClient(c,
		zrpc.WithUnaryClientInterceptor(grpcx.UnaryMetadataInterceptor),
		zrpc.WithDialOption(grpc.WithDefaultCallOptions(
			//grpc.MaxCallSendMsgSize(math.MaxInt32), // 发送最大2GB
			grpc.MaxCallSendMsgSize(50*1024*1024), // 发送最大50MB
			//grpc.MaxCallRecvMsgSize(100 * 1024 * 1024),  // 接收最大100MB
		)),
	)
	if err != nil {
		return socketClient{}, fmt.Errorf("create socket client %s: %w", endpoint, err)
	}

	conn := cli.Conn()
	if conn == nil {
		return socketClient{}, fmt.Errorf("create socket client %s: nil connection", endpoint)
	}

	return socketClient{
		client: socketgtw.NewSocketGtwClient(conn),
		close:  conn.Close,
	}, nil
}

// newDiscoveredSocketClient 为动态发现创建的客户端强制非阻塞拨号：
// 避免不可达节点阻塞发现回调，连接就绪前的 RPC 快速失败，由 gRPC 自动重连。
func newDiscoveredSocketClient(endpoint string, c zrpc.RpcClientConf) (socketClient, error) {
	c.NonBlock = true
	return newSocketClient(endpoint, c)
}

func (p *SocketContainer) isClosed() bool {
	p.lock.RLock()
	defer p.lock.RUnlock()
	return p.closed
}

// syncClientMap 同步客户端集合：连接在锁外创建，避免阻塞读请求；被摘除的地址会关闭连接。
func (p *SocketContainer) syncClientMap(addrs []string, c zrpc.RpcClientConf) {
	target := make(map[string]struct{}, len(addrs))
	for _, addr := range addrs {
		target[addr] = struct{}{}
	}

	p.lock.RLock()
	pending := make([]string, 0, len(target))
	for addr := range target {
		if _, ok := p.ClientMap[addr]; !ok {
			pending = append(pending, addr)
		}
	}
	p.lock.RUnlock()

	created := make(map[string]socketClient, len(pending))
	for _, addr := range pending {
		client, err := newDiscoveredSocketClient(addr, c)
		if err != nil {
			logx.Errorf("[socketio] create client for %s failed: %v", addr, err)
			continue
		}
		created[addr] = client
	}

	p.lock.Lock()
	if p.closed {
		p.lock.Unlock()
		for _, client := range created {
			_ = client.close()
		}
		return
	}

	removed := 0
	closeAfter := make([]func() error, 0, len(p.ClientMap))
	for addr := range p.ClientMap {
		if _, ok := target[addr]; ok {
			continue
		}
		delete(p.ClientMap, addr)
		if closer, ok := p.closers[addr]; ok {
			closeAfter = append(closeAfter, closer)
			delete(p.closers, addr)
		}
		removed++
	}

	added := 0
	for addr, client := range created {
		if _, exist := p.ClientMap[addr]; exist {
			closeAfter = append(closeAfter, client.close) // 并发更新已创建
			continue
		}
		if _, ok := target[addr]; !ok {
			closeAfter = append(closeAfter, client.close) // 地址已被后续更新移除
			continue
		}
		p.ClientMap[addr] = client.client
		p.closers[addr] = client.close
		added++
	}
	total := len(p.ClientMap)
	p.lock.Unlock()

	for _, closer := range closeAfter {
		_ = closer()
	}

	if added > 0 || removed > 0 {
		logx.Infof("[socketio] clients updated: total=%d added=%d removed=%d",
			total, added, removed)
	}
}

func (p *SocketContainer) getConn4Etcd(c zrpc.RpcClientConf) error {
	sub, err := discov.NewSubscriber(c.Etcd.Hosts, c.Etcd.Key)
	if err != nil {
		return err
	}
	p.etcdSub = sub
	update := func() {
		if p.isClosed() {
			return
		}
		addrs := subset(sub.Values(), subsetSize)
		p.syncClientMap(addrs, c)
	}
	sub.AddListener(update)
	update()
	return nil
}

func (p *SocketContainer) getConn4Direct(c zrpc.RpcClientConf) error {
	for _, val := range c.Endpoints {
		if _, ok := p.ClientMap[val]; ok {
			continue
		}
		client, err := newSocketClient(val, c)
		if err != nil {
			return err
		}
		p.ClientMap[val] = client.client
		p.closers[val] = client.close
	}
	return nil
}

func (p *SocketContainer) getConn4Nacos(c zrpc.RpcClientConf) error {
	discovery, err := nacosx.Discover(context.Background(), c.Target, func(addrs []string) {
		p.updateClientMap(addrs, c)
	})
	if err != nil {
		return err
	}
	p.discovery = discovery
	return nil
}

func (p *SocketContainer) updateClientMap(addrs []string, c zrpc.RpcClientConf) {
	if p.isClosed() {
		return
	}
	p.syncClientMap(addrs, c)
}

func subset(set []string, sub int) []string {
	// Values() 返回的是 subscriber 内部快照，先复制再打乱，避免污染订阅端数据。
	shuffled := make([]string, len(set))
	copy(shuffled, set)

	rand.Shuffle(len(shuffled), func(i, j int) {
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
	})
	if len(shuffled) <= sub {
		return shuffled
	}
	return shuffled[:sub]
}
