package logic

import (
	"context"
	"encoding/json"

	"zero-service/socketapp/socketgtw/internal/svc"
	"zero-service/socketapp/socketgtw/socketgtw"

	"github.com/zeromicro/go-zero/core/jsonx"
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
		var payload any
		raw := []byte(in.Payload)
		var js json.RawMessage
		if jsonx.Unmarshal(raw, &js) == nil {
			payload = json.RawMessage(raw)
		} else {
			payload = in.Payload
		}
		err := session.EmitDown(in.Event, payload, in.ReqId)
		if err != nil {
			return nil, err
		}
	}
	return &socketgtw.SendToSessionRes{
		ReqId: in.ReqId,
	}, nil
}
