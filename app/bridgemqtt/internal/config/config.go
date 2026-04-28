package config

import (
	"zero-service/common/mqttx"

	"github.com/zeromicro/go-zero/zrpc"
)

type Config struct {
	zrpc.RpcServerConf
	NacosConfig struct {
		IsRegister  bool
		Host        string
		Port        uint64
		Username    string
		PassWord    string
		NamespaceId string
		ServiceName string
	} `json:",optional"`
	MqttConfig      mqttx.MqttConfig
	StreamEventConf zrpc.RpcClientConf
	SocketPushConf  zrpc.RpcClientConf
	LogConfig       mqttx.TopicLogConfig `json:",optional"`
	EventMapping    []EventMapping       `json:",optional"`
	DefaultEvent    string               `json:",default=mqtt"`
}

type EventMapping struct {
	TopicTemplate string `json:"topicTemplate"`
	Event         string `json:"event"`
}
