package svc

import (
	"zero-service/aiapp/mcpserver/internal/config"
	"zero-service/aiapp/mcpserver/internal/skills"
	"zero-service/app/bridgemodbus/bridgemodbus"
	interceptor "zero-service/common/Interceptor/rpcclient"

	"github.com/zeromicro/go-zero/zrpc"
)

type ServiceContext struct {
	Config          config.Config
	BridgeModbusCli bridgemodbus.BridgeModbusClient
	SkillsLoader    *skills.Loader
}

func NewServiceContext(c config.Config) (*ServiceContext, error) {
	// 初始化 skills loader
	var skillsLoader *skills.Loader
	if c.Skills.Dir != "" {
		var err error
		skillsLoader, err = skills.NewLoader(c.Skills.Dir, c.Skills.AutoReload)
		if err != nil {
			return nil, err
		}
	}

	return &ServiceContext{
		Config: c,
		BridgeModbusCli: bridgemodbus.NewBridgeModbusClient(
			zrpc.MustNewClient(c.BridgeModbusRpcConf,
				zrpc.WithUnaryClientInterceptor(interceptor.UnaryMetadataInterceptor),
				zrpc.WithStreamClientInterceptor(interceptor.StreamTracingInterceptor),
			).Conn()),
		SkillsLoader: skillsLoader,
	}, nil
}
