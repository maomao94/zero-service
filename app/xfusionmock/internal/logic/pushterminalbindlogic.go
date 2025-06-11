package logic

import (
	"context"
	"encoding/json"
	"github.com/dromara/carbon/v2"
	"zero-service/model"

	"zero-service/app/xfusionmock/internal/svc"
	"zero-service/app/xfusionmock/xfusionmock"

	"github.com/zeromicro/go-zero/core/logx"
)

type PushTerminalBindLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewPushTerminalBindLogic(ctx context.Context, svcCtx *svc.ServiceContext) *PushTerminalBindLogic {
	return &PushTerminalBindLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *PushTerminalBindLogic) PushTerminalBind(in *xfusionmock.ReqPushTerminalBind) (*xfusionmock.ResPushTerminalBind, error) {
	l.Info("PushTerminalBind")
	var jsonData []byte
	var err error
	if in.PushMode {
		jsonData, err = json.Marshal(in.Data)
		if err != nil {
			return nil, err
		}
	} else {
		terminalNo := randomTerminal(l.svcCtx.Config.TerminalList)
		trackNo, b := l.svcCtx.Config.TerminalBind[terminalNo]
		if b {
			data := model.TerminalBind{
				DataTagV1:     l.svcCtx.Config.Name,
				Action:        "BIND",
				TerminalID:    600000000001,
				TerminalNo:    terminalNo,
				StaffIdCardNo: "11011100011",
				TrackID:       5001,
				TrackNo:       trackNo,
				TrackType:     "STAFF",
				TrackName:     l.svcCtx.Config.Name,
				ActionTime:    carbon.Now().Format("Y-m-d H:i:s"),
			}
			jsonData, err = json.Marshal(data)
			if err != nil {
				return nil, err
			}
			l.svcCtx.KafkaTerminalBindPusher.Push(l.ctx, string(jsonData))
		}
	}
	return &xfusionmock.ResPushTerminalBind{}, nil
}
