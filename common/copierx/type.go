package copierx

import (
	"errors"
	"github.com/dromara/carbon/v2"
	"github.com/jinzhu/copier"
	"strconv"
	"time"
	"zero-service/common"
)

var Option = copier.Option{
	IgnoreEmpty: true,
	DeepCopy:    true,
	Converters: []copier.TypeConverter{
		{
			SrcType: time.Time{},
			DstType: copier.String,
			Fn: func(src interface{}) (interface{}, error) {
				s, ok := src.(time.Time)

				if !ok {
					return nil, errors.New("src type not matching")
				}

				return carbon.CreateFromStdTime(s).Format(carbon.DateTimeMicroFormat), nil
			},
		},
		{
			SrcType: copier.String,
			DstType: copier.Int,
			Fn: func(src interface{}) (interface{}, error) {
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
			Fn: func(src interface{}) (interface{}, error) {
				s, ok := src.(time.Time)

				if !ok {
					return nil, errors.New("src type not matching")
				}

				return common.DateTime(s), nil
			},
		},
	},
}
