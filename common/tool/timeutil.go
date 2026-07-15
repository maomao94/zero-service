package tool

import (
	"time"

	"github.com/dromara/carbon/v2"
)

// NowStartOfSecond returns the current time with sub-second precision cleared.
func NowStartOfSecond() *carbon.Carbon {
	return carbon.Now().StartOfSecond()
}

// CarbonFromTimeStartOfSecond converts a time.Time to carbon and clears sub-second precision.
func CarbonFromTimeStartOfSecond(t time.Time) *carbon.Carbon {
	return carbon.CreateFromStdTime(t).StartOfSecond()
}

// GenSecondTS returns the current Unix timestamp in seconds.
func GenSecondTS() int64 {
	return time.Now().Unix() // 示例：1734429580（对应2025-12-17 09:59:40）
}

// GenMilliTS returns the current Unix timestamp in milliseconds.
func GenMilliTS() int64 {
	return time.Now().UnixMilli() // 示例：1734429580020（对应2025-12-17 09:59:40.020）
}

// GenMicroTS returns the current Unix timestamp in microseconds.
func GenMicroTS() int64 {
	return time.Now().UnixMicro() // 示例：1734429580020123
}
