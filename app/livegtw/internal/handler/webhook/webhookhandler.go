package webhook

import (
	"net/http"

	"zero-service/common/livekitx"
	webhooklogic "zero-service/app/livegtw/internal/logic/webhook"
	"zero-service/app/livegtw/internal/svc"

	"github.com/livekit/protocol/webhook"
	"github.com/zeromicro/go-zero/core/logx"
	xhttp "github.com/zeromicro/x/http"
)

// LiveKitWebhookHandler 接收 LiveKit 推送的 webhook：
//  1. 用 livekitx.NewWebhookKeyProvider 验签（失败返回 401，不执行任何业务）；
//  2. 验签通过后把 *livekit.WebhookEvent 原始 proto 字节经 WebhookNotify RPC
//     透传给 app/live 处理（幂等/事件全 case 由业务侧负责）。
func LiveKitWebhookHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		event, err := webhook.ReceiveWebhookEvent(r, livekitx.NewWebhookKeyProvider(svcCtx.Config.LiveKit.WebhookKey))
		if err != nil {
			// 验签失败：不调用任何业务代码
			http.Error(w, "invalid webhook signature", http.StatusUnauthorized)
			return
		}
		l := webhooklogic.NewWebhookNotifyLogic(r.Context(), svcCtx)
		if err := l.WebhookNotify(event); err != nil {
			// 业务处理失败：记录日志并返回 500，让 LiveKit 重推（幂等在业务侧保证），
			// 避免返回 200 造成事件丢失。
			logx.WithContext(r.Context()).Errorf("webhook notify failed: id=%s event=%s err=%v",
				event.GetId(), event.GetEvent(), err)
			http.Error(w, "webhook notify failed", http.StatusInternalServerError)
			return
		}
		xhttp.JsonBaseResponseCtx(r.Context(), w, nil)
	}
}
