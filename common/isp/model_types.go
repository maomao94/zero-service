package isp

import "encoding/xml"

// DevicePointModel 为 ISP 设备点位模型（表 B.1）的一行 <Item>。
// 通过 xml struct tag 控制字段名与转义，由 WriteDeviceModel / WriteDeviceModelStream 输出。
type DevicePointModel struct {
	XMLName xml.Name `xml:"Item"`

	// StationName 变电站名称
	StationName string `xml:"station_name,attr" json:"station_name"`
	// StationCode 变电站编码，和中台设备编码保持一致
	StationCode string `xml:"station_code,attr" json:"station_code"`
	// AreaID 区域 ID，自定义的站内区域，可为空
	AreaID string `xml:"area_id,attr" json:"area_id"`
	// AreaName 区域名称
	AreaName string `xml:"area_name,attr" json:"area_name"`
	// BayID 间隔 ID，一次设备点位所在间隔，和中台设备编码保持一致
	BayID string `xml:"bay_id,attr" json:"bay_id"`
	// BayName 间隔名称
	BayName string `xml:"bay_name,attr" json:"bay_name"`
	// MainDeviceID 主设备 ID，点位所属主设备，和中台设备编码保持一致
	MainDeviceID string `xml:"main_device_id,attr" json:"main_device_id"`
	// MainDeviceName 主设备名称
	MainDeviceName string `xml:"main_device_name,attr" json:"main_device_name"`
	// ComponentID 部件 ID，一次设备点位对应所属部件或部位；对应部件时和中台设备编码保持一致，可为空
	ComponentID string `xml:"component_id,attr" json:"component_id"`
	// ComponentName 部件名称
	ComponentName string `xml:"component_name,attr" json:"component_name"`
	// DeviceID 设备点位 ID
	DeviceID string `xml:"device_id,attr" json:"device_id"`
	// DeviceName 设备点位名称
	DeviceName string `xml:"device_name,attr" json:"device_name"`
	// DeviceType 主设备类型
	DeviceType string `xml:"device_type,attr" json:"device_type"`
	// MeterType 表计类型
	MeterType string `xml:"meter_type,attr" json:"meter_type"`
	// AppearanceType 辅助设施类型
	AppearanceType string `xml:"appearance_type,attr" json:"appearance_type"`
	// SaveTypeList 采集/保存文件类型列表，多个类型用逗号分隔
	SaveTypeList string `xml:"save_type_list,attr" json:"save_type_list"`
	// RecognitionTypeList 识别类型列表，多个类型用逗号分隔
	RecognitionTypeList string `xml:"recognition_type_list,attr" json:"recognition_type_list"`
	// Phase 相位，多个相位用逗号分隔
	Phase string `xml:"phase,attr" json:"phase"`
	// DeviceInfo 备注信息，用于描述设备点位的文字信息；可能包含 JSON 扩展数据
	DeviceInfo string `xml:"device_info,attr" json:"device_info"`
	// DataType 设备点位支持的数据来源，按位定义
	DataType string `xml:"data_type,attr" json:"data_type"`
	// LowerValue 正常范围下限，选填
	LowerValue string `xml:"lower_value,attr" json:"lower_value"`
	// UpperValue 正常范围上限，选填
	UpperValue string `xml:"upper_value,attr" json:"upper_value"`
	// VideoPos 关联视频编码及预置位，JSON 格式
	VideoPos string `xml:"video_pos,attr" json:"video_pos"`
	// PointType 重要等级（1=I类, 2=II类, 3=III类）
	PointType string `xml:"point_type,attr" json:"point_type"`
	// LabelAttri 标签属性，点位标签属性，多个附加属性逗号分隔（1=人工关注）
	LabelAttri string `xml:"label_attri,attr" json:"label_attri"`
}

// PatrolDeviceModel 为 ISP 巡视装置模型的一行 <Item>。
// 通过 xml struct tag 控制字段名与转义，由 WritePatrolDeviceModel / WritePatrolDeviceModelStream 输出。
type PatrolDeviceModel struct {
	XMLName xml.Name `xml:"Item"`

	// PatrolDeviceName 巡视装置名称
	PatrolDeviceName string `xml:"patroldevice_name,attr"`
	// PatrolDeviceCode 巡视装置编码
	PatrolDeviceCode string `xml:"patroldevice_code,attr"`
	// StationName 变电站名称
	StationName string `xml:"station_name,attr"`
	// StationCode 变电站编码
	StationCode string `xml:"station_code,attr"`
	// DeviceModel 装置型号
	DeviceModel string `xml:"device_model,attr"`
	// Manufacturer 生产厂家
	Manufacturer string `xml:"manufacturer,attr"`
	// UseUnit 使用单位
	UseUnit string `xml:"use_unit,attr"`
	// DeviceSource 装置来源
	DeviceSource string `xml:"device_source,attr"`
	// ProductionDate 生产日期
	ProductionDate string `xml:"production_date,attr"`
	// ProductionCode 出厂编号
	ProductionCode string `xml:"production_code,attr"`
	// IsTransport 是否轮转，当装置类型为机器人时传递（0=不轮转, 1=轮转）
	IsTransport string `xml:"istransport,attr"`
	// UseMode 使用类型，当装置类型为摄像机时传递（10=枪机, 11=球机, 12=云台, 13=微型摄像机）
	UseMode string `xml:"use_mode,attr"`
	// VideoMode 视频类型（1=可见光, 2=红外, 3=可见光与红外）
	VideoMode string `xml:"video_mode,attr"`
	// Place 安装位置，摄像机的安装位置，其余种类的采集终端传空值
	Place string `xml:"place,attr"`
	// Type 装置类型（1=室外轮式机器人, 2=室内轮式机器人, 3=挂轨机器人, 10=摄像机, 11=硬盘录像机,
	// 12=智能分析主机, 13=无人机, 14=声纹, 15=无人机机巢, 20=区域巡视主机, 21=边缘节点装置）
	Type string `xml:"type,attr"`
	// PatrolDeviceInfo 备注信息，用于描述巡视装置的文字信息
	PatrolDeviceInfo string `xml:"patroldevice_info,attr"`
	// MountPatrolDeviceCode 所属挂载装置编码；当摄像机为机器人上挂载时填机器人编码，
	// 当摄像机为无人机上挂载时填无人机编码，当摄像机为无人机机巢上挂载时填无人机机巢编码，
	// 当摄像机独立使用时填硬盘录像机编码，其他情况为空
	MountPatrolDeviceCode string `xml:"mount_patroldevice_code,attr"`
}
