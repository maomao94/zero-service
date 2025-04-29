package types

import (
	"encoding/json"
	"errors"
	"github.com/golang-module/carbon/v2"
	"github.com/jinzhu/copier"
	"strconv"
	"time"
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

				return carbon.CreateFromStdTime(s).Format("Y-m-d H:i:s.u"), nil
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
			DstType: DateTime{},
			Fn: func(src interface{}) (interface{}, error) {
				s, ok := src.(time.Time)

				if !ok {
					return nil, errors.New("src type not matching")
				}

				return DateTime(s), nil
			},
		},
	},
}

type MsgBody struct {
	TypeId int `json:"typeId"`
	Body   any `json:"body"`
}

// DateTime 定义 time.Time 的别名
type DateTime time.Time

// 序列化为 yyyy-MM-dd HH:mm:ss
func (t DateTime) MarshalJSON() ([]byte, error) {
	ts := carbon.CreateFromStdTime(time.Time(t)).ToDateTimeString()
	return json.Marshal(ts) // 直接返回格式化后的字符串
}

func (t *DateTime) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	c := carbon.Parse(s)
	*t = DateTime(c.StdTime())
	return nil
}

type SinglePointInfo struct {
	Ioa   uint   `json:"ioa"`   // 信息对象地址
	Value bool   `json:"value"` // 状态值
	Qds   byte   `json:"qds"`
	Time  string `json:"time"`
}

type DoublePointInfo struct {
	Ioa   uint   `json:"ioa"`   // 信息对象地址
	Value bool   `json:"value"` // 状态值
	Qds   byte   `json:"qds"`
	Time  string `json:"time"`
}
