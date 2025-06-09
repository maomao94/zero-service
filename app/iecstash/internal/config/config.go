package config

import (
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
	IecStreamRpcConf   zrpc.RpcClientConf
	PushAsduChunkBytes int `json:",default=10485760"` // 10M
}
