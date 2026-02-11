package config

import "github.com/zeromicro/go-zero/zrpc"

type Config struct {
	zrpc.RpcServerConf
	JwtAuth struct {
		AccessSecret     string
		PrevAccessSecret string `json:",optional"`
		AccessExpire     int64
	}
	NacosConfig struct {
		IsRegister  bool
		Host        string
		Port        uint64
		Username    string
		PassWord    string
		NamespaceId string
		ServiceName string
	} `json:",optional"`
	SocketGtwConf zrpc.RpcClientConf `json:",optional"`
}
