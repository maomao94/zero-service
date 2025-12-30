package logic

import (
	"context"

	"zero-service/socketapp/socketgtw/internal/svc"
	"zero-service/socketapp/socketgtw/socketgtw"

	"github.com/zeromicro/go-zero/core/logx"
)

type SendToSessionLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewSendToSessionLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SendToSessionLogic {
	return &SendToSessionLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 向指定 session 发送消息
func (l *SendToSessionLogic) SendToSession(in *socketgtw.SendToSessionReq) (*socketgtw.SendToSessionRes, error) {
	session := l.svcCtx.SocketServer.GetSession(in.SId)
	if session != nil {
		err := session.EmitDown(in.Event, in.Payload, in.ReqId)
		if err != nil {
			return nil, err
		}
	}
	return &socketgtw.SendToSessionRes{}, nil
}
