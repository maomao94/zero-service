package logic

import (
	"context"

	"zero-service/socketapp/socketgtw/internal/svc"
	"zero-service/socketapp/socketgtw/socketgtw"

	"github.com/zeromicro/go-zero/core/logx"
)

type SendToMetaSessionsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewSendToMetaSessionsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SendToMetaSessionsLogic {
	return &SendToMetaSessionsLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 向指定元数据 session 批量发送消息
func (l *SendToMetaSessionsLogic) SendToMetaSessions(in *socketgtw.SendToMetaSessionsReq) (*socketgtw.SendToMetaSessionsRes, error) {
	if len(in.MetaSessions) != 0 {
		for _, metaSession := range in.MetaSessions {
			sessions, ok := l.svcCtx.SocketServer.GetSessionByKey(metaSession.Key, metaSession.Value)
			if ok {
				for _, session := range sessions {
					err := session.EmitDown(in.Event, in.Payload, in.ReqId)
					if err != nil {
						l.Errorf("SendToMetaSession error: %v", err)
						continue
					}
				}
			}
		}
	}
	return &socketgtw.SendToMetaSessionsRes{
		ReqId: in.ReqId,
	}, nil
}
