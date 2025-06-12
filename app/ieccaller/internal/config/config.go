package config

import (
	"github.com/zeromicro/go-zero/zrpc"
	"zero-service/common/config"
	"zero-service/iec104/iec104client"
)

type Config struct {
	zrpc.RpcServerConf
	DeployMode      string `json:",default=standalone,options=standalone|cluster"` // 可选值：standalone 或 cluster
	KafkaConfig     config.KqConfig
	IecServerConfig []iec104client.IecServerConfig
	//IecCoaConfig         []iec104client.CoaConfig
	InterrogationCmdCron string
	// 模式字段，支持 cluster / standalone

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
