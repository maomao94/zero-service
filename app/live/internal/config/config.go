package config

import (
	"time"

	"zero-service/common/gormx"

	"github.com/zeromicro/go-zero/zrpc"
)

// RecordConf 会议录制（LiveKit Egress）配置。
type RecordConf struct {
	// 是否启用录制接口，默认 true
	Enabled bool `json:",default=true"`
	// 合成布局（空则使用服务端默认，如 "grid"）
	Layout string `json:",optional"`
	// Egress 输出目录（容器内路径，需与 egress 的 volume 映射一致）
	OutputDir string `json:",default=/out"`
	// 播放基址（录制文件对外访问前缀，如 http://host/recordings/）。
	// 为空时不返回 file_url，避免暴露本地路径。
	PlayURLBase string `json:",optional"`
}

// LiveKitConf LiveKit 管理 API、webhook 与录制配置。
type LiveKitConf struct {
	// 管理 API 地址，如 http://127.0.0.1:7880
	Url string
	// 管理 API key/secret
	ApiKey    string
	ApiSecret string
	// Webhook 验签 key（与 LiveKit 服务端配置的 signing key 一致）
	WebhookKey string
	// 入会 token 有效期，默认 2h
	TokenValidFor time.Duration `json:",default=2h"`
	// 跳过 TLS 证书校验（内网自签证书环境），默认 false 走正常校验
	InsecureSkipVerify bool `json:",optional"`
	// 录制配置（LiveKit Egress 会议录制）
	Record RecordConf
}

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
	// LiveKit 配置（管理 API、webhook 与录制）
	LiveKit LiveKitConf
	// 数据库配置（会议单据与参会记录）
	DB gormx.Config `json:",optional"`
	// SocketPushConf socketpush 推送服务（可选，未配置时通知接口不可用）
	SocketPushConf zrpc.RpcClientConf `json:",optional"`
}
