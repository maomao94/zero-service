package djisdk

import (
	"fmt"
	"strconv"
	"strings"
)

// DeviceDomain 表示 DJI 设备身份三元组的产品域。
type DeviceDomain int

const (
	// DeviceDomainAircraft 表示飞机类，官方 domain 值为 0。
	DeviceDomainAircraft DeviceDomain = iota
	// DeviceDomainPayload 表示负载类，官方 domain 值为 1。
	DeviceDomainPayload
	// DeviceDomainRemoteController 表示遥控器类，官方 domain 值为 2。
	DeviceDomainRemoteController
	// DeviceDomainDock 表示机场类，官方 domain 值为 3。
	DeviceDomainDock
)

// DeviceType 按官方 {domain}-{type}-{sub_type} 三元组唯一标识 DJI 设备型号。
type DeviceType struct {
	Domain  DeviceDomain
	Type    int
	SubType int
}

// ParseDeviceType 严格解析官方 {domain}-{type}-{sub_type} 格式。
func ParseDeviceType(raw string) (DeviceType, error) {
	parts := strings.Split(raw, "-")
	if len(parts) != 3 {
		return DeviceType{}, fmt.Errorf("invalid DJI device type %q: expected domain-type-sub_type", raw)
	}
	values := [3]int{}
	for i, part := range parts {
		value, err := strconv.Atoi(part)
		if err != nil || value < 0 {
			return DeviceType{}, fmt.Errorf("invalid DJI device type %q: tuple values must be non-negative integers", raw)
		}
		values[i] = value
	}
	if values[0] < int(DeviceDomainAircraft) || values[0] > int(DeviceDomainDock) {
		return DeviceType{}, fmt.Errorf("invalid DJI device type %q: unsupported domain %d", raw, values[0])
	}
	return DeviceType{Domain: DeviceDomain(values[0]), Type: values[1], SubType: values[2]}, nil
}

// String 返回官方 {domain}-{type}-{sub_type} 格式。
func (d DeviceType) String() string {
	return fmt.Sprintf("%d-%d-%d", d.Domain, d.Type, d.SubType)
}

// DeviceTypeDefinition 表示官方设备三元组及其产品展示名称。
type DeviceTypeDefinition struct {
	DeviceType DeviceType
	Name       string
}

var deviceTypeRegistry = map[DeviceType]DeviceTypeDefinition{
	// 飞机类。
	{Domain: DeviceDomainAircraft, Type: 103, SubType: 0}: {DeviceType: DeviceType{Domain: DeviceDomainAircraft, Type: 103, SubType: 0}, Name: "Matrice 400"},
	{Domain: DeviceDomainAircraft, Type: 89, SubType: 0}:  {DeviceType: DeviceType{Domain: DeviceDomainAircraft, Type: 89, SubType: 0}, Name: "Matrice 350 RTK"},
	{Domain: DeviceDomainAircraft, Type: 60, SubType: 0}:  {DeviceType: DeviceType{Domain: DeviceDomainAircraft, Type: 60, SubType: 0}, Name: "Matrice 300 RTK"},
	{Domain: DeviceDomainAircraft, Type: 67, SubType: 0}:  {DeviceType: DeviceType{Domain: DeviceDomainAircraft, Type: 67, SubType: 0}, Name: "Matrice 30"},
	{Domain: DeviceDomainAircraft, Type: 67, SubType: 1}:  {DeviceType: DeviceType{Domain: DeviceDomainAircraft, Type: 67, SubType: 1}, Name: "Matrice 30T"},
	{Domain: DeviceDomainAircraft, Type: 77, SubType: 0}:  {DeviceType: DeviceType{Domain: DeviceDomainAircraft, Type: 77, SubType: 0}, Name: "Mavic 3 行业系列（M3E 相机）"},
	{Domain: DeviceDomainAircraft, Type: 77, SubType: 1}:  {DeviceType: DeviceType{Domain: DeviceDomainAircraft, Type: 77, SubType: 1}, Name: "Mavic 3 行业系列（M3T 相机）"},
	{Domain: DeviceDomainAircraft, Type: 77, SubType: 3}:  {DeviceType: DeviceType{Domain: DeviceDomainAircraft, Type: 77, SubType: 3}, Name: "Mavic 3 行业系列（M3TA 相机）"},
	{Domain: DeviceDomainAircraft, Type: 91, SubType: 0}:  {DeviceType: DeviceType{Domain: DeviceDomainAircraft, Type: 91, SubType: 0}, Name: "Matrice 3D"},
	{Domain: DeviceDomainAircraft, Type: 91, SubType: 1}:  {DeviceType: DeviceType{Domain: DeviceDomainAircraft, Type: 91, SubType: 1}, Name: "Matrice 3TD"},
	{Domain: DeviceDomainAircraft, Type: 100, SubType: 0}: {DeviceType: DeviceType{Domain: DeviceDomainAircraft, Type: 100, SubType: 0}, Name: "Matrice 4D"},
	{Domain: DeviceDomainAircraft, Type: 100, SubType: 1}: {DeviceType: DeviceType{Domain: DeviceDomainAircraft, Type: 100, SubType: 1}, Name: "Matrice 4TD"},
	{Domain: DeviceDomainAircraft, Type: 99, SubType: 0}:  {DeviceType: DeviceType{Domain: DeviceDomainAircraft, Type: 99, SubType: 0}, Name: "DJI Matrice 4 系列（M4E 相机）"},
	{Domain: DeviceDomainAircraft, Type: 99, SubType: 1}:  {DeviceType: DeviceType{Domain: DeviceDomainAircraft, Type: 99, SubType: 1}, Name: "DJI Matrice 4 系列（M4T 相机）"},

	// 负载类。共用三元组名称不包含宿主飞机、机场代际或舱内外语境。
	{Domain: DeviceDomainPayload, Type: 39, SubType: 0}:  {DeviceType: DeviceType{Domain: DeviceDomainPayload, Type: 39, SubType: 0}, Name: "飞行器 FPV 相机"},
	{Domain: DeviceDomainPayload, Type: 176, SubType: 0}: {DeviceType: DeviceType{Domain: DeviceDomainPayload, Type: 176, SubType: 0}, Name: "飞行器辅助影像"},
	{Domain: DeviceDomainPayload, Type: 20, SubType: 0}:  {DeviceType: DeviceType{Domain: DeviceDomainPayload, Type: 20, SubType: 0}, Name: "禅思 Z30"},
	{Domain: DeviceDomainPayload, Type: 26, SubType: 0}:  {DeviceType: DeviceType{Domain: DeviceDomainPayload, Type: 26, SubType: 0}, Name: "禅思 XT2"},
	{Domain: DeviceDomainPayload, Type: 41, SubType: 0}:  {DeviceType: DeviceType{Domain: DeviceDomainPayload, Type: 41, SubType: 0}, Name: "禅思 XTS"},
	{Domain: DeviceDomainPayload, Type: 42, SubType: 0}:  {DeviceType: DeviceType{Domain: DeviceDomainPayload, Type: 42, SubType: 0}, Name: "禅思 H20"},
	{Domain: DeviceDomainPayload, Type: 43, SubType: 0}:  {DeviceType: DeviceType{Domain: DeviceDomainPayload, Type: 43, SubType: 0}, Name: "禅思 H20T"},
	{Domain: DeviceDomainPayload, Type: 61, SubType: 0}:  {DeviceType: DeviceType{Domain: DeviceDomainPayload, Type: 61, SubType: 0}, Name: "禅思 H20N"},
	{Domain: DeviceDomainPayload, Type: 82, SubType: 0}:  {DeviceType: DeviceType{Domain: DeviceDomainPayload, Type: 82, SubType: 0}, Name: "禅思 H30"},
	{Domain: DeviceDomainPayload, Type: 83, SubType: 0}:  {DeviceType: DeviceType{Domain: DeviceDomainPayload, Type: 83, SubType: 0}, Name: "禅思 H30T"},
	{Domain: DeviceDomainPayload, Type: 52, SubType: 0}:  {DeviceType: DeviceType{Domain: DeviceDomainPayload, Type: 52, SubType: 0}, Name: "Matrice 30 相机"},
	{Domain: DeviceDomainPayload, Type: 53, SubType: 0}:  {DeviceType: DeviceType{Domain: DeviceDomainPayload, Type: 53, SubType: 0}, Name: "Matrice 30T 相机"},
	{Domain: DeviceDomainPayload, Type: 88, SubType: 0}:  {DeviceType: DeviceType{Domain: DeviceDomainPayload, Type: 88, SubType: 0}, Name: "DJI Matrice 4E 相机"},
	{Domain: DeviceDomainPayload, Type: 89, SubType: 0}:  {DeviceType: DeviceType{Domain: DeviceDomainPayload, Type: 89, SubType: 0}, Name: "DJI Matrice 4T 相机"},
	{Domain: DeviceDomainPayload, Type: 66, SubType: 0}:  {DeviceType: DeviceType{Domain: DeviceDomainPayload, Type: 66, SubType: 0}, Name: "Mavic 3E 相机"},
	{Domain: DeviceDomainPayload, Type: 67, SubType: 0}:  {DeviceType: DeviceType{Domain: DeviceDomainPayload, Type: 67, SubType: 0}, Name: "Mavic 3T 相机"},
	{Domain: DeviceDomainPayload, Type: 129, SubType: 0}: {DeviceType: DeviceType{Domain: DeviceDomainPayload, Type: 129, SubType: 0}, Name: "Mavic 3TA 相机"},
	{Domain: DeviceDomainPayload, Type: 80, SubType: 0}:  {DeviceType: DeviceType{Domain: DeviceDomainPayload, Type: 80, SubType: 0}, Name: "Matrice 3D 相机"},
	{Domain: DeviceDomainPayload, Type: 81, SubType: 0}:  {DeviceType: DeviceType{Domain: DeviceDomainPayload, Type: 81, SubType: 0}, Name: "Matrice 3TD 相机"},
	{Domain: DeviceDomainPayload, Type: 98, SubType: 0}:  {DeviceType: DeviceType{Domain: DeviceDomainPayload, Type: 98, SubType: 0}, Name: "Matrice 4D 相机"},
	{Domain: DeviceDomainPayload, Type: 99, SubType: 0}:  {DeviceType: DeviceType{Domain: DeviceDomainPayload, Type: 99, SubType: 0}, Name: "Matrice 4TD 相机"},
	{Domain: DeviceDomainPayload, Type: 165, SubType: 0}: {DeviceType: DeviceType{Domain: DeviceDomainPayload, Type: 165, SubType: 0}, Name: "大疆机场相机"},

	// 遥控器类。
	{Domain: DeviceDomainRemoteController, Type: 56, SubType: 0}:  {DeviceType: DeviceType{Domain: DeviceDomainRemoteController, Type: 56, SubType: 0}, Name: "DJI 带屏遥控器行业版"},
	{Domain: DeviceDomainRemoteController, Type: 119, SubType: 0}: {DeviceType: DeviceType{Domain: DeviceDomainRemoteController, Type: 119, SubType: 0}, Name: "DJI RC Plus"},
	{Domain: DeviceDomainRemoteController, Type: 174, SubType: 0}: {DeviceType: DeviceType{Domain: DeviceDomainRemoteController, Type: 174, SubType: 0}, Name: "DJI RC Plus 2"},
	{Domain: DeviceDomainRemoteController, Type: 144, SubType: 0}: {DeviceType: DeviceType{Domain: DeviceDomainRemoteController, Type: 144, SubType: 0}, Name: "DJI RC Pro 行业版"},

	// 机场类。
	{Domain: DeviceDomainDock, Type: 1, SubType: 0}: {DeviceType: DeviceType{Domain: DeviceDomainDock, Type: 1, SubType: 0}, Name: "大疆机场"},
	{Domain: DeviceDomainDock, Type: 2, SubType: 0}: {DeviceType: DeviceType{Domain: DeviceDomainDock, Type: 2, SubType: 0}, Name: "大疆机场 2"},
	{Domain: DeviceDomainDock, Type: 3, SubType: 0}: {DeviceType: DeviceType{Domain: DeviceDomainDock, Type: 3, SubType: 0}, Name: "大疆机场 3"},
}

// LookupDeviceType 按官方设备三元组查询产品定义。
func LookupDeviceType(deviceType DeviceType) (DeviceTypeDefinition, bool) {
	definition, ok := deviceTypeRegistry[deviceType]
	return definition, ok
}

// LookupDeviceTypeName 解析官方三元组字符串并查询产品展示名称。
func LookupDeviceTypeName(raw string) (string, bool) {
	deviceType, err := ParseDeviceType(raw)
	if err != nil {
		return "", false
	}
	definition, ok := LookupDeviceType(deviceType)
	return definition.Name, ok
}

// PayloadGimbalIndex 表示负载挂载位置，不属于设备身份三元组。
type PayloadGimbalIndex int

const (
	// PayloadGimbalMain 表示主云台，M300 RTK 上表示机身下方左云台。
	PayloadGimbalMain PayloadGimbalIndex = 0
	// PayloadGimbalLowerRight 表示 M300 RTK 机身下方右云台。
	PayloadGimbalLowerRight PayloadGimbalIndex = 1
	// PayloadGimbalUpper 表示 M300 RTK 机身上方云台。
	PayloadGimbalUpper PayloadGimbalIndex = 2
	// PayloadGimbalFPV 表示 FPV 相机位置。
	PayloadGimbalFPV PayloadGimbalIndex = 7
)

// PayloadGimbalPosition 是负载挂载位置的中文描述。
type PayloadGimbalPosition string

const (
	// PayloadPositionMain 描述非 M300 RTK 飞机的主云台。
	PayloadPositionMain PayloadGimbalPosition = "主云台"
	// PayloadPositionLowerLeft 描述 M300 RTK 机身下方左云台。
	PayloadPositionLowerLeft PayloadGimbalPosition = "机身下方左云台"
	// PayloadPositionLowerRight 描述 M300 RTK 机身下方右云台。
	PayloadPositionLowerRight PayloadGimbalPosition = "机身下方右云台"
	// PayloadPositionUpper 描述 M300 RTK 机身上方云台。
	PayloadPositionUpper PayloadGimbalPosition = "机身上方云台"
	// PayloadPositionFPV 描述索引 7 的官方语义。
	PayloadPositionFPV PayloadGimbalPosition = "FPV 相机"
)

// Position 根据宿主飞机型号返回官方 gimbalindex 对应的中文挂载位置。
func (index PayloadGimbalIndex) Position(hostAircraftType DeviceType) (PayloadGimbalPosition, bool) {
	if index == PayloadGimbalFPV {
		return PayloadPositionFPV, true
	}
	if hostAircraftType == (DeviceType{Domain: DeviceDomainAircraft, Type: 60, SubType: 0}) {
		switch index {
		case PayloadGimbalMain:
			return PayloadPositionLowerLeft, true
		case PayloadGimbalLowerRight:
			return PayloadPositionLowerRight, true
		case PayloadGimbalUpper:
			return PayloadPositionUpper, true
		default:
			return "", false
		}
	}
	if definition, ok := LookupDeviceType(hostAircraftType); !ok || definition.DeviceType.Domain != DeviceDomainAircraft {
		return "", false
	}
	if index == PayloadGimbalMain {
		return PayloadPositionMain, true
	}
	return "", false
}

// PayloadPlacement 将负载三元组、宿主飞机型号与独立的 gimbalindex 挂载位置关联。
type PayloadPlacement struct {
	DeviceType       DeviceType
	HostAircraftType DeviceType
	GimbalIndex      PayloadGimbalIndex
}

// DescribePayload 查询负载产品定义及其独立挂载位置。
func DescribePayload(placement PayloadPlacement) (DeviceTypeDefinition, PayloadGimbalPosition, bool) {
	if placement.DeviceType.Domain != DeviceDomainPayload {
		return DeviceTypeDefinition{}, "", false
	}
	definition, ok := LookupDeviceType(placement.DeviceType)
	if !ok {
		return DeviceTypeDefinition{}, "", false
	}
	position, ok := placement.GimbalIndex.Position(placement.HostAircraftType)
	return definition, position, ok
}
