package config

import (
	"time"

	"github.com/zeromicro/go-zero/zrpc"
	"zero-service/common/nacosx"
)

type Config struct {
	zrpc.RpcServerConf
	JwtAuth struct {
		AccessSecret     string
		PrevAccessSecret string `json:",optional"`
		AccessExpire     int64
	}
	// PushTimeout 是向全部网关扇出推送的总超时，未配置时默认 2s。
	PushTimeout   time.Duration        `json:",optional"`
	NacosConfig   nacosx.ServiceConfig `json:",optional"`
	SocketGtwConf zrpc.RpcClientConf   `json:",optional"`
}
