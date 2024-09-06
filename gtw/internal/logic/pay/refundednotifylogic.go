package pay

import (
	"context"
	"github.com/ArtisanCloud/PowerWeChat/v3/src/kernel/models"
	"github.com/ArtisanCloud/PowerWeChat/v3/src/payment/notify/request"
	"net/http"

	"github.com/zeromicro/go-zero/core/logx"
	"zero-service/gtw/internal/svc"
)

type RefundedNotifyLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
	r      *http.Request
	w      http.ResponseWriter
}

// 微信退款通知
func NewRefundedNotifyLogic(ctx context.Context, svcCtx *svc.ServiceContext, r *http.Request, w http.ResponseWriter) *RefundedNotifyLogic {
	return &RefundedNotifyLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
		r:      r,
		w:      w,
	}
}

func (l *RefundedNotifyLogic) RefundedNotify() error {
	res, err := l.svcCtx.WxPayCli.HandleRefundedNotify(
		l.r,
		func(message *request.RequestNotify, transaction *models.Refund, fail func(message string)) interface{} {
			if message.EventType != "REFUND.SUCCESS" {
				return true
			}
			if transaction.OutTradeNo != "" {
				l.Infof("订单号: %s 退款成功", transaction.OutTradeNo)
			} else {
				fail("payment fail")
				return nil
			}
			return true
		},
	)
	if err != nil {
		panic(err)
	}
	res.Write(l.w)
	return nil
}
