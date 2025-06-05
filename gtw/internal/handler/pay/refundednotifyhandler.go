package pay

import (
	"net/http"

	xhttp "github.com/zeromicro/x/http"
	"zero-service/gtw/internal/logic/pay"
	"zero-service/gtw/internal/svc"
)

// 微信退款通知
func RefundedNotifyHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := pay.NewRefundedNotifyLogic(r.Context(), svcCtx, r, w)
		err := l.RefundedNotify()
		if err != nil {
			xhttp.JsonBaseResponseCtx(r.Context(), w, err)
		} else {
			//xhttp.JsonBaseResponseCtx(r.Context(), w, resp)
		}
	}
}
