package logic

import (
	"context"
	"encoding/json"
	"github.com/duke-git/lancet/v2/random"
	"time"
	"zero-service/model"

	"zero-service/app/xfusionmock/internal/svc"
	"zero-service/app/xfusionmock/xfusionmock"

	"github.com/zeromicro/go-zero/core/logx"
)

type PushEventLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewPushEventLogic(ctx context.Context, svcCtx *svc.ServiceContext) *PushEventLogic {
	return &PushEventLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *PushEventLogic) PushEvent(in *xfusionmock.ReqPushEvent) (*xfusionmock.ResPushEvent, error) {
	l.Info("PushEvent")
	var jsonData []byte
	var err error
	if in.PushMode {
		jsonData, err = json.Marshal(in.Data)
		if err != nil {
			return nil, err
		}
	} else {
		uuid, _ := random.UUIdV4()
		data := model.EventData{
			DataTagV1:  l.svcCtx.Config.Name,
			ID:         uuid,
			EventTitle: "进⼊围栏",
			EventCode:  "IN_FENCE",
			ServerTime: time.Now().UnixMilli(),
			EpochTime:  time.Now().UnixMilli(),
			TerminalInfo: model.TerminalInfo{
				TerminalID: 100001,
				TerminalNo: randomTerminal(),
				TrackID:    5001,
				TrackNo:    "沪A12345",
				TrackType:  "CAR",
				TrackName:  l.svcCtx.Config.Name,
			},
			Position: model.Position{
				Lat: 37.61774353704819,
				Lon: 100.41165033341075,
				Alt: 15.5,
			},
		}
		jsonData, err = json.Marshal(data)
		if err != nil {
			return nil, err
		}
	}
	l.svcCtx.KafkaEventPusher.Push(l.ctx, string(jsonData))
	return &xfusionmock.ResPushEvent{}, nil
}
