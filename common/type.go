package common

import (
	"encoding/json"
	"github.com/dromara/carbon/v2"
	"time"
)

const (
	ExpireTime      = 30 * 60
	PayType_Wxpay   = "wxpay"
	PayType_Alipay  = "alipay"
	TxnType_Consume = 1000
	TxnType_Refund  = 2000
)

// 定义交易结果的常量
const (
	ResultUnprocessed string = "U" // 未处理
	ResultProcessing  string = "P" // 交易处理中
	ResultFailed      string = "F" // 失败
	ResultTimedOut    string = "T" // 超时
	ResultClosed      string = "C" // 关闭
	ResultSuccessful  string = "S" // 成功
)

// DateTime 定义 time.Time 的别名
type DateTime time.Time

// 序列化为 yyyy-MM-dd HH:mm:ss
func (t DateTime) MarshalJSON() ([]byte, error) {
	ts := carbon.CreateFromStdTime(time.Time(t)).ToDateTimeMicroString()
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
