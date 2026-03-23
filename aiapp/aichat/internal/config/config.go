package config

import "github.com/zeromicro/go-zero/zrpc"

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
	Providers []ProviderConfig
	Models    []ModelConfig
}
