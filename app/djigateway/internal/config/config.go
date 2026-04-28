package config

import (
	"time"

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
	MqttConfig    mqttx.MqttConfig
	PendingTTL    time.Duration       `json:",default=30s"`
	UpstreamReply UpstreamReplyConfig `json:",optional"`
	DangerousOps  DangerousOpsConfig  `json:",optional"`
}

type UpstreamReplyConfig struct {
	EnableEventsReply   bool `json:",default=true"`
	EnableStatusReply   bool `json:",default=true"`
	EnableRequestsReply bool `json:",default=true"`
}

// DangerousOpsConfig 敏感/危险操作开关配置。
// 默认全部关闭，需要在配置文件中显式开启才能调用对应接口。
type DangerousOpsConfig struct {
	// EnableDroneEmergencyStop 是否启用紧急停桨接口。
	// 紧急停桨会立即停止所有电机，飞行器将失去动力坠落，属于极端危险操作。
	// 默认 false，生产环境请谨慎开启。
	EnableDroneEmergencyStop bool `json:",default=false"`
}
