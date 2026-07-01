package config

import (
	"zero-service/common/djisdk"
	"zero-service/common/gormx"

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
	Dji            djisdk.Config       `json:",optional"`
	DB             gormx.Config        `json:",optional"`
	Oss            *OssConfig          `json:",optional"`
	Telemetry      TelemetryConfig     `json:",optional"`
	DangerousOps   DangerousOpsConfig  `json:",optional"`
	SocketPushConf zrpc.RpcClientConf  `json:",optional"`
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

// OssConfig OSS 对象存储配置（可选）。
// 仅支持 Minio，用于上传自定义飞行区 GeoJSON 文件。
type OssConfig struct {
	Category   int64  `json:",default=1"`    // OSS 类型，1=Minio
	Endpoint   string `json:",optional"`      // OSS 服务地址
	AccessKey  string `json:",optional"`      // Access Key
	SecretKey  string `json:",optional"`      // Secret Key
	BucketName string `json:",optional"`      // 默认存储桶名称
	Region     string `json:",optional"`      // 区域
}
