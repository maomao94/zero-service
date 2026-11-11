package config

import (
	"time"

	"github.com/zeromicro/go-zero/zrpc"
)

type Config struct {
	zrpc.RpcServerConf
	JwtAuth struct {
		AccessSecret     string
		PrevAccessSecret string `json:",optional"`
		AccessExpire     int64
	}
	// PushTimeout 是向全部网关扇出推送的总超时，未配置时默认 2s。
	PushTimeout time.Duration `json:",optional"`
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
