package config

import "github.com/zeromicro/go-zero/zrpc"

type Config struct {
	zrpc.RpcServerConf
	IspConf IspConf
}

type IspConf struct {
	ListenAddr         string        `json:",default=:7100"`
	MaxFrameLength     int           `json:",default=1048576"`
	HeartbeatInterval  int           `json:",default=60"`
	DeviceRunInterval  int           `json:",default=10"`
	NestRunInterval    int           `json:",default=500"`
	WeatherInterval    int           `json:",default=500"`
	DebugLog           bool          `json:",default=true"`
	RootName           string        `json:",default=PatrolHost"`
	IdleTimeoutSeconds int           `json:",default=300"`
}

func (c *IspConf) ApplyDefaults() {
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
		c.RootName = "PatrolHost"
	}
	if c.IdleTimeoutSeconds <= 0 {
		c.IdleTimeoutSeconds = 300
	}
}
