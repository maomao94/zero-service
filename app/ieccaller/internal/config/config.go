package config

import (
	"zero-service/common/iec104/iec104client"
	"zero-service/common/mqttx"

	"github.com/zeromicro/go-zero/zrpc"
)

type Config struct {
	zrpc.RpcServerConf
	// 模式字段，支持 cluster / standalone
	DeployMode      string `json:",default=standalone,options=standalone|cluster"` // 可选值：standalone 或 cluster
	IecServerConfig []iec104client.IecServerConfig
	//IecCoaConfig         []iec104client.CoaConfig
	InterrogationCmdCron    string `json:",optional"`
	CounterInterrogationCmd string `json:",optional"`

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
	} `json:",optional"`

	MqttConfig struct {
		mqttx.MqttConfig
		Topic  []string `json:",optional"`
		IsPush bool     `json:",optional"`
	} `json:",optional"`
}
