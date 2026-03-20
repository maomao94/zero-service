// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package ssegtw

import (
	"context"

	"zero-service/aiapp/ssegtw/internal/svc"
	"zero-service/aiapp/ssegtw/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type PingLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// ping
func NewPingLogic(ctx context.Context, svcCtx *svc.ServiceContext) *PingLogic {
	return &PingLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *PingLogic) Ping() (resp *types.PingReply, err error) {
	return &types.PingReply{Msg: "ssegtw pong"}, nil
}
