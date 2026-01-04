package logic

import (
	"context"

	"zero-service/socketapp/socketgtw/internal/svc"
	"zero-service/socketapp/socketgtw/socketgtw"

	"github.com/zeromicro/go-zero/core/logx"
)

type SendToSessionsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewSendToSessionsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SendToSessionsLogic {
	return &SendToSessionsLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 向指定 session 批量发送消息
func (l *SendToSessionsLogic) SendToSessions(in *socketgtw.SendToSessionsReq) (*socketgtw.SendToSessionsRes, error) {
	if len(in.SIds) != 0 {
		for _, sId := range in.SIds {
			session := l.svcCtx.SocketServer.GetSession(sId)
			if session != nil {
				err := session.EmitDown(in.Event, in.Payload, in.ReqId)
				if err != nil {
					l.Errorf("SendToSessionsLogic.SendToSessions error: %v", err)
					continue
				}
			}
		}
	}
	return &socketgtw.SendToSessionsRes{
		ReqId: in.ReqId,
	}, nil
}
