package logic

import (
	"context"
	"encoding/json"
	"time"

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
	data := &TerminalData{
		TerminalInfo: &TerminalInfo{
			TerminalID: 100001,
			TerminalNo: "T12345678901",
			TrackID:    5001,
			TrackNo:    "沪A12345",
			TrackType:  "CAR",
		},
		EpochTime: time.Now().UnixMilli(),
		Location: &Location{
			Position: &Position{
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
		BuildingInfo: &BuildingInfo{
			BuildingID: 2001,
			FloorNo:    3,
		},
		Status: &Status{
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
	l.Info("PushPoint")
	l.svcCtx.KafkaPointPusher.Push(l.ctx, string(jsonData))
	return &xfusionmock.ResPushPoint{}, nil
}

type TerminalData struct {
	// 终端信息
	TerminalInfo *TerminalInfo `json:"terminalInfo"`
	// 位置点上报时间（Unix时间戳，毫秒）
	EpochTime int64 `json:"epochTime"`
	// 定位信息
	Location *Location `json:"location"`
	// 建筑信息
	BuildingInfo *BuildingInfo `json:"buildingInfo"`
	// 设备状态
	Status *Status `json:"status"`
}

// TerminalInfo 终端详细信息
type TerminalInfo struct {
	// 终端ID（唯一标识）
	TerminalID int64 `json:"terminalId"`
	// 终端唯一编号（12位字符）
	TerminalNo string `json:"terminalNo"`
	// 跟踪对象ID（关联业务系统）
	TrackID int64 `json:"trackId"`
	// 对象编号（如车牌号"沪A12345"）
	TrackNo string `json:"trackNo"`
	// 对象类型：CAR-车辆, STAFF-人员
	TrackType string `json:"trackType"`
}

// Location 定位数据
type Location struct {
	// 经纬度坐标
	Position *Position `json:"position"`
	// 速度（千米/小时，保留4位小数）
	Speed float64 `json:"speed"`
	// 方向角度（0-360度，正北为0）
	Direction float64 `json:"direction"`
	// 定位模式（如GNSS、LBS等）
	LocationMode string `json:"locationMode"`
	// 卫星数量（GPS定位时有效）
	SatelliteNum int `json:"satelliteNum"`
	// GGA状态：1-单点定位，4-固定解
	GGAStatus int `json:"ggaStatus"`
}

// Position 经纬度坐标点
type Position struct {
	// 纬度（WGS84坐标系）
	Lat float64 `json:"lat"`
	// 经度（WGS84坐标系）
	Lon float64 `json:"lon"`
	// 海拔高度（米）
	Alt float64 `json:"alt"`
}

// BuildingInfo 建筑信息
type BuildingInfo struct {
	// 建筑ID（地理围栏标识）
	BuildingID int64 `json:"buildingId"`
	// 楼层编号（地下层用负数表示）
	FloorNo int `json:"floorNo"`
}

// Status 设备实时状态
type Status struct {
	// ACC点火状态：true-车辆启动
	ACC bool `json:"acc"`
	// 紧急报警状态：true-触发报警
	Emergency bool `json:"emergency"`
	// 主电源状态：true-电源断开
	MainSourceDown bool `json:"mainSourceDown"`
	// 信号强度（0-31，越大越好）
	Signal int `json:"signal"`
	// 剩余电量百分比（0-100）
	Battery int `json:"battery"`
	// 运动状态：0-静止，1-移动
	MoveState int `json:"moveState"`
}
