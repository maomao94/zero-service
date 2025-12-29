package socketio

import (
	"context"
	"fmt"
	"math/rand/v2"
	"net"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	interceptor "zero-service/common/Interceptor/rpcclient"
	"zero-service/gateway/socketgtw/socketgtw"

	"github.com/nacos-group/nacos-sdk-go/v2/common/logger"
	"github.com/nacos-group/nacos-sdk-go/v2/model"
	"github.com/pkg/errors"
	"google.golang.org/grpc/resolver"

	"github.com/nacos-group/nacos-sdk-go/v2/clients"
	"github.com/nacos-group/nacos-sdk-go/v2/common/constant"
	"github.com/nacos-group/nacos-sdk-go/v2/vo"
	"github.com/zeromicro/go-zero/core/discov"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/mapping"
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
	tgt, err := parseURL(c.Target)
	if err != nil {
		return errors.Wrap(err, "Wrong nacos URL")
	}

	host, ports, err := net.SplitHostPort(tgt.Addr)
	if err != nil {
		return fmt.Errorf("failed parsing address error: %v", err)
	}
	port, _ := strconv.ParseUint(ports, 10, 16)

	sc := []constant.ServerConfig{
		*constant.NewServerConfig(host, port),
	}

	cc := &constant.ClientConfig{
		AppName:     tgt.AppName,
		NamespaceId: tgt.NamespaceID,
		Username:    tgt.User,
		Password:    tgt.Password,
		TimeoutMs:   uint64(tgt.Timeout),
		//NotLoadCacheAtStart: tgt.NotLoadCacheAtStart,
		//UpdateCacheWhenEmpty: tgt.UpdateCacheWhenEmpty,
		NotLoadCacheAtStart:  true,  // 不用旧缓存启动
		UpdateCacheWhenEmpty: false, // 查询不到就返回空，别用缓存兜底
	}

	if tgt.CacheDir != "" {
		cc.CacheDir = tgt.CacheDir
	}
	if tgt.LogDir != "" {
		cc.LogDir = tgt.LogDir
	}
	if tgt.LogLevel != "" {
		cc.LogLevel = tgt.LogLevel
	}

	cli, err := clients.NewNamingClient(vo.NacosClientParam{
		ServerConfigs: sc,
		ClientConfig:  cc,
	})
	if err != nil {
		return errors.Wrap(err, "Couldn't connect to the nacos API")
	}

	ctx, cancel := context.WithCancel(context.Background())
	pipe := make(chan []string)

	go cli.Subscribe(&vo.SubscribeParam{
		ServiceName:       tgt.Service,
		Clusters:          tgt.Clusters,
		GroupName:         tgt.GroupName,
		SubscribeCallback: newWatcher(ctx, cancel, pipe).CallBackHandle, // required
	})

	go populateEndpoints(ctx, conn, pipe)

	go func() {
		ticker := time.NewTicker(60 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				instances, err := cli.SelectAllInstances(vo.SelectAllInstancesParam{
					ServiceName: tgt.Service,
					Clusters:    tgt.Clusters,
					GroupName:   tgt.GroupName,
				})
				if err != nil {
					logx.Errorf("failed to pull nacos service instances: %v", err)
					continue
				}

				addrs := extractHealthyGRPCInstances(instances)
				pipe <- addrs
			}
		}
	}()
	return nil
}

type watcher struct {
	ctx    context.Context
	cancel context.CancelFunc
	out    chan<- []string
}

func newWatcher(ctx context.Context, cancel context.CancelFunc, out chan<- []string) *watcher {
	return &watcher{
		ctx:    ctx,
		cancel: cancel,
		out:    out,
	}
}

func (nw *watcher) CallBackHandle(services []model.Instance, err error) {
	if err != nil {
		logger.Error("[Nacos resolver] watcher call back handle error:%v", err)
		return
	}
	addrs := extractHealthyGRPCInstances(services)
	nw.out <- addrs
}

func populateEndpoints(ctx context.Context, clientConn resolver.ClientConn, input <-chan []string) {
	for {
		select {
		case cc := <-input:
			connsSet := make(map[string]struct{}, len(cc))
			for _, c := range cc {
				connsSet[c] = struct{}{}
			}
			conns := make([]resolver.Address, 0, len(connsSet))
			for c := range connsSet {
				conns = append(conns, resolver.Address{Addr: c})
			}
			sort.Sort(byAddressString(conns)) // Don't replace the same address list in the balancer
			_ = clientConn.UpdateState(resolver.State{Addresses: conns})
		case <-ctx.Done():
			logx.Info("[Nacos resolver] Watch has been finished")
			return
		}
	}
}

func extractHealthyGRPCInstances(instances []model.Instance) []string {
	addrs := make([]string, 0, len(instances))
	for _, s := range instances {
		if s.Metadata == nil || s.Metadata["gRPC_port"] == "" {
			logx.Errorf("[Nacos] 忽略实例: %s:%d (无gRPC_port配置)", s.Ip, s.Port)
			continue
		}

		if !s.Healthy || !s.Enable {
			logx.Debugf("[Nacos] 忽略实例: %s:%s (健康: %t, 启用: %t)",
				s.Ip, s.Metadata["gRPC_port"], s.Healthy, s.Enable)
			continue
		}
		logx.Debugf("[Nacos] 发现健康实例: %s|%s:%s (权重: %.1f)",
			s.InstanceId, s.Ip, s.Metadata["gRPC_port"], s.Weight)
		addrs = append(addrs, fmt.Sprintf("%s:%s", s.Ip, s.Metadata["gRPC_port"]))
	}
	return addrs
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

type target struct {
	Addr                 string        `key:",optional"`
	User                 string        `key:",optional"`
	Password             string        `key:",optional"`
	Service              string        `key:",optional"`
	GroupName            string        `key:",optional"`
	Clusters             []string      `key:",optional"`
	NamespaceID          string        `key:"namespaceid,optional"`
	Timeout              time.Duration `key:"timeout,optional"`
	AppName              string        `key:"appName,optional"`
	LogLevel             string        `key:",optional"`
	LogDir               string        `key:",optional"`
	CacheDir             string        `key:",optional"`
	NotLoadCacheAtStart  bool          `key:"notLoadCacheAtStart,optional,string"`
	UpdateCacheWhenEmpty bool          `key:"updateCacheWhenEmpty,optional,string"`
}

const schemeName = "nacos"

func parseURL(rawURL url.URL) (target, error) {
	if rawURL.Scheme != schemeName ||
		len(rawURL.Host) == 0 || len(strings.TrimLeft(rawURL.Path, "/")) == 0 {
		return target{},
			errors.Errorf("Malformed URL('%s'). Must be in the next format: 'nacos://[user:passwd]@host/service?param=value'", rawURL.String())
	}

	var tgt target
	params := make(map[string]interface{}, len(rawURL.Query()))
	for name, value := range rawURL.Query() {
		params[name] = value[0]
	}

	err := mapping.UnmarshalKey(params, &tgt)
	if err != nil {
		return target{}, errors.Wrap(err, "Malformed URL parameters")
	}

	if tgt.NamespaceID == "" {
		tgt.NamespaceID = "public"
	}

	tgt.LogLevel = os.Getenv("NACOS_LOG_LEVEL")
	tgt.LogDir = os.Getenv("NACOS_LOG_DIR")
	tgt.CacheDir = os.Getenv("NACOS_CACHE_DIR")

	tgt.User = rawURL.User.Username()
	tgt.Password, _ = rawURL.User.Password()
	tgt.Addr = rawURL.Host
	tgt.Service = strings.TrimLeft(rawURL.Path, "/")

	if logLevel, exists := os.LookupEnv("NACOS_LOG_LEVEL"); exists {
		tgt.LogLevel = logLevel
	}

	if logDir, exists := os.LookupEnv("NACOS_LOG_DIR"); exists {
		tgt.LogDir = logDir
	}

	if notLoadCacheAtStart, exists := os.LookupEnv("NACOS_NOT_LOAD_CACHE_AT_START"); exists {
		tgt.NotLoadCacheAtStart = notLoadCacheAtStart == "true"
	}

	if updateCacheWhenEmpty, exists := os.LookupEnv("NACOS_UPDATE_CACHE_WHEN_EMPTY"); exists {
		tgt.UpdateCacheWhenEmpty = updateCacheWhenEmpty == "true"
	}

	return tgt, nil
}
