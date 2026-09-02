package copierx

import (
	"errors"
	"strconv"
	"time"

	"github.com/jinzhu/copier"

	"zero-service/common"
	"zero-service/common/carbonx"
)

// TimeFormat 时间格式精度
type TimeFormat int

const (
	TimeFormatSecond TimeFormat = iota // "2006-01-02 15:04:05"
	TimeFormatMilli                    // "2006-01-02 15:04:05.000"
	TimeFormatMicro                    // "2006-01-02 15:04:05.000000"
)

// OptionWithFormat 返回指定时间格式精度的 copier.Option
func OptionWithFormat(format TimeFormat) copier.Option {
	var formatFn func(time.Time, ...string) string

	switch format {
	case TimeFormatMilli:
		formatFn = carbonx.FormatDateTimeMilli
	case TimeFormatSecond:
		formatFn = carbonx.FormatDateTime
	default:
		formatFn = carbonx.FormatDateTimeMicro
	}

	return copier.Option{
		IgnoreEmpty: true,
		DeepCopy:    true,
		Converters: []copier.TypeConverter{
			{
				SrcType: time.Time{},
				DstType: copier.String,
				Fn: func(src any) (any, error) {
					s, ok := src.(time.Time)

					if !ok {
						return nil, errors.New("src type not matching")
					}

					return formatFn(s), nil
				},
			},
			{
				SrcType: copier.String,
				DstType: copier.Int,
				Fn: func(src any) (any, error) {
					s, ok := src.(string)

					if !ok {
						return nil, errors.New("src type not matching")
					}

					return strconv.Atoi(s)
				},
			},
			{
				SrcType: time.Time{},
				DstType: common.DateTime{},
				Fn: func(src any) (any, error) {
					s, ok := src.(time.Time)

					if !ok {
						return nil, errors.New("src type not matching")
					}

					return common.DateTime(s), nil
				},
			},
		},
	}
}

// Option 默认微秒精度的 copier.Option（向后兼容）
var Option = OptionWithFormat(TimeFormatMicro)

// OptionSecond 秒精度的 copier.Option
var OptionSecond = OptionWithFormat(TimeFormatSecond)

// OptionMilli 毫秒精度的 copier.Option
var OptionMilli = OptionWithFormat(TimeFormatMilli)

// OptionMicro 微秒精度的 copier.Option（显式命名，等同于 Option）
var OptionMicro = OptionWithFormat(TimeFormatMicro)
