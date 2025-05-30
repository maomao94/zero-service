package logic

import (
	"context"
	"encoding/json"
	"time"
	"zero-service/app/xfusionmock/internal/svc"
	"zero-service/app/xfusionmock/xfusionmock"
	"zero-service/model"

	"github.com/zeromicro/go-zero/core/logx"
)

type PushPointLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewPushPointLogic(ctx context.Context, svcCtx *svc.ServiceContext) *PushPointLogic {
	return &PushPointLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *PushPointLogic) PushPoint(in *xfusionmock.ReqPushPoint) (*xfusionmock.ResPushPoint, error) {
	l.Info("PushPoint")
	var jsonData []byte
	var err error
	if in.PushMode {
		jsonData, err = json.Marshal(in.Data)
		if err != nil {
			return nil, err
		}
	} else {
		data := &model.TerminalData{
			DataTagV1: l.svcCtx.Config.Name,
			TerminalInfo: &model.TerminalInfo{
				TerminalID: 100001,
				TerminalNo: randomTerminal(),
				TrackID:    5001,
				TrackNo:    "b88ca6b10d3f098f0c2cccab1ef7afa2",
				TrackType:  "STAFF",
				TrackName:  l.svcCtx.Config.Name,
			},
			EpochTime: time.Now().UnixMilli(),
			Location: &model.Location{
				Position: &model.Position{
					Lat: 37.61774353704819,
					Lon: 100.41165033341075,
					Alt: 15.5,
				},
				Speed:        55.1234,
				Direction:    182.5,
				LocationMode: "GNSS",
				SatelliteNum: 8,
				GGAStatus:    4,
			},
			BuildingInfo: &model.BuildingInfo{
				BuildingID: 2001,
				FloorNo:    3,
			},
			Status: &model.Status{
				ACC:            true,
				Emergency:      false,
				MainSourceDown: false,
				Signal:         28,
				Battery:        85,
				MoveState:      1,
			},
		}
		jsonData, err = json.Marshal(data)
		if err != nil {
			return nil, err
		}
	}
	l.svcCtx.KafkaPointPusher.Push(l.ctx, string(jsonData))
	return &xfusionmock.ResPushPoint{}, nil
}
