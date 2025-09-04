package logic

import (
	"context"
	"fmt"
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

// 关闭GB28181 RTP接收端口（注：根据lalserver文档，当前需通过KickSession接口实现，本接口暂未开放）
func (l *StopRtpPubLogic) StopRtpPub(in *lalproxy.StopRtpPubReq) (*lalproxy.StopRtpPubRes, error) {
	return nil, fmt.Errorf("未开放")
}
