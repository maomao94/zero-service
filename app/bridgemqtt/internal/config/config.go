package config

import (
	"zero-service/common/mqttx"

	"github.com/zeromicro/go-zero/zrpc"
	"zero-service/common/nacosx"
)

type Config struct {
	zrpc.RpcServerConf
	NacosConfig     nacosx.ServiceConfig `json:",optional"`
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
