package gtw

import (
	"context"
	"zero-service/zerorpc/zerorpc"

	"zero-service/gtw/internal/svc"
	"zero-service/gtw/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type PingLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewPingLogic(ctx context.Context, svcCtx *svc.ServiceContext) *PingLogic {
	return &PingLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *PingLogic) Ping() (resp *types.PingReply, err error) {
	res, err := l.svcCtx.ZeroRpcCli.Ping(l.ctx, &zerorpc.Req{Ping: "gtw"})
	if err != nil {
		return nil, err
	}
	return &types.PingReply{Msg: res.Pong}, nil
}
