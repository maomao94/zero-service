package logic

import (
	"context"
	"encoding/json"
	"time"
	"zero-service/model"

	"zero-service/app/xfusionmock/internal/svc"
	"zero-service/app/xfusionmock/xfusionmock"

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
	data := &model.TerminalData{
		DataTagV1: l.svcCtx.Config.Name,
		TerminalInfo: &model.TerminalInfo{
			TerminalID: 100001,
			TerminalNo: randomTerminal(),
			TrackID:    5001,
			TrackNo:    "沪A12345",
			TrackType:  "CAR",
			TrackName:  l.svcCtx.Config.Name,
		},
		EpochTime: time.Now().UnixMilli(),
		Location: &model.Location{
			Position: &model.Position{
				Lat: 31.2304,
				Lon: 121.4737,
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
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	l.svcCtx.KafkaPointPusher.Push(l.ctx, string(jsonData))
	return &xfusionmock.ResPushPoint{}, nil
}
