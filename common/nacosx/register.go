package nacosx

import (
	"errors"
	"fmt"
	"net"
	"os"
	"strings"

	"github.com/nacos-group/nacos-sdk-go/v2/common/constant"
	"github.com/nacos-group/nacos-sdk-go/v2/vo"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/proc"

	"github.com/zeromicro/go-zero/core/netx"
)

// RegisterService register service to nacos
func RegisterService(opts *Options) error {
	if opts == nil || len(opts.ServiceName) == 0 {
		return errors.New("nacos register: empty service name")
	}

	_, port, err := parseServerAddr(opts.ListenOn)
	if err != nil {
		return fmt.Errorf("nacos register %s: %w", opts.ServiceName, err)
	}

	host := opts.AdvertiseIP
	if host == "" {
		host, err = resolveAdvertiseIP("", opts.ListenOn)
		if err != nil {
			return fmt.Errorf("nacos register %s: %w", opts.ServiceName, err)
		}
	}

	clientConfig := constant.ClientConfig{}
	if opts.ClientConfig != nil {
		clientConfig = *opts.ClientConfig
	}
	client, err := newNamingClient(opts.ServerConfig, clientConfig)
	if err != nil {
		return fmt.Errorf("nacos register %s: %w", opts.ServiceName, err)
	}

	// service register
	_, err = client.RegisterInstance(vo.RegisterInstanceParam{
		ServiceName: opts.ServiceName,
		Ip:          host,
		Port:        port,
		Weight:      opts.Weight,
		Enable:      true,
		Healthy:     true,
		Ephemeral:   true,
		Metadata:    opts.Metadata,
		ClusterName: opts.Cluster,
		GroupName:   opts.Group,
	})

	if err != nil {
		// 调用方普遍忽略返回值，这里必须留下日志，否则 Nacos 不可用时会静默不注册。
		logx.Errorf("[nacos] register service %s failed: %v", opts.ServiceName, err)
		client.CloseClient()
		return fmt.Errorf("nacos register %s: %w", opts.ServiceName, err)
	}

	// service deregister
	proc.AddWrapUpListener(func() {
		defer client.CloseClient()
		_, err := client.DeregisterInstance(vo.DeregisterInstanceParam{
			Ip:          host,
			Port:        port,
			ServiceName: opts.ServiceName,
			Cluster:     opts.Cluster,
			GroupName:   opts.Group,
			Ephemeral:   true,
		})
		if err != nil {
			logx.Errorf("[nacos] deregister service %s failed: %v", opts.ServiceName, err)
		} else {
			logx.Infof("[nacos] deregistered service %s", opts.ServiceName)
		}
	})

	return nil
}

// resolveAdvertiseIP 依次从显式配置、实例监听地址、gRPC 监听地址、POD_IP、内网 IP
// 中选取可用的非回环 IP；通配地址（0.0.0.0 / ::）和回环地址会被跳过。
func resolveAdvertiseIP(explicit string, listenOns ...string) (string, error) {
	if explicit != "" {
		ip := net.ParseIP(strings.Trim(explicit, "[]"))
		if ip == nil {
			return "", fmt.Errorf("invalid advertise IP %q", explicit)
		}
		return ip.String(), nil
	}

	for _, listenOn := range listenOns {
		host, _, err := net.SplitHostPort(listenOn)
		if err != nil {
			continue
		}
		if ip := usableIP(host); ip != "" {
			return ip, nil
		}
	}

	if ip := usableIP(os.Getenv(envPodIP)); ip != "" {
		return ip, nil
	}
	if ip := usableIP(netx.InternalIp()); ip != "" {
		return ip, nil
	}

	return "", errors.New("cannot derive advertise IP from listen address, POD_IP or internal IP")
}

func usableIP(host string) string {
	ip := net.ParseIP(strings.Trim(host, "[]"))
	if ip == nil || ip.IsUnspecified() || ip.IsLoopback() {
		return ""
	}
	return ip.String()
}
