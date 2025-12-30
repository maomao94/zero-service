package logic

import (
	"context"

	"zero-service/socketapp/socketgtw/internal/svc"
	"zero-service/socketapp/socketgtw/socketgtw"

	"github.com/zeromicro/go-zero/core/logx"
)

type SendToMetaSessionLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewSendToMetaSessionLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SendToMetaSessionLogic {
	return &SendToMetaSessionLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 向指定元数据session 发送消息
func (l *SendToMetaSessionLogic) SendToMetaSession(in *socketgtw.SendToMetaSessionReq) (*socketgtw.SendToMetaSessionRes, error) {
	sessions, ok := l.svcCtx.SocketServer.GetSessionByKey(in.Key, in.Value)
	if ok {
		for _, session := range sessions {
			session.Emit(in.Event, in.Payload)
		}
	}
	return &socketgtw.SendToMetaSessionRes{}, nil
}
