package config

import (
	"time"

	"github.com/zeromicro/go-zero/zrpc"
)

// Config 为 ispagent gRPC 服务配置，包含 zrpc 基础配置和 ISP TCP 客户端参数。
type Config struct {
	zrpc.RpcServerConf
	IspSetting IspSetting
}

// IspSetting 定义 ISP TCP 客户端连接及协议参数。
// 零值字段会在 ApplyDefaults() 中填充以下最佳配置。
type IspSetting struct {
	ServerAddr          string        `json:",optional"`                           // 上级 ISP 系统地址
	SendCode            string        `json:",optional"`                           // 本端标识
	RegisterReceiveCode string        `json:",optional"`                           // 注册包中的对端标识（可选）
	RootName            string        `json:",default=PatrolDevice"`               // XML 根元素
	HeartbeatInterval   time.Duration `json:",default=60s"`                        // 心跳间隔
	RequestTimeout      time.Duration `json:",default=10s"`                        // 单次请求超时
	ReconnectInterval   time.Duration `json:",default=3s"`                         // 断线重连间隔
	MaxFrameLength      int           `json:",default=1048576"`                    // 单帧最大字节数
	DebugLog            bool          `json:",optional"`                           // 帧级 debug 日志
}

// ApplyDefaults 对零值字段填充最佳默认配置。
func (s *IspSetting) ApplyDefaults() {
	if s.RootName == "" {
		s.RootName = "PatrolDevice"
	}
	if s.HeartbeatInterval <= 0 {
		s.HeartbeatInterval = 60 * time.Second
	}
	if s.RequestTimeout <= 0 {
		s.RequestTimeout = 10 * time.Second
	}
	if s.ReconnectInterval <= 0 {
		s.ReconnectInterval = 3 * time.Second
	}
	if s.MaxFrameLength <= 0 {
		s.MaxFrameLength = 1 << 20
	}
}
