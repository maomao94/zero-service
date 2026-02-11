package logic

import (
	"context"
	"encoding/json"

	"zero-service/socketapp/socketgtw/internal/svc"
	"zero-service/socketapp/socketgtw/socketgtw"

	"github.com/zeromicro/go-zero/core/jsonx"
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
		var payload any
		raw := []byte(in.Payload)
		var js json.RawMessage
		if jsonx.Unmarshal(raw, &js) == nil {
			payload = json.RawMessage(raw)
		} else {
			payload = in.Payload
		}
		for _, session := range sessions {
			err := session.EmitDown(in.Event, payload, in.ReqId)
			if err != nil {
				l.Errorf("SendToMetaSession error: %v", err)
				continue
			}
		}
	}
	return &socketgtw.SendToMetaSessionRes{
		ReqId: in.ReqId,
	}, nil
}
