package gisx

// validate.go — 坐标合法性校验
//
// 本文件提供经纬度合法范围校验。
// 坐标约定与全项目一致：{lon, lat} 顺序。

import "fmt"

// ValidationError 坐标校验专用错误类型，便于上层按类型区分校验失败与系统异常。
type ValidationError struct {
	Msg string
}

func (e *ValidationError) Error() string {
	return e.Msg
}

// ValidateCoordinate 校验第 idx 个坐标点的经纬度是否在合法范围内。
// 参数顺序：(lon, lat)，与 orb.Point {lon, lat} 一致。
// 经度有效范围: [-180, 180]，纬度有效范围: [-90, 90]。
func ValidateCoordinate(lon, lat float64, idx int) error {
	if lon < -180 || lon > 180 {
		return &ValidationError{fmt.Sprintf("第 %d 个 point 的经度超出范围：lon=%.8f（有效范围 -180~180）", idx, lon)}
	}
	if lat < -90 || lat > 90 {
		return &ValidationError{fmt.Sprintf("第 %d 个 point 的纬度超出范围：lat=%.8f（有效范围 -90~90）", idx, lat)}
	}
	return nil
}
