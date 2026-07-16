package config

import (
	"zero-service/common/isp"

	"github.com/zeromicro/go-zero/zrpc"
)

type Config struct {
	zrpc.RpcServerConf
	IspConf isp.ServerConfig
}
