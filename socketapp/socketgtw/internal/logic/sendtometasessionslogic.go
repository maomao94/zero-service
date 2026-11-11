package logic

import (
	"context"

	"zero-service/common/socketiox"
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
		var sessions []*socketiox.Session
		for _, metaSession := range in.MetaSessions {
			if matched, ok := l.svcCtx.SocketServer.GetSessionByKey(metaSession.Key, metaSession.Value); ok {
				sessions = append(sessions, matched...)
			}
		}
		if len(sessions) > 0 {
			payload := parseJsonPayload(in.Payload)
			if err := emitSessions(l.svcCtx.EmitRunner, l.Logger, sessions, in.Event, payload, in.ReqId); err != nil {
				return nil, err
			}
		}
	}
	return &socketgtw.SendToMetaSessionsRes{
		ReqId: in.ReqId,
	}, nil
}
