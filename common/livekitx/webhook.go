package livekitx

import (
	"context"
	"errors"
	"net/http"

	"github.com/livekit/protocol/livekit"
	"github.com/livekit/protocol/webhook"
)

// WebhookEvent 是 SDK 原生事件的 Hook 包装，保留完整事件 ID、类型和字段。
type WebhookEvent = livekit.WebhookEvent

// ReceiveWebhook 校验 Authorization 与原始 body，校验成功后按注册顺序分发到
// OnWebhook Hook；handler 非 nil 时作为一次性 handler 在 Hook 之后执行，
// 兼容旧版单 handler 调用方式。验签失败不会调用任何 handler。
// 业务应在 handler 中持久化 event ID，以决定重试、去重和乱序处理策略。
func (c *Client) ReceiveWebhook(ctx context.Context, request *http.Request, signingKey string, handler WebhookHandler) error {
	if c == nil || c.isClosed() {
		return ErrClosed
	}
	if request == nil || signingKey == "" || ctx == nil {
		return ErrInvalidConfig
	}
	event, err := webhook.ReceiveWebhookEvent(request, webhookKeyProvider{secret: signingKey})
	if err != nil {
		return err
	}
	var errs []error
	if err := c.hooks.webhook.dispatch(ctx, event); err != nil {
		errs = append(errs, err)
	}
	if handler != nil {
		if err := callHandler(ctx, "webhook", handler, event); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

// webhookKeyProvider 对任意 API key claim 返回同一个 signing key。
// 服务器签名时使用自己的 API key（如 dev server 的 devkey），而 ReceiveWebhook
// 只接收 signing key；API key claim 仅作标识，不增加安全性，真正校验的是
// body 摘要必须由 signing key 签名。
type webhookKeyProvider struct{ secret string }

func (p webhookKeyProvider) GetSecret(string) string { return p.secret }
func (p webhookKeyProvider) NumKeys() int            { return 1 }
