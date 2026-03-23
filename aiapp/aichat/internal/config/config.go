package config

import (
	"time"

	"github.com/zeromicro/go-zero/zrpc"
)

type ProviderConfig struct {
	Name     string
	Type     string // "openai_compatible"
	Endpoint string
	ApiKey   string
}

type ModelConfig struct {
	Id                string
	Provider          string
	BackendModel      string
	DisplayName       string `json:",optional"`
	Description       string `json:",optional"`
	MaxTokens         int    `json:",optional,default=8192"`
	SupportsStreaming bool   `json:",optional,default=true"`
}

type Config struct {
	zrpc.RpcServerConf
	StreamTimeout     time.Duration `json:",default=10m"` // 单次流的总时长上限
	StreamIdleTimeout time.Duration `json:",default=90s"` // chunk 间最大空闲时间
	Providers         []ProviderConfig
	Models            []ModelConfig
}
