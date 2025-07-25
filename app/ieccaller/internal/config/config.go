package config

import (
	"github.com/zeromicro/go-zero/zrpc"
	"zero-service/common/iec104/iec104client"
)

type Config struct {
	zrpc.RpcServerConf
	// 模式字段，支持 cluster / standalone
	DeployMode      string `json:",default=standalone,options=standalone|cluster"` // 可选值：standalone 或 cluster
	IecServerConfig []iec104client.IecServerConfig
	//IecCoaConfig         []iec104client.CoaConfig
	InterrogationCmdCron    string `json:",default=@every 60s"`
	CounterInterrogationCmd string `json:",default=@every 60s"`

	NacosConfig struct {
		IsRegister  bool
		Host        string
		Port        uint64
		Username    string
		PassWord    string
		NamespaceId string
		ServiceName string
	} `json:",optional"`

	KafkaConfig struct {
		Brokers          []string
		Topic            string
		BroadcastTopic   string `json:",optional,default=iec-broadcast"`
		BroadcastGroupId string `json:",optional,default=iec-caller"`
		IsPush           bool   `json:",optional"`
	}
}
