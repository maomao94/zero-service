package logic

import (
	"strconv"
	"time"

	"zero-service/app/ispagent/ispagent"
	"zero-service/common/isp"
)

func protoItems(items []*ispagent.Item) []isp.Item {
	out := make([]isp.Item, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		attrs := make(isp.Item, len(item.GetAttributes()))
		for k, v := range item.GetAttributes() {
			attrs[k] = v
		}
		out = append(out, attrs)
	}
	return out
}

func patrolDeviceCoordinatesToItems(items []*ispagent.PatrolDeviceCoordinate) []isp.Item {
	now := time.Now().Format("2006-01-02 15:04:05")
	out := make([]isp.Item, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		out = append(out, isp.Item{
			"patroldevice_name":    item.GetPatrolDeviceName(),
			"patroldevice_code":    item.GetPatrolDeviceCode(),
			"time":                 now,
			"coordinate_pixel":     item.GetCoordinatePixel(),
			"coordinate_geography": item.GetCoordinateGeography(),
			"task_patrolled_id":    item.GetTaskPatrolledId(),
		})
	}
	return out
}

func patrolDeviceRunDataToItems(items []*ispagent.PatrolDeviceRunData) []isp.Item {
	now := time.Now().Format("2006-01-02 15:04:05")
	out := make([]isp.Item, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		out = append(out, isp.Item{
			"patroldevice_name": item.GetPatrolDeviceName(),
			"patroldevice_code": item.GetPatrolDeviceCode(),
			"time":              now,
			"type":              strconv.Itoa(int(item.GetType())),
			"value":             item.GetValue(),
			"value_unit":        item.GetValueUnit(),
			"unit":              item.GetUnit(),
		})
	}
	return out
}

func patrolDeviceStatusDataToItems(items []*ispagent.PatrolDeviceStatusData) []isp.Item {
	now := time.Now().Format("2006-01-02 15:04:05")
	out := make([]isp.Item, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		out = append(out, isp.Item{
			"patroldevice_name": item.GetPatrolDeviceName(),
			"patroldevice_code": item.GetPatrolDeviceCode(),
			"time":              now,
			"type":              item.GetType(),
			"value":             item.GetValue(),
			"value_unit":        item.GetValueUnit(),
			"unit":              item.GetUnit(),
		})
	}
	return out
}

// commandResponse 将 ISP 响应转换为 gRPC CommandRes。
func commandResponse(msg *isp.Message) *ispagent.CommandRes {
	items := make([]*ispagent.Item, 0, len(msg.Items))
	for _, item := range msg.Items {
		attrs := make(map[string]string, len(item))
		for k, v := range item {
			attrs[k] = v
		}
		items = append(items, &ispagent.Item{Attributes: attrs})
	}
	success := true
	if msg.Type == isp.TypeSystem {
		success = msg.Code == "" || msg.Code == "0" || msg.Code == "200"
	}
	return &ispagent.CommandRes{
		Success: success,
		Code:    msg.Code,
		Items:   items,
		RawXml:  msg.RawXML,
	}
}
