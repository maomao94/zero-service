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

type PushAlarmLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewPushAlarmLogic(ctx context.Context, svcCtx *svc.ServiceContext) *PushAlarmLogic {
	return &PushAlarmLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *PushAlarmLogic) PushAlarm(in *xfusionmock.ReqPushAlarm) (*xfusionmock.ResPushAlarm, error) {
	l.Info("PushAlarm")
	uuid, _ := random.UUIdV4()
	data := model.AlarmData{
		ID:             uuid,
		Name:           "区域闯入报警",
		AlarmNo:        "ALARM-20240530-0123",
		AlarmCode:      "CROSS_IN",
		Level:          1,
		TerminalNoList: []string{"T123456789013"},
		TrackInfoList: []model.TrackInfo{
			{
				TerminalInfo: &model.TerminalInfo{
					TerminalID: 1,
					TerminalNo: "123456789013",
					TrackID:    1,
					TrackNo:    "沪A999991",
					TrackType:  "CAR",
				},
				TrackName: "测试车辆001",
			},
		},
		Position: &model.LocationPosition{
			Lat: 31.31464578,
			Lon: 121.31891978,
			Alt: 30.12,
		},
		StartTime:   time.Now().Add(-10 * time.Minute).UnixMilli(),
		EndTime:     time.Now().UnixMilli(),
		Duration:    600,
		AlarmStatus: "OFF",
	}
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	l.svcCtx.KafkaAlarmPusher.Push(l.ctx, string(jsonData))
	return &xfusionmock.ResPushAlarm{}, nil
}
