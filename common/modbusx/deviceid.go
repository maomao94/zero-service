package modbusx

// Standard Modbus Device Identification object IDs
const (
	DeviceIDVendorName          byte = 0x00 // 厂商名称
	DeviceIDProductCode         byte = 0x01 // 产品代码
	DeviceIDMajorMinorRevision  byte = 0x02 // 版本号 (Major.Minor)
	DeviceIDVendorURL           byte = 0x03 // 厂商网址
	DeviceIDProductName         byte = 0x04 // 产品名称
	DeviceIDModelName           byte = 0x05 // 型号名称
	DeviceIDUserApplicationName byte = 0x06 // 用户应用名称
)

// deviceIDObjectNames 是标准对象 ID 到语义名称的映射。
// 保持包内私有：调用方只通过 DeviceIDObjectName 查询，避免误改全局映射。
var deviceIDObjectNames = map[byte]string{
	DeviceIDVendorName:          "VendorName",
	DeviceIDProductCode:         "ProductCode",
	DeviceIDMajorMinorRevision:  "MajorMinorRevision",
	DeviceIDVendorURL:           "VendorURL",
	DeviceIDProductName:         "ProductName",
	DeviceIDModelName:           "ModelName",
	DeviceIDUserApplicationName: "UserApplicationName",
}

// DeviceIDObjectName 返回对象 ID 对应的标准语义名称。
// ok 为 false 表示非标准对象（例如厂商私有对象），调用方应自行回退命名。
func DeviceIDObjectName(id byte) (name string, ok bool) {
	name, ok = deviceIDObjectNames[id]
	return name, ok
}
