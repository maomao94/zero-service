package sockethandler

import (
	"context"
	"zero-service/common/socketiox"
	"zero-service/facade/streamevent/streamevent"

	"github.com/doquangtan/socketio/v4"
	"github.com/duke-git/lancet/v2/convertor"
	"github.com/zeromicro/go-zero/core/jsonx"
)

type SocketUpHandler struct {
	cli streamevent.StreamEventClient
}

func NewSocketUpHandler(cli streamevent.StreamEventClient) *SocketUpHandler {
	return &SocketUpHandler{
		cli: cli,
	}
}

func (l *SocketUpHandler) Handle(ctx context.Context, event string, upPayload *socketio.EventPayload) (downPayload string, error error) {
	data, err := convertor.ToBytes(upPayload.Data[0])
	if err != nil {
		return "", err
	}
	var upReq socketiox.SocketUpReq
	err = jsonx.Unmarshal(data, &upReq)
	if err != nil {
		return "", err
	}
	jsonx.Marshal(upReq.Payload)
	res, err := l.cli.UpSocketMessage(ctx, &streamevent.UpSocketMessageReq{
		ReqId:   upReq.ReqId,
		SId:     upPayload.SID,
		Event:   event,
		Payload: string(data),
	})
	if err != nil {
		return "", err
	}
	return res.Payload, nil
}
