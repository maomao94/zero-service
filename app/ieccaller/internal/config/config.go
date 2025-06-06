package config

import (
	"github.com/zeromicro/go-zero/zrpc"
	"zero-service/common/config"
	"zero-service/iec104/iec104client"
)

type Config struct {
	zrpc.RpcServerConf
	KafkaASDUConfig config.KqConfig
	IecServerConfig []iec104client.IecServerConfig
	//IecCoaConfig         []iec104client.CoaConfig
	InterrogationCmdCron string

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
