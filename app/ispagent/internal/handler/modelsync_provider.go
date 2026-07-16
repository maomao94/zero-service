package handler

import (
	"context"
	"encoding/xml"
	"os"
	"path/filepath"

	"zero-service/common/isp"

	"github.com/zeromicro/go-zero/core/logx"
)

// ModelDataProvider 提供生成模型 XML 所需的点位数据和巡视装置数据。
type ModelDataProvider interface {
	DevicePoints(ctx context.Context, stationCode string) ([]isp.DevicePointModel, error)
	PatrolDevices(ctx context.Context, stationCode string) ([]isp.PatrolDeviceModel, error)
}

type DefaultModelDataProvider struct{}

type deviceModelXML struct {
	Items []isp.DevicePointModel `xml:"Item"`
}

type patrolDeviceModelXML struct {
	Items []isp.PatrolDeviceModel `xml:"Item"`
}

func (d DefaultModelDataProvider) DevicePoints(_ context.Context, stationCode string) ([]isp.DevicePointModel, error) {
	if err := validateSafePathComponent(stationCode); err != nil {
		return nil, err
	}
	path := filepath.Join("local", stationCode, "device_model.xml")
	data, err := os.ReadFile(path)
	if err != nil {
		logx.Errorf("[ispagent] device_model.xml not found for %s: %v", stationCode, err)
		return nil, err
	}
	var model deviceModelXML
	if err := xml.Unmarshal(data, &model); err != nil {
		return nil, err
	}
	return model.Items, nil
}

func (d DefaultModelDataProvider) PatrolDevices(_ context.Context, stationCode string) ([]isp.PatrolDeviceModel, error) {
	if err := validateSafePathComponent(stationCode); err != nil {
		return nil, err
	}
	path := filepath.Join("local", stationCode, "patrol_device_model.xml")
	data, err := os.ReadFile(path)
	if err != nil {
		logx.Errorf("[ispagent] patrol_device_model.xml not found for %s: %v", stationCode, err)
		return nil, err
	}
	var model patrolDeviceModelXML
	if err := xml.Unmarshal(data, &model); err != nil {
		return nil, err
	}
	return model.Items, nil
}
