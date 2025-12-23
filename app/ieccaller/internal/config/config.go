package config

import (
	"time"
	"zero-service/common/iec104/iec104client"
	"zero-service/common/mqttx"

	"github.com/zeromicro/go-zero/zrpc"
)

type Config struct {
	zrpc.RpcServerConf
	// 模式字段，支持 cluster / standalone
	DeployMode      string `json:",default=standalone,options=standalone|cluster"` // 可选值：standalone 或 cluster
	IecServerConfig []client.IecServerConfig
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
		Brokers          []string `json:",optional"`
		Topic            string   `json:",optional,default=asdu"`
		BroadcastTopic   string   `json:",optional,default=iec-broadcast"`
		BroadcastGroupId string   `json:",optional,default=iec-caller"`
		IsPush           bool     `json:",optional,default=false"`
	} `json:",optional"`

	MqttConfig struct {
		mqttx.MqttConfig
		Topic  []string `json:",optional"`
		IsPush bool     `json:",optional,default=false"`
	} `json:",optional"`

	StreamEventConf zrpc.RpcClientConf

	DisableStmtLog bool `json:",optional,default=false"`
	SqliteDB       struct {
		DataSource string `json:",optional"`
	}
	PushAsduChunkBytes int           `json:",default=1048576"` // 1M
	GracePeriod        time.Duration `json:",default=10s"`
}
