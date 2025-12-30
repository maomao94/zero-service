package config

import (
	"github.com/zeromicro/go-zero/rest"
	"github.com/zeromicro/go-zero/zrpc"
)

type Config struct {
	zrpc.RpcServerConf
	Http    rest.RestConf
	JwtAuth struct {
		AccessSecret string
		AccessExpire int64
	} `json:",optional"`
	NacosConfig struct {
		IsRegister  bool
		Host        string
		Port        uint64
		Username    string
		PassWord    string
		NamespaceId string
		ServiceName string
	} `json:",optional"`
	SocketGtwConf  zrpc.RpcClientConf `json:",optional"`
	SocketMetaData []string           `json:",optional"`
}
