package model

type TerminalBind struct {
	// kafka tag
	DataTagV1 string `json:"dataTagV1"`
	// 绑定动作： BIND ｜ UNBIND
	Action string `json:"action"`
	// 终端ID（唯一标识）
	TerminalID int64 `json:"terminalId"`
	// 终端唯一编号（12位字符）
	TerminalNo string `json:"terminalNo"`
	// 员工身份证号
	StaffIdCardNo string `json:"staffIdCardNo"`
	// 跟踪对象ID（关联业务系统）
	TrackID int64 `json:"trackId"`
	// 对象编号（如车牌号"沪A12345"）
	TrackNo string `json:"trackNo"`
	// 对象类型：CAR-车辆, STAFF-人员
	TrackType string `json:"trackType"`
	// 监控对象显示名称（如车牌号"沪A12345"）
	TrackName string `json:"trackName"`
	// 操作时间，北京时间 eg: 2024-07-01 10:00:00
	ActionTime string `json:"actionTime"`
}

type EventData struct {
	// kafka tag
	DataTagV1 string `json:"dataTagV1"`
	// 事件ID
	ID string `json:"id"`
	// 事件名称
	EventTitle string `json:"eventTitle"`
	// 事件类型
	EventCode string `json:"eventCode"`
	// 事件时间（服务端）
	ServerTime int64 `json:"serverTime"`
	// 事件时间（终端）
	EpochTime int64 `json:"epochTime"`
	// 终端信息
	TerminalInfo TerminalInfo `json:"terminalInfo"`
	// 位置
	Position Position `json:"position"`
}

type TerminalData struct {
	// kafka tag
	DataTagV1 string `json:"dataTagV1"`
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

type AlarmData struct {
	// kafka tag
	DataTagV1 string `json:"dataTagV1"`
	// 报警唯一标识
	ID string `json:"id"`
	// 报警自定义名称（最大长度50字符）
	Name string `json:"name"`
	// 报警编号（格式：ALARM-日期-序号）
	AlarmNo string `json:"alarmNo"`
	// 报警类型编码（见AlarmType枚举）
	AlarmCode string `json:"alarmCode"`
	// 报警等级：1-紧急 2-严重 3-警告
	Level int32 `json:"level"`
	// 关联终端编号列表（至少包含一个有效终端号）
	TerminalNoList []string `json:"terminalNoList"`
	// 报警涉及的主体信息列表
	TrackInfoList []TerminalInfo `json:"trackInfoList"`
	// 监控对象类型：CAR-车辆 STAFF-人员
	TrackType string `json:"trackType"`
	// 报警触发位置（WGS84坐标系）
	Position LocationPosition `json:"position"`
	// 报警开始围栏code列表
	StartFences []FenceInfo `json:"startFences"`
	// 报警结束围栏code列表
	EndFences []FenceInfo `json:"endFences"`
	// 报警开始时间（Unix时间戳，毫秒级）
	StartTime int64 `json:"startTime"`
	// 报警结束时间（Unix时间戳，毫秒级）
	EndTime int64 `json:"endTime"`
	// 报警持续时长（单位：秒），由结束时间-开始时间计算得出
	Duration int `json:"duration"`
	// 报警当前状态：ON-进行中 OFF-已结束
	AlarmStatus string `json:"alarmStatus"`
}

// FenceInfo 围栏信息
type FenceInfo struct {
	FenceCode string `json:"fenceCode"` // 围栏code
	OrgCode   string `json:"orgCode"`
}

// LocationPosition 位置坐标
type LocationPosition struct {
	// 纬度（-90~90）
	Lat float64 `json:"lat"`
	// 经度（-180~180）
	Lon float64 `json:"lon"`
	// 海拔高度（米）
	Alt float64 `json:"alt"`
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
	// 监控对象显示名称（如车牌号"沪A12345"）
	TrackName string `json:"trackName"`
	OrgCode   string `json:"orgCode"`
	OrgName   string `json:"orgName"`
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

type BridgeMsgBody struct {
	TraceId  string `json:"traceId"`
	Body     string `json:"body"`
	Time     string `json:"time"`
	FilePath string `json:"filePath"`
}
