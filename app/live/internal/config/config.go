package config

import (
	"time"

	"zero-service/common/gormx"

	"github.com/zeromicro/go-zero/zrpc"
)

type Config struct {
	zrpc.RpcServerConf
	// GracePeriod 优雅关闭时的强制退出等待时间（超时后强制 kill）
	GracePeriod time.Duration `json:",default=500s"`
	// 部署模式：standalone / cluster
	DeployMode string `json:",default=standalone,options=standalone|cluster"`
	// Nacos 服务注册（可选）
	NacosConfig struct {
		IsRegister  bool
		Host        string
		Port        uint64
		Username    string
		PassWord    string
		NamespaceId string
		ServiceName string
	} `json:",optional"`
	// LiveKit 配置（管理 API 与 webhook）
	LiveKit struct {
		// 管理 API 地址，如 http://127.0.0.1:7880
		Url string
		// 管理 API key/secret
		ApiKey    string
		ApiSecret string
		// Webhook 验签 key（与 LiveKit 服务端配置的 signing key 一致）
		WebhookKey string
		// 入会 token 有效期，默认 2h
		TokenValidFor time.Duration `json:",default=2h"`
	}
	// 数据库配置（会议单据与参会记录）
	DB gormx.Config `json:",optional"`
	// SocketPushConf socketpush 推送服务（可选，未配置时通知接口不可用）
	SocketPushConf zrpc.RpcClientConf `json:",optional"`
}
