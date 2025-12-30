package logic

import (
	"context"

	"zero-service/socketapp/socketgtw/internal/svc"
	"zero-service/socketapp/socketgtw/socketgtw"

	"github.com/zeromicro/go-zero/core/logx"
)

type SendOneToSessionLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewSendOneToSessionLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SendOneToSessionLogic {
	return &SendOneToSessionLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 向指定 session 发送消息
func (l *SendOneToSessionLogic) SendOneToSession(in *socketgtw.SendOneToSessionReq) (*socketgtw.SendOneToSessionRes, error) {
	session := l.svcCtx.SocketServer.GetSession(in.SId)
	if session != nil {
		err := session.EmitString(in.Event, string(in.Payload))
		if err != nil {
			return nil, err
		}
	}
	return &socketgtw.SendOneToSessionRes{}, nil
}
