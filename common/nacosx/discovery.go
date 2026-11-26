package nacosx

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/nacos-group/nacos-sdk-go/v2/clients"
	"github.com/nacos-group/nacos-sdk-go/v2/clients/naming_client"
	"github.com/nacos-group/nacos-sdk-go/v2/common/constant"
	"github.com/nacos-group/nacos-sdk-go/v2/model"
	"github.com/nacos-group/nacos-sdk-go/v2/vo"
	"github.com/pkg/errors"
	"github.com/zeromicro/go-zero/core/logx"
)

const (
	// 订阅失败时指数退避重试，订阅成功后由 Nacos 推送驱动，不再轮询实例。
	// 抽成变量便于测试注入更短的间隔。
	defaultSubscribeRetryMin = time.Second
	defaultSubscribeRetryMax = 30 * time.Second
)

var (
	subscribeRetryMin = defaultSubscribeRetryMin
	subscribeRetryMax = defaultSubscribeRetryMax
)

// Discovery 订阅 Nacos 服务实例，实例集合变化时回调 onChange。
// 除 gRPC resolver 外，需要自行维护客户端集合的场景（如 socket 网关扇出）
// 也可以直接复用，避免重复实现一套订阅逻辑。
type Discovery struct {
	cancel context.CancelFunc
	client interface {
		CloseClient()
	}
	once sync.Once
}

// Discover 解析 nacos:// 地址并启动订阅。onChange 在独立协程中串行回调，
// 地址已去重并按字典序排序，仅在集合发生变化时触发；实例全部下线时
// 回调空列表（非 nil 的空切片），调用方据此清空客户端集合。
func Discover(ctx context.Context, rawTarget string, onChange func([]string)) (*Discovery, error) {
	if onChange == nil {
		onChange = func([]string) {}
	}

	rawURL, err := url.Parse(rawTarget)
	if err != nil {
		return nil, errors.Wrap(err, "parse nacos target")
	}
	tgt, err := parseURL(*rawURL)
	if err != nil {
		return nil, errors.Wrap(err, "Wrong nacos URL")
	}

	client, err := newWatchClient(tgt)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(ctx)
	pipe := make(chan []string)

	callback := func(instances []model.Instance, err error) {
		if err != nil {
			logx.Errorf("[nacos] watcher callback error: %v", err)
			return
		}
		select {
		case pipe <- extractHealthyGRPCInstances(instances):
		case <-ctx.Done():
		}
	}

	go subscribeNacos(ctx, client, tgt, callback)
	go deliverAddrs(ctx, pipe, onChange)

	return &Discovery{
		cancel: cancel,
		client: client,
	}, nil
}

// Close 停止订阅并释放 Nacos client，可安全重复调用。
func (d *Discovery) Close() {
	d.once.Do(func() {
		d.cancel()
		d.client.CloseClient()
	})
}

// newNamingClient 统一注册与发现侧的 naming client 创建：补齐缓存/日志目录后交给 SDK。
func newNamingClient(serverConfigs []constant.ServerConfig, clientConfig constant.ClientConfig) (naming_client.INamingClient, error) {
	applyClientDefaults(&clientConfig)
	client, err := clients.NewNamingClient(vo.NacosClientParam{
		ServerConfigs: serverConfigs,
		ClientConfig:  &clientConfig,
	})
	if err != nil {
		return nil, errors.Wrap(err, "Couldn't connect to the nacos API")
	}
	return client, nil
}

// newWatchClient 按 target 构建 Nacos naming client，供 gRPC resolver 与 Discover 复用。
func newWatchClient(tgt target) (naming_client.INamingClient, error) {
	host, port, err := parseServerAddr(tgt.Addr)
	if err != nil {
		return nil, err
	}

	sc := []constant.ServerConfig{
		*constant.NewServerConfig(host, port),
	}

	cc := constant.ClientConfig{
		AppName:     tgt.AppName,
		NamespaceId: tgt.NamespaceID,
		Username:    tgt.User,
		Password:    tgt.Password,
		// tgt.Timeout 是 time.Duration（纳秒），SDK 按毫秒解释 TimeoutMs。
		TimeoutMs:           uint64(tgt.Timeout / time.Millisecond),
		NotLoadCacheAtStart: targetBool(tgt.NotLoadCacheAtStart, true),
		// 默认允许空实例列表进入回调：SDK 在 UpdateCacheWhenEmpty=false 时
		// 会直接丢弃“实例清空”的推送，消费方将永远保留已下线的节点。
		UpdateCacheWhenEmpty: targetBool(tgt.UpdateCacheWhenEmpty, true),
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

	return newNamingClient(sc, cc)
}

// subscribeNacos 建立订阅；订阅成功时 SDK 会立即把当前实例列表回调一次，
// 之后由服务端推送驱动。首次失败按指数退避重试，避免 Nacos 短暂不可用时
// 服务永久拿不到地址（SDK 只在连接建立后对已确认的订阅做 redo）。
func subscribeNacos(ctx context.Context, client naming_client.INamingClient, tgt target,
	callback func([]model.Instance, error)) {
	// 复用同一个 SubscribeParam：SDK 以 &param.SubscribeCallback 作为回调标识，
	// 复用后 Unsubscribe 才能精确摘掉上一次失败注册的回调。
	param := &vo.SubscribeParam{
		ServiceName:       tgt.Service,
		Clusters:          tgt.Clusters,
		GroupName:         tgt.GroupName,
		SubscribeCallback: callback, // required
	}

	backoff := subscribeRetryMin
	for attempt := 1; ; attempt++ {
		select {
		case <-ctx.Done():
			return
		default:
		}

		if attempt > 1 {
			// SDK 的 Subscribe 每次都会追加回调且不去重；失败重试前先摘掉上一次注册的
			// 回调，否则同一次推送会触发 N 次回调并泄漏 listener。
			_ = client.Unsubscribe(param)
		}

		if err := client.Subscribe(param); err == nil {
			logx.Infof("[nacos] subscribed service %s", tgt.Service)
			return
		} else {
			logx.Errorf("[nacos] subscribe %s failed, retry in %s: %v", tgt.Service, backoff, err)
		}

		timer := time.NewTimer(backoff)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
		backoff = min(backoff*2, subscribeRetryMax)
	}
}

func deliverAddrs(ctx context.Context, input <-chan []string, onChange func([]string)) {
	first := true
	var last string
	for {
		select {
		case addrs, ok := <-input:
			if !ok {
				return
			}
			normalized := normalizeAddrs(addrs)
			key := strings.Join(normalized, ",")
			if !first && key == last {
				continue
			}
			first, last = false, key
			onChange(normalized)
		case <-ctx.Done():
			logx.Debug("[nacos] watch has been finished")
			return
		}
	}
}

func normalizeAddrs(addrs []string) []string {
	set := make(map[string]struct{}, len(addrs))
	for _, addr := range addrs {
		if addr = strings.TrimSpace(addr); addr != "" {
			set[addr] = struct{}{}
		}
	}
	normalized := make([]string, 0, len(set))
	for addr := range set {
		normalized = append(normalized, addr)
	}
	sort.Strings(normalized)
	return normalized
}

func parseServerAddr(addr string) (string, uint64, error) {
	host, portStr, err := net.SplitHostPort(addr)
	if err != nil {
		return "", 0, fmt.Errorf("invalid nacos server address %q: %w", addr, err)
	}
	port, err := strconv.ParseUint(portStr, 10, 16)
	if err != nil || port == 0 {
		return "", 0, fmt.Errorf("invalid nacos server address %q: invalid port %q", addr, portStr)
	}
	return host, port, nil
}

// extractHealthyGRPCInstances 返回可用地址：gRPC_port 元数据优先，
// 缺失时回退实例 port（与 Java 侧 Feign + gRPC 的约定一致）。
func extractHealthyGRPCInstances(instances []model.Instance) []string {
	addrs := make([]string, 0, len(instances))
	for _, s := range instances {
		if !s.Healthy || !s.Enable {
			logx.Debugf("[nacos] 忽略实例: %s:%d (健康: %t, 启用: %t)", s.Ip, s.Port, s.Healthy, s.Enable)
			continue
		}

		port := s.Port
		if v := s.Metadata["gRPC_port"]; v != "" {
			if parsed, err := strconv.ParseUint(v, 10, 16); err == nil && parsed > 0 {
				port = parsed
			}
		}
		logx.Debugf("[nacos] 发现健康实例: %s|%s:%d (权重: %.1f)", s.InstanceId, s.Ip, port, s.Weight)
		addrs = append(addrs, net.JoinHostPort(s.Ip, strconv.FormatUint(port, 10)))
	}
	return addrs
}
