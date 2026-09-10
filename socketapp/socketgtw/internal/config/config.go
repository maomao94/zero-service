package config

import (
	"github.com/zeromicro/go-zero/rest"
	"github.com/zeromicro/go-zero/zrpc"
)

type Config struct {
	zrpc.RpcServerConf
	Http    rest.RestConf
	JwtAuth struct {
		AccessSecret     string
		PrevAccessSecret string `json:",optional"`
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
	SocketGtwConf zrpc.RpcClientConf `json:",optional"`
	// SocketMetaData 为空时使用内置默认身份键提取（authctx.DefaultClaimAliases）；
	// 非空时在默认基础上增补额外 claim 名（按原名存入 session metadata）。
	SocketMetaData          []string `json:",optional"`
	StreamEventConf         zrpc.RpcClientConf
	EnableStreamEventNotify bool `json:",default=false"`
}
