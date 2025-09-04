package logic

import (
	"context"

	"zero-service/app/lalproxy/internal/svc"
	"zero-service/app/lalproxy/lalproxy"

	"github.com/zeromicro/go-zero/core/logx"
)

type StartRtpPubLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewStartRtpPubLogic(ctx context.Context, svcCtx *svc.ServiceContext) *StartRtpPubLogic {
	return &StartRtpPubLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 打开GB28181接收端口
func (l *StartRtpPubLogic) StartRtpPub(in *lalproxy.StartRtpPubReq) (*lalproxy.StartRtpPubRes, error) {
	// todo: add your logic here and delete this line

	return &lalproxy.StartRtpPubRes{}, nil
}
