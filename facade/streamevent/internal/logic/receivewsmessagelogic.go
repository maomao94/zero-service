package logic

import (
	"context"

	"zero-service/facade/streamevent/internal/svc"
	"zero-service/facade/streamevent/streamevent"

	"github.com/zeromicro/go-zero/core/logx"
)

type ReceiveWSMessageLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewReceiveWSMessageLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ReceiveWSMessageLogic {
	return &ReceiveWSMessageLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 接收WS消息
func (l *ReceiveWSMessageLogic) ReceiveWSMessage(in *streamevent.ReceiveWSMessageReq) (*streamevent.ReceiveWSMessageRes, error) {
	// todo: add your logic here and delete this line

	return &streamevent.ReceiveWSMessageRes{}, nil
}
