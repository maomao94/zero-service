package config

import (
	"time"

	"zero-service/common/gormx"
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
	MqttConfig     mqttx.MqttConfig
	DB             gormx.Config        `json:",optional"`
	PendingTTL     time.Duration       `json:",default=30s"`
	UpstreamReply  UpstreamReplyConfig `json:",optional"`
	Telemetry      TelemetryConfig     `json:",optional"`
	DangerousOps   DangerousOpsConfig  `json:",optional"`
	DrcConfig      DrcConfig           `json:",optional"`
	SocketPushConf zrpc.RpcClientConf  `json:",optional"`
}

// DrcConfig DRC 平台化功能配置。
type DrcConfig struct {
	// HeartbeatInterval 心跳发送间隔，默认 2s。
	HeartbeatInterval time.Duration `json:",default=2s"`
	// HeartbeatTimeout 设备心跳超时时间，超过此时间未收到设备心跳则判定离线，默认 300s。
	// 同时作为 cache TTL，收到设备心跳时自动续期。
	HeartbeatTimeout time.Duration `json:",default=300s"`
}

type UpstreamReplyConfig struct {
	EnableEventsReply   bool `json:",default=true"`
	EnableStatusReply   bool `json:",default=true"`
	EnableRequestsReply bool `json:",default=true"`
}

type TelemetryConfig struct {
	DisableOsdSQLTrace bool `json:",default=false"`
}

// DangerousOpsConfig 敏感/危险操作开关配置。
// 默认全部关闭，需要在配置文件中显式开启才能调用对应接口。
type DangerousOpsConfig struct {
	// EnableDroneEmergencyStop 是否启用紧急停桨接口。
	// 紧急停桨会立即停止所有电机，飞行器将失去动力坠落，属于极端危险操作。
	// 默认 false，生产环境请谨慎开启。
	EnableDroneEmergencyStop bool `json:",default=false"`
}
