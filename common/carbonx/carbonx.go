package carbonx

import (
	"github.com/dromara/carbon/v2"
)

func init() {
	// 设置Carbon的全局默认配置
	carbon.SetDefault(carbon.Default{
		Layout:       carbon.DateTimeLayout,
		Timezone:     carbon.Shanghai,
		Locale:       "zh-CN",
		WeekStartsAt: carbon.Monday,
		WeekendDays:  []carbon.Weekday{carbon.Saturday, carbon.Sunday},
	})
}
