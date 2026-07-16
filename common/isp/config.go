package isp

import "time"

// ClientConfig 定义 ISP TCP 客户端连接及协议参数。
// 零值字段会在 ApplyDefaults 中填充默认配置。
type ClientConfig struct {
	ServerAddr          string        `json:",optional"`             // 上级 ISP 系统地址
	SendCode            string        `json:",optional"`             // 本端标识
	RegisterReceiveCode string        `json:",optional"`             // 注册包中的对端标识（可选）
	RootName            string        `json:",default=PatrolDevice"` // XML 根元素
	HeartbeatInterval   time.Duration `json:",default=60s"`          // 心跳间隔
	RequestTimeout      time.Duration `json:",default=10s"`          // 单次请求超时
	ReconnectInterval   time.Duration `json:",default=3s"`           // 断线重连间隔
	MaxFrameLength      int           `json:",default=1048576"`      // 单帧最大字节数
	DebugLog            bool          `json:",optional"`             // 帧级 debug 日志
}

// ApplyDefaults 对零值字段填充 ISP 客户端默认配置。
func (c *ClientConfig) ApplyDefaults() {
	if c.RootName == "" {
		c.RootName = RootPatrolDevice
	}
	if c.HeartbeatInterval <= 0 {
		c.HeartbeatInterval = 60 * time.Second
	}
	if c.RequestTimeout <= 0 {
		c.RequestTimeout = 10 * time.Second
	}
	if c.ReconnectInterval <= 0 {
		c.ReconnectInterval = 3 * time.Second
	}
	if c.MaxFrameLength <= 0 {
		c.MaxFrameLength = 1 << 20
	}
}

// ServerConfig 定义 ISP TCP 服务端监听、协议和注册响应参数。
// 零值字段会在 ApplyDefaults 中填充默认配置。
type ServerConfig struct {
	ListenAddr         string `json:",default=:7100"`      // TCP 监听地址
	MaxFrameLength     int    `json:",default=1048576"`    // 单帧最大字节数
	HeartbeatInterval  int    `json:",default=60"`         // 注册响应 heart_beat_interval，单位秒
	DeviceRunInterval  int    `json:",default=10"`         // 注册响应 patroldevice_run_interval，单位秒
	NestRunInterval    int    `json:",default=500"`        // 注册响应 nest_run_interval，单位秒
	WeatherInterval    int    `json:",default=500"`        // 注册响应 weather_interval，单位秒
	DebugLog           bool   `json:",default=true"`       // 帧级 debug 日志
	RootName           string `json:",default=PatrolHost"` // XML 根元素
	IdleTimeoutSeconds int    `json:",default=300"`        // TCP 空闲连接超时，单位秒
}

// ApplyDefaults 对零值字段填充 ISP 服务端默认配置。
func (c *ServerConfig) ApplyDefaults() {
	if c.ListenAddr == "" {
		c.ListenAddr = ":7100"
	}
	if c.MaxFrameLength <= 0 {
		c.MaxFrameLength = 1 << 20
	}
	if c.HeartbeatInterval <= 0 {
		c.HeartbeatInterval = 60
	}
	if c.DeviceRunInterval <= 0 {
		c.DeviceRunInterval = 10
	}
	if c.NestRunInterval <= 0 {
		c.NestRunInterval = 500
	}
	if c.WeatherInterval <= 0 {
		c.WeatherInterval = 500
	}
	if c.RootName == "" {
		c.RootName = RootPatrolHost
	}
	if c.IdleTimeoutSeconds <= 0 {
		c.IdleTimeoutSeconds = 300
	}
}
