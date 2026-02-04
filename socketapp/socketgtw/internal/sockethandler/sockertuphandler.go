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

func (l *SocketUpHandler) Handle(ctx context.Context, event string, payload *socketio.EventPayload) error {
	data, err := convertor.ToBytes(payload.Data[0])
	if err != nil {
		return err
	}
	var upReq socketiox.SocketUpReq
	err = jsonx.Unmarshal(data, &upReq)
	if err != nil {
		return err
	}
	_, err = l.cli.UpSocketMessage(ctx, &streamevent.UpSocketMessageReq{
		ReqId:   upReq.ReqId,
		SId:     payload.SID,
		Event:   payload.Name,
		Payload: upReq.Payload,
	})
	if err != nil {
		return err
	}
	return nil
}
