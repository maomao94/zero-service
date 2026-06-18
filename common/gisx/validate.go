package gisx

import "fmt"

// validationError 坐标校验专用错误类型，便于上层按类型区分校验失败与系统异常。
type validationError struct {
	msg string
}

func (e *validationError) Error() string {
	return e.msg
}

// ValidateCoordinate 校验第 idx 个坐标点的经纬度是否在合法范围内。
// 纬度有效范围: [-90, 90]，经度有效范围: [-180, 180]。
func ValidateCoordinate(lat, lon float64, idx int) error {
	if lat < -90 || lat > 90 {
		return &validationError{fmt.Sprintf("第 %d 个 point 的纬度超出范围：lat=%.8f（有效范围 -90~90）", idx, lat)}
	}
	if lon < -180 || lon > 180 {
		return &validationError{fmt.Sprintf("第 %d 个 point 的经度超出范围：lon=%.8f（有效范围 -180~180）", idx, lon)}
	}
	return nil
}
