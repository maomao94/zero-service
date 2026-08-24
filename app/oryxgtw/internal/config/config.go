package config

import (
	"github.com/zeromicro/go-zero/rest"
	"github.com/zeromicro/go-zero/zrpc"
)

type Config struct {
	rest.RestConf
	// HookOpaque 回调凭证，与 Oryx hooks/apply 里配置的 opaque 一致，用于校验回调合法性（为空则跳过校验）
	HookOpaque string `json:",optional"`
	// OryxServerConf oryxserver gRPC 配置（record 生命周期落库通道）
	OryxServerConf zrpc.RpcClientConf
	// NacosConfig Nacos 注册配置（本地默认不注册）
	NacosConfig struct {
		IsRegister  bool
		Host        string
		Port        uint64
		Username    string
		PassWord    string
		NamespaceId string
		ServiceName string
	} `json:",optional"`
}
