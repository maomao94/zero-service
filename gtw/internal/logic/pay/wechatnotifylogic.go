package pay

import (
	"context"

	"zero-service/gtw/internal/svc"
	"zero-service/gtw/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type WechatNotifyLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 微信支付通知
func NewWechatNotifyLogic(ctx context.Context, svcCtx *svc.ServiceContext) *WechatNotifyLogic {
	return &WechatNotifyLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *WechatNotifyLogic) WechatNotify() (resp *types.EmptyReply, err error) {
	// todo: add your logic here and delete this line

	return
}
