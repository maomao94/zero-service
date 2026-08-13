package carbonx

import (
	"database/sql"
	"time"

	"github.com/dromara/carbon/v2"
)

func init() {
	// 设置 Carbon 的全局默认配置。
	carbon.SetDefault(carbon.Default{
		Layout:       carbon.DateTimeLayout,
		Timezone:     carbon.Shanghai,
		Locale:       "zh-CN",
		WeekStartsAt: carbon.Monday,
		WeekendDays:  []carbon.Weekday{carbon.Saturday, carbon.Sunday},
	})
}

// NowStartOfSecond 返回清除秒以下精度的当前时间。
func NowStartOfSecond() *carbon.Carbon {
	return carbon.Now().StartOfSecond()
}

// NowDateTime 返回 Carbon 默认时区下的当前秒级日期时间文本。
func NowDateTime() string {
	return carbon.Now().ToDateTimeString()
}

// NowDateTimeMilli 返回 Carbon 默认时区下的当前毫秒级日期时间文本。
func NowDateTimeMilli() string {
	return carbon.Now().ToDateTimeMilliString()
}

// NowDateTimeMicro 返回 Carbon 默认时区下的当前微秒级日期时间文本。
func NowDateTimeMicro() string {
	return carbon.Now().ToDateTimeMicroString()
}

// FromTime 创建 Carbon 值；未指定 timezone 时保留 value 的时区，指定时转换同一时刻。
// timezone 无效时，错误记录在返回值的 Error 字段中。
func FromTime(value time.Time, timezone ...string) *carbon.Carbon {
	return carbon.CreateFromStdTime(value, timezone...)
}

// FromTimeStartOfSecond 创建 Carbon 值并清除秒以下精度，时区行为与 FromTime 一致。
func FromTimeStartOfSecond(value time.Time, timezone ...string) *carbon.Carbon {
	return FromTime(value, timezone...).StartOfSecond()
}

// FormatDateTime 以秒精度格式化 value；默认保留原时区，指定 timezone 时转换同一时刻。
// timezone 或 Carbon 输入无效时，返回 Carbon 定义的空字符串。
func FormatDateTime(value time.Time, timezone ...string) string {
	return FromTime(value, timezone...).ToDateTimeString()
}

// FormatDateTimeMilli 使用 Carbon 的毫秒日期时间格式输出 value。
// 默认保留原时区；timezone 或 Carbon 输入无效时返回 Carbon 定义的空字符串。
func FormatDateTimeMilli(value time.Time, timezone ...string) string {
	return FromTime(value, timezone...).ToDateTimeMilliString()
}

// FormatDateTimeMicro 使用 Carbon 的微秒日期时间格式输出 value。
// 默认保留原时区；timezone 或 Carbon 输入无效时返回 Carbon 定义的空字符串。
func FormatDateTimeMicro(value time.Time, timezone ...string) string {
	return FromTime(value, timezone...).ToDateTimeMicroString()
}

// FormatDateTimeMicroOrEmpty 以微秒精度格式化 value，Go 零时间返回空字符串。
func FormatDateTimeMicroOrEmpty(value time.Time, timezone ...string) string {
	if value.IsZero() {
		return ""
	}
	return FormatDateTimeMicro(value, timezone...)
}

// FormatDateTimeOrEmpty 以秒精度格式化 value，Go 零时间返回空字符串。
func FormatDateTimeOrEmpty(value time.Time, timezone ...string) string {
	if value.IsZero() {
		return ""
	}
	return FormatDateTime(value, timezone...)
}

// FormatNullDateTime 以秒精度格式化有效的 SQL 时间，仅以 Valid 判断是否为空。
// Valid 为 true 的 Go 零时间仍按有效值格式化。
func FormatNullDateTime(value sql.NullTime, timezone ...string) string {
	if !value.Valid {
		return ""
	}
	return FormatDateTime(value.Time, timezone...)
}

// ToNullTime 将 Go 零时间转换为无效 SQL 时间，非零时间保持原值并标记有效。
func ToNullTime(value time.Time) sql.NullTime {
	return sql.NullTime{Time: value, Valid: !value.IsZero()}
}

// NowUnix 返回当前 Unix 秒时间戳。
func NowUnix() int64 {
	return time.Now().Unix()
}

// NowUnixMilli 返回当前 Unix 毫秒时间戳。
func NowUnixMilli() int64 {
	return time.Now().UnixMilli()
}

// NowUnixMicro 返回当前 Unix 微秒时间戳。
func NowUnixMicro() int64 {
	return time.Now().UnixMicro()
}
