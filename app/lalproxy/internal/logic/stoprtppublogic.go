package logic

import (
	"context"

	"zero-service/app/lalproxy/internal/svc"
	"zero-service/app/lalproxy/lalproxy"

	"github.com/zeromicro/go-zero/core/logx"
)

type StopRtpPubLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewStopRtpPubLogic(ctx context.Context, svcCtx *svc.ServiceContext) *StopRtpPubLogic {
	return &StopRtpPubLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 关闭GB28181接收端口
func (l *StopRtpPubLogic) StopRtpPub(in *lalproxy.StopRtpPubReq) (*lalproxy.StopRtpPubRes, error) {
	// todo: add your logic here and delete this line

	return &lalproxy.StopRtpPubRes{}, nil
}
