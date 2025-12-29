package socketio

import (
	"math/rand/v2"
	"strings"
	"sync"
	interceptor "zero-service/common/Interceptor/rpcclient"
	"zero-service/gateway/socketgtw/socketgtw"

	"github.com/zeromicro/go-zero/core/discov"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/zrpc"
	"google.golang.org/grpc"
)

type SocketContainer struct {
	ClientMap map[string]socketgtw.SocketGtwClient
	lock      sync.Mutex
}

func NewPubContainer(c zrpc.RpcClientConf) *SocketContainer {
	p := &SocketContainer{
		ClientMap: make(map[string]socketgtw.SocketGtwClient),
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

func (p *SocketContainer) GetClient(key string) socketgtw.SocketGtwClient {
	p.lock.Lock()
	defer p.lock.Unlock()
	return p.ClientMap[key]
}

func (p *SocketContainer) GetClients() map[string]socketgtw.SocketGtwClient {
	p.lock.Lock()
	defer p.lock.Unlock()
	clients := make(map[string]socketgtw.SocketGtwClient, len(p.ClientMap))
	for k, v := range p.ClientMap {
		clients[k] = v
	}
	return clients
}

const (
	subsetSize = 32
)

func (p *SocketContainer) getConn4Etcd(c zrpc.RpcClientConf) error {
	sub, err := discov.NewSubscriber(c.Etcd.Hosts, c.Etcd.Key)
	if err != nil {
		return err
	}
	update := func() {
		var add []string
		var remove []string
		p.lock.Lock()
		m := make(map[string]bool)
		for _, val := range subset(sub.Values(), subsetSize) {
			m[val] = true
		}
		for k, _ := range p.ClientMap {
			if _, ok := m[k]; !ok {
				remove = append(remove, k)
			}
		}
		for k, _ := range m {
			if _, ok := p.ClientMap[k]; !ok {
				add = append(add, k)
			}
		}
		for _, val := range add {
			endpoints := make([]string, 1)
			endpoints[0] = val
			c.Endpoints = endpoints
			client := socketgtw.NewSocketGtwClient(zrpc.MustNewClient(c,
				zrpc.WithUnaryClientInterceptor(interceptor.UnaryMetadataInterceptor),
				// 添加最大消息配置
				zrpc.WithDialOption(grpc.WithDefaultCallOptions(
					//grpc.MaxCallSendMsgSize(math.MaxInt32), // 发送最大2GB
					grpc.MaxCallSendMsgSize(50*1024*1024), // 发送最大50MB
					//grpc.MaxCallRecvMsgSize(100 * 1024 * 1024),  // 接收最大100MB
				)),
			).Conn())
			p.ClientMap[val] = client
		}
		for _, val := range remove {
			delete(p.ClientMap, val)
		}
		logx.Infof("update len(pubMap)=%d", len(p.ClientMap))
		p.lock.Unlock()
	}
	sub.AddListener(update)
	update()
	return nil
}

func (p *SocketContainer) getConn4Direct(c zrpc.RpcClientConf) error {
	p.lock.Lock()
	for _, val := range c.Endpoints {
		if _, ok := p.ClientMap[val]; ok {
			continue
		}
		endpoints := make([]string, 1)
		endpoints[0] = val
		c.Endpoints = endpoints
		client := socketgtw.NewSocketGtwClient(zrpc.MustNewClient(c,
			zrpc.WithUnaryClientInterceptor(interceptor.UnaryMetadataInterceptor),
			// 添加最大消息配置
			zrpc.WithDialOption(grpc.WithDefaultCallOptions(
				//grpc.MaxCallSendMsgSize(math.MaxInt32), // 发送最大2GB
				grpc.MaxCallSendMsgSize(50*1024*1024), // 发送最大50MB
				//grpc.MaxCallRecvMsgSize(100 * 1024 * 1024),  // 接收最大100MB
			)),
		).Conn())
		p.ClientMap[val] = client
	}
	p.lock.Unlock()
	return nil
}

func (p *SocketContainer) getConn4Nacos(c zrpc.RpcClientConf) error {
	return nil
}

func subset(set []string, sub int) []string {
	rand.Shuffle(len(set), func(i, j int) {
		set[i], set[j] = set[j], set[i]
	})
	if len(set) <= sub {
		return set
	}
	return set[:sub]
}
