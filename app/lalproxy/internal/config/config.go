package config

import (
	"github.com/zeromicro/go-zero/zrpc"
	"zero-service/common/nacosx"
)

type Config struct {
	zrpc.RpcServerConf
	NacosConfig nacosx.ServiceConfig `json:",optional"`
	// LAL服务器配置
	LalServer struct {
		// LAL服务器的IP地址
		Ip string `json:",default=127.0.0.1"`
		// LAL服务器的HTTP API端口
		Port int `json:",default=8083"`
		// 超时时间，单位：毫秒
		Timeout int `json:",default=5000"`
	}
}
