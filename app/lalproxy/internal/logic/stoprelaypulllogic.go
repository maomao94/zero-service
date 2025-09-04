package logic

import (
	"context"

	"zero-service/app/lalproxy/internal/svc"
	"zero-service/app/lalproxy/lalproxy"

	"github.com/zeromicro/go-zero/core/logx"
)

type StopRelayPullLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewStopRelayPullLogic(ctx context.Context, svcCtx *svc.ServiceContext) *StopRelayPullLogic {
	return &StopRelayPullLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 停止从远端拉流
func (l *StopRelayPullLogic) StopRelayPull(in *lalproxy.StopRelayPullReq) (*lalproxy.StopRelayPullRes, error) {
	// todo: add your logic here and delete this line

	return &lalproxy.StopRelayPullRes{}, nil
}
