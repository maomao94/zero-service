package config

import (
	"time"

	"zero-service/common/gormx"
	"zero-service/common/mqttx"

	"github.com/zeromicro/go-zero/zrpc"
)

type Config struct {
	zrpc.RpcServerConf
	// GracePeriod 优雅关闭时的强制退出等待时间（超时后强制 kill）
	GracePeriod time.Duration `json:",default=500s"`
	// 部署模式：standalone / cluster（对齐 ieccaller；cluster 模式下跨节点停止转推走 MQTT 广播）
	DeployMode  string `json:",default=standalone,options=standalone|cluster"`
	NacosConfig struct {
		IsRegister  bool
		Host        string
		Port        uint64
		Username    string
		PassWord    string
		NamespaceId string
		ServiceName string
	} `json:",optional"`
	// Oryx 服务器配置
	OryxConfig struct {
		// Oryx 服务器的 IP 地址
		Ip string `json:",default=127.0.0.1"`
		// Oryx 服务器的 HTTP API 端口
		Port int `json:",default=80"`
		// 超时时间，单位：毫秒
		Timeout int `json:",default=5000"`
		// API Secret，即 SRS_PLATFORM_SECRET，用于 Bearer 鉴权
		Secret string
	}
	// 数据库配置（record 生命周期落库）
	DB gormx.Config `json:",optional"`
	// Asynq Redis DB（与 trigger 同 Redis 实例时设不同值隔离队列）
	RedisDB int `json:",default=5"`
	// Asynq worker 并发度
	Concurrency int `json:",default=20"`
	// 转推配置（FFmpeg 拉流转推到 SRS）
	RelayConfig struct {
		// SRS RTMP 地址，如 rtmp://127.0.0.1:1935（默认本机 1935）
		SrsRtmpAddr string `json:",default=rtmp://127.0.0.1:1935"`
		// 默认目标应用名（request.app 为空时使用）
		DefaultApp string `json:",default=live"`
		// 默认推流鉴权 key（request.secret_key 为空时使用）
		SecretKey string `json:",default=secret"`
		// 默认推流鉴权 value（request.secret_value 为空时使用）
		SecretValue string `json:",optional"`
	} `json:",optional"`
	// MQTT 配置（可选；cluster 模式跨节点停止转推必需，standalone 无需配置）
	MqttConfig struct {
		mqttx.MqttConfig
	} `json:",optional"`
}
