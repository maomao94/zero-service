package logic

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/duke-git/lancet/v2/random"
	"sync/atomic"
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
		DataTagV1:      l.svcCtx.Config.Name,
		ID:             uuid,
		Name:           "区域闯入报警",
		AlarmNo:        GenerateAlarmNo(),
		AlarmCode:      "CROSS_IN",
		Level:          1,
		TerminalNoList: []string{"T123456789013"},
		TrackInfoList: []model.TerminalInfo{
			{
				TerminalID: 100001,
				TerminalNo: "T12345678901",
				TrackID:    5001,
				TrackNo:    "沪A12345",
				TrackType:  "CAR",
				TrackName:  l.svcCtx.Config.Name,
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

// 生成报警编号的工具函数
func GenerateAlarmNo() string {
	// 1. 获取当前日期（格式：yyyymmdd）
	now := time.Now().Format("20060102")

	// 2. 生成流水号（线程安全）
	// 实际项目中可以用 Redis/数据库 等持久化计数
	var counter uint64
	seq := atomic.AddUint64(&counter, 1) // 原子操作避免并发冲突

	return fmt.Sprintf("ALARM-%s-%04d", now, seq)
}
