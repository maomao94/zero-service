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
	ServerAddr          string        // 上级 ISP 系统地址（如 127.0.0.1:7100）
	SendCode            string        // 本端标识，需显式配置
	RegisterReceiveCode string        // 注册包中的对端标识（可选，注册响应后以服务端实际返回为准）
	RootName            string        // XML 根元素（PatrolHost / PatrolDevice），默认 PatrolDevice
	HeartbeatInterval   time.Duration // 心跳间隔，默认 30s（注册响应可覆盖）
	RequestTimeout      time.Duration // 单次请求超时，默认 10s
	ReconnectInterval   time.Duration // 断线重连间隔，默认 3s
	MaxFrameLength      int           // 单帧最大字节数，默认 1MB
	DebugLog            bool          // 是否打印帧级 debug 日志（base64 编码）
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
