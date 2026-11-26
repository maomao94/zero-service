package config

import (
	"zero-service/common/nacosx"

	"github.com/zeromicro/go-zero/zrpc"
)

type Config struct {
	zrpc.RpcServerConf
	NacosConfig nacosx.ServiceConfig `json:",optional"`
	// PodEngineConf podengine 下游 gRPC 配置
	PodEngineConf zrpc.RpcClientConf
	// BridgeModbusConf bridgemodbus 下游 gRPC 配置
	BridgeModbusConf zrpc.RpcClientConf
	// GisConf 为 GIS 下游连接配置。省略时不注册 GIS 方法。
	GisConf *zrpc.RpcClientConf `json:",optional"`
}
