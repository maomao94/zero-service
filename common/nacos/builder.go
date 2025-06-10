package nacos

import (
	"context"
	"fmt"
	"github.com/nacos-group/nacos-sdk-go/v2/clients"
	"github.com/nacos-group/nacos-sdk-go/v2/common/constant"
	"github.com/nacos-group/nacos-sdk-go/v2/model"
	"github.com/nacos-group/nacos-sdk-go/v2/vo"
	"github.com/pkg/errors"
	"github.com/zeromicro/go-zero/core/logx"
	"google.golang.org/grpc/resolver"
	"net"
	"strconv"
	"time"
)

func init() {
	resolver.Register(&builder{})
}

// schemeName for the urls
// All target URLs like 'nacos://.../...' will be resolved by this resolver
const schemeName = "nacos"

// builder implements resolver.Builder and use for constructing all consul resolvers
type builder struct{}

func (b *builder) Build(url resolver.Target, conn resolver.ClientConn, opts resolver.BuildOptions) (resolver.Resolver, error) {
	tgt, err := parseURL(url.URL)
	if err != nil {
		return nil, errors.Wrap(err, "Wrong nacos URL")
	}

	host, ports, err := net.SplitHostPort(tgt.Addr)
	if err != nil {
		return nil, fmt.Errorf("failed parsing address error: %v", err)
	}
	port, _ := strconv.ParseUint(ports, 10, 16)

	sc := []constant.ServerConfig{
		*constant.NewServerConfig(host, port),
	}

	cc := &constant.ClientConfig{
		AppName:              tgt.AppName,
		NamespaceId:          tgt.NamespaceID,
		Username:             tgt.User,
		Password:             tgt.Password,
		TimeoutMs:            uint64(tgt.Timeout),
		NotLoadCacheAtStart:  tgt.NotLoadCacheAtStart,
		UpdateCacheWhenEmpty: tgt.UpdateCacheWhenEmpty,
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
		return nil, errors.Wrap(err, "Couldn't connect to the nacos API")
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
		ticker := time.NewTicker(5 * time.Second)
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

	return &resolvr{cancelFunc: cancel}, nil
}

// Scheme returns the scheme supported by this resolver.
// Scheme is defined at https://github.com/grpc/grpc/blob/master/doc/naming.md.
func (b *builder) Scheme() string {
	return schemeName
}

func extractHealthyGRPCInstances(instances []model.Instance) []string {
	addrs := make([]string, 0, len(instances))
	for _, s := range instances {
		if !s.Healthy || !s.Enable {
			logx.Statf("[Nacos] 忽略不健康/禁用实例: %s:%d (健康: %t, 启用: %t)",
				s.Ip, s.Port, s.Healthy, s.Enable)
			continue
		}

		logx.Statf("[Nacos] 发现健康实例: %s|%s:%d (权重: %.1f)",
			s.InstanceId, s.Ip, s.Port, s.Weight)

		if s.Metadata != nil && s.Metadata["gRPC_port"] != "" {
			addrs = append(addrs, fmt.Sprintf("%s:%s", s.Ip, s.Metadata["gRPC_port"]))
		} else {
			addrs = append(addrs, fmt.Sprintf("%s:%d", s.Ip, s.Port))
		}
	}
	return addrs
}
