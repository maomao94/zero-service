package gtw

import (
	"context"
	"zero-service/admin/guns"
	"zero-service/gtw/internal/svc"
	"zero-service/gtw/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type PingJavaLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// pingJava
func NewPingJavaLogic(ctx context.Context, svcCtx *svc.ServiceContext) *PingJavaLogic {
	return &PingJavaLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *PingJavaLogic) PingJava() (resp *types.PingReply, err error) {
	res, err := l.svcCtx.AdminRpcCli.Ping(l.ctx, &guns.Req{Ping: "gtw"})
	if err != nil {
		return nil, err
	}
	return &types.PingReply{Msg: res.Pong}, nil
}
