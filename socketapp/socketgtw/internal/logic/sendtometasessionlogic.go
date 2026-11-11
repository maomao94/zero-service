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
		payload := parseJsonPayload(in.Payload)
		if err := emitSessions(l.svcCtx.EmitRunner, l.Logger, sessions, in.Event, payload, in.ReqId); err != nil {
			return nil, err
		}
	}
	return &socketgtw.SendToMetaSessionRes{
		ReqId: in.ReqId,
	}, nil
}
