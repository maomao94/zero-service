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

var (
	alarmCodes = []string{
		"CROSS_IN",
		"CROSS_OUT",
		"CLUSTER",
		"CROWDED",
		"LACKED",
		"LOW_BATTERY",
		"OVER_SPEED",
		"RETENTION",
		"SOS",
		"STATIC",
		"STAY",
		"CRASH",
		"VEHICLE_ILLEGAL_MOVE",
	}
	alarmNameMap = map[string]string{
		"CROSS_IN":             "区域闯入报警",
		"CROSS_OUT":            "区域离开报警",
		"CLUSTER":              "人员聚集报警",
		"CROWDED":              "车辆超员报警",
		"LACKED":               "人员缺员报警",
		"LOW_BATTERY":          "设备低电量报警",
		"OVER_SPEED":           "车辆超速报警",
		"RETENTION":            "人员滞留报警",
		"SOS":                  "SOS紧急报警",
		"STATIC":               "设备静止报警",
		"STAY":                 "车辆停留报警",
		"CRASH":                "车辆碰撞报警",
		"VEHICLE_ILLEGAL_MOVE": "车辆设备位移报警",
	}
	alarmLevels = []int{1, 2, 3}

	userIds    = []string{"b88ca6b10d3f098f0c2cccab1ef7afa2", "92cf0f46966a2dc5432ba02048ca57fb", "9ae27e8b3218fee13f4c7c872d5e9a86"}
	fenceCodes = []string{"SZQapVxmaX", "cRLqgcjzfh", "iNNMLtLtHw"}
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
	var jsonData []byte
	var err error
	if in.PushMode {
		jsonData, err = json.Marshal(in.Data)
		if err != nil {
			return nil, err
		}
	} else {
		uuid, _ := random.UUIdV4()
		alarmCode := randomAlarmCode()
		data := model.AlarmData{
			DataTagV1:      l.svcCtx.Config.Name,
			ID:             uuid,
			Name:           getAlarmName(alarmCode),
			AlarmNo:        generateAlarmNo(),
			AlarmCode:      alarmCode,
			Level:          randomLevel(),
			TerminalNoList: []string{"600000000001"},
			TrackInfoList: []model.TerminalInfo{
				{
					TerminalID: 100001,
					TerminalNo: "600000000001",
					TrackID:    5001,
					TrackNo:    randomUserId(),
					TrackType:  "CAR",
					TrackName:  l.svcCtx.Config.Name,
					OrgCode:    "001013002",
					OrgName:    "orgName",
				},
			},
			Position: model.LocationPosition{
				Lat: 37.61774353704819,
				Lon: 100.41165033341075,
				Alt: 30.12,
			},
			StartFences: []model.FenceInfo{model.FenceInfo{FenceCode: randomFenceCode(), OrgCode: "001013002"}},
			EndFences:   []model.FenceInfo{model.FenceInfo{FenceCode: randomFenceCode(), OrgCode: "001013002"}},
			StartTime:   time.Now().Add(-30 * time.Minute).UnixMilli(),
			EndTime:     time.Now().UnixMilli(),
			Duration:    600,
			AlarmStatus: "ON",
		}
		jsonData, err = json.Marshal(data)
		if err != nil {
			return nil, err
		}
	}
	l.svcCtx.KafkaAlarmPusher.Push(l.ctx, string(jsonData))
	return &xfusionmock.ResPushAlarm{}, nil
}

// 生成报警编号的工具函数
func generateAlarmNo() string {
	// 1. 获取当前日期（格式：yyyymmdd）
	now := time.Now().Format("20060102")

	// 2. 生成流水号（线程安全）
	// 实际项目中可以用 Redis/数据库 等持久化计数
	var counter uint64
	seq := atomic.AddUint64(&counter, 1) // 原子操作避免并发冲突

	return fmt.Sprintf("ALARM-%s-%04d", now, seq)
}

func randomAlarmCode() string {
	return random.RandFromGivenSlice(alarmCodes)
}

func randomLevel() int32 {
	return int32(random.RandFromGivenSlice(alarmLevels))
}

func randomUserId() string {
	return string(random.RandFromGivenSlice(userIds))
}

func randomFenceCode() string {
	return string(random.RandFromGivenSlice(fenceCodes))
}

func getAlarmName(code string) string {
	if name, ok := alarmNameMap[code]; ok {
		return name
	}
	return "未知报警类型"
}
