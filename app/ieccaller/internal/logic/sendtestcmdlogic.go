package logic

import (
	"context"
	"zero-service/app/ieccaller/ieccaller"
	"zero-service/app/ieccaller/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type SendTestCmdLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewSendTestCmdLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SendTestCmdLogic {
	return &SendTestCmdLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *SendTestCmdLogic) SendTestCmd(in *ieccaller.SendTestCmdReq) (*ieccaller.SendTestCmdRes, error) {
	return &ieccaller.SendTestCmdRes{}, nil
}
