package nacosx

import "github.com/nacos-group/nacos-sdk-go/v2/common/constant"

const (
	envPodIP = "POD_IP"

	envLogLevel             = "NACOS_LOG_LEVEL"
	envLogDir               = "NACOS_LOG_DIR"
	envCacheDir             = "NACOS_CACHE_DIR"
	envNotLoadCacheAtStart  = "NACOS_NOT_LOAD_CACHE_AT_START"
	envUpdateCacheWhenEmpty = "NACOS_UPDATE_CACHE_WHEN_EMPTY"
)

// options
type Options struct {
	ListenOn    string
	ServiceName string
	// AdvertiseIP 是注册到 Nacos 的实例 IP；为空时按监听地址、POD_IP、内网 IP 依次推导。
	AdvertiseIP string
	Weight      float64
	Cluster     string
	Group       string
	Metadata    map[string]string

	ServerConfig []constant.ServerConfig
	ClientConfig *constant.ClientConfig
}

type Option func(*Options)

func newNacosConfig(serviceName, listenOn string, sc []constant.ServerConfig, cc *constant.ClientConfig, opts ...Option) *Options {
	options := &Options{
		ServiceName:  serviceName,
		ListenOn:     listenOn,
		ServerConfig: sc,
		ClientConfig: cc,
		Weight:       1.0,
		Metadata:     make(map[string]string),
	}

	for _, opt := range opts {
		opt(options)
	}

	return options
}

func WithWeight(weight float64) Option {
	return func(o *Options) {
		o.Weight = weight
	}
}

func WithCluster(cluster string) Option {
	return func(o *Options) {
		o.Cluster = cluster
	}
}

func WithGroup(group string) Option {
	return func(o *Options) {
		o.Group = group
	}
}

func WithMetadata(metadata map[string]string) Option {
	return func(o *Options) {
		o.Metadata = metadata
	}
}
