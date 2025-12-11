package config

import (
	"time"

	"github.com/zeromicro/go-queue/kq"
	"github.com/zeromicro/go-zero/zrpc"
)

type Config struct {
	zrpc.RpcServerConf
	KafkaASDUConfig kq.KqConf
	NacosConfig     struct {
		IsRegister  bool
		Host        string
		Port        uint64
		Username    string
		PassWord    string
		NamespaceId string
		ServiceName string
	} `json:",optional"`
	StreamEventConf    zrpc.RpcClientConf
	PushAsduChunkBytes int           `json:",default=1048576"` // 1M
	GracePeriod        time.Duration `json:",default=10s"`
}
