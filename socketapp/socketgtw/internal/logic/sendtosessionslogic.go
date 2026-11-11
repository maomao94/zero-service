package logic

import (
	"context"

	"zero-service/common/socketiox"
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
	sessions := make([]*socketiox.Session, 0, len(in.SocketIds))
	for _, socketId := range in.SocketIds {
		if session := l.svcCtx.SocketServer.GetSession(socketId); session != nil {
			sessions = append(sessions, session)
		}
	}
	if len(sessions) > 0 {
		payload := parseJsonPayload(in.Payload)
		if err := emitSessions(l.svcCtx.EmitRunner, l.Logger, sessions, in.Event, payload, in.ReqId); err != nil {
			return nil, err
		}
	}
	return &socketgtw.SendToSessionsRes{
		ReqId: in.ReqId,
	}, nil
}
