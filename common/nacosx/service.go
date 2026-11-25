package nacosx

import (
	"fmt"
	"maps"
	"strconv"

	"github.com/nacos-group/nacos-sdk-go/v2/common/constant"
	"github.com/zeromicro/go-zero/core/logx"
)

// ServiceConfig 配置服务实例在 Nacos 中的注册信息，注册与发现共用同一套默认值。
type ServiceConfig struct {
	IsRegister  bool              `json:",optional"`
	Host        string            `json:",default=127.0.0.1"`
	Port        uint64            `json:",default=8848"`
	Username    string            `json:",optional"`
	PassWord    string            `json:",optional"`
	NamespaceId string            `json:",default=public"`
	ServiceName string            `json:",optional"`
	AdvertiseIP string            `json:",optional"`
	Group       string            `json:",optional"`
	Cluster     string            `json:",optional"`
	Weight      float64           `json:",optional"`
	TimeoutMs   uint64            `json:",optional"`
	Metadata    map[string]string `json:",optional"`
}

// WithMetadata 返回合并了 kv 的副本，不会修改调用方的配置；
// 同 key 时 kv 覆盖原 Metadata。
func (c ServiceConfig) WithMetadata(kv map[string]string) ServiceConfig {
	if len(kv) == 0 {
		return c
	}
	merged := make(map[string]string, len(c.Metadata)+len(kv))
	maps.Copy(merged, c.Metadata)
	maps.Copy(merged, kv)
	c.Metadata = merged
	return c
}

// Register 注册服务实例：
//   - listenOn 是实例端口（Nacos 中的 port）。HTTP 与 gRPC 并存时传 HTTP/主端口；
//   - grpcListenOn 是 gRPC 端口，写入 metadata.gRPC_port；为空表示不写，
//     消费方回退使用实例端口（与 Java 侧 Feign + gRPC 的约定一致）。
func (c ServiceConfig) Register(defaultName, listenOn, grpcListenOn string) error {
	if !c.IsRegister {
		return nil
	}
	opts, err := c.buildOptions(defaultName, listenOn, grpcListenOn)
	if err != nil {
		logx.Errorf("[nacos] %v", err)
		return err
	}
	return RegisterService(opts)
}

// MustRegister 与 Register 相同，注册失败时直接退出进程，适合启动阶段调用。
func (c ServiceConfig) MustRegister(defaultName, listenOn, grpcListenOn string) {
	logx.Must(c.Register(defaultName, listenOn, grpcListenOn))
}

// MustRegisterGRPC 注册仅 gRPC 的实例，实例端口与 gRPC_port 元数据一致。
func (c ServiceConfig) MustRegisterGRPC(defaultName, listenOn string) {
	c.MustRegister(defaultName, listenOn, listenOn)
}

// MustRegisterHTTP 注册仅 HTTP 的实例，不写 gRPC_port 元数据。
func (c ServiceConfig) MustRegisterHTTP(defaultName, listenOn string) {
	c.MustRegister(defaultName, listenOn, "")
}

// buildOptions 生成注册参数；监听地址校验与发现侧共用 parseServerAddr。
func (c ServiceConfig) buildOptions(defaultName, listenOn, grpcListenOn string) (*Options, error) {
	name := c.ServiceName
	if name == "" {
		name = defaultName
	}
	if name == "" {
		return nil, fmt.Errorf("nacos register: empty service name (ServiceName and defaultName are both empty)")
	}

	if _, _, err := parseServerAddr(listenOn); err != nil {
		return nil, fmt.Errorf("nacos register %s: %w", name, err)
	}

	metadata := make(map[string]string, len(c.Metadata)+2)
	maps.Copy(metadata, c.Metadata)
	metadata["preserved.register.source"] = "go-zero"
	if grpcListenOn != "" {
		_, grpcPort, err := parseServerAddr(grpcListenOn)
		if err != nil {
			return nil, fmt.Errorf("nacos register %s: invalid gRPC address: %w", name, err)
		}
		metadata["gRPC_port"] = strconv.FormatUint(grpcPort, 10)
	}

	advertiseIP, err := resolveAdvertiseIP(c.AdvertiseIP, listenOn, grpcListenOn)
	if err != nil {
		return nil, fmt.Errorf("nacos register %s: %w", name, err)
	}

	timeout := c.TimeoutMs
	if timeout == 0 {
		timeout = 5000
	}

	servers := []constant.ServerConfig{*constant.NewServerConfig(c.Host, c.Port)}
	client := &constant.ClientConfig{
		NamespaceId:         c.NamespaceId,
		Username:            c.Username,
		Password:            c.PassWord,
		TimeoutMs:           timeout,
		NotLoadCacheAtStart: true,
	}

	opts := []Option{
		WithCluster(c.Cluster),
		WithGroup(c.Group),
		WithMetadata(metadata),
	}
	if c.Weight > 0 {
		opts = append(opts, WithWeight(c.Weight))
	}
	nacosOpts := newNacosConfig(name, listenOn, servers, client, opts...)
	nacosOpts.AdvertiseIP = advertiseIP
	return nacosOpts, nil
}
