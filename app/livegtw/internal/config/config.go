package config

import (
	"github.com/zeromicro/go-zero/rest"
	"github.com/zeromicro/go-zero/zrpc"
)

type Config struct {
	rest.RestConf
	// JwtAuth 参考 aigtw：ClaimMapping 映射外部 JWT claim key 到 authctx 标准 key
	JwtAuth struct {
		AccessSecret     string
		PrevAccessSecret string            `json:",optional"`
		ClaimMapping     map[string]string `json:",optional"`
	} `json:",optional"`
	// LiveRpcConf app/live 会议业务服务
	LiveRpcConf zrpc.RpcClientConf
	// LiveKit webhook 验签 key（与 LiveKit 服务端配置的 signing key 一致）
	LiveKit struct {
		WebhookKey string
	}
	// 测试页路由开关（默认开启，便于联调）
	EnableTestPage bool `json:",default=true"`
}
