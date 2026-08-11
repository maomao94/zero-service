package logic

import (
	"encoding/json"
	"time"

	"zero-service/app/trigger/internal/cronjob"
	"zero-service/app/trigger/trigger"
	"zero-service/common/crontask"
	"zero-service/common/holiday"
	"zero-service/common/tool"
	"zero-service/third_party/extproto"

	"github.com/dromara/carbon/v2"
)

type cronJobTaskData struct {
	taskCode       string
	taskName       string
	taskType       string
	groupID        string
	description    string
	deptCode       string
	rule           *trigger.PlanRulePb
	startTime      string
	endTime        string
	excludeDates   []string
	priority       int32
	payload        string
	bizExtra       string
	lockTimeout    int64
	maxDelay       int64
	skipTimeFilter bool
	ext1           string
	ext2           string
	ext3           string
	ext4           string
	ext5           string
}

func buildCronJobTask(data cronJobTaskData) (*crontask.TaskConfig, error) {
	payload, err := optionalJSON(data.payload, "payload")
	if err != nil {
		return nil, err
	}
	bizExtra, err := optionalJSON(data.bizExtra, "extra")
	if err != nil {
		return nil, err
	}
	schedule, err := cronjob.CompileSchedule(
		data.rule,
		data.startTime,
		data.endTime,
		data.excludeDates,
		data.skipTimeFilter,
		tool.NowStartOfSecond().StdTime(),
	)
	if err != nil {
		return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_01_PARAM_INVALID, err, "Cron Job 规则无效")
	}
	extra, err := cronjob.MarshalExtra(&cronjob.CronJobExtra{
		DeptCode:     data.deptCode,
		Type:         data.taskType,
		GroupId:      data.groupID,
		Description:  data.description,
		StartTime:    data.startTime,
		EndTime:      data.endTime,
		Rule:         schedule.RuleJSON,
		ExcludeDates: append([]string(nil), data.excludeDates...),
		BizExtra:     bizExtra,
		Ext1:         data.ext1,
		Ext2:         data.ext2,
		Ext3:         data.ext3,
		Ext4:         data.ext4,
		Ext5:         data.ext5,
	})
	if err != nil {
		return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_01_PARAM_INVALID, err, "Cron Job 扩展字段无效")
	}
	return &crontask.TaskConfig{
		TaskCode:    data.taskCode,
		TaskName:    data.taskName,
		RRuleStr:    schedule.RRuleStr,
		Priority:    int(data.priority),
		LockTimeout: time.Duration(data.lockTimeout) * time.Millisecond,
		MaxDelay:    time.Duration(data.maxDelay) * time.Second,
		Payload:     payload,
		Extra:       extra,
		Status:      crontask.StatusEnabled,
		NextRun:     schedule.NextRun,
	}, nil
}

func optionalJSON(value, field string) (json.RawMessage, error) {
	if value == "" {
		return nil, nil
	}
	if !json.Valid([]byte(value)) {
		return nil, tool.NewErrorByPbCode(extproto.Code__1_01_PARAM_INVALID, field+" 必须是合法 JSON")
	}
	return json.RawMessage(value), nil
}

func parseHolidayDate(date string) (time.Time, error) {
	if date == "" {
		return carbon.Now().StdTime(), nil
	}
	c := carbon.ParseByFormat(date, carbon.DateFormat)
	if c.Error != nil || c.IsInvalid() {
		if c.Error != nil {
			return time.Time{}, tool.NewErrorByPbCodeWrap(extproto.Code__1_01_PARAM_INVALID, c.Error, "日期格式错误，应为 yyyy-MM-dd")
		}
		return time.Time{}, tool.NewErrorByPbCode(extproto.Code__1_01_PARAM_INVALID, "日期格式错误，应为 yyyy-MM-dd")
	}
	return c.StdTime(), nil
}

func validateHolidayDate(date string) error {
	if date == "" {
		return tool.NewErrorByPbCode(extproto.Code__1_01_PARAM_INVALID, "日期不能为空")
	}
	_, err := parseHolidayDate(date)
	return err
}

func toHolidayDayPb(info holiday.DayInfo) *trigger.HolidayDayPb {
	return &trigger.HolidayDayPb{
		Date:          info.Date,
		Name:          info.Name,
		Type:          string(info.Type),
		Kind:          toHolidayDayKindPb(info.Kind),
		Note:          info.Note,
		IsFestivalDay: info.IsFestivalDay,
		IsHoliday:     info.IsHoliday,
		IsWorkday:     info.IsWorkday,
	}
}

func toHolidayDayKindPb(kind holiday.DayKind) trigger.HolidayDayKindPb {
	switch kind {
	case holiday.DayKindStatutoryHoliday:
		return trigger.HolidayDayKindPb_HOLIDAY_DAY_KIND_STATUTORY_HOLIDAY
	case holiday.DayKindWeekend:
		return trigger.HolidayDayKindPb_HOLIDAY_DAY_KIND_WEEKEND
	case holiday.DayKindMakeupWorkday:
		return trigger.HolidayDayKindPb_HOLIDAY_DAY_KIND_MAKEUP_WORKDAY
	case holiday.DayKindNormalWorkday:
		return trigger.HolidayDayKindPb_HOLIDAY_DAY_KIND_NORMAL_WORKDAY
	default:
		return trigger.HolidayDayKindPb_HOLIDAY_DAY_KIND_UNSPECIFIED
	}
}

func toHolidayFestivalPb(info holiday.FestivalInfo) *trigger.HolidayFestivalPb {
	return &trigger.HolidayFestivalPb{
		Year:           int32(info.Year),
		Name:           info.Name,
		StartDate:      info.StartDate,
		EndDate:        info.EndDate,
		HolidayDays:    append([]string(nil), info.HolidayDays...),
		MakeupWorkdays: append([]string(nil), info.MakeupWorkdays...),
		FestivalDays:   append([]string(nil), info.FestivalDays...),
	}
}

func toHolidayYearSummaryPb(info holiday.YearSummaryInfo) *trigger.HolidayYearSummaryPb {
	return &trigger.HolidayYearSummaryPb{
		Year:           int32(info.Year),
		HolidayDays:    append([]string(nil), info.HolidayDays...),
		MakeupWorkdays: append([]string(nil), info.MakeupWorkdays...),
		FestivalDays:   append([]string(nil), info.FestivalDays...),
		Names:          append([]string(nil), info.Names...),
	}
}

func toHolidaySourcePb(item holiday.StoredEntry) *trigger.HolidaySourcePb {
	return &trigger.HolidaySourcePb{
		Date:          item.Date,
		Name:          item.Entry.Name,
		Type:          string(item.Entry.Type),
		Note:          item.Entry.Note,
		IsFestivalDay: item.Entry.IsFestivalDay,
		Enabled:       item.Enabled,
	}
}
