package handler

import (
	"context"

	"zero-service/app/ispagent/internal/modelxml"
)

// ModelDataProvider 提供生成模型 XML 所需的点位数据和巡视装置数据。
// 后续可替换为数据库实现；当前默认实现返回示例数据用于验证通路。
type ModelDataProvider interface {
	// DevicePoints 返回指定变电站的所有设备点位。
	DevicePoints(ctx context.Context, stationCode string) ([]modelxml.DevicePointModel, error)
	// PatrolDevices 返回指定变电站的所有巡视装置。
	PatrolDevices(ctx context.Context, stationCode string) ([]modelxml.PatrolDeviceModel, error)
}

// DefaultModelDataProvider 返回示例数据，用于未接入 DB 前的通路验证。
type DefaultModelDataProvider struct{}

func (d DefaultModelDataProvider) DevicePoints(_ context.Context, stationCode string) ([]modelxml.DevicePointModel, error) {
	return []modelxml.DevicePointModel{
		{
			StationName: "500kV变电站", StationCode: stationCode,
			AreaID: "1000", AreaName: "110kVGIS",
			BayID: "1001", BayName: "110kV热水II线",
			MainDeviceID: "1004", MainDeviceName: "110kV热水II线气室SF6压力表1",
			ComponentID: "1005", ComponentName: "表计读取",
			DeviceID: "1000001", DeviceName: "110kV热水II线气室SF6压力表1表计读取",
			RecognitionTypeList: "1",
			DeviceInfo:          `{"building":"building5","floor":"L1","patrolDeviceCode":"Q1_P484","point_type":"taskpoint","x":1468,"y":289,"z":0}`,
			DataType:            "2",
		},
		{
			StationName: "500kV变电站", StationCode: stationCode,
			AreaID: "1000", AreaName: "110kVGIS",
			BayID: "1001", BayName: "110kV热水II线",
			MainDeviceID: "1006", MainDeviceName: "110kV热水II线DS22刀闸分合",
			ComponentID: "1007", ComponentName: "分合指示",
			DeviceID: "1000002", DeviceName: "110kV热水II线DS22刀闸分合分合指示",
			RecognitionTypeList: "2",
			DeviceInfo:          `{"building":"building5","floor":"L1","patrolDeviceCode":"Q1_P484","point_type":"taskpoint","x":1468,"y":289,"z":0}`,
			DataType:            "2",
		},
	}, nil
}

func (d DefaultModelDataProvider) PatrolDevices(_ context.Context, stationCode string) ([]modelxml.PatrolDeviceModel, error) {
	return []modelxml.PatrolDeviceModel{
		{
			PatrolDeviceName: "xgrobot", PatrolDeviceCode: "Q1_P484",
			StationName: "500kV变电站", StationCode: stationCode,
			Manufacturer: "联想",
			Type:         "1",
		},
	}, nil
}
