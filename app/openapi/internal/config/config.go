package config

import "github.com/zeromicro/go-zero/zrpc"

type Config struct {
	zrpc.RpcServerConf
	// PodEngineConf podengine 下游 gRPC 配置
	PodEngineConf zrpc.RpcClientConf
	// BridgeModbusConf bridgemodbus 下游 gRPC 配置
	BridgeModbusConf zrpc.RpcClientConf
	// Mock 为 true 时路由表返回固定数据，仅允许 dev/test 使用
	Mock bool `json:",optional"`
}
