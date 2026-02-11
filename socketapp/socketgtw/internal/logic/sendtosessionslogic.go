package logic

import (
	"context"
	"encoding/json"

	"zero-service/socketapp/socketgtw/internal/svc"
	"zero-service/socketapp/socketgtw/socketgtw"

	"github.com/zeromicro/go-zero/core/jsonx"
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
		var payload any
		raw := []byte(in.Payload)
		var js json.RawMessage
		if jsonx.Unmarshal(raw, &js) == nil {
			payload = json.RawMessage(raw)
		} else {
			payload = in.Payload
		}
		for _, sId := range in.SIds {
			session := l.svcCtx.SocketServer.GetSession(sId)
			if session != nil {
				err := session.EmitDown(in.Event, payload, in.ReqId)
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
