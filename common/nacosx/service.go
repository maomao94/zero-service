package nacosx

import (
	"fmt"
	"net"
	"strconv"

	"github.com/nacos-group/nacos-sdk-go/v2/common/constant"
)

// ServiceConfig 配置服务实例在 Nacos 中的注册信息。
type ServiceConfig struct {
	IsRegister  bool   `json:",optional"`
	Host        string `json:",default=127.0.0.1"`
	Port        uint64 `json:",default=8848"`
	Username    string `json:",optional"`
	PassWord    string `json:",optional"`
	NamespaceId string `json:",default=public"`
	ServiceName string `json:",optional"`
}

// Register 根据配置注册 HTTP 或 gRPC 实例，省略服务名时使用 defaultName。
func (c ServiceConfig) Register(defaultName, listenOn string, grpcService bool) error {
	if !c.IsRegister {
		return nil
	}
	opts, err := c.registrationOptions(defaultName, listenOn, grpcService)
	if err != nil {
		return err
	}
	return RegisterService(opts)
}

func (c ServiceConfig) registrationOptions(defaultName, listenOn string, grpcService bool) (*Options, error) {
	name := c.ServiceName
	if name == "" {
		name = defaultName
	}
	_, port, err := net.SplitHostPort(listenOn)
	if err != nil {
		return nil, fmt.Errorf("nacos register %s: %w", name, err)
	}
	if value, err := strconv.ParseUint(port, 10, 16); err != nil || value == 0 {
		return nil, fmt.Errorf("nacos register %s: invalid port %q", name, port)
	}
	metadata := map[string]string{"preserved.register.source": "go-zero"}
	if grpcService {
		metadata["gRPC_port"] = port
	}
	servers := []constant.ServerConfig{*constant.NewServerConfig(c.Host, c.Port)}
	client := &constant.ClientConfig{
		NamespaceId:         c.NamespaceId,
		Username:            c.Username,
		Password:            c.PassWord,
		TimeoutMs:           5000,
		NotLoadCacheAtStart: true,
	}
	return NewNacosConfig(name, listenOn, servers, client, WithMetadata(metadata)), nil
}
