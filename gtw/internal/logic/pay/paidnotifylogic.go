package pay

import (
	"context"
	"github.com/ArtisanCloud/PowerWeChat/v3/src/kernel/models"
	"github.com/ArtisanCloud/PowerWeChat/v3/src/payment/notify/request"
	"net/http"

	"github.com/zeromicro/go-zero/core/logx"
	"zero-service/gtw/internal/svc"
)

type PaidNotifyLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
	r      *http.Request
	w      http.ResponseWriter
}

// 微信支付通知
func NewPaidNotifyLogic(ctx context.Context, svcCtx *svc.ServiceContext, r *http.Request, w http.ResponseWriter) *PaidNotifyLogic {
	return &PaidNotifyLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
		r:      r,
		w:      w,
	}
}

func (l *PaidNotifyLogic) PaidNotify() error {
	res, err := l.svcCtx.WxPayCli.HandlePaidNotify(
		l.r,
		func(message *request.RequestNotify, transaction *models.Transaction, fail func(message string)) interface{} {
			// 看下支付通知事件状态
			// 这里可能是微信支付失败的通知，所以可能需要在数据库做一些记录，然后告诉微信我处理完成了。
			if message.EventType != "TRANSACTION.SUCCESS" {
				return true
			}
			if transaction.OutTradeNo != "" {
				// 这里对照自有数据库里面的订单做查询以及支付状态改变
				l.Infof("订单号: %s 支付成功", transaction.OutTradeNo)
			} else {
				// 因为微信这个回调不存在订单号，所以可以告诉微信我还没处理成功，等会它会重新发起通知
				// 如果不需要，直接返回true即可
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
