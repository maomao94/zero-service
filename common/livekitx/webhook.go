package livekitx

import "github.com/livekit/protocol/auth"

// NewWebhookKeyProvider 返回校验 LiveKit Webhook 签名所需的 KeyProvider。
//
// LiveKit 服务器用自身的 API key 作为 token 的 key claim 签名，而业务只
// 持有 signing key（secret），因此返回的 KeyProvider 对任意 key claim 都
// 返回同一个 signing key；真正校验的是 body 摘要必须由该 secret 签名。
// 业务用法：
//
//	event, err := webhook.ReceiveWebhookEvent(req, livekitx.NewWebhookKeyProvider(cfg.WebhookKey))
//
// 验签失败时 ReceiveWebhookEvent 直接返回签名错误，不调用任何业务代码；
// 成功返回的 event 以 event.GetId() 持久化去重，事件可能重复、迟到、乱序。
func NewWebhookKeyProvider(signingKey string) auth.KeyProvider {
	return webhookKeyProvider{secret: signingKey}
}

// webhookKeyProvider 对任意 API key claim 返回同一个 signing key。
// 不能使用 auth.NewSimpleKeyProvider("", key)：空 API key 的 KeyProvider
// 永远验签失败。API key claim 仅作标识，不增加安全性。
type webhookKeyProvider struct{ secret string }

func (p webhookKeyProvider) GetSecret(string) string { return p.secret }
func (p webhookKeyProvider) NumKeys() int            { return 1 }
