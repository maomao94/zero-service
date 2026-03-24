// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package config

import (
	"github.com/zeromicro/go-zero/rest"
	"github.com/zeromicro/go-zero/zrpc"
)

type AbilityConfig struct {
	Id                string `json:",optional"`
	Ability           string `json:",optional"`
	DisplayName       string `json:",optional"`
	Description       string `json:",optional"`
	MaxTokens         int    `json:",optional,default=8192"`
	SupportsStreaming bool   `json:",optional,default=true"`
}

type Config struct {
	rest.RestConf
	JwtAuth struct {
		AccessSecret string
	}
	AiChatRpcConf zrpc.RpcClientConf
	Abilities     []AbilityConfig `json:",optional"`
}
